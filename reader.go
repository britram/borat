package borat

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
)

var ShortReadError error
var CBORTypeReadError error
var InvalidCBORError error

func init() {
	ShortReadError = errors.New("short read")
	CBORTypeReadError = errors.New("invalid CBOR type for typed read")
	InvalidCBORError = errors.New("invalid CBOR")
}

type CBORReader struct {
	in       io.Reader
	pushback byte
	pushed   bool
}

func NewCBORReader(in io.Reader) *CBORReader {
	r := new(CBORReader)
	r.in = in
	return r
}

func (r *CBORReader) readType() (byte, error) {
	b := make([]byte, 1)
	if r.pushed {
		b[0] = r.pushback
		r.pushed = false
	} else {
		n, err := r.in.Read(b)
		if n != 1 {
			return 0, ShortReadError
		} else if err != nil {
			return 0, err
		}
	}

	return b[0], nil
}

func (r *CBORReader) pushbackType(pushback byte) {
	r.pushback = pushback
	r.pushed = true
}

func (r *CBORReader) readBasicUnsigned(mt byte) (uint64, byte, bool, error) {
	// read the first byte to see how much int to read

	// byte 0 is the CBOR type
	ct, err := r.readType()
	if err != nil {
		return 0, 0, false, err
	}

	// check for negative if this is a straight integer
	var neg bool
	if mt == majorUnsigned {
		switch ct & majorMask {
		case majorUnsigned:
			neg = false
		case majorNegative:
			neg = true
		default:
			// type mismatch, push back
			r.pushbackType(ct)
			return 0, ct, false, CBORTypeReadError
		}
	} else {
		if ct&majorMask != mt {
			// type mismatch, push back
			r.pushbackType(ct)
			return 0, ct, false, CBORTypeReadError
		}
	}

	var u uint64
	switch {
	case ct&majorMask < 23:
		u = uint64(ct & majorMask)

	case ct&majorMask == 24:
		b := make([]byte, 1)
		n, err := r.in.Read(b)
		if n != len(b) {
			return 0, 0, false, ShortReadError
		} else if err != nil {
			return 0, 0, false, err
		}
		u = uint64(b[0])

	case ct&majorMask == 25:
		b := make([]byte, 2)
		n, err := r.in.Read(b)
		if n != len(b) {
			return 0, 0, false, ShortReadError
		} else if err != nil {
			return 0, 0, false, err
		}
		u = uint64(binary.BigEndian.Uint16(b))

	case ct&majorMask == 26:
		b := make([]byte, 4)
		n, err := r.in.Read(b)
		if n != len(b) {
			return 0, 0, false, ShortReadError
		} else if err != nil {
			return 0, 0, false, err
		}
		u = uint64(binary.BigEndian.Uint32(b))

	case ct&majorMask == 27:
		b := make([]byte, 8)
		n, err := r.in.Read(b)
		if n != len(b) {
			return 0, 0, false, ShortReadError
		} else if err != nil {
			return 0, 0, false, err
		}
		u = uint64(binary.BigEndian.Uint64(b))

	default:
		return 0, 0, false, InvalidCBORError
	}

	return u, ct, neg, nil
}

func (r *CBORReader) ReadInt() (int, error) {
	var i int
	u, _, neg, err := r.readBasicUnsigned(majorUnsigned)
	if err != nil {
		return 0, err
	}

	// negate if necessary and return
	if neg {
		i = -1 - int(u)
	} else {
		i = int(u)
	}

	return i, nil
}

func (r *CBORReader) ReadTag() (CBORTag, error) {
	u, _, _, err := r.readBasicUnsigned(majorTag)
	if err != nil {
		return 0, err
	}

	return CBORTag(u), nil
}

func (r *CBORReader) ReadFloat() (float64, error) {
	u, ct, _, err := r.readBasicUnsigned(majorOther)
	if err != nil {
		return 0, err
	}

	var f float64
	switch ct {
	case majorOther | 26:
		f = float64(math.Float32frombits(uint32(u)))
	case majorOther | 27:
		f = math.Float64frombits(u)
	default:
		r.pushbackType(ct)
		return 0, CBORTypeReadError
	}

	return f, nil
}

func (r *CBORReader) ReadBytes() ([]byte, error) {
	// read length
	u, _, _, err := r.readBasicUnsigned(majorBytes)
	if err != nil {
		return nil, err
	}

	// read u bytes and return them
	b := make([]byte, u)
	n, err := r.in.Read(b)
	if n != 1 {
		return nil, ShortReadError
	} else if err != nil {
		return nil, err
	}

	return b, nil
}

func (r *CBORReader) ReadString() (string, error) {
	// read length
	u, _, _, err := r.readBasicUnsigned(majorString)
	if err != nil {
		return "", err
	}

	// read u bytes and return them as a string
	b := make([]byte, u)
	n, err := r.in.Read(b)
	if n != 1 {
		return "", ShortReadError
	} else if err != nil {
		return "", err
	}

	return string(b), nil
}

func (r *CBORReader) ReadArray() ([]interface{}, error) {
	// read length
	u, _, _, err := r.readBasicUnsigned(majorArray)
	if err != nil {
		return nil, err
	}

	arraylen := int(u)

	// create an output value
	out := make([]interface{}, arraylen)

	// now read that many values
	for i := 0; i < arraylen; i++ {
		v, err := r.Read()
		if err != nil {
			return nil, err
		}
		out[i] = v
	}

	return out, nil
}

func (r *CBORReader) ReadStringArray() ([]string, error) {
	// read length
	u, _, _, err := r.readBasicUnsigned(majorArray)
	if err != nil {
		return nil, err
	}

	arraylen := int(u)

	// create an output value
	out := make([]string, arraylen)

	// now read that many values
	for i := 0; i < arraylen; i++ {
		v, err := r.ReadString()
		if err != nil {
			return nil, err
		}
		out[i] = v
	}

	return out, nil
}

