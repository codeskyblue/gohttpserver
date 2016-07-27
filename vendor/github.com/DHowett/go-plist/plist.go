package plist

import (
	"reflect"
	"sort"
)

// Property list format constants
const (
	// Used by Decoder to represent an invalid property list.
	InvalidFormat int = 0

	// Used to indicate total abandon with regards to Encoder's output format.
	AutomaticFormat = 0

	XMLFormat      = 1
	BinaryFormat   = 2
	OpenStepFormat = 3
	GNUStepFormat  = 4
)

var FormatNames = map[int]string{
	InvalidFormat:  "unknown/invalid",
	XMLFormat:      "XML",
	BinaryFormat:   "Binary",
	OpenStepFormat: "OpenStep",
	GNUStepFormat:  "GNUStep",
}

type plistKind uint

const (
	Invalid plistKind = iota
	Dictionary
	Array
	String
	Integer
	Real
	Boolean
	Data
	Date
)

var plistKindNames map[plistKind]string = map[plistKind]string{
	Invalid:    "invalid",
	Dictionary: "dictionary",
	Array:      "array",
	String:     "string",
	Integer:    "integer",
	Real:       "real",
	Boolean:    "boolean",
	Data:       "data",
	Date:       "date",
}

type plistValue struct {
	kind  plistKind
	value interface{}
}

type signedInt struct {
	value  uint64
	signed bool
}

type sizedFloat struct {
	value float64
	bits  int
}

type dictionary struct {
	count  int
	m      map[string]*plistValue
	keys   sort.StringSlice
	values []*plistValue
}

func (d *dictionary) Len() int {
	return d.count
}

func (d *dictionary) Less(i, j int) bool {
	return d.keys.Less(i, j)
}

func (d *dictionary) Swap(i, j int) {
	d.keys.Swap(i, j)
	d.values[i], d.values[j] = d.values[j], d.values[i]
}

func (d *dictionary) populateArrays() {
	if d.count > 0 {
		return
	}

	l := len(d.m)
	d.count = l
	d.keys = make([]string, l)
	d.values = make([]*plistValue, l)
	i := 0
	for k, v := range d.m {
		d.keys[i] = k
		d.values[i] = v
		i++
	}
	sort.Sort(d)
}

type unknownTypeError struct {
	typ reflect.Type
}

func (u *unknownTypeError) Error() string {
	return "plist: can't marshal value of type " + u.typ.String()
}

type invalidPlistError struct {
	format string
	err    error
}

func (e invalidPlistError) Error() string {
	s := "plist: invalid " + e.format + " property list"
	if e.err != nil {
		s += ": " + e.err.Error()
	}
	return s
}

type plistParseError struct {
	format string
	err    error
}

func (e plistParseError) Error() string {
	s := "plist: error parsing " + e.format + " property list"
	if e.err != nil {
		s += ": " + e.err.Error()
	}
	return s
}
