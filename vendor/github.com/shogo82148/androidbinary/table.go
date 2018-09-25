package androidbinary

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unsafe"
)

// ResID is ID for resources.
type ResID uint32

// TableFile is a resrouce table file.
type TableFile struct {
	stringPool    *ResStringPool
	tablePackages map[uint32]*TablePackage
}

// ResTableHeader is a header of TableFile.
type ResTableHeader struct {
	Header       ResChunkHeader
	PackageCount uint32
}

// ResTablePackage is a header of table packages.
type ResTablePackage struct {
	Header         ResChunkHeader
	ID             uint32
	Name           [128]uint16
	TypeStrings    uint32
	LastPublicType uint32
	KeyStrings     uint32
	LastPublicKey  uint32
}

// TablePackage is a table package.
type TablePackage struct {
	Header      ResTablePackage
	TypeStrings *ResStringPool
	KeyStrings  *ResStringPool
	TableTypes  []*TableType
}

// ResTableType is a type of a table.
type ResTableType struct {
	Header       ResChunkHeader
	ID           uint8
	Res0         uint8
	Res1         uint16
	EntryCount   uint32
	EntriesStart uint32
	Config       ResTableConfig
}

// ScreenLayout describes screen layout.
type ScreenLayout uint8

// ScreenLayout bits
const (
	MaskScreenSize   ScreenLayout = 0x0f
	ScreenSizeAny    ScreenLayout = 0x01
	ScreenSizeSmall  ScreenLayout = 0x02
	ScreenSizeNormal ScreenLayout = 0x03
	ScreenSizeLarge  ScreenLayout = 0x04
	ScreenSizeXLarge ScreenLayout = 0x05

	MaskScreenLong  ScreenLayout = 0x30
	ShiftScreenLong              = 4
	ScreenLongAny   ScreenLayout = 0x00
	ScreenLongNo    ScreenLayout = 0x10
	ScreenLongYes   ScreenLayout = 0x20

	MaskLayoutDir  ScreenLayout = 0xC0
	ShiftLayoutDir              = 6
	LayoutDirAny   ScreenLayout = 0x00
	LayoutDirLTR   ScreenLayout = 0x40
	LayoutDirRTL   ScreenLayout = 0x80
)

// UIMode describes UI mode.
type UIMode uint8

// UIMode bits
const (
	MaskUIModeType   UIMode = 0x0f
	UIModeTypeAny    UIMode = 0x01
	UIModeTypeNormal UIMode = 0x02
	UIModeTypeDesk   UIMode = 0x03
	UIModeTypeCar    UIMode = 0x04

	MaskUIModeNight  UIMode = 0x30
	ShiftUIModeNight        = 4
	UIModeNightAny   UIMode = 0x00
	UIModeNightNo    UIMode = 0x10
	UIModeNightYes   UIMode = 0x20
)

// InputFlags are input flags.
type InputFlags uint8

// input flags
const (
	MaskKeysHidden InputFlags = 0x03
	KeysHiddenAny  InputFlags = 0x00
	KeysHiddenNo   InputFlags = 0x01
	KeysHiddenYes  InputFlags = 0x02
	KeysHiddenSoft InputFlags = 0x03

	MaskNavHidden InputFlags = 0x0c
	NavHiddenAny  InputFlags = 0x00
	NavHiddenNo   InputFlags = 0x04
	NavHiddenYes  InputFlags = 0x08
)

// ResTableConfig is a configuration of a table.
type ResTableConfig struct {
	Size uint32
	// imsi
	Mcc uint16
	Mnc uint16

	// locale
	Language [2]uint8
	Country  [2]uint8

	// screen type
	Orientation uint8
	Touchscreen uint8
	Density     uint16

	// inout
	Keyboard   uint8
	Navigation uint8
	InputFlags InputFlags
	InputPad0  uint8

	// screen size
	ScreenWidth  uint16
	ScreenHeight uint16

	// version
	SDKVersion   uint16
	MinorVersion uint16

	// screen config
	ScreenLayout          ScreenLayout
	UIMode                UIMode
	SmallestScreenWidthDp uint16

	// screen size dp
	ScreenWidthDp  uint16
	ScreenHeightDp uint16
}

