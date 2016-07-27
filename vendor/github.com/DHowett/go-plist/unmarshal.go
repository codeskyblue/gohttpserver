package plist

import (
	"encoding"
	"fmt"
	"reflect"
	"time"
)

type incompatibleDecodeTypeError struct {
	typ   reflect.Type
	pKind plistKind
}

func (u *incompatibleDecodeTypeError) Error() string {
	return fmt.Sprintf("plist: type mismatch: tried to decode %v into value of type %v", plistKindNames[u.pKind], u.typ)
}

var (
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

func isEmptyInterface(v reflect.Value) bool {
	return v.Kind() == reflect.Interface && v.NumMethod() == 0
}

func (p *Decoder) unmarshalTextInterface(pval *plistValue, unmarshalable encoding.TextUnmarshaler) {
	err := unmarshalable.UnmarshalText([]byte(pval.value.(string)))
	if err != nil {
		panic(err)
	}
}

func (p *Decoder) unmarshalTime(pval *plistValue, val reflect.Value) {
	val.Set(reflect.ValueOf(pval.value.(time.Time)))
}

func (p *Decoder) unmarshalLaxString(s string, val reflect.Value) {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i := mustParseInt(s, 10, 64)
		val.SetInt(i)
		return
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		i := mustParseUint(s, 10, 64)
		val.SetUint(i)
		return
	case reflect.Float32, reflect.Float64:
		f := mustParseFloat(s, 64)
		val.SetFloat(f)
		return
	case reflect.Bool:
		b := mustParseBool(s)
		val.SetBool(b)
		return
	case reflect.Struct:
		if val.Type() == timeType {
			t, err := time.Parse(textPlistTimeLayout, s)
			if err != nil {
				panic(err)
			}
			val.Set(reflect.ValueOf(t.In(time.UTC)))
			return
		}
		fallthrough
	default:
		panic(&incompatibleDecodeTypeError{val.Type(), String})
	}
}

func (p *Decoder) unmarshal(pval *plistValue, val reflect.Value) {
	if pval == nil {
		return
	}

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			val.Set(reflect.New(val.Type().Elem()))
		}
		val = val.Elem()
	}

	if isEmptyInterface(val) {
		v := p.valueInterface(pval)
		val.Set(reflect.ValueOf(v))
		return
	}

	incompatibleTypeError := &incompatibleDecodeTypeError{val.Type(), pval.kind}

	// time.Time implements TextMarshaler, but we need to parse it as RFC3339
	if pval.kind == Date {
		if val.Type() == timeType {
			p.unmarshalTime(pval, val)
			return
		}
		panic(incompatibleTypeError)
	}

	if val.CanInterface() && val.Type().Implements(textUnmarshalerType) && val.Type() != timeType {
		p.unmarshalTextInterface(pval, val.Interface().(encoding.TextUnmarshaler))
		return
	}

	if val.CanAddr() {
		pv := val.Addr()
		if pv.CanInterface() && pv.Type().Implements(textUnmarshalerType) && val.Type() != timeType {
			p.unmarshalTextInterface(pval, pv.Interface().(encoding.TextUnmarshaler))
			return
		}
	}

	typ := val.Type()

	switch pval.kind {
	case String:
		if val.Kind() == reflect.String {
			val.SetString(pval.value.(string))
			return
		}
		if p.lax {
			p.unmarshalLaxString(pval.value.(string), val)
			return
		}

		panic(incompatibleTypeError)
	case Integer:
		switch val.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val.SetInt(int64(pval.value.(signedInt).value))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			val.SetUint(pval.value.(signedInt).value)
		default:
			panic(incompatibleTypeError)
		}
	case Real:
		if val.Kind() == reflect.Float32 || val.Kind() == reflect.Float64 {
			val.SetFloat(pval.value.(sizedFloat).value)
		} else {
			panic(incompatibleTypeError)
		}
	case Boolean:
		if val.Kind() == reflect.Bool {
			val.SetBool(pval.value.(bool))
		} else {
			panic(incompatibleTypeError)
		}
	case Data:
		if val.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Uint8 {
			val.SetBytes(pval.value.([]byte))
		} else {
			panic(incompatibleTypeError)
		}
	case Array:
		p.unmarshalArray(pval, val)
	case Dictionary:
		p.unmarshalDictionary(pval, val)
	}
}

