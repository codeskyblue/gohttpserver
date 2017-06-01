package androidbinary

import (
	"bytes"
	"encoding/binary"
	"io"
	"unicode/utf16"
)

// ChunkType is a type of a resource chunk.
type ChunkType uint16

// Chunk types.
const (
	ResNullChunkType       ChunkType = 0x0000
	ResStringPoolChunkType ChunkType = 0x0001
	ResTableChunkType      ChunkType = 0x0002
	ResXMLChunkType        ChunkType = 0x0003

	// Chunk types in RES_XML_TYPE
	ResXMLFirstChunkType     ChunkType = 0x0100
	ResXMLStartNamespaceType ChunkType = 0x0100
	ResXMLEndNamespaceType   ChunkType = 0x0101
	ResXMLStartElementType   ChunkType = 0x0102
	ResXMLEndElementType     ChunkType = 0x0103
	ResXMLCDataType          ChunkType = 0x0104
	ResXMLLastChunkType      ChunkType = 0x017f

	// This contains a uint32_t array mapping strings in the string
	// pool back to resource identifiers.  It is optional.
	ResXMLResourceMapType ChunkType = 0x0180

	// Chunk types in RES_TABLE_TYPE
	ResTablePackageType  ChunkType = 0x0200
	ResTableTypeType     ChunkType = 0x0201
	ResTableTypeSpecType ChunkType = 0x0202
)

// ResChunkHeader is a header of a resource chunk.
type ResChunkHeader struct {
	Type       ChunkType
	HeaderSize uint16
	Size       uint32
}

// Flags are flags for string pool header.
type Flags uint32

// the values of Flags.
const (
	SortedFlag Flags = 1 << 0
	UTF8Flag   Flags = 1 << 8
)

// ResStringPoolHeader is a chunk header of string pool.
type ResStringPoolHeader struct {
	Header      ResChunkHeader
	StringCount uint32
	StyleCount  uint32
	Flags       Flags
	StringStart uint32
	StylesStart uint32
}

// ResStringPoolSpan is a span of style information associated with
// a string in the pool.
type ResStringPoolSpan struct {
	FirstChar, LastChar uint32
}

// ResStringPool is a string pool resrouce.
type ResStringPool struct {
	Header  ResStringPoolHeader
	Strings []string
	Styles  []ResStringPoolSpan
}

// NilResStringPoolRef is nil reference for string pool.
const NilResStringPoolRef = ResStringPoolRef(0xFFFFFFFF)

// ResStringPoolRef is a type representing a reference to a string.
type ResStringPoolRef uint32

// DataType is a type of the data value.
type DataType uint8

// The constants for DataType
const (
	TypeNull          DataType = 0x00
	TypeReference     DataType = 0x01
	TypeAttribute     DataType = 0x02
	TypeString        DataType = 0x03
	TypeFloat         DataType = 0x04
	TypeDemention     DataType = 0x05
	TypeFraction      DataType = 0x06
	TypeFirstInt      DataType = 0x10
	TypeIntDec        DataType = 0x10
	TypeIntHex        DataType = 0x11
	TypeIntBoolean    DataType = 0x12
	TypeFirstColorInt DataType = 0x1c
	TypeIntColorARGB8 DataType = 0x1c
	TypeIntColorRGB8  DataType = 0x1d
	TypeIntColorARGB4 DataType = 0x1e
	TypeIntColorRGB4  DataType = 0x1f
	TypeLastColorInt  DataType = 0x1f
	TypeLastInt       DataType = 0x1f
)

// ResValue is a representation of a value in a resource
type ResValue struct {
	Size     uint16
	Res0     uint8
	DataType DataType
	Data     uint32
}

// GetString returns a string referenced by ref.
func (pool *ResStringPool) GetString(ref ResStringPoolRef) string {
	return pool.Strings[int(ref)]
}

func readStringPool(sr *io.SectionReader) (*ResStringPool, error) {
	sp := new(ResStringPool)
	if err := binary.Read(sr, binary.LittleEndian, &sp.Header); err != nil {
		return nil, err
	}

	stringStarts := make([]uint32, sp.Header.StringCount)
	if err := binary.Read(sr, binary.LittleEndian, stringStarts); err != nil {
		return nil, err
	}

	styleStarts := make([]uint32, sp.Header.StyleCount)
	if err := binary.Read(sr, binary.LittleEndian, styleStarts); err != nil {
		return nil, err
	}

	sp.Strings = make([]string, sp.Header.StringCount)
	for i, start := range stringStarts {
		var str string
		var err error
		if _, err := sr.Seek(int64(sp.Header.StringStart+start), seekStart); err != nil {
			return nil, err
		}
		if (sp.Header.Flags & UTF8Flag) == 0 {
			str, err = readUTF16(sr)
		} else {
			str, err = readUTF8(sr)
		}
		if err != nil {
			return nil, err
		}
		sp.Strings[i] = str
	}

	sp.Styles = make([]ResStringPoolSpan, sp.Header.StyleCount)
	for i, start := range styleStarts {
		if _, err := sr.Seek(int64(sp.Header.StylesStart+start), seekStart); err != nil {
			return nil, err
		}
		if err := binary.Read(sr, binary.LittleEndian, &sp.Styles[i]); err != nil {
			return nil, err
		}
	}

	return sp, nil
}

func readUTF16(sr *io.SectionReader) (string, error) {
	// read lenth of string
	size, err := readUTF16length(sr)
	if err != nil {
		return "", err
	}

	// read string value
	buf := make([]uint16, size)
	if err := binary.Read(sr, binary.LittleEndian, buf); err != nil {
		return "", err
	}
	return string(utf16.Decode(buf)), nil
}

func readUTF16length(sr *io.SectionReader) (int, error) {
	var size int
	var first, second uint16
	if err := binary.Read(sr, binary.LittleEndian, &first); err != nil {
		return 0, err
	}
	if (first & 0x8000) != 0 {
		if err := binary.Read(sr, binary.LittleEndian, &second); err != nil {
			return 0, err
		}
		size = (int(first&0x7FFF) << 16) + int(second)
	} else {
		size = int(first)
	}
	return size, nil
}

func readUTF8(sr *io.SectionReader) (string, error) {
	// skip utf16 length
	_, err := readUTF8length(sr)
	if err != nil {
		return "", err
	}

	// read lenth of string
	size, err := readUTF8length(sr)
	if err != nil {
		return "", err
	}

	buf := make([]uint8, size)
	if err := binary.Read(sr, binary.LittleEndian, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func readUTF8length(sr *io.SectionReader) (int, error) {
	var size int
	var first, second uint8
	if err := binary.Read(sr, binary.LittleEndian, &first); err != nil {
		return 0, err
	}
	if (first & 0x80) != 0 {
		if err := binary.Read(sr, binary.LittleEndian, &second); err != nil {
			return 0, err
		}
		size = (int(first&0x7F) << 8) + int(second)
	} else {
		size = int(first)
	}
	return size, nil
}

func newZeroFilledReader(r io.Reader, actual int64, expected int64) (io.Reader, error) {
	if actual >= expected {
		// no need to fill
		return r, nil
	}

	// read `actual' bytes from r, and
	buf := new(bytes.Buffer)
	if _, err := io.CopyN(buf, r, actual); err != nil {
		return nil, err
	}

	// fill zero until `expected' bytes
	for i := actual; i < expected; i++ {
		if err := buf.WriteByte(0x00); err != nil {
			return nil, err
		}
	}

	return buf, nil
}