// TableType is a collection of resource entries for a particular resource data type.
type TableType struct {
	Header  *ResTableType
	Entries []TableEntry
}

// ResTableEntry is the beginning of information about an entry in the resource table.
type ResTableEntry struct {
	Size  uint16
	Flags uint16
	Key   ResStringPoolRef
}

// TableEntry is a entry in a recource table.
type TableEntry struct {
	Key   *ResTableEntry
	Value *ResValue
	Flags uint32
}

// ResTableTypeSpec is specification of the resources defined by a particular type.
type ResTableTypeSpec struct {
	Header     ResChunkHeader
	ID         uint8
	Res0       uint8
	Res1       uint16
	EntryCount uint32
}

// IsResID returns whether s is ResId.
func IsResID(s string) bool {
	return strings.HasPrefix(s, "@0x")
}

// ParseResID parses ResId.
func ParseResID(s string) (ResID, error) {
	if !IsResID(s) {
		return 0, fmt.Errorf("androidbinary: %s is not ResID", s)
	}
	id, err := strconv.ParseUint(s[3:], 16, 32)
	if err != nil {
		return 0, err
	}
	return ResID(id), nil
}

func (id ResID) String() string {
	return fmt.Sprintf("@0x%08X", uint32(id))
}

// Package returns the package index of id.
func (id ResID) Package() uint32 {
	return uint32(id) >> 24
}

// Type returns the type index of id.
func (id ResID) Type() int {
	return (int(id) >> 16) & 0xFF
}

// Entry returns the entry index of id.
func (id ResID) Entry() int {
	return int(id) & 0xFFFF
}

// NewTableFile returns new TableFile.
func NewTableFile(r io.ReaderAt) (*TableFile, error) {
	f := new(TableFile)
	sr := io.NewSectionReader(r, 0, 1<<63-1)

	header := new(ResTableHeader)
	binary.Read(sr, binary.LittleEndian, header)
	f.tablePackages = make(map[uint32]*TablePackage)

	offset := int64(header.Header.HeaderSize)
	for offset < int64(header.Header.Size) {
		chunkHeader, err := f.readChunk(sr, offset)
		if err != nil {
			return nil, err
		}
		offset += int64(chunkHeader.Size)
	}
	return f, nil
}

func (f *TableFile) findPackage(id uint32) *TablePackage {
	if f == nil {
		return nil
	}
	return f.tablePackages[id]
}

func (p *TablePackage) findEntry(typeIndex, entryIndex int, config *ResTableConfig) TableEntry {
	var best *TableType
	for _, t := range p.TableTypes {
		switch {
		case int(t.Header.ID) != typeIndex:
			// nothing to do
		case !t.Header.Config.Match(config):
			// nothing to do
		case entryIndex >= len(t.Entries):
			// nothing to do
		case t.Entries[entryIndex].Value == nil:
			// nothing to do
		case best == nil || t.Header.Config.IsBetterThan(&best.Header.Config, config):
			best = t
		}
	}
	if best == nil || entryIndex >= len(best.Entries) {
		return TableEntry{}
	}
	return best.Entries[entryIndex]
}

// GetResource returns a resrouce referenced by id.
func (f *TableFile) GetResource(id ResID, config *ResTableConfig) (interface{}, error) {
	p := f.findPackage(id.Package())
	if p == nil {
		return nil, fmt.Errorf("androidbinary: package 0x%02X not found", id.Package())
	}
	e := p.findEntry(id.Type(), id.Entry(), config)
	v := e.Value
	if v == nil {
		return nil, fmt.Errorf("androidbinary: entry 0x%04X not found", id.Entry())
	}
	switch v.DataType {
	case TypeNull:
		return nil, nil
	case TypeString:
		return f.GetString(ResStringPoolRef(v.Data)), nil
	case TypeIntDec:
		return v.Data, nil
	case TypeIntHex:
		return v.Data, nil
	case TypeIntBoolean:
		return v.Data != 0, nil
	}
	return v.Data, nil
}