func (p *Decoder) unmarshalArray(pval *plistValue, val reflect.Value) {
	subvalues := pval.value.([]*plistValue)

	var n int
	if val.Kind() == reflect.Slice {
		// Slice of element values.
		// Grow slice.
		cnt := len(subvalues) + val.Len()
		if cnt >= val.Cap() {
			ncap := 2 * cnt
			if ncap < 4 {
				ncap = 4
			}
			new := reflect.MakeSlice(val.Type(), val.Len(), ncap)
			reflect.Copy(new, val)
			val.Set(new)
		}
		n = val.Len()
		val.SetLen(cnt)
	} else if val.Kind() == reflect.Array {
		if len(subvalues) > val.Cap() {
			panic(fmt.Errorf("plist: attempted to unmarshal %d values into an array of size %d", len(subvalues), val.Cap()))
		}
	} else {
		panic(&incompatibleDecodeTypeError{val.Type(), pval.kind})
	}

	// Recur to read element into slice.
	for _, sval := range subvalues {
		p.unmarshal(sval, val.Index(n))
		n++
	}
	return
}

func (p *Decoder) unmarshalDictionary(pval *plistValue, val reflect.Value) {
	typ := val.Type()
	switch val.Kind() {
	case reflect.Struct:
		tinfo, err := getTypeInfo(typ)
		if err != nil {
			panic(err)
		}

		subvalues := pval.value.(*dictionary).m
		for _, finfo := range tinfo.fields {
			p.unmarshal(subvalues[finfo.name], finfo.value(val))
		}
	case reflect.Map:
		if val.IsNil() {
			val.Set(reflect.MakeMap(typ))
		}

		subvalues := pval.value.(*dictionary).m
		for k, sval := range subvalues {
			keyv := reflect.ValueOf(k).Convert(typ.Key())
			mapElem := val.MapIndex(keyv)
			if !mapElem.IsValid() {
				mapElem = reflect.New(typ.Elem()).Elem()
			}

			p.unmarshal(sval, mapElem)
			val.SetMapIndex(keyv, mapElem)
		}
	default:
		panic(&incompatibleDecodeTypeError{typ, pval.kind})
	}
}

/* *Interface is modelled after encoding/json */
func (p *Decoder) valueInterface(pval *plistValue) interface{} {
	switch pval.kind {
	case String:
		return pval.value.(string)
	case Integer:
		if pval.value.(signedInt).signed {
			return int64(pval.value.(signedInt).value)
		}
		return pval.value.(signedInt).value
	case Real:
		bits := pval.value.(sizedFloat).bits
		switch bits {
		case 32:
			return float32(pval.value.(sizedFloat).value)
		case 64:
			return pval.value.(sizedFloat).value
		}
	case Boolean:
		return pval.value.(bool)
	case Array:
		return p.arrayInterface(pval.value.([]*plistValue))
	case Dictionary:
		return p.dictionaryInterface(pval.value.(*dictionary))
	case Data:
		return pval.value.([]byte)
	case Date:
		return pval.value.(time.Time)
	}
	return nil
}

func (p *Decoder) arrayInterface(subvalues []*plistValue) []interface{} {
	out := make([]interface{}, len(subvalues))
	for i, subv := range subvalues {
		out[i] = p.valueInterface(subv)
	}
	return out
}

func (p *Decoder) dictionaryInterface(dict *dictionary) map[string]interface{} {
	out := make(map[string]interface{})
	for k, subv := range dict.m {
		out[k] = p.valueInterface(subv)
	}
	return out
}
