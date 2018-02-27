package borat

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

func cborDecoderHarness(t *testing.T, in []byte, expected interface{}) {
	r := NewCBORReader(bytes.NewReader(in))
	result, err := r.Read()
	if err != nil {
		t.Errorf("failed to decode %v: want %v, got error: %v", expected, in, err)
		return
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Decoder returned unexpected result: want %v, got %v", expected, result)
	}
}

func TestReadInt(t *testing.T) {
	testPatterns := []struct {
		cbor  []byte
		value interface{}
	}{
		{
			[]byte{0x01},
			uint64(1),
		},
		{
			[]byte{0x0a},
			uint64(10),
		},
		{
			[]byte{0x17},
			uint64(23),
		},
		{
			[]byte{0x18, 0x18},
			uint64(24),
		},
		{
			[]byte{0x18, 0x19},
			uint64(25),
		},
		{
			[]byte{0x18, 0x64},
			uint64(100),
		},
		{
			[]byte{0x19, 0x03, 0xe8},
			uint64(1000),
		},
		{
			[]byte{0x1a, 0x00, 0x0f, 0x42, 0x40},
			uint64(1000000),
		},
		{
			[]byte{0x1b, 0x00, 0x00, 0x00, 0xe8, 0xd4, 0xa5, 0x10, 0x00},
			uint64(1000000000000),
		},
		{
			[]byte{0x1b, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			uint64(18446744073709551615),
		},
	}

	for i := range testPatterns {
		cborDecoderHarness(t, testPatterns[i].cbor, testPatterns[i].value)
	}
}

func TestReadTime(t *testing.T) {
	testPatterns := []struct {
		cbor  []byte
		value time.Time
	}{
		{
			[]byte{0xc1, 0x1a, 0x51, 0x4b, 0x67, 0xb0},
			time.Unix(1363896240, 0),
		},
	}

	for i := range testPatterns {
		cborDecoderHarness(t, testPatterns[i].cbor, testPatterns[i].value)
	}
}