// GetString returns a string referenced by ref.
func (f *TableFile) GetString(ref ResStringPoolRef) string {
	return f.stringPool.GetString(ref)
}

func (f *TableFile) readChunk(r io.ReaderAt, offset int64) (*ResChunkHeader, error) {
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
	case ResTablePackageType:
		var tablePackage *TablePackage
		tablePackage, err = readTablePackage(sr)
		f.tablePackages[tablePackage.Header.ID] = tablePackage
	}
	if err != nil {
		return nil, err
	}

	return chunkHeader, nil
}

func readTablePackage(sr *io.SectionReader) (*TablePackage, error) {
	tablePackage := new(TablePackage)
	header := new(ResTablePackage)
	if err := binary.Read(sr, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	tablePackage.Header = *header

	srTypes := io.NewSectionReader(sr, int64(header.TypeStrings), int64(header.Header.Size-header.TypeStrings))
	if typeStrings, err := readStringPool(srTypes); err == nil {
		tablePackage.TypeStrings = typeStrings
	} else {
		return nil, err
	}

	srKeys := io.NewSectionReader(sr, int64(header.KeyStrings), int64(header.Header.Size-header.KeyStrings))
	if keyStrings, err := readStringPool(srKeys); err == nil {
		tablePackage.KeyStrings = keyStrings
	} else {
		return nil, err
	}

	offset := int64(header.Header.HeaderSize)
	for offset < int64(header.Header.Size) {
		chunkHeader := &ResChunkHeader{}
		if _, err := sr.Seek(offset, seekStart); err != nil {
			return nil, err
		}
		if err := binary.Read(sr, binary.LittleEndian, chunkHeader); err != nil {
			return nil, err
		}

		var err error
		chunkReader := io.NewSectionReader(sr, offset, int64(chunkHeader.Size))
		if _, err := sr.Seek(offset, seekStart); err != nil {
			return nil, err
		}
		switch chunkHeader.Type {
		case ResTableTypeType:
			var tableType *TableType
			tableType, err = readTableType(chunkHeader, chunkReader)
			tablePackage.TableTypes = append(tablePackage.TableTypes, tableType)
		case ResTableTypeSpecType:
			_, err = readTableTypeSpec(chunkReader)
		}
		if err != nil {
			return nil, err
		}
		offset += int64(chunkHeader.Size)
	}

	return tablePackage, nil
}

func readTableType(chunkHeader *ResChunkHeader, sr *io.SectionReader) (*TableType, error) {
	// TableType header may be omitted
	header := new(ResTableType)
	if _, err := sr.Seek(0, seekStart); err != nil {
		return nil, err
	}
	buf, err := newZeroFilledReader(sr, int64(chunkHeader.HeaderSize), int64(unsafe.Sizeof(*header)))
	if err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, header); err != nil {
		return nil, err
	}

	entryIndexes := make([]uint32, header.EntryCount)
	if _, err := sr.Seek(int64(header.Header.HeaderSize), seekStart); err != nil {
		return nil, err
	}
	if err := binary.Read(sr, binary.LittleEndian, entryIndexes); err != nil {
		return nil, err
	}

	entries := make([]TableEntry, header.EntryCount)
	for i, index := range entryIndexes {
		if index == 0xFFFFFFFF {
			continue
		}
		if _, err := sr.Seek(int64(header.EntriesStart+index), seekStart); err != nil {
			return nil, err
		}
		var key ResTableEntry
		binary.Read(sr, binary.LittleEndian, &key)
		entries[i].Key = &key

		var val ResValue
		binary.Read(sr, binary.LittleEndian, &val)
		entries[i].Value = &val
	}
	return &TableType{
		header,
		entries,
	}, nil
}

