package plist

import (
	"bufio"
	"encoding/hex"
	"errors"
	"io"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type textPlistGenerator struct {
	writer io.Writer
	format int

	quotableTable *[4]uint64

	indent string
	depth  int

	dictKvDelimiter, dictEntryDelimiter, arrayDelimiter []byte
}

var (
	textPlistTimeLayout = "2006-01-02 15:04:05 -0700"
	padding             = "0000"
)

func (p *textPlistGenerator) generateDocument(pval *plistValue) {
	p.writePlistValue(pval)
}

func (p *textPlistGenerator) plistQuotedString(str string) string {
	if str == "" {
		return `""`
	}
	s := ""
	quot := false
	for _, r := range str {
		if r > 0xFF {
			quot = true
			s += `\U`
			us := strconv.FormatInt(int64(r), 16)
			s += padding[len(us):]
			s += us
		} else if r > 0x7F {
			quot = true
			s += `\`
			us := strconv.FormatInt(int64(r), 8)
			s += padding[1+len(us):]
			s += us
		} else {
			c := uint8(r)
			if (*p.quotableTable)[c/64]&(1<<(c%64)) > 0 {
				quot = true
			}

			switch c {
			case '\a':
				s += `\a`
			case '\b':
				s += `\b`
			case '\v':
				s += `\v`
			case '\f':
				s += `\f`
			case '\\':
				s += `\\`
			case '"':
				s += `\"`
			case '\t', '\r', '\n':
				fallthrough
			default:
				s += string(c)
			}
		}
	}
	if quot {
		s = `"` + s + `"`
	}
	return s
}

func (p *textPlistGenerator) deltaIndent(depthDelta int) {
	if depthDelta < 0 {
		p.depth--
	} else if depthDelta > 0 {
		p.depth++
	}
}

func (p *textPlistGenerator) writeIndent() {
	if len(p.indent) == 0 {
		return
	}
	if len(p.indent) > 0 {
		p.writer.Write([]byte("\n"))
		for i := 0; i < p.depth; i++ {
			io.WriteString(p.writer, p.indent)
		}
	}
}

func (p *textPlistGenerator) writePlistValue(pval *plistValue) {
	if pval == nil {
		return
	}

	switch pval.kind {
	case Dictionary:
		p.writer.Write([]byte(`{`))
		p.deltaIndent(1)
		dict := pval.value.(*dictionary)
		dict.populateArrays()
		for i, k := range dict.keys {
			p.writeIndent()
			io.WriteString(p.writer, p.plistQuotedString(k))
			p.writer.Write(p.dictKvDelimiter)
			p.writePlistValue(dict.values[i])
			p.writer.Write(p.dictEntryDelimiter)
		}
		p.deltaIndent(-1)
		p.writeIndent()
		p.writer.Write([]byte(`}`))
	case Array:
		p.writer.Write([]byte(`(`))
		p.deltaIndent(1)
		values := pval.value.([]*plistValue)
		for _, v := range values {
			p.writeIndent()
			p.writePlistValue(v)
			p.writer.Write(p.arrayDelimiter)
		}
		p.deltaIndent(-1)
		p.writeIndent()
		p.writer.Write([]byte(`)`))
	case String:
		io.WriteString(p.writer, p.plistQuotedString(pval.value.(string)))
	case Integer:
		if p.format == GNUStepFormat {
			p.writer.Write([]byte(`<*I`))
		}
		if pval.value.(signedInt).signed {
			io.WriteString(p.writer, strconv.FormatInt(int64(pval.value.(signedInt).value), 10))
		} else {
			io.WriteString(p.writer, strconv.FormatUint(pval.value.(signedInt).value, 10))
		}
		if p.format == GNUStepFormat {
			p.writer.Write([]byte(`>`))
		}
	case Real:
		if p.format == GNUStepFormat {
			p.writer.Write([]byte(`<*R`))
		}
		io.WriteString(p.writer, strconv.FormatFloat(pval.value.(sizedFloat).value, 'g', -1, 64))
		if p.format == GNUStepFormat {
			p.writer.Write([]byte(`>`))
		}
	case Boolean:
		b := pval.value.(bool)
		if p.format == GNUStepFormat {
			if b {
				p.writer.Write([]byte(`<*BY>`))
			} else {
				p.writer.Write([]byte(`<*BN>`))
			}
		} else {
			if b {
				p.writer.Write([]byte(`1`))
			} else {
				p.writer.Write([]byte(`0`))
			}
		}
	case Data:
		b := pval.value.([]byte)
		var hexencoded [9]byte
		var l int
		var asc = 9
		hexencoded[8] = ' '

		p.writer.Write([]byte(`<`))
		for i := 0; i < len(b); i += 4 {
			l = i + 4
			if l >= len(b) {
				l = len(b)
				// We no longer need the space - or the rest of the buffer.
				// (we used >= above to get this part without another conditional :P)
				asc = (l - i) * 2
			}
			// Fill the buffer (only up to 8 characters, to preserve the space we implicitly include
			// at the end of every encode)
			hex.Encode(hexencoded[:8], b[i:l])
			io.WriteString(p.writer, string(hexencoded[:asc]))
		}
		p.writer.Write([]byte(`>`))
	case Date:
		if p.format == GNUStepFormat {
			p.writer.Write([]byte(`<*D`))
			io.WriteString(p.writer, pval.value.(time.Time).In(time.UTC).Format(textPlistTimeLayout))
			p.writer.Write([]byte(`>`))
		} else {
			io.WriteString(p.writer, p.plistQuotedString(pval.value.(time.Time).In(time.UTC).Format(textPlistTimeLayout)))
		}
	}
}

