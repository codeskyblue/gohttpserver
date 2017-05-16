package androidbinary

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

type XMLFile struct {
	stringPool     *ResStringPool
	resourceMap    []uint32
	notPrecessedNS map[ResStringPoolRef]ResStringPoolRef
	namespaces     map[ResStringPoolRef]ResStringPoolRef
	xmlBuffer      bytes.Buffer
}

type ResXMLTreeNode struct {
	Header     ResChunkHeader
	LineNumber uint32
	Comment    ResStringPoolRef
}

type ResXMLTreeNamespaceExt struct {
	Prefix ResStringPoolRef
	Uri    ResStringPoolRef
}

type ResXMLTreeAttrExt struct {
	NS             ResStringPoolRef
	Name           ResStringPoolRef
	AttributeStart uint16
	AttributeSize  uint16
	AttributeCount uint16
	IdIndex        uint16
	ClassIndex     uint16
	StyleIndex     uint16
}

type ResXMLTreeAttribute struct {
	NS         ResStringPoolRef
	Name       ResStringPoolRef
	RawValue   ResStringPoolRef
	TypedValue ResValue
}

type ResXMLTreeEndElementExt struct {
	NS   ResStringPoolRef
	Name ResStringPoolRef
}

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

func (f *XMLFile) Reader() *bytes.Reader {
	return bytes.NewReader(f.xmlBuffer.Bytes())
}

func (f *XMLFile) readChunk(r io.ReaderAt, offset int64) (*ResChunkHeader, error) {
	sr := io.NewSectionReader(r, offset, 1<<63-1-offset)
	chunkHeader := &ResChunkHeader{}
	sr.Seek(0, os.SEEK_SET)
	if err := binary.Read(sr, binary.LittleEndian, chunkHeader); err != nil {
		return nil, err
	}

	var err error
	sr.Seek(0, os.SEEK_SET)
	switch chunkHeader.Type {
	case RES_STRING_POOL_TYPE:
		f.stringPool, err = readStringPool(sr)
	case RES_XML_START_NAMESPACE_TYPE:
		err = f.readStartNamespace(sr)
	case RES_XML_END_NAMESPACE_TYPE:
		err = f.readEndNamespace(sr)
	case RES_XML_START_ELEMENT_TYPE:
		err = f.readStartElement(sr)
	case RES_XML_END_ELEMENT_TYPE:
		err = f.readEndElement(sr)
	}
	if err != nil {
		return nil, err
	}

	return chunkHeader, nil
}

func (f *XMLFile) GetString(ref ResStringPoolRef) string {
	return f.stringPool.GetString(ref)
}

func (f *XMLFile) readStartNamespace(sr *io.SectionReader) error {
	header := new(ResXMLTreeNode)
	if err := binary.Read(sr, binary.LittleEndian, header); err != nil {
		return err
	}

	sr.Seek(int64(header.Header.HeaderSize), os.SEEK_SET)
	namespace := new(ResXMLTreeNamespaceExt)
	if err := binary.Read(sr, binary.LittleEndian, namespace); err != nil {
		return err
	}

	if f.notPrecessedNS == nil {
		f.notPrecessedNS = make(map[ResStringPoolRef]ResStringPoolRef)
	}
	f.notPrecessedNS[namespace.Uri] = namespace.Prefix

	if f.namespaces == nil {
		f.namespaces = make(map[ResStringPoolRef]ResStringPoolRef)
	}
	f.namespaces[namespace.Uri] = namespace.Prefix

	return nil
}

func (f *XMLFile) readEndNamespace(sr *io.SectionReader) error {
	header := new(ResXMLTreeNode)
	if err := binary.Read(sr, binary.LittleEndian, header); err != nil {
		return err
	}

	sr.Seek(int64(header.Header.HeaderSize), os.SEEK_SET)
	namespace := new(ResXMLTreeNamespaceExt)
	if err := binary.Read(sr, binary.LittleEndian, namespace); err != nil {
		return err
	}
	delete(f.namespaces, namespace.Uri)
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

	sr.Seek(int64(header.Header.HeaderSize), os.SEEK_SET)
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
		sr.Seek(offset, os.SEEK_SET)
		attr := new(ResXMLTreeAttribute)
		binary.Read(sr, binary.LittleEndian, attr)

		var value string
		if attr.RawValue != NilResStringPoolRef {
			value = f.GetString(attr.RawValue)
		} else {
			data := attr.TypedValue.Data
			switch attr.TypedValue.DataType {
			case TYPE_NULL:
				value = ""
			case TYPE_REFERENCE:
				value = fmt.Sprintf("@0x%08X", data)
			case TYPE_INT_DEC:
				value = fmt.Sprintf("%d", data)
			case TYPE_INT_HEX:
				value = fmt.Sprintf("0x%08X", data)
			case TYPE_INT_BOOLEAN:
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
	sr.Seek(int64(header.Header.HeaderSize), os.SEEK_SET)
	ext := new(ResXMLTreeEndElementExt)
	if err := binary.Read(sr, binary.LittleEndian, ext); err != nil {
		return err
	}
	fmt.Fprintf(&f.xmlBuffer, "</%s>", f.addNamespacePrefix(ext.NS, ext.Name))
	return nil
}