func readTableTypeSpec(sr *io.SectionReader) ([]uint32, error) {
	header := new(ResTableTypeSpec)
	if err := binary.Read(sr, binary.LittleEndian, header); err != nil {
		return nil, err
	}

	flags := make([]uint32, header.EntryCount)
	if _, err := sr.Seek(int64(header.Header.HeaderSize), seekStart); err != nil {
		return nil, err
	}
	if err := binary.Read(sr, binary.LittleEndian, flags); err != nil {
		return nil, err
	}
	return flags, nil
}

// IsMoreSpecificThan returns true if c is more specific than o.
func (c *ResTableConfig) IsMoreSpecificThan(o *ResTableConfig) bool {
	// nil ResTableConfig is never more specific than any ResTableConfig
	if c == nil {
		return false
	}
	if o == nil {
		return false
	}

	// imsi
	if c.Mcc != o.Mcc {
		if c.Mcc == 0 {
			return false
		}
		if o.Mnc == 0 {
			return true
		}
	}
	if c.Mnc != o.Mnc {
		if c.Mnc == 0 {
			return false
		}
		if o.Mnc == 0 {
			return true
		}
	}

	// locale
	if diff := c.IsLocaleMoreSpecificThan(o); diff < 0 {
		return false
	} else if diff > 0 {
		return true
	}

	// screen layout
	if c.ScreenLayout != 0 || o.ScreenLayout != 0 {
		if ((c.ScreenLayout ^ o.ScreenLayout) & MaskLayoutDir) != 0 {
			if (c.ScreenLayout & MaskLayoutDir) == 0 {
				return false
			}
			if (o.ScreenLayout & MaskLayoutDir) == 0 {
				return true
			}
		}
	}

	// smallest screen width dp
	if c.SmallestScreenWidthDp != 0 || o.SmallestScreenWidthDp != 0 {
		if c.SmallestScreenWidthDp != o.SmallestScreenWidthDp {
			if c.SmallestScreenWidthDp == 0 {
				return false
			}
			if o.SmallestScreenWidthDp == 0 {
				return true
			}
		}
	}

	// screen size dp
	if c.ScreenWidthDp != 0 || o.ScreenWidthDp != 0 ||
		c.ScreenHeightDp != 0 || o.ScreenHeightDp != 0 {
		if c.ScreenWidthDp != o.ScreenWidthDp {
			if c.ScreenWidthDp == 0 {
				return false
			}
			if o.ScreenWidthDp == 0 {
				return true
			}
		}
		if c.ScreenHeightDp != o.ScreenHeightDp {
			if c.ScreenHeightDp == 0 {
				return false
			}
			if o.ScreenHeightDp == 0 {
				return true
			}
		}
	}

	// screen layout
	if c.ScreenLayout != 0 || o.ScreenLayout != 0 {
		if ((c.ScreenLayout ^ o.ScreenLayout) & MaskScreenSize) != 0 {
			if (c.ScreenLayout & MaskScreenSize) == 0 {
				return false
			}
			if (o.ScreenLayout & MaskScreenSize) == 0 {
				return true
			}
		}
		if ((c.ScreenLayout ^ o.ScreenLayout) & MaskScreenLong) != 0 {
			if (c.ScreenLayout & MaskScreenLong) == 0 {
				return false
			}
			if (o.ScreenLayout & MaskScreenLong) == 0 {
				return true
			}
		}
	}

	// orientation
	if c.Orientation != o.Orientation {
		if c.Orientation == 0 {
			return false
		}
		if o.Orientation == 0 {
			return true
		}
	}

	// uimode
	if c.UIMode != 0 || o.UIMode != 0 {
		diff := c.UIMode ^ o.UIMode
		if (diff & MaskUIModeType) != 0 {
			if (c.UIMode & MaskUIModeType) == 0 {
				return false
			}
			if (o.UIMode & MaskUIModeType) == 0 {
				return true
			}
		}
		if (diff & MaskUIModeNight) != 0 {
			if (c.UIMode & MaskUIModeNight) == 0 {
				return false
			}
			if (o.UIMode & MaskUIModeNight) == 0 {
				return true
			}
		}
	}

	// touchscreen
	if c.Touchscreen != o.Touchscreen {
		if c.Touchscreen == 0 {
			return false
		}
		if o.Touchscreen == 0 {
			return true
		}
	}

	// input
	if c.InputFlags != 0 || o.InputFlags != 0 {
		myKeysHidden := c.InputFlags & MaskKeysHidden
		oKeysHidden := o.InputFlags & MaskKeysHidden
		if (myKeysHidden ^ oKeysHidden) != 0 {
			if myKeysHidden == 0 {
				return false
			}
			if oKeysHidden == 0 {
				return true
			}
		}
		myNavHidden := c.InputFlags & MaskNavHidden
		oNavHidden := o.InputFlags & MaskNavHidden
		if (myNavHidden ^ oNavHidden) != 0 {
			if myNavHidden == 0 {
				return false
			}
			if oNavHidden == 0 {
				return true
			}
		}
	}

	if c.Keyboard != o.Keyboard {
		if c.Keyboard == 0 {
			return false
		}
		if o.Keyboard == 0 {
			return true
		}
	}

	if c.Navigation != o.Navigation {
		if c.Navigation == 0 {
			return false
		}
		if o.Navigation == 0 {
			return true
		}
	}

	// screen size
	if c.ScreenWidth != 0 || o.ScreenWidth != 0 ||
		c.ScreenHeight != 0 || o.ScreenHeight != 0 {
		if c.ScreenWidth != o.ScreenWidth {
			if c.ScreenWidth == 0 {
				return false
			}
			if o.ScreenWidth == 0 {
				return true
			}
		}
		if c.ScreenHeight != o.ScreenHeight {
			if c.ScreenHeight == 0 {
				return false
			}
			if o.ScreenHeight == 0 {
				return true
			}
		}
	}

	//version
	if c.SDKVersion != o.SDKVersion {
		if c.SDKVersion == 0 {
			return false
		}
		if o.SDKVersion == 0 {
			return true
		}
	}
	if c.MinorVersion != o.MinorVersion {
		if c.MinorVersion == 0 {
			return false
		}
		if o.MinorVersion == 0 {
			return true
		}
	}

	return false
}