func (p *textPlistGenerator) Indent(i string) {
	p.indent = i
	if i == "" {
		p.dictKvDelimiter = []byte(`=`)
	} else {
		// For pretty-printing
		p.dictKvDelimiter = []byte(` = `)
	}
}

func newTextPlistGenerator(w io.Writer, format int) *textPlistGenerator {
	table := &osQuotable
	if format == GNUStepFormat {
		table = &gsQuotable
	}
	return &textPlistGenerator{
		writer:             mustWriter{w},
		format:             format,
		quotableTable:      table,
		dictKvDelimiter:    []byte(`=`),
		arrayDelimiter:     []byte(`,`),
		dictEntryDelimiter: []byte(`;`),
	}
}

type byteReader interface {
	io.Reader
	io.ByteScanner
	Peek(n int) ([]byte, error)
	ReadBytes(delim byte) ([]byte, error)
}

type textPlistParser struct {
	reader             byteReader
	whitespaceReplacer *strings.Replacer
	format             int
}

func (p *textPlistParser) parseDocument() (pval *plistValue, parseError error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			if _, ok := r.(invalidPlistError); ok {
				parseError = r.(error)
			} else {
				// Wrap all non-invalid-plist errors.
				parseError = plistParseError{"text", r.(error)}
			}
		}
	}()
	pval = p.parsePlistValue()
	return
}

func (p *textPlistParser) chugWhitespace() {
ws:
	for {
		c, err := p.reader.ReadByte()
		if err != nil && err != io.EOF {
			panic(err)
		}
		if whitespace[c/64]&(1<<(c%64)) == 0 {
			if c == '/' && err != io.EOF {
				// A / at the end of the file is not the begining of a comment.
				cs, err := p.reader.Peek(1)
				if err != nil && err != io.EOF {
					panic(err)
				}
				c = cs[0]
				switch c {
				case '/':
					for {
						c, err = p.reader.ReadByte()
						if err != nil && err != io.EOF {
							panic(err)
						} else if err == io.EOF {
							break
						}
						// TODO: UTF-8
						if c == '\n' || c == '\r' {
							break
						}
					}
				case '*':
					// Peek returned a value here, so it is safe to read.
					_, _ = p.reader.ReadByte()
					star := false
					for {
						c, err = p.reader.ReadByte()
						if err != nil {
							panic(err)
						}
						if c == '*' {
							star = true
						} else if c == '/' && star {
							break
						} else {
							star = false
						}
					}
				default:
					p.reader.UnreadByte() // Not the beginning of a // or /* comment
					break ws
				}
				continue
			}
			p.reader.UnreadByte()
			break
		}
	}
}

func (p *textPlistParser) parseQuotedString() *plistValue {
	escaping := false
	s := ""
	for {
		byt, err := p.reader.ReadByte()
		// EOF here is an error: we're inside a quoted string!
		if err != nil {
			panic(err)
		}
		c := rune(byt)
		if !escaping {
			if c == '"' {
				break
			} else if c == '\\' {
				escaping = true
				continue
			}
		} else {
			escaping = false
			// Everything that is not listed here passes through unharmed.
			switch c {
			case 'a':
				c = '\a'
			case 'b':
				c = '\b'
			case 'v':
				c = '\v'
			case 'f':
				c = '\f'
			case 't':
				c = '\t'
			case 'r':
				c = '\r'
			case 'n':
				c = '\n'
			case 'x', 'u', 'U': // hex and unicode
				l := 4
				if c == 'x' {
					l = 2
				}
				hex := make([]byte, l)
				p.reader.Read(hex)
				newc := mustParseInt(string(hex), 16, 16)
				c = rune(newc)
			case '0', '1', '2', '3', '4', '5', '6', '7': // octal!
				oct := make([]byte, 3)
				oct[0] = uint8(c)
				p.reader.Read(oct[1:])
				newc := mustParseInt(string(oct), 8, 16)
				c = rune(newc)
			}
		}
		s += string(c)
	}
	return &plistValue{String, s}
}

