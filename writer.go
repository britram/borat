package borat

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CBORWriter writes CBOR to an output stream. It provides a relatively
// low-level interface, allowing the caller to write typed data to the stream as
// CBOR, as well as a higher-level Marshal interface which uses reflection to
// properly encode arbitrary objects as CBOR.
type CBORWriter struct {
	out io.Writer
}

func NewCBORWriter(out io.Writer) *CBORWriter {
	w := new(CBORWriter)
	w.out = out
	return w
}

const (
	TagDateTimeString = 0
	TagDateTimeEpoch  = 1
	TagURI            = 32
	TagBase64URL      = 33
	TagBase64         = 34
	TagUUID           = 37
)

const (
	majorUnsigned = 0x00
	majorNegative = 0x20
	majorBytes    = 0x40
	majorString   = 0x60
	majorArray    = 0x80
	majorMap      = 0xa0
	majorTag      = 0xc0
	majorOther    = 0xe0
)

func (w *CBORWriter) writeBasicInt(u uint, mt byte) error {
	var out []byte

	if u < 24 {
		out = []byte{mt | byte(u)}
	} else if u < math.MaxUint8 {
		out = []byte{mt | 24, byte(u)}
	} else if u < math.MaxUint16 {
		out = []byte{mt | 25, 0, 0}
		binary.BigEndian.PutUint16(out[1:3], uint16(u))
	} else if u < math.MaxUint32 {
		out = []byte{mt | 26, 0, 0, 0, 0}
		binary.BigEndian.PutUint32(out[1:5], uint32(u))
	} else {
		out = []byte{mt | 27, 0, 0, 0, 0, 0, 0, 0, 0}
		binary.BigEndian.PutUint64(out[1:9], uint64(u))
	}

	_, err := w.out.Write(out)
	return err
}

// WriteTag writes a CBOR tag to the output stream. CBOR tags are used to note the semantics of the following object.
func (w *CBORWriter) WriteTag(t uint) error {
	return w.writeBasicInt(t, majorTag)
}

// WriteInt writes an integer to the output stream.
func (w *CBORWriter) WriteInt(i int) error {
	var u uint
	var mt byte
	if i >= 0 {
		u = uint(i)
		mt = majorUnsigned
	} else {
		u = uint(-1 - i)
		mt = majorNegative
	}

	return w.writeBasicInt(u, mt)
}

// WriteFloat writes a floating point number to the output stream.
func (w *CBORWriter) WriteFloat(f float64) error {
	out := []byte{majorOther | 27, 0, 0, 0, 0, 0, 0, 0, 0}
	u := math.Float64bits(f)
	binary.BigEndian.PutUint64(out[1:9], u)

	_, err := w.out.Write(out)
	return err
}

func (w *CBORWriter) writeBasicBytes(b []byte, mt byte) error {
	if err := w.writeBasicInt(uint(len(b)), mt); err != nil {
		return err
	}

	_, err := w.out.Write(b)
	return err
}

// WriteBytes writes a byte array to the output stream.
func (w *CBORWriter) WriteBytes(b []byte) error {
	return w.writeBasicBytes(b, majorBytes)
}

// WriteString writes a string to the output stream.
func (w *CBORWriter) WriteString(s string) error {
	return w.writeBasicBytes([]byte(s), majorString)
}

// WriteBool writes a boolean value to the output stream.
func (w *CBORWriter) WriteBool(b bool) error {
	out := []byte{0xf4}
	if b {
		out[0] = 0xf5
	}

	_, err := w.out.Write(out)
	return err
}

func (w *CBORWriter) WriteTimeNumeric(t time.Time) error {
	if err := w.WriteTag(TagDateTimeEpoch); err != nil {
		return err
	}

	return w.WriteInt(int(t.Unix()))
}

// WriteNil writes a nil to the output stream
func (w *CBORWriter) WriteNil() error {
	out := []byte{0xf6}
	_, err := w.out.Write(out)
	return err
}

// WriteArray writes an arbitrary array to the output stream. Each of the
// elements of the array will be reflected and written as appropriate.
func (w *CBORWriter) WriteArray(a []interface{}) error {
	if err := w.writeBasicInt(uint(len(a)), majorArray); err != nil {
		return err
	}

	for i := range a {
		if err := w.Marshal(a[i]); err != nil {
			return err
		}
	}

	return nil
}

// WriteStringMap writes a map keyed by strings to arbitrary types to the output
// stream. Each of the values of the map will be reflected and written as
// appropriate.
func (w *CBORWriter) WriteStringMap(m map[string]interface{}) error {
	if err := w.writeBasicInt(uint(len(m)), majorMap); err != nil {
		return err
	}

	// get sorted keys array
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	// serialize based on ordered keys
	for _, k := range keys {
		if err := w.WriteString(k); err != nil {
			return err
		}
		if err := w.Marshal(m[k]); err != nil {
			return err
		}
	}

	return nil
}

