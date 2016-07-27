package plist

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"strings"
	"time"
)

const xmlDOCTYPE = `<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
`

type xmlPlistGenerator struct {
	writer     io.Writer
	xmlEncoder *xml.Encoder
}

func (p *xmlPlistGenerator) generateDocument(pval *plistValue) {
	io.WriteString(p.writer, xml.Header)
	io.WriteString(p.writer, xmlDOCTYPE)

	plistStartElement := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: "plist",
		},
		Attr: []xml.Attr{{
			Name: xml.Name{
				Space: "",
				Local: "version"},
			Value: "1.0"},
		},
	}

	p.xmlEncoder.EncodeToken(plistStartElement)

	p.writePlistValue(pval)

	p.xmlEncoder.EncodeToken(plistStartElement.End())
	p.xmlEncoder.Flush()
}

func (p *xmlPlistGenerator) writePlistValue(pval *plistValue) {
	if pval == nil {
		return
	}

	defer p.xmlEncoder.Flush()

	key := ""
	encodedValue := pval.value
	switch pval.kind {
	case Dictionary:
		startElement := xml.StartElement{Name: xml.Name{Local: "dict"}}
		p.xmlEncoder.EncodeToken(startElement)
		dict := encodedValue.(*dictionary)
		dict.populateArrays()
		for i, k := range dict.keys {
			p.xmlEncoder.EncodeElement(k, xml.StartElement{Name: xml.Name{Local: "key"}})
			p.writePlistValue(dict.values[i])
		}
		p.xmlEncoder.EncodeToken(startElement.End())
	case Array:
		startElement := xml.StartElement{Name: xml.Name{Local: "array"}}
		p.xmlEncoder.EncodeToken(startElement)
		values := encodedValue.([]*plistValue)
		for _, v := range values {
			p.writePlistValue(v)
		}
		p.xmlEncoder.EncodeToken(startElement.End())
	case String:
		key = "string"
	case Integer:
		key = "integer"
		if pval.value.(signedInt).signed {
			encodedValue = int64(pval.value.(signedInt).value)
		} else {
			encodedValue = pval.value.(signedInt).value
		}
	case Real:
		key = "real"
		encodedValue = pval.value.(sizedFloat).value
		switch {
		case math.IsInf(pval.value.(sizedFloat).value, 1):
			encodedValue = "inf"
		case math.IsInf(pval.value.(sizedFloat).value, -1):
			encodedValue = "-inf"
		case math.IsNaN(pval.value.(sizedFloat).value):
			encodedValue = "nan"
		}
	case Boolean:
		key = "false"
		b := pval.value.(bool)
		if b {
			key = "true"
		}
		encodedValue = ""
	case Data:
		key = "data"
		encodedValue = xml.CharData(base64.StdEncoding.EncodeToString(pval.value.([]byte)))
	case Date:
		key = "date"
		encodedValue = pval.value.(time.Time).In(time.UTC).Format(time.RFC3339)
	}
	if key != "" {
		err := p.xmlEncoder.EncodeElement(encodedValue, xml.StartElement{Name: xml.Name{Local: key}})
		if err != nil {
			panic(err)
		}
	}
}

func (p *xmlPlistGenerator) Indent(i string) {
	p.xmlEncoder.Indent("", i)
}

func newXMLPlistGenerator(w io.Writer) *xmlPlistGenerator {
	mw := mustWriter{w}
	return &xmlPlistGenerator{mw, xml.NewEncoder(mw)}
}

type xmlPlistParser struct {
	reader             io.Reader
	xmlDecoder         *xml.Decoder
	whitespaceReplacer *strings.Replacer
	ntags              int
}

func (p *xmlPlistParser) parseDocument() (pval *plistValue, parseError error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			if _, ok := r.(invalidPlistError); ok {
				parseError = r.(error)
			} else {
				// Wrap all non-invalid-plist errors.
				parseError = plistParseError{"XML", r.(error)}
			}
		}
	}()
	for {
		if token, err := p.xmlDecoder.Token(); err == nil {
			if element, ok := token.(xml.StartElement); ok {
				pval = p.parseXMLElement(element)
				if p.ntags == 0 {
					panic(invalidPlistError{"XML", errors.New("no elements encountered")})
				}
				return
			}
		} else {
			// The first XML parse turned out to be invalid:
			// we do not have an XML property list.
			panic(invalidPlistError{"XML", err})
		}
	}
}

