package plist

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"runtime"
	"time"
	"unicode/utf16"
)

type bplistTrailer struct {
	Unused            [5]uint8
	SortVersion       uint8
	OffsetIntSize     uint8
	ObjectRefSize     uint8
	NumObjects        uint64
	TopObject         uint64
	OffsetTableOffset uint64
}

const (
	bpTagNull        uint8 = 0x00
	bpTagBoolFalse         = 0x08
	bpTagBoolTrue          = 0x09
	bpTagInteger           = 0x10
	bpTagReal              = 0x20
	bpTagDate              = 0x30
	bpTagData              = 0x40
	bpTagASCIIString       = 0x50
	bpTagUTF16String       = 0x60
	bpTagUID               = 0x80
	bpTagArray             = 0xA0
	bpTagDictionary        = 0xD0
)

type bplistGenerator struct {
	writer   *countedWriter
	uniqmap  map[interface{}]uint64
	objmap   map[*plistValue]uint64
	objtable []*plistValue
	nobjects uint64
	trailer  bplistTrailer
}

func (p *bplistGenerator) flattenPlistValue(pval *plistValue) {
	switch pval.kind {
	case String, Integer, Real:
		if _, ok := p.uniqmap[pval.value]; ok {
			return
		}
		p.uniqmap[pval.value] = p.nobjects
	case Date:
		k := pval.value.(time.Time).UnixNano()
		if _, ok := p.uniqmap[k]; ok {
			return
		}
		p.uniqmap[k] = p.nobjects
	case Data:
		// Data are uniqued by their checksums.
		// The wonderful difference between uint64 (which we use for numbers)
		// and uint32 makes this possible.
		// Todo: Look at calculating this only once and storing it somewhere;
		// crc32 is fairly quick, however.
		uniqkey := crc32.ChecksumIEEE(pval.value.([]byte))
		if _, ok := p.uniqmap[uniqkey]; ok {
			return
		}
		p.uniqmap[uniqkey] = p.nobjects
	}

	p.objtable = append(p.objtable, pval)
	p.objmap[pval] = p.nobjects
	p.nobjects++

	switch pval.kind {
	case Dictionary:
		dict := pval.value.(*dictionary)
		dict.populateArrays()
		for _, k := range dict.keys {
			p.flattenPlistValue(&plistValue{String, k})
		}
		for _, v := range dict.values {
			p.flattenPlistValue(v)
		}
	case Array:
		subvalues := pval.value.([]*plistValue)
		for _, v := range subvalues {
			p.flattenPlistValue(v)
		}
	}
}

func (p *bplistGenerator) indexForPlistValue(pval *plistValue) (uint64, bool) {
	var v uint64
	var ok bool
	switch pval.kind {
	case String, Integer, Real:
		v, ok = p.uniqmap[pval.value]
	case Date:
		v, ok = p.uniqmap[pval.value.(time.Time).UnixNano()]
	case Data:
		v, ok = p.uniqmap[crc32.ChecksumIEEE(pval.value.([]byte))]
	default:
		v, ok = p.objmap[pval]
	}
	return v, ok
}

func (p *bplistGenerator) generateDocument(rootpval *plistValue) {
	p.objtable = make([]*plistValue, 0, 15)
	p.uniqmap = make(map[interface{}]uint64)
	p.objmap = make(map[*plistValue]uint64)
	p.flattenPlistValue(rootpval)

	p.trailer.NumObjects = uint64(len(p.objtable))
	p.trailer.ObjectRefSize = uint8(minimumSizeForInt(p.trailer.NumObjects))

	p.writer.Write([]byte("bplist00"))

	offtable := make([]uint64, p.trailer.NumObjects)
	for i, pval := range p.objtable {
		offtable[i] = uint64(p.writer.BytesWritten())
		p.writePlistValue(pval)
	}

	p.trailer.OffsetIntSize = uint8(minimumSizeForInt(uint64(p.writer.BytesWritten())))
	p.trailer.TopObject = p.objmap[rootpval]
	p.trailer.OffsetTableOffset = uint64(p.writer.BytesWritten())

	for _, offset := range offtable {
		p.writeSizedInt(offset, int(p.trailer.OffsetIntSize))
	}

	binary.Write(p.writer, binary.BigEndian, p.trailer)
}

