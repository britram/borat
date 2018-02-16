package borat

import (
	"encoding/binary"
	"errors"
	"io"
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
	in io.Reader
}

func NewCBORReader(in io.Reader) *CBORReader {
	r := new(CBORReader)
	r.in = in
	return r
}

func (r *CBORReader) readInt() (int, error) {
	// read the first byte to see how much int to read
	b := make([]byte, 1)
	n, err := r.in.Read(b)
	if n != 1 {
		return 0, ShortReadError
	} else if err != nil {
		return 0, err
	}

	// determine if negative
	var negate bool
	switch b[0] & ^byte(majorMask) {
	case majorUnsigned:
		negate = false
	case majorNegative:
		negate = true
	default:
		return 0, CBORTypeReadError
	}

	// read the integer
	var i int
	switch {

	case b[0]&majorMask < 23:
		i = int(b[0] & majorMask)

	case b[0]&majorMask == 24:
		b := make([]byte, 1)
		n, err := r.in.Read(b)
		if n != len(b) {
			return 0, ShortReadError
		} else if err != nil {
			return 0, err
		}
		i = int(b[0])

	case b[0]&majorMask == 25:
		b := make([]byte, 2)
		n, err := r.in.Read(b)
		if n != len(b) {
			return 0, ShortReadError
		} else if err != nil {
			return 0, err
		}
		i = int(binary.BigEndian.Uint16(b))

	case b[0]&majorMask == 26:
		b := make([]byte, 4)
		n, err := r.in.Read(b)
		if n != len(b) {
			return 0, ShortReadError
		} else if err != nil {
			return 0, err
		}
		i = int(binary.BigEndian.Uint32(b))

	case b[0]&majorMask == 27:
		b := make([]byte, 8)
		n, err := r.in.Read(b)
		if n != len(b) {
			return 0, ShortReadError
		} else if err != nil {
			return 0, err
		}
		i = int(binary.BigEndian.Uint64(b))

	default:
		return 0, InvalidCBORError
	}

	// negate if necessary and return
	if negate {
		i = -1 - i
	}

	return i, nil
}

func (r *CBORReader) readFloat() (float64, error) {
	// read the first byte to see how much float to read
	b := make([]byte, 1)
	n, err := r.in.Read(b)
	if n != 1 {
		return 0, ShortReadError
	} else if err != nil {
		return 0, err
	}

	return 0, nil

}