func (r *CBORReader) ReadIntArray() ([]int, error) {
	// read length
	u, _, _, err := r.readBasicUnsigned(majorArray)
	if err != nil {
		return nil, err
	}

	arraylen := int(u)

	// create an output value
	out := make([]int, arraylen)

	// now read as many values as there should be
	for i := 0; i < arraylen; i++ {
		v, err := r.ReadInt()
		if err != nil {
			return nil, err
		}
		out[i] = v
	}

	return out, nil
}

func (r *CBORReader) ReadStringMap() (map[string]interface{}, error) {
	// read length
	u, _, _, err := r.readBasicUnsigned(majorArray)
	if err != nil {
		return nil, err
	}

	maplen := int(u)

	// create an output value
	out := make(map[string]interface{})

	// now read as many key/value pairs as there should be
	for i := 0; i < maplen; i++ {
		var ks string
		k, err := r.Read()
		if err != nil {
			return nil, err
		}
		switch k.(type) {
		case string:
			ks = k.(string)
		default:
			ks = fmt.Sprintf("%v", k)
		}

		v, err := r.Read()
		if err != nil {
			return nil, err
		}

		out[ks] = v
	}

	return out, nil
}

func (r *CBORReader) ReadIntMap() (map[int]interface{}, error) {
	// read length
	u, _, _, err := r.readBasicUnsigned(majorArray)
	if err != nil {
		return nil, err
	}

	maplen := int(u)

	// create an output value
	out := make(map[int]interface{})

	// now read as many key/value pairs as there should be
	for i := 0; i < maplen; i++ {
		k, err := r.ReadInt()
		if err != nil {
			return nil, err
		}

		v, err := r.Read()
		if err != nil {
			return nil, err
		}

		out[k] = v
	}

	return out, nil
}

// Read reads the next value as an arbitrary object from the CBOR reader. It
// returns a single interface{} of one of the following types, depending on the
// major type of the next CBOR object in the stream:
//
// - Unsigned (major 0): uint
// - Negative (major 1): int
// - Byte array (major 2): []byte
// - String (major 3): string
// - Array (major 4): []interface{}
// - Map (major 5): map[string]interface{}, with keys coerced to strings via Sprintf("%v").
// - Tag (major 6): CBORTag type
// - Other (major 7) float: float64
// - Other (major 7) true or false: bool
// - Other (major 7) nil: nil
// - anything else: currently an error

func (r *CBORReader) Read() (interface{}, error) {
	ct, err := r.readType()
	if err != nil {
		return nil, err
	}

	switch ct & majorMask {
	case majorUnsigned:
		r.pushbackType(ct)
		return r.ReadInt()
	case majorNegative:
		r.pushbackType(ct)
		return r.ReadInt()
	case majorBytes:
		r.pushbackType(ct)
		return r.ReadBytes()
	case majorString:
		r.pushbackType(ct)
		return r.ReadString()
	case majorArray:
		r.pushbackType(ct)
		return r.ReadArray()
	case majorMap:
		r.pushbackType(ct)
		return r.ReadStringMap()
	case majorTag:
		r.pushbackType(ct)
		return r.ReadTag()
	case majorOther:
		switch {
		case ct == majorOther|25 || ct == majorOther|26:
			r.pushbackType(ct)
			return r.ReadFloat()
		case ct == 0xf4:
			return false, nil
		case ct == 0xf5:
			return true, nil
		case ct == 0xf6:
			return nil, nil
		}
	}

	// if we're here, pretend this isn't CBOR
	return nil, InvalidCBORError
}

// Unmarshal attempts to read the next value from the CBOR reader and store it
// in the value pointed to by v, according to v's type. Returns
// CBORTypeReadError if the type does not match or cannot be made to match.
// Values are handled as in Marshal().
func (r *CBORReader) Unmarshal(x interface{}) error {

	pv := reflect.ValueOf(x)

	// make sure we have a pointer to a thing
	if pv.Kind() != reflect.Ptr || pv.IsNil() {
		return fmt.Errorf("cannot unmarshal CBOR to non-pointer type %v", pv.Type())
	}

	// if the type implements unmarshaler, just do that
	if pv.Type().Elem().Implements(reflect.TypeOf((*CBORMarshaler)(nil)).Elem()) {
		return pv.Elem().Interface().(CBORUnmarshaler).UnmarshalCBOR(r)
	}

	// otherwise, read value based on value's element kind
	switch pv.Elem().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := r.ReadInt()
		if err != nil {
			return err
		}
		pv.Elem().SetInt(int64(i))
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		i, err := r.ReadInt()
		if err != nil {
			return err
		}
		pv.Elem().SetUint(uint64(i))
		return nil
	case reflect.String:
		s, err := r.ReadString()
		if err != nil {
			return err
		}
		pv.Elem().SetString(s)
		return nil
	case reflect.Slice:
		// Work Pointer
		return fmt.Errorf("Cannot unmarshal objects of type %v from CBOR", pv.Type().Elem())
	case reflect.Array:
		return fmt.Errorf("Cannot unmarshal objects of type %v from CBOR", pv.Type().Elem())

	case reflect.Struct:
		return fmt.Errorf("Cannot unmarshal objects of type %v from CBOR", pv.Type().Elem())

	case reflect.Bool:
		return fmt.Errorf("Cannot unmarshal objects of type %v from CBOR", pv.Type().Elem())

	default:
		return fmt.Errorf("Cannot unmarshal objects of type %v from CBOR", pv.Type().Elem())
	}
}

type CBORUnmarshaler interface {
	UnmarshalCBOR(r *CBORReader) error
}