func (p *bplistGenerator) writePlistValue(pval *plistValue) {
	if pval == nil {
		return
	}

	switch pval.kind {
	case Dictionary:
		p.writeDictionaryTag(pval.value.(*dictionary))
	case Array:
		p.writeArrayTag(pval.value.([]*plistValue))
	case String:
		p.writeStringTag(pval.value.(string))
	case Integer:
		p.writeIntTag(pval.value.(signedInt).value)
	case Real:
		p.writeRealTag(pval.value.(sizedFloat).value, pval.value.(sizedFloat).bits)
	case Boolean:
		p.writeBoolTag(pval.value.(bool))
	case Data:
		p.writeDataTag(pval.value.([]byte))
	case Date:
		p.writeDateTag(pval.value.(time.Time))
	}
}

func minimumSizeForInt(n uint64) int {
	switch {
	case n <= uint64(0xff):
		return 1
	case n <= uint64(0xffff):
		return 2
	case n <= uint64(0xffffffff):
		return 4
	default:
		return 8
	}
	panic(errors.New("illegal integer size"))
}

func (p *bplistGenerator) writeSizedInt(n uint64, nbytes int) {
	var val interface{}
	switch nbytes {
	case 1:
		val = uint8(n)
	case 2:
		val = uint16(n)
	case 4:
		val = uint32(n)
	case 8:
		val = n
	default:
		panic(errors.New("illegal integer size"))
	}
	binary.Write(p.writer, binary.BigEndian, val)
}

func (p *bplistGenerator) writeBoolTag(v bool) {
	tag := uint8(bpTagBoolFalse)
	if v {
		tag = bpTagBoolTrue
	}
	binary.Write(p.writer, binary.BigEndian, tag)
}

func (p *bplistGenerator) writeIntTag(n uint64) {
	var tag uint8
	var val interface{}
	switch {
	case n <= uint64(0xff):
		val = uint8(n)
		tag = bpTagInteger | 0x0
	case n <= uint64(0xffff):
		val = uint16(n)
		tag = bpTagInteger | 0x1
	case n <= uint64(0xffffffff):
		val = uint32(n)
		tag = bpTagInteger | 0x2
	default:
		val = n
		tag = bpTagInteger | 0x3
	}

	binary.Write(p.writer, binary.BigEndian, tag)
	binary.Write(p.writer, binary.BigEndian, val)
}

func (p *bplistGenerator) writeRealTag(n float64, bits int) {
	var tag uint8 = bpTagReal | 0x3
	var val interface{} = n
	if bits == 32 {
		val = float32(n)
		tag = bpTagReal | 0x2
	}

	binary.Write(p.writer, binary.BigEndian, tag)
	binary.Write(p.writer, binary.BigEndian, val)
}

func (p *bplistGenerator) writeDateTag(t time.Time) {
	tag := uint8(bpTagDate) | 0x3
	val := float64(t.In(time.UTC).UnixNano()) / float64(time.Second)
	val -= 978307200 // Adjust to Apple Epoch

	binary.Write(p.writer, binary.BigEndian, tag)
	binary.Write(p.writer, binary.BigEndian, val)
}

func (p *bplistGenerator) writeCountedTag(tag uint8, count uint64) {
	marker := tag
	if count >= 0xF {
		marker |= 0xF
	} else {
		marker |= uint8(count)
	}

	binary.Write(p.writer, binary.BigEndian, marker)

	if count >= 0xF {
		p.writeIntTag(count)
	}
}

