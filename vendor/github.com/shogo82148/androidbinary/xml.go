package androidbinary

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
)

// XMLFile is an XML file expressed in binary format.
type XMLFile struct {
	stringPool     *ResStringPool
	resourceMap    []uint32
	notPrecessedNS map[ResStringPoolRef]ResStringPoolRef
	namespaces     map[ResStringPoolRef]ResStringPoolRef
	xmlBuffer      bytes.Buffer
}

// ResXMLTreeNode is basic XML tree node.
type ResXMLTreeNode struct {
	Header     ResChunkHeader
	LineNumber uint32
	Comment    ResStringPoolRef
}

// ResXMLTreeNamespaceExt is extended XML tree node for namespace start/end nodes.
type ResXMLTreeNamespaceExt struct {
	Prefix ResStringPoolRef
	URI    ResStringPoolRef
}

// ResXMLTreeAttrExt is extended XML tree node for start tags -- includes attribute.
type ResXMLTreeAttrExt struct {
	NS             ResStringPoolRef
	Name           ResStringPoolRef
	AttributeStart uint16
	AttributeSize  uint16
	AttributeCount uint16
	IDIndex        uint16
	ClassIndex     uint16
	StyleIndex     uint16
}

// ResXMLTreeAttribute is an attribute of start tags.
type ResXMLTreeAttribute struct {
	NS         ResStringPoolRef
	Name       ResStringPoolRef
	RawValue   ResStringPoolRef
	TypedValue ResValue
}

// ResXMLTreeEndElementExt is extended XML tree node for element start/end nodes.
type ResXMLTreeEndElementExt struct {
	NS   ResStringPoolRef
	Name ResStringPoolRef
}