// WriteIntMap writes a map keyed by integers to arbitrary types to the output
// stream. Each of the values of the map will be reflected and written as
// appropriate.
func (w *CBORWriter) WriteIntMap(m map[int]interface{}) error {
	if err := w.writeBasicInt(uint(len(m)), majorMap); err != nil {
		return err
	}

	// get sorted keys array
	keys := make([]int, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Ints(keys)

	// serialize based on ordered keys
	for _, k := range keys {
		if err := w.WriteInt(k); err != nil {
			return err
		}
		if err := w.Marshal(m[k]); err != nil {
			return err
		}
	}

	return nil
}

// Marshal marshals an arbitrary object to the output stream using reflection.
// If the object is a primitive type, it will be marshaled as such. If it
// implements CBORMarshaler, its MarshalCBOR function will be called. If the
// object is a structure with CBOR struct tags, those struct tags will be used.
// If the object is a struct without CBOR struct tags, the struct will be
// marshaled as a map of strings to objects using the names of the public
// members of the struct.
//
func (w *CBORWriter) Marshal(x interface{}) error {

	v := reflect.ValueOf(x)

	// if the type implements marshaler, just do that
	if v.Type().Implements(reflect.TypeOf((*CBORMarshaler)(nil)).Elem()) {
		return v.Interface().(CBORMarshaler).MarshalCBOR(w)
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return w.WriteInt(int(v.Int()))
	case reflect.Bool:
		return w.WriteBool(v.Bool())
	case reflect.String:
		return w.WriteString(v.String())
	case reflect.Slice:
		// treat byte slices specially
		if v.Type() == reflect.TypeOf([]byte{}) {
			return w.WriteBytes(v.Bytes())
		} else {
			return w.WriteArray(v.Interface().([]interface{}))
		}
	case reflect.Array:
		// FIXME basically, treat these as fixed-length slices...
		panic("array marshaling not yet supported")
	case reflect.Struct:
		// treat times sepcially
		if v.Type() == reflect.TypeOf(time.Time{}) {
			return w.WriteTimeNumeric(v.Interface().(time.Time))
		} else {
			return w.writeReflectedStruct(v)
		}
	default:
	}

	return nil
}

type structFieldKeyMap struct {
	intKeyForField map[string]int
	strKeyForField map[string]string
}

func (sfk *structFieldKeyMap) usingIntKeys() bool {
	return sfk.intKeyForField != nil
}

func (sfk *structFieldKeyMap) learnStruct(t reflect.Type) {
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
					if sfk.strKeyForField != nil {
						panic(fmt.Sprintf("cannot mix integer and string keys in %s", t.Name()))
					}
					if sfk.intKeyForField == nil {
						sfk.intKeyForField = make(map[string]int)
					}
					sfk.intKeyForField[f.Name] = intKey
				} else {
					if sfk.intKeyForField != nil {
						panic(fmt.Sprintf("cannot mix integer and string keys in %s", t.Name()))
					}
					if sfk.strKeyForField == nil {
						sfk.strKeyForField = make(map[string]string)
					}
					sfk.strKeyForField[f.Name] = tag
				}
			} else {
				// generate map key from name
				if sfk.intKeyForField != nil {
					panic(fmt.Sprintf("cannot mix integer and string keys in %s", t.Name()))
				}
				if sfk.strKeyForField == nil {
					sfk.strKeyForField = make(map[string]string)
				}
				sfk.strKeyForField[f.Name] = f.Name
			}
		}
	}
}

func (sfk *structFieldKeyMap) convertStructToIntMap(v reflect.Value) map[int]interface{} {
	if sfk.intKeyForField == nil {
		panic(fmt.Sprintf("can't convert %s to integer-keyed map", v.Type().Name()))
	}

	out := make(map[int]interface{})

	for i, n := 0, v.NumField(); i < n; i++ {
		fieldName := v.Type().Field(i).Name
		fieldVal := v.Field(i)
		out[sfk.intKeyForField[fieldName]] = fieldVal.Interface()
	}

	return out
}

func (sfk *structFieldKeyMap) convertStructToStringMap(v reflect.Value) map[string]interface{} {
	if sfk.strKeyForField == nil {
		panic(fmt.Sprintf("can't convert %s to string-keyed map", v.Type().Name()))
	}

	out := make(map[string]interface{})

	for i, n := 0, v.NumField(); i < n; i++ {
		fieldName := v.Type().Field(i).Name
		fieldVal := v.Field(i)
		out[sfk.strKeyForField[fieldName]] = fieldVal.Interface()
	}

	return out
}

func (w *CBORWriter) writeReflectedStruct(v reflect.Value) error {

	// FIXME we probably want to cache this
	// Generate map containing serialization names for fields
	var sfk structFieldKeyMap
	sfk.learnStruct(v.Type())

	// and write either an int map or a string map
	if sfk.usingIntKeys() {
		return w.WriteIntMap(sfk.convertStructToIntMap(v))
	} else {
		return w.WriteStringMap(sfk.convertStructToStringMap(v))
	}

}

// CBORMarshaler represents an object that can write itself to a CBORWriter
type CBORMarshaler interface {
	MarshalCBOR(w *CBORWriter) error
}
