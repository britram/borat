package borat

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// structCBORSpec represents metadata for writing structures.
type structCBORSpec struct {
	tag            uint
	hasTag         bool
	intKeyForField map[string]int
	strKeyForField map[string]string
}

// TaggedElement is used to wrap elements which may be tagged for writing.
type TaggedElement struct {
	tag   CBORTag
	value interface{}
}

func (scs *structCBORSpec) usingIntKeys() bool {
	return scs.intKeyForField != nil
}

func (scs *structCBORSpec) learnStruct(t reflect.Type) {
	for i, n := 0, t.NumField(); i < n; i++ {
		f := t.Field(i)

		// only process fields that are exportable
		if f.PkgPath == "" {
			// check for a struct tag
			tag := f.Tag.Get("cbor")
			if tag != "" {
				// generate map key from tag
				if strings.HasPrefix(tag, "#") {
					// Integer tag; parse it
					intKey, err := strconv.Atoi(tag[1:len(tag)])
					if err != nil {
						panic(fmt.Sprintf("invalid integer key tag for %s.%s", t.Name(), f.Name))
					}
					if scs.strKeyForField != nil {
						panic(fmt.Sprintf("cannot mix integer and string keys in %s", t.Name()))
					}
					if scs.intKeyForField == nil {
						scs.intKeyForField = make(map[string]int)
					}
					scs.intKeyForField[f.Name] = intKey
				} else {
					if scs.intKeyForField != nil {
						panic(fmt.Sprintf("cannot mix integer and string keys in %s", t.Name()))
					}
					if scs.strKeyForField == nil {
						scs.strKeyForField = make(map[string]string)
					}
					scs.strKeyForField[f.Name] = tag
				}
			} else {
				// generate map key from name
				if scs.intKeyForField != nil {
					panic(fmt.Sprintf("cannot mix integer and string keys in %s", t.Name()))
				}
				if scs.strKeyForField == nil {
					scs.strKeyForField = make(map[string]string)
				}
				scs.strKeyForField[f.Name] = f.Name
			}
		} else if f.Name == "cborTag" {
			// structure indicates it would like to be tagged
			tag := f.Tag.Get("cbor")
			if tag != "" {
				// parse tag value as a base-10 int
				ct, err := strconv.Atoi(tag)
				if err != nil || ct < 0 {
					panic(fmt.Sprintf("cannot parse special struct member cborTag %s in %s", tag, t.Name()))
				}
				scs.tag = uint(ct)
				scs.hasTag = true
			}
		}
	}
}

func (scs *structCBORSpec) convertStructToIntMap(v reflect.Value) map[int]interface{} {
	if scs.intKeyForField == nil {
		panic(fmt.Sprintf("can't convert %s to integer-keyed map", v.Type().Name()))
	}

	out := make(map[int]interface{})

	for i, n := 0, v.NumField(); i < n; i++ {
		fieldName := v.Type().Field(i).Name
		fieldVal := v.Field(i)
		out[scs.intKeyForField[fieldName]] = fieldVal.Interface()
	}

	return out
}

func (scs *structCBORSpec) convertIntMapToStruct(in map[int]interface{}, out reflect.Value) {
	if scs.intKeyForField == nil {
		panic(fmt.Sprintf("can't parse int map for struct type %s", out.Type().Name()))
	}

	for i, n := 0, out.NumField(); i < n; i++ {
		fieldName := out.Type().Field(i).Name
		mapIdx := scs.intKeyForField[fieldName]
		// If this is a struct we should do something special here.
		if value, ok := in[mapIdx]; ok {
			// If this field is of type int but we have a uint64, we can cast
			// it, provided that it fits.
			if out.Field(i).Kind() == reflect.Int && reflect.ValueOf(value).Kind() == reflect.Uint64 {
				value = int(value.(uint64))
			}
			out.Field(i).Set(reflect.ValueOf(value))
		}
	}
}

func (scs *structCBORSpec) convertStructToStringMap(v reflect.Value, tagTypeMap map[reflect.Type]uint64) (map[string]TaggedElement, error) {
	if scs.strKeyForField == nil {
		panic(fmt.Sprintf("can't convert %s to string-keyed map", v.Type().Name()))
	}

	out := make(map[string]TaggedElement)

	for i, n := 0, v.NumField(); i < n; i++ {
		fieldName := v.Type().Field(i).Name
		if v.Type().Field(i).PkgPath != "" {
			continue
		}
		fieldVal := v.Field(i)
		// Do not tag structs over here because Marshal does that.
		elem := TaggedElement{}
		elem.value = fieldVal.Interface()
		out[scs.strKeyForField[fieldName]] = elem
	}

	return out, nil
}

