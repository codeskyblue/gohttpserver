package plist

import (
	"encoding"
	"reflect"
	"time"
)

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

var (
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	timeType          = reflect.TypeOf((*time.Time)(nil)).Elem()
)

func (p *Encoder) marshalTextInterface(marshalable encoding.TextMarshaler) *plistValue {
	s, err := marshalable.MarshalText()
	if err != nil {
		panic(err)
	}
	return &plistValue{String, string(s)}
}

func (p *Encoder) marshalStruct(typ reflect.Type, val reflect.Value) *plistValue {
	tinfo, _ := getTypeInfo(typ)

	dict := &dictionary{
		m: make(map[string]*plistValue, len(tinfo.fields)),
	}
	for _, finfo := range tinfo.fields {
		value := finfo.value(val)
		if !value.IsValid() || finfo.omitEmpty && isEmptyValue(value) {
			continue
		}
		dict.m[finfo.name] = p.marshal(value)
	}

	return &plistValue{Dictionary, dict}
}

func (p *Encoder) marshalTime(val reflect.Value) *plistValue {
	time := val.Interface().(time.Time)
	return &plistValue{Date, time}
}

func (p *Encoder) marshal(val reflect.Value) *plistValue {
	if !val.IsValid() {
		return nil
	}

	// time.Time implements TextMarshaler, but we need to store it in RFC3339
	if val.Type() == timeType {
		return p.marshalTime(val)
	}
	if val.Kind() == reflect.Ptr || (val.Kind() == reflect.Interface && val.NumMethod() == 0) {
		ival := val.Elem()
		if ival.IsValid() && ival.Type() == timeType {
			return p.marshalTime(ival)
		}
	}

	// Check for text marshaler.
	if val.CanInterface() && val.Type().Implements(textMarshalerType) {
		return p.marshalTextInterface(val.Interface().(encoding.TextMarshaler))
	}
	if val.CanAddr() {
		pv := val.Addr()
		if pv.CanInterface() && pv.Type().Implements(textMarshalerType) {
			return p.marshalTextInterface(pv.Interface().(encoding.TextMarshaler))
		}
	}

	// Descend into pointers or interfaces
	if val.Kind() == reflect.Ptr || (val.Kind() == reflect.Interface && val.NumMethod() == 0) {
		val = val.Elem()
	}

	// We got this far and still may have an invalid anything or nil ptr/interface
	if !val.IsValid() || ((val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface) && val.IsNil()) {
		return nil
	}

	typ := val.Type()

	if val.Kind() == reflect.Struct {
		return p.marshalStruct(typ, val)
	}

	switch val.Kind() {
	case reflect.String:
		return &plistValue{String, val.String()}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &plistValue{Integer, signedInt{uint64(val.Int()), true}}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return &plistValue{Integer, signedInt{uint64(val.Uint()), false}}
	case reflect.Float32, reflect.Float64:
		return &plistValue{Real, sizedFloat{val.Float(), val.Type().Bits()}}
	case reflect.Bool:
		return &plistValue{Boolean, val.Bool()}
	case reflect.Slice, reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			bytes := []byte(nil)
			if val.CanAddr() {
				bytes = val.Bytes()
			} else {
				bytes = make([]byte, val.Len())
				reflect.Copy(reflect.ValueOf(bytes), val)
			}
			return &plistValue{Data, bytes}
		} else {
			subvalues := make([]*plistValue, val.Len())
			for idx, length := 0, val.Len(); idx < length; idx++ {
				if subpval := p.marshal(val.Index(idx)); subpval != nil {
					subvalues[idx] = subpval
				}
			}
			return &plistValue{Array, subvalues}
		}
	case reflect.Map:
		if typ.Key().Kind() != reflect.String {
			panic(&unknownTypeError{typ})
		}

		l := val.Len()
		dict := &dictionary{
			m: make(map[string]*plistValue, l),
		}
		for _, keyv := range val.MapKeys() {
			if subpval := p.marshal(val.MapIndex(keyv)); subpval != nil {
				dict.m[keyv.String()] = subpval
			}
		}
		return &plistValue{Dictionary, dict}
	default:
		panic(&unknownTypeError{typ})
	}
	return nil
}