func (p *textPlistParser) parseUnquotedString() *plistValue {
	s := ""
	for {
		c, err := p.reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		// if we encounter a character that must be quoted, we're done.
		// the GNUStep quote table is more lax here, so we use it instead of the OpenStep one.
		if gsQuotable[c/64]&(1<<(c%64)) > 0 {
			p.reader.UnreadByte()
			break
		}
		s += string(c)
	}
	return &plistValue{String, s}
}

func (p *textPlistParser) parseDictionary() *plistValue {
	var keypv *plistValue
	subval := make(map[string]*plistValue)
	for {
		p.chugWhitespace()

		c, err := p.reader.ReadByte()
		// EOF here is an error: we're inside a dictionary!
		if err != nil {
			panic(err)
		}

		if c == '}' {
			break
		} else if c == '"' {
			keypv = p.parseQuotedString()
		} else {
			p.reader.UnreadByte() // Whoops, ate part of the string
			keypv = p.parseUnquotedString()
		}
		if keypv == nil {
			// TODO better error
			panic(errors.New("missing dictionary key"))
		}

		p.chugWhitespace()
		c, err = p.reader.ReadByte()
		if err != nil {
			panic(err)
		}

		if c != '=' {
			panic(errors.New("missing = in dictionary"))
		}

		// whitespace is guzzled within
		val := p.parsePlistValue()

		p.chugWhitespace()
		c, err = p.reader.ReadByte()
		if err != nil {
			panic(err)
		}

		if c != ';' {
			panic(errors.New("missing ; in dictionary"))
		}

		subval[keypv.value.(string)] = val
	}
	return &plistValue{Dictionary, &dictionary{m: subval}}
}

func (p *textPlistParser) parseArray() *plistValue {
	subval := make([]*plistValue, 0, 10)
	for {
		c, err := p.reader.ReadByte()
		// EOF here is an error: we're inside an array!
		if err != nil {
			panic(err)
		}

		if c == ')' {
			break
		} else if c == ',' {
			continue
		}

		p.reader.UnreadByte()
		pval := p.parsePlistValue()
		if pval.kind == String && pval.value.(string) == "" {
			continue
		}
		subval = append(subval, pval)
	}
	return &plistValue{Array, subval}
}

func (p *textPlistParser) parseGNUStepValue(v []byte) *plistValue {
	if len(v) < 2 {
		panic(errors.New("invalid GNUStep extended value"))
	}
	typ := v[1]
	v = v[2:]
	switch typ {
	case 'I':
		if v[0] == '-' {
			n := mustParseInt(string(v), 10, 64)
			return &plistValue{Integer, signedInt{uint64(n), true}}
		} else {
			n := mustParseUint(string(v), 10, 64)
			return &plistValue{Integer, signedInt{n, false}}
		}
	case 'R':
		n := mustParseFloat(string(v), 64)
		return &plistValue{Real, sizedFloat{n, 64}}
	case 'B':
		b := v[0] == 'Y'
		return &plistValue{Boolean, b}
	case 'D':
		t, err := time.Parse(textPlistTimeLayout, string(v))
		if err != nil {
			panic(err)
		}

		return &plistValue{Date, t.In(time.UTC)}
	}
	panic(errors.New("invalid GNUStep type " + string(typ)))
	return nil
}

func (p *textPlistParser) parsePlistValue() *plistValue {
	for {
		p.chugWhitespace()

		c, err := p.reader.ReadByte()
		if err != nil && err != io.EOF {
			panic(err)
		}
		switch c {
		case '<':
			bytes, err := p.reader.ReadBytes('>')
			if err != nil {
				panic(err)
			}
			bytes = bytes[:len(bytes)-1]

			if bytes[0] == '*' {
				p.format = GNUStepFormat
				return p.parseGNUStepValue(bytes)
			} else {
				s := p.whitespaceReplacer.Replace(string(bytes))
				data, err := hex.DecodeString(s)
				if err != nil {
					panic(err)
				}
				return &plistValue{Data, data}
			}
		case '"':
			return p.parseQuotedString()
		case '{':
			return p.parseDictionary()
		case '(':
			return p.parseArray()
		default:
			p.reader.UnreadByte() // Place back in buffer for parseUnquotedString
			return p.parseUnquotedString()
		}
	}
	return nil
}

func newTextPlistParser(r io.Reader) *textPlistParser {
	var reader byteReader
	if rd, ok := r.(byteReader); ok {
		reader = rd
	} else {
		reader = bufio.NewReader(r)
	}
	return &textPlistParser{
		reader:             reader,
		whitespaceReplacer: strings.NewReplacer("\t", "", "\n", "", " ", "", "\r", ""),
		format:             OpenStepFormat,
	}
}