// handleSlice sets the field referenced by out to the data in in,
// out should be a slice of some type and in should also be a []interface{}
func (scs *structCBORSpec) handleSlice(out reflect.Value, in []interface{}, registry map[uint64]reflect.Type) {
	if out.Kind() != reflect.Slice {
		panic("called nandleSlice on a non-slice")
	}
	k := out.Type().Elem().Kind()
	if k == reflect.Ptr {
		k = out.Type().Elem().Elem().Kind()
	}
	// If the slice is a slice of pointers,
	switch k {
	case reflect.Uint64, reflect.String:
		for i, e := range in {
			out.Index(i).Set(reflect.ValueOf(e))
		}
	case reflect.Struct:
		for i, e := range in {
			scs.convertStringMapToStruct(e.(map[string]TaggedElement), out.Index(i), registry)
		}
	case reflect.Slice:
		inIter := in
		for i, inElem := range inIter {
			slen := len(inElem.([]interface{}))
			slice := reflect.MakeSlice(out.Type().Elem(), slen, slen)
			out.Index(i).Set(slice)
			scs.handleSlice(out.Index(i), inElem.([]interface{}), registry)
		}
	default:
		panic(fmt.Sprintf("unsupported slice type: %v", k))
	}
}

func (scs *structCBORSpec) handleArray(out reflect.Value, in TaggedElement) {
	if out.Kind() != reflect.Array {
		panic(fmt.Sprintf("called handleArray on non-array type: %v", out.Kind()))
	}
	switch out.Type().Elem().Kind() {
	case reflect.Uint64, reflect.String:
		for i, e := range in.value.([]interface{}) {
			out.Index(i).Set(reflect.ValueOf(e))
		}
	case reflect.Uint32:
		for i, e := range in.value.([]interface{}) {
			dc := uint32(e.(uint64))
			out.Index(i).Set(reflect.ValueOf(dc))
		}
	case reflect.Uint16:
		for i, e := range in.value.([]interface{}) {
			dc := uint16(e.(uint64))
			out.Index(i).Set(reflect.ValueOf(dc))
		}
	case reflect.Uint8:
		for i, e := range in.value.([]interface{}) {
			dc := uint8(e.(uint64))
			out.Index(i).Set(reflect.ValueOf(dc))
		}
	case reflect.Struct:
		panic("not yet implemented")
	}
}

// out must be a value of type struct. If the current thing is an interface then it should be
// resolved to the actual type by the caller.
func (scs *structCBORSpec) convertStringMapToStruct(in map[string]TaggedElement, out reflect.Value, registry map[uint64]reflect.Type) {
	if scs.strKeyForField == nil {
		panic(fmt.Sprintf("cant parse string map for struct type %s", out.Type().Name()))
	}
	// Allocate the pointer if we were actually given a pointer to a struct.
	// Then we set out to the indirect of that pointer to reuse our exisitng logic.
	if out.Kind() == reflect.Ptr {
		out.Set(reflect.New(out.Type().Elem()))
		out = reflect.Indirect(out)
	}
	if out.Kind() != reflect.Struct {
		panic(fmt.Sprintf("cannot convertStringMapToStruct on non-struct: %v", out.Kind()))
	}
	for i, n := 0, out.NumField(); i < n; i++ {
		fieldName := out.Type().Field(i).Name
		mapIdx := scs.strKeyForField[fieldName]
		// If this is a struct we should do something special here.
		if elem, ok := in[mapIdx]; ok {
			// If this field is of type int but we have a uint64, we can cast
			// it, provided that it fits.
			if out.Field(i).Kind() == reflect.Int && reflect.ValueOf(elem.value).Kind() == reflect.Uint64 {
				elem.value = int(elem.value.(uint64))
			}
			if out.Field(i).Kind() == reflect.Slice {
				// We need to make a slice with the correct length and type.
				slen := len(elem.value.([]interface{}))
				slice := reflect.MakeSlice(out.Field(i).Type(), slen, slen)
				out.Field(i).Set(slice)
				scs.handleSlice(out.Field(i), elem.value.([]interface{}), registry) // Handle the elements of the slice.
			} else if out.Field(i).Kind() == reflect.Array {
				innerType := out.Field(i).Type().Elem()
				arr := reflect.New(reflect.ArrayOf(len(elem.value.([]interface{})), innerType)).Elem()
				out.Field(i).Set(arr)
				scs.handleArray(out.Field(i), elem)
			} else if out.Field(i).Kind() == reflect.Struct {
				scs.convertStringMapToStruct(elem.value.(map[string]TaggedElement), out.Field(i), registry)
			} else if out.Field(i).Kind() == reflect.Interface {
				concrete, ok := registry[uint64(elem.tag)]
				if !ok {
					panic(fmt.Sprintf("unsupported tag %d", elem.tag))
				}
				inst := reflect.New(concrete)
				childScs := structCBORSpec{}
				childScs.learnStruct(concrete)
				childScs.convertStringMapToStruct(elem.value.(map[string]TaggedElement), inst.Elem(), registry)
				out.Field(i).Set(inst)
			} else {
				out.Field(i).Set(reflect.ValueOf(elem.value))
			}
		}
	}
}