func (p *bplistGenerator) writeDataTag(data []byte) {
	p.writeCountedTag(bpTagData, uint64(len(data)))
	binary.Write(p.writer, binary.BigEndian, data)
}

func (p *bplistGenerator) writeStringTag(str string) {
	for _, r := range str {
		if r > 0xFF {
			utf16Runes := utf16.Encode([]rune(str))
			p.writeCountedTag(bpTagUTF16String, uint64(len(utf16Runes)))
			binary.Write(p.writer, binary.BigEndian, utf16Runes)
			return
		}
	}

	p.writeCountedTag(bpTagASCIIString, uint64(len(str)))
	binary.Write(p.writer, binary.BigEndian, []byte(str))
}

func (p *bplistGenerator) writeDictionaryTag(dict *dictionary) {
	p.writeCountedTag(bpTagDictionary, uint64(dict.count))
	vals := make([]uint64, dict.count*2)
	cnt := dict.count
	for i, k := range dict.keys {
		keyIdx, ok := p.uniqmap[k]
		if !ok {
			panic(errors.New("failed to find key " + k + " in object map during serialization"))
		}
		vals[i] = keyIdx
	}
	for i, v := range dict.values {
		objIdx, ok := p.indexForPlistValue(v)
		if !ok {
			panic(errors.New("failed to find value in object map during serialization"))
		}
		vals[i+cnt] = objIdx
	}

	for _, v := range vals {
		p.writeSizedInt(v, int(p.trailer.ObjectRefSize))
	}
}

func (p *bplistGenerator) writeArrayTag(arr []*plistValue) {
	p.writeCountedTag(bpTagArray, uint64(len(arr)))
	for _, v := range arr {
		objIdx, ok := p.indexForPlistValue(v)
		if !ok {
			panic(errors.New("failed to find value in object map during serialization"))
		}

		p.writeSizedInt(objIdx, int(p.trailer.ObjectRefSize))
	}
}

func (p *bplistGenerator) Indent(i string) {
	// There's nothing to indent.
}

func newBplistGenerator(w io.Writer) *bplistGenerator {
	return &bplistGenerator{
		writer: &countedWriter{Writer: mustWriter{w}},
	}
}

type bplistParser struct {
	reader        io.ReadSeeker
	version       int
	buf           []byte
	objrefs       map[uint64]*plistValue
	offtable      []uint64
	trailer       bplistTrailer
	trailerOffset int64
}

func (p *bplistParser) parseDocument() (pval *plistValue, parseError error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			if _, ok := r.(invalidPlistError); ok {
				parseError = r.(error)
			} else {
				// Wrap all non-invalid-plist errors.
				parseError = plistParseError{"binary", r.(error)}
			}
		}
	}()

	magic := make([]byte, 6)
	ver := make([]byte, 2)
	p.reader.Seek(0, 0)
	p.reader.Read(magic)
	if !bytes.Equal(magic, []byte("bplist")) {
		panic(invalidPlistError{"binary", errors.New("mismatched magic")})
	}

	_, err := p.reader.Read(ver)
	if err != nil {
		panic(err)
	}

	p.version = int(mustParseInt(string(ver), 10, 0))

	if p.version > 1 {
		panic(fmt.Errorf("unexpected version %d", p.version))
	}

	p.objrefs = make(map[uint64]*plistValue)
	p.trailerOffset, err = p.reader.Seek(-32, 2)
	if err != nil && err != io.EOF {
		panic(err)
	}

	err = binary.Read(p.reader, binary.BigEndian, &p.trailer)
	if err != nil && err != io.EOF {
		panic(err)
	}

	if p.trailer.NumObjects > uint64(math.Pow(2, 8*float64(p.trailer.ObjectRefSize))) {
		panic(fmt.Errorf("binary property list contains more objects (%v) than its object ref size (%v bytes) can support", p.trailer.NumObjects, p.trailer.ObjectRefSize))
	}

	if p.trailer.TopObject >= p.trailer.NumObjects {
		panic(fmt.Errorf("top object index %v is out of range (only %v objects exist)", p.trailer.TopObject, p.trailer.NumObjects))
	}
	p.offtable = make([]uint64, p.trailer.NumObjects)

	// SEEK_SET
	_, err = p.reader.Seek(int64(p.trailer.OffsetTableOffset), 0)
	if err != nil && err != io.EOF {
		panic(err)
	}

	for i := uint64(0); i < p.trailer.NumObjects; i++ {
		off := p.readSizedInt(int(p.trailer.OffsetIntSize))
		if off >= uint64(p.trailerOffset) {
			panic(fmt.Errorf("object %v starts beyond end of plist trailer (%v vs %v)", i, off, p.trailerOffset))
		}
		p.offtable[i] = off
	}

	for _, off := range p.offtable {
		p.valueAtOffset(off)
	}

	pval = p.valueAtOffset(p.offtable[p.trailer.TopObject])
	return
}