func (p *xmlPlistParser) parseXMLElement(element xml.StartElement) *plistValue {
	var charData xml.CharData
	switch element.Name.Local {
	case "plist":
		p.ntags++
		for {
			token, err := p.xmlDecoder.Token()
			if err != nil {
				panic(err)
			}

			if el, ok := token.(xml.EndElement); ok && el.Name.Local == "plist" {
				break
			}

			if el, ok := token.(xml.StartElement); ok {
				return p.parseXMLElement(el)
			}
		}
		return nil
	case "string":
		p.ntags++
		err := p.xmlDecoder.DecodeElement(&charData, &element)
		if err != nil {
			panic(err)
		}

		return &plistValue{String, string(charData)}
	case "integer":
		p.ntags++
		err := p.xmlDecoder.DecodeElement(&charData, &element)
		if err != nil {
			panic(err)
		}

		s := string(charData)
		if s[0] == '-' {
			n := mustParseInt(string(charData), 10, 64)
			return &plistValue{Integer, signedInt{uint64(n), true}}
		} else {
			n := mustParseUint(string(charData), 10, 64)
			return &plistValue{Integer, signedInt{n, false}}
		}
	case "real":
		p.ntags++
		err := p.xmlDecoder.DecodeElement(&charData, &element)
		if err != nil {
			panic(err)
		}

		n := mustParseFloat(string(charData), 64)
		return &plistValue{Real, sizedFloat{n, 64}}
	case "true", "false":
		p.ntags++
		p.xmlDecoder.Skip()

		b := element.Name.Local == "true"
		return &plistValue{Boolean, b}
	case "date":
		p.ntags++
		err := p.xmlDecoder.DecodeElement(&charData, &element)
		if err != nil {
			panic(err)
		}

		t, err := time.ParseInLocation(time.RFC3339, string(charData), time.UTC)
		if err != nil {
			panic(err)
		}

		return &plistValue{Date, t}
	case "data":
		p.ntags++
		err := p.xmlDecoder.DecodeElement(&charData, &element)
		if err != nil {
			panic(err)
		}

		str := p.whitespaceReplacer.Replace(string(charData))

		l := base64.StdEncoding.DecodedLen(len(str))
		bytes := make([]uint8, l)
		l, err = base64.StdEncoding.Decode(bytes, []byte(str))
		if err != nil {
			panic(err)
		}

		return &plistValue{Data, bytes[:l]}
	case "dict":
		p.ntags++
		var key *string
		var subvalues map[string]*plistValue = make(map[string]*plistValue)
		for {
			token, err := p.xmlDecoder.Token()
			if err != nil {
				panic(err)
			}

			if el, ok := token.(xml.EndElement); ok && el.Name.Local == "dict" {
				if key != nil {
					panic(errors.New("missing value in dictionary"))
				}
				break
			}

			if el, ok := token.(xml.StartElement); ok {
				if el.Name.Local == "key" {
					var k string
					p.xmlDecoder.DecodeElement(&k, &el)
					key = &k
				} else {
					if key == nil {
						panic(errors.New("missing key in dictionary"))
					}
					subvalues[*key] = p.parseXMLElement(el)
					key = nil
				}
			}
		}
		return &plistValue{Dictionary, &dictionary{m: subvalues}}
	case "array":
		p.ntags++
		var subvalues []*plistValue = make([]*plistValue, 0, 10)
		for {
			token, err := p.xmlDecoder.Token()
			if err != nil {
				panic(err)
			}

			if el, ok := token.(xml.EndElement); ok && el.Name.Local == "array" {
				break
			}

			if el, ok := token.(xml.StartElement); ok {
				subvalues = append(subvalues, p.parseXMLElement(el))
			}
		}
		return &plistValue{Array, subvalues}
	}
	err := fmt.Errorf("encountered unknown element %s", element.Name.Local)
	if p.ntags == 0 {
		// If out first XML tag is invalid, it might be an openstep data element, ala <abab> or <0101>
		panic(invalidPlistError{"XML", err})
	}
	panic(err)
}

func newXMLPlistParser(r io.Reader) *xmlPlistParser {
	return &xmlPlistParser{r, xml.NewDecoder(r), strings.NewReplacer("\t", "", "\n", "", " ", "", "\r", ""), 0}
}