// IsBetterThan returns true if c is better than o for the r configuration.
func (c *ResTableConfig) IsBetterThan(o *ResTableConfig, r *ResTableConfig) bool {
	if r == nil {
		return c.IsMoreSpecificThan(o)
	}

	// nil ResTableConfig is never better than any ResTableConfig
	if c == nil {
		return false
	}
	if o == nil {
		return false
	}

	// imsi
	if c.Mcc != 0 || c.Mnc != 0 || o.Mcc != 0 || o.Mnc != 0 {
		if c.Mcc != o.Mcc && r.Mcc != 0 {
			return c.Mcc != 0
		}
		if c.Mnc != o.Mnc && r.Mnc != 0 {
			return c.Mnc != 0
		}
	}

	// locale
	if c.IsLocaleBetterThan(o, r) {
		return true
	}

	// screen layout
	if c.ScreenLayout != 0 || o.ScreenLayout != 0 {
		myLayoutdir := c.ScreenLayout & MaskLayoutDir
		oLayoutdir := o.ScreenLayout & MaskLayoutDir
		if (myLayoutdir^oLayoutdir) != 0 && (r.ScreenLayout&MaskLayoutDir) != 0 {
			return myLayoutdir > oLayoutdir
		}
	}

	// smallest screen width dp
	if c.SmallestScreenWidthDp != 0 || o.SmallestScreenWidthDp != 0 {
		if c.SmallestScreenWidthDp != o.SmallestScreenWidthDp {
			return c.SmallestScreenWidthDp > o.SmallestScreenWidthDp
		}
	}

	// screen size dp
	if c.ScreenWidthDp != 0 || c.ScreenHeightDp != 0 || o.ScreenWidthDp != 0 || o.ScreenHeightDp != 0 {
		myDelta := 0
		otherDelta := 0
		if r.ScreenWidthDp != 0 {
			myDelta += int(r.ScreenWidthDp) - int(c.ScreenWidthDp)
			otherDelta += int(r.ScreenWidthDp) - int(o.ScreenWidthDp)
		}
		if r.ScreenHeightDp != 0 {
			myDelta += int(r.ScreenHeightDp) - int(c.ScreenHeightDp)
			otherDelta += int(r.ScreenHeightDp) - int(o.ScreenHeightDp)
		}
		if myDelta != otherDelta {
			return myDelta < otherDelta
		}
	}

	// screen layout
	if c.ScreenLayout != 0 || o.ScreenLayout != 0 {
		mySL := c.ScreenLayout & MaskScreenSize
		oSL := o.ScreenLayout & MaskScreenSize
		if (mySL^oSL) != 0 && (r.ScreenLayout&MaskScreenSize) != 0 {
			fixedMySL := mySL
			fixedOSL := oSL
			if (r.ScreenLayout & MaskScreenSize) >= ScreenSizeNormal {
				if fixedMySL == 0 {
					fixedMySL = ScreenSizeNormal
				}
				if fixedOSL == 0 {
					fixedOSL = ScreenSizeNormal
				}
			}
			if fixedMySL == fixedOSL {
				return mySL != 0
			}
			return fixedMySL > fixedOSL
		}

		if ((c.ScreenLayout^o.ScreenLayout)&MaskScreenLong) != 0 &&
			(r.ScreenLayout&MaskScreenLong) != 0 {
			return (c.ScreenLayout & MaskScreenLong) != 0
		}
	}

	// orientation
	if c.Orientation != o.Orientation && r.Orientation != 0 {
		return c.Orientation != 0
	}

	// uimode
	if c.UIMode != 0 || o.UIMode != 0 {
		diff := c.UIMode ^ o.UIMode
		if (diff&MaskUIModeType) != 0 && (r.UIMode&MaskUIModeType) != 0 {
			return (c.UIMode & MaskUIModeType) != 0
		}
		if (diff&MaskUIModeNight) != 0 && (r.UIMode&MaskUIModeNight) != 0 {
			return (c.UIMode & MaskUIModeNight) != 0
		}
	}

	// screen type
	if c.Density != o.Density {
		h := int(c.Density)
		if h == 0 {
			h = 160
		}
		l := int(o.Density)
		if l == 0 {
			l = 160
		}
		blmBigger := true
		if l > h {
			h, l = l, h
			blmBigger = false
		}

		reqValue := int(r.Density)
		if reqValue == 0 {
			reqValue = 160
		}
		if reqValue >= h {
			return blmBigger
		}
		if l >= reqValue {
			return !blmBigger
		}
		if (2*l-reqValue)*h > reqValue*reqValue {
			return !blmBigger
		}
		return blmBigger
	}
	if c.Touchscreen != o.Touchscreen && r.Touchscreen != 0 {
		return c.Touchscreen != 0
	}

	// input
	if c.InputFlags != 0 || o.InputFlags != 0 {
		myKeysHidden := c.InputFlags & MaskKeysHidden
		oKeysHidden := o.InputFlags & MaskKeysHidden
		reqKeysHidden := r.InputFlags & MaskKeysHidden
		if myKeysHidden != oKeysHidden && reqKeysHidden != 0 {
			switch {
			case myKeysHidden == 0:
				return false
			case oKeysHidden == 0:
				return true
			case reqKeysHidden == myKeysHidden:
				return true
			case reqKeysHidden == oKeysHidden:
				return false
			}
		}
		myNavHidden := c.InputFlags & MaskNavHidden
		oNavHidden := o.InputFlags & MaskNavHidden
		reqNavHidden := r.InputFlags & MaskNavHidden
		if myNavHidden != oNavHidden && reqNavHidden != 0 {
			switch {
			case myNavHidden == 0:
				return false
			case oNavHidden == 0:
				return true
			}
		}
	}
	if c.Keyboard != o.Keyboard && r.Keyboard != 0 {
		return c.Keyboard != 0
	}
	if c.Navigation != o.Navigation && r.Navigation != 0 {
		return c.Navigation != 0
	}

	// screen size
	if c.ScreenWidth != 0 || c.ScreenHeight != 0 || o.ScreenWidth != 0 || o.ScreenHeight != 0 {
		myDelta := 0
		otherDelta := 0
		if r.ScreenWidth != 0 {
			myDelta += int(r.ScreenWidth) - int(c.ScreenWidth)
			otherDelta += int(r.ScreenWidth) - int(o.ScreenWidth)
		}
		if r.ScreenHeight != 0 {
			myDelta += int(r.ScreenHeight) - int(c.ScreenHeight)
			otherDelta += int(r.ScreenHeight) - int(o.ScreenHeight)
		}
		if myDelta != otherDelta {
			return myDelta < otherDelta
		}
	}

	// version
	if c.SDKVersion != 0 || o.MinorVersion != 0 {
		if c.SDKVersion != o.SDKVersion && r.SDKVersion != 0 {
			return c.SDKVersion > o.SDKVersion
		}
		if c.MinorVersion != o.MinorVersion && r.MinorVersion != 0 {
			return c.MinorVersion != 0
		}
	}

	return false
}