func (p *bplistParser) readSizedInt(nbytes int) uint64 {
	switch nbytes {
	case 1:
		var val uint8
		binary.Read(p.reader, binary.BigEndian, &val)
		return uint64(val)
	case 2:
		var val uint16
		binary.Read(p.reader, binary.BigEndian, &val)
		return uint64(val)
	case 4:
		var val uint32
		binary.Read(p.reader, binary.BigEndian, &val)
		return uint64(val)
	case 8:
		var val uint64
		binary.Read(p.reader, binary.BigEndian, &val)
		return uint64(val)
	case 16:
		var high, low uint64
		binary.Read(p.reader, binary.BigEndian, &high)
		binary.Read(p.reader, binary.BigEndian, &low)
		// TODO: int128 support (!)
		return uint64(low)
	}
	panic(errors.New("illegal integer size"))
}

func (p *bplistParser) countForTag(tag uint8) uint64 {
	cnt := uint64(tag & 0x0F)
	if cnt == 0xF {
		var intTag uint8
		binary.Read(p.reader, binary.BigEndian, &intTag)
		cnt = p.readSizedInt(1 << (intTag & 0xF))
	}
	return cnt
}

func (p *bplistParser) valueAtOffset(off uint64) *plistValue {
	if pval, ok := p.objrefs[off]; ok {
		return pval
	}
	pval := p.parseTagAtOffset(int64(off))
	p.objrefs[off] = pval
	return pval
}