// NewXMLFile returns a new XMLFile.
func NewXMLFile(r io.ReaderAt) (*XMLFile, error) {
	f := new(XMLFile)
	sr := io.NewSectionReader(r, 0, 1<<63-1)

	fmt.Fprintf(&f.xmlBuffer, xml.Header)

	header := new(ResChunkHeader)
	if err := binary.Read(sr, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	offset := int64(header.HeaderSize)
	for offset < int64(header.Size) {
		chunkHeader, err := f.readChunk(r, offset)
		if err != nil {
			return nil, err
		}
		offset += int64(chunkHeader.Size)
	}
	return f, nil
}

// Reader returns a reader of XML file expressed in text format.
func (f *XMLFile) Reader() *bytes.Reader {
	return bytes.NewReader(f.xmlBuffer.Bytes())
}

func (f *XMLFile) readChunk(r io.ReaderAt, offset int64) (*ResChunkHeader, error) {
	sr := io.NewSectionReader(r, offset, 1<<63-1-offset)
	chunkHeader := &ResChunkHeader{}
	if _, err := sr.Seek(0, seekStart); err != nil {
		return nil, err
	}
	if err := binary.Read(sr, binary.LittleEndian, chunkHeader); err != nil {
		return nil, err
	}

	var err error
	if _, err := sr.Seek(0, seekStart); err != nil {
		return nil, err
	}
	switch chunkHeader.Type {
	case ResStringPoolChunkType:
		f.stringPool, err = readStringPool(sr)
	case ResXMLStartNamespaceType:
		err = f.readStartNamespace(sr)
	case ResXMLEndNamespaceType:
		err = f.readEndNamespace(sr)
	case ResXMLStartElementType:
		err = f.readStartElement(sr)
	case ResXMLEndElementType:
		err = f.readEndElement(sr)
	}
	if err != nil {
		return nil, err
	}

	return chunkHeader, nil
}

// GetString returns a string referenced by ref.
func (f *XMLFile) GetString(ref ResStringPoolRef) string {
	return f.stringPool.GetString(ref)
}

func (f *XMLFile) readStartNamespace(sr *io.SectionReader) error {
	header := new(ResXMLTreeNode)
	if err := binary.Read(sr, binary.LittleEndian, header); err != nil {
		return err
	}

	if _, err := sr.Seek(int64(header.Header.HeaderSize), seekStart); err != nil {
		return err
	}
	namespace := new(ResXMLTreeNamespaceExt)
	if err := binary.Read(sr, binary.LittleEndian, namespace); err != nil {
		return err
	}

	if f.notPrecessedNS == nil {
		f.notPrecessedNS = make(map[ResStringPoolRef]ResStringPoolRef)
	}
	f.notPrecessedNS[namespace.URI] = namespace.Prefix

	if f.namespaces == nil {
		f.namespaces = make(map[ResStringPoolRef]ResStringPoolRef)
	}
	f.namespaces[namespace.URI] = namespace.Prefix

	return nil
}

func (f *XMLFile) readEndNamespace(sr *io.SectionReader) error {
	header := new(ResXMLTreeNode)
	if err := binary.Read(sr, binary.LittleEndian, header); err != nil {
		return err
	}

	if _, err := sr.Seek(int64(header.Header.HeaderSize), seekStart); err != nil {
		return err
	}
	namespace := new(ResXMLTreeNamespaceExt)
	if err := binary.Read(sr, binary.LittleEndian, namespace); err != nil {
		return err
	}
	delete(f.namespaces, namespace.URI)
	return nil
}

func (f *XMLFile) addNamespacePrefix(ns, name ResStringPoolRef) string {
	if ns != NilResStringPoolRef {
		prefix := f.GetString(f.namespaces[ns])
		return fmt.Sprintf("%s:%s", prefix, f.GetString(name))
	}
	return f.GetString(name)
}

func (f *XMLFile) readStartElement(sr *io.SectionReader) error {
	header := new(ResXMLTreeNode)
	if err := binary.Read(sr, binary.LittleEndian, header); err != nil {
		return err
	}

	if _, err := sr.Seek(int64(header.Header.HeaderSize), seekStart); err != nil {
		return err
	}
	ext := new(ResXMLTreeAttrExt)
	if err := binary.Read(sr, binary.LittleEndian, ext); err != nil {
		return nil
	}

	fmt.Fprintf(&f.xmlBuffer, "<%s", f.addNamespacePrefix(ext.NS, ext.Name))

	// output XML namespaces
	if f.notPrecessedNS != nil {
		for uri, prefix := range f.notPrecessedNS {
			fmt.Fprintf(&f.xmlBuffer, " xmlns:%s=\"", f.GetString(prefix))
			xml.Escape(&f.xmlBuffer, []byte(f.GetString(uri)))
			fmt.Fprint(&f.xmlBuffer, "\"")
		}
		f.notPrecessedNS = nil
	}

	// process attributes
	offset := int64(ext.AttributeStart + header.Header.HeaderSize)
	for i := 0; i < int(ext.AttributeCount); i++ {
		if _, err := sr.Seek(offset, seekStart); err != nil {
			return err
		}
		attr := new(ResXMLTreeAttribute)
		binary.Read(sr, binary.LittleEndian, attr)

		var value string
		if attr.RawValue != NilResStringPoolRef {
			value = f.GetString(attr.RawValue)
		} else {
			data := attr.TypedValue.Data
			switch attr.TypedValue.DataType {
			case TypeNull:
				value = ""
			case TypeReference:
				value = fmt.Sprintf("@0x%08X", data)
			case TypeIntDec:
				value = fmt.Sprintf("%d", data)
			case TypeIntHex:
				value = fmt.Sprintf("0x%08X", data)
			case TypeIntBoolean:
				if data != 0 {
					value = "true"
				} else {
					value = "false"
				}
			default:
				value = fmt.Sprintf("@0x%08X", data)
			}
		}

		fmt.Fprintf(&f.xmlBuffer, " %s=\"", f.addNamespacePrefix(attr.NS, attr.Name))
		xml.Escape(&f.xmlBuffer, []byte(value))
		fmt.Fprint(&f.xmlBuffer, "\"")
		offset += int64(ext.AttributeSize)
	}
	fmt.Fprint(&f.xmlBuffer, ">")
	return nil
}

func (f *XMLFile) readEndElement(sr *io.SectionReader) error {
	header := new(ResXMLTreeNode)
	if err := binary.Read(sr, binary.LittleEndian, header); err != nil {
		return err
	}
	if _, err := sr.Seek(int64(header.Header.HeaderSize), seekStart); err != nil {
		return err
	}
	ext := new(ResXMLTreeEndElementExt)
	if err := binary.Read(sr, binary.LittleEndian, ext); err != nil {
		return err
	}
	fmt.Fprintf(&f.xmlBuffer, "</%s>", f.addNamespacePrefix(ext.NS, ext.Name))
	return nil
}