// IsLocaleMoreSpecificThan a positive integer if this config is more specific than o,
// a negative integer if |o| is more specific
// and 0 if they're equally specific.
func (c *ResTableConfig) IsLocaleMoreSpecificThan(o *ResTableConfig) int {
	if (c.Language != [2]uint8{} || c.Country != [2]uint8{}) || (o.Language != [2]uint8{} || o.Country != [2]uint8{}) {
		if c.Language != o.Language {
			if c.Language == [2]uint8{} {
				return -1
			}
			if o.Language == [2]uint8{} {
				return 1
			}
		}

		if c.Country != o.Country {
			if c.Country == [2]uint8{} {
				return -1
			}
			if o.Country == [2]uint8{} {
				return 1
			}
		}
	}
	return 0
}

// IsLocaleBetterThan returns true if c is a better locale match than o for the r configuration.
func (c *ResTableConfig) IsLocaleBetterThan(o *ResTableConfig, r *ResTableConfig) bool {
	if r.Language == [2]uint8{} && r.Country == [2]uint8{} {
		// The request doesn't have a locale, so no resource is better
		// than the other.
		return false
	}

	if c.Language == [2]uint8{} && c.Country == [2]uint8{} && o.Language == [2]uint8{} && o.Country == [2]uint8{} {
		// The locales parts of both resources are empty, so no one is better
		// than the other.
		return false
	}

	if c.Language != o.Language {
		// The languages of the two resources are not the same.

		// the US English resource have traditionally lived for most apps.
		if r.Language == [2]uint8{'e', 'n'} {
			if r.Country == [2]uint8{'U', 'S'} {
				if c.Language != [2]uint8{} {
					return c.Country == [2]uint8{} || c.Country == [2]uint8{'U', 'S'}
				}
				return !(c.Country == [2]uint8{} || c.Country == [2]uint8{'U', 'S'})
			}
		}
		return c.Language != [2]uint8{}
	}

	if c.Country != o.Country {
		return c.Country != [2]uint8{}
	}

	return false
}