func (p *bplistParser) parseTagAtOffset(off int64) *plistValue {
	var tag uint8
	_, err := p.reader.Seek(off, 0)
	if err != nil {
		panic(err)
	}
	err = binary.Read(p.reader, binary.BigEndian, &tag)
	if err != nil {
		panic(err)
	}

	switch tag & 0xF0 {
	case bpTagNull:
		switch tag & 0x0F {
		case bpTagBoolTrue, bpTagBoolFalse:
			return &plistValue{Boolean, tag == bpTagBoolTrue}
		}
		return nil
	case bpTagInteger:
		val := p.readSizedInt(1 << (tag & 0xF))
		return &plistValue{Integer, signedInt{val, false}}
	case bpTagReal:
		nbytes := 1 << (tag & 0x0F)
		switch nbytes {
		case 4:
			var val float32
			binary.Read(p.reader, binary.BigEndian, &val)
			return &plistValue{Real, sizedFloat{float64(val), 32}}
		case 8:
			var val float64
			binary.Read(p.reader, binary.BigEndian, &val)
			return &plistValue{Real, sizedFloat{float64(val), 64}}
		}
		panic(errors.New("illegal float size"))
	case bpTagDate:
		var val float64
		binary.Read(p.reader, binary.BigEndian, &val)

		// Apple Epoch is 20110101000000Z
		// Adjust for UNIX Time
		val += 978307200

		sec, fsec := math.Modf(val)
		time := time.Unix(int64(sec), int64(fsec*float64(time.Second))).In(time.UTC)
		return &plistValue{Date, time}
	case bpTagData:
		cnt := p.countForTag(tag)
		if int64(cnt) > p.trailerOffset-int64(off) {
			panic(fmt.Errorf("data at %x longer than file (%v bytes, max is %v)", off, cnt, p.trailerOffset-int64(off)))
		}

		bytes := make([]byte, cnt)
		binary.Read(p.reader, binary.BigEndian, bytes)
		return &plistValue{Data, bytes}
	case bpTagASCIIString, bpTagUTF16String:
		cnt := p.countForTag(tag)
		if int64(cnt) > p.trailerOffset-int64(off) {
			panic(fmt.Errorf("string at %x longer than file (%v bytes, max is %v)", off, cnt, p.trailerOffset-int64(off)))
		}

		if tag&0xF0 == bpTagASCIIString {
			bytes := make([]byte, cnt)
			binary.Read(p.reader, binary.BigEndian, bytes)
			return &plistValue{String, string(bytes)}
		} else {
			bytes := make([]uint16, cnt)
			binary.Read(p.reader, binary.BigEndian, bytes)
			runes := utf16.Decode(bytes)
			return &plistValue{String, string(runes)}
		}
	case bpTagUID: // Somehow different than int: low half is nbytes - 1 instead of log2(nbytes)
		val := p.readSizedInt(int(tag&0xF) + 1)
		return &plistValue{Integer, signedInt{val, false}}
	case bpTagDictionary:
		cnt := p.countForTag(tag)

		subvalues := make(map[string]*plistValue)
		indices := make([]uint64, cnt*2)
		for i := uint64(0); i < cnt*2; i++ {
			idx := p.readSizedInt(int(p.trailer.ObjectRefSize))

			if idx >= p.trailer.NumObjects {
				panic(fmt.Errorf("dictionary contains invalid entry index %d (max %d)", idx, p.trailer.NumObjects))
			}

			indices[i] = idx
		}
		for i := uint64(0); i < cnt; i++ {
			keyOffset := p.offtable[indices[i]]
			valueOffset := p.offtable[indices[i+cnt]]
			if keyOffset == uint64(off) {
				panic(fmt.Errorf("dictionary contains self-referential key %x (index %d)", off, i))
			}
			if valueOffset == uint64(off) {
				panic(fmt.Errorf("dictionary contains self-referential value %x (index %d)", off, i))
			}

			kval := p.valueAtOffset(keyOffset)
			if kval == nil || kval.kind != String {
				panic(fmt.Errorf("dictionary contains non-string key at index %d", i))
			}

			key, ok := kval.value.(string)
			if !ok {
				panic(fmt.Errorf("string-type plist value contains non-string at index %d", i))
			}
			subvalues[key] = p.valueAtOffset(valueOffset)
		}

		return &plistValue{Dictionary, &dictionary{m: subvalues}}
	case bpTagArray:
		cnt := p.countForTag(tag)

		arr := make([]*plistValue, cnt)
		indices := make([]uint64, cnt)
		for i := uint64(0); i < cnt; i++ {
			idx := p.readSizedInt(int(p.trailer.ObjectRefSize))

			if idx >= p.trailer.NumObjects {
				panic(fmt.Errorf("array contains invalid entry index %d (max %d)", idx, p.trailer.NumObjects))
			}

			indices[i] = idx
		}
		for i := uint64(0); i < cnt; i++ {
			valueOffset := p.offtable[indices[i]]
			if valueOffset == uint64(off) {
				panic(fmt.Errorf("array contains self-referential value %x (index %d)", off, i))
			}
			arr[i] = p.valueAtOffset(valueOffset)
		}

		return &plistValue{Array, arr}
	}
	panic(fmt.Errorf("unexpected atom 0x%2.02x at offset %d", tag, off))
}

func newBplistParser(r io.ReadSeeker) *bplistParser {
	return &bplistParser{reader: r}
}