// Match returns true if c can be considered a match for the parameters in settings.
func (c *ResTableConfig) Match(settings *ResTableConfig) bool {
	// nil ResTableConfig always matches.
	if settings == nil {
		return true
	} else if c == nil {
		return *settings == ResTableConfig{}
	}

	// match imsi
	if settings.Mcc == 0 {
		if c.Mcc != 0 {
			return false
		}
	} else {
		if c.Mcc != 0 && c.Mcc != settings.Mcc {
			return false
		}
	}
	if settings.Mnc == 0 {
		if c.Mnc != 0 {
			return false
		}
	} else {
		if c.Mnc != 0 && c.Mnc != settings.Mnc {
			return false
		}
	}

	// match locale
	if c.Language != [2]uint8{0, 0} {
		// Don't consider country and variants when deciding matches.
		// If two configs differ only in their country and variant,
		// they can be weeded out in the isMoreSpecificThan test.
		if c.Language != settings.Language {
			return false
		}

		if c.Country != [2]uint8{0, 0} {
			if c.Country != settings.Country {
				return false
			}
		}
	}

	// screen layout
	layoutDir := c.ScreenLayout & MaskLayoutDir
	setLayoutDir := settings.ScreenLayout & MaskLayoutDir
	if layoutDir != 0 && layoutDir != setLayoutDir {
		return false
	}

	screenSize := c.ScreenLayout & MaskScreenSize
	setScreenSize := settings.ScreenLayout & MaskScreenSize
	if screenSize != 0 && screenSize > setScreenSize {
		return false
	}

	screenLong := c.ScreenLayout & MaskScreenLong
	setScreenLong := settings.ScreenLayout & MaskScreenLong
	if screenLong != 0 && screenLong != setScreenLong {
		return false
	}

	// ui mode
	uiModeType := c.UIMode & MaskUIModeType
	setUIModeType := settings.UIMode & MaskUIModeType
	if uiModeType != 0 && uiModeType != setUIModeType {
		return false
	}

	uiModeNight := c.UIMode & MaskUIModeNight
	setUIModeNight := settings.UIMode & MaskUIModeNight
	if uiModeNight != 0 && uiModeNight != setUIModeNight {
		return false
	}

	// smallest screen width dp
	if c.SmallestScreenWidthDp != 0 &&
		c.SmallestScreenWidthDp > settings.SmallestScreenWidthDp {
		return false
	}

	// screen size dp
	if c.ScreenWidthDp != 0 &&
		c.ScreenWidthDp > settings.ScreenWidthDp {
		return false
	}
	if c.ScreenHeightDp != 0 &&
		c.ScreenHeightDp > settings.ScreenHeightDp {
		return false
	}

	// screen type
	if c.Orientation != 0 && c.Orientation != settings.Orientation {
		return false
	}
	if c.Touchscreen != 0 && c.Touchscreen != settings.Touchscreen {
		return false
	}

	// input
	if c.InputFlags != 0 {
		myKeysHidden := c.InputFlags & MaskKeysHidden
		oKeysHidden := settings.InputFlags & MaskKeysHidden
		if myKeysHidden != 0 && myKeysHidden != oKeysHidden {
			if myKeysHidden != KeysHiddenNo || oKeysHidden != KeysHiddenSoft {
				return false
			}
		}
		myNavHidden := c.InputFlags & MaskNavHidden
		oNavHidden := settings.InputFlags & MaskNavHidden
		if myNavHidden != 0 && myNavHidden != oNavHidden {
			return false
		}
	}
	if c.Keyboard != 0 && c.Keyboard != settings.Keyboard {
		return false
	}
	if c.Navigation != 0 && c.Navigation != settings.Navigation {
		return false
	}

	// screen size
	if c.ScreenWidth != 0 &&
		c.ScreenWidth > settings.ScreenWidth {
		return false
	}
	if c.ScreenHeight != 0 &&
		c.ScreenHeight > settings.ScreenHeight {
		return false
	}

	// version
	if settings.SDKVersion != 0 && c.SDKVersion != 0 &&
		c.SDKVersion > settings.SDKVersion {
		return false
	}
	if settings.MinorVersion != 0 && c.MinorVersion != 0 &&
		c.MinorVersion != settings.MinorVersion {
		return false
	}

	return true
}

// Locale returns the locale of the configuration.
func (c *ResTableConfig) Locale() string {
	if c.Language[0] == 0 {
		return ""
	}
	if c.Country[0] == 0 {
		return fmt.Sprintf("%c%c", c.Language[0], c.Language[1])
	}
	return fmt.Sprintf("%c%c-%c%c", c.Language[0], c.Language[1], c.Country[0], c.Country[1])
}
