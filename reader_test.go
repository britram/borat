package borat

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"testing"

	"gopkg.in/d4l3k/messagediff.v1"
)

type UnmarshalablePtr struct {
	a int
}

func (u *UnmarshalablePtr) UnmarshalCBOR(r *CBORReader) error {
	fmt.Printf("called our thing")
	a, err := r.ReadInt()
	if err != nil {
		return fmt.Errorf("expected no error but got: %v", err)
	}
	u.a = a
	return nil
}

func TestUnmarshalable(t *testing.T) {
	b := []byte{0x18, 0x7b}
	var dst UnmarshalablePtr
	r := NewCBORReader(bytes.NewReader(b))
	if err := r.Unmarshal(&dst); err != nil {
		t.Fatalf("expected no error unmarshaling but got %v", err)
	}
}

func cborDecoderHarness(t *testing.T, in []byte, expected interface{}) {
	r := NewCBORReader(bytes.NewReader(in))
	result, err := r.Read()
	if err != nil {
		t.Errorf("failed to decode %v: input % x, got error: %v", expected, in, err)
		return
	}
	if diff, equal := messagediff.PrettyDiff(result, expected); !equal {
		t.Errorf("decoder returned unexpected result: %#v diff=%s", result, diff)
	}
}

func cborDecoderUntagHarness(t *testing.T, in []byte, expected interface{}) {
	r := NewCBORReader(bytes.NewReader(in))
	result, err := r.Read()
	if err != nil {
		t.Errorf("failed to decode %v: input % x, got error %v", expected, in, err)
		return
	}
	if m, ok := result.(map[string]TaggedElement); ok {
		result = r.UntagStringMap(m)
	} else if a, ok := result.([]TaggedElement); ok {
		result = r.UntagArray(a)
	}
	if diff, equal := messagediff.PrettyDiff(result, expected); !equal {
		t.Errorf("decoder returned unexpected result: %#v diff=%s", result, diff)
	}
}

func cborDecoderHarnessExpectErr(t *testing.T, in []byte, errExpect error) {
	r := NewCBORReader(bytes.NewReader(in))
	if _, err := r.Read(); err != errExpect {
		t.Errorf("expected error %v but got %v", errExpect, err)
	}
}

func TestReadInt(t *testing.T) {
	testPatterns := []struct {
		cbor  []byte
		value interface{}
	}{
		{
			[]byte{0x01},
			1,
		},
		{
			[]byte{0x0a},
			10,
		},
		{
			[]byte{0x17},
			23,
		},
		{
			[]byte{0x18, 0x18},
			24,
		},
		{
			[]byte{0x18, 0x19},
			25,
		},
		{
			[]byte{0x18, 0x64},
			100,
		},
		{
			[]byte{0x19, 0x03, 0xe8},
			1000,
		},
		{
			[]byte{0x1a, 0x00, 0x0f, 0x42, 0x40},
			1000000,
		},
		{
			[]byte{0x1b, 0x00, 0x00, 0x00, 0xe8, 0xd4, 0xa5, 0x10, 0x00},
			1000000000000,
		},
		{
			[]byte{0x1b, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			9223372036854775807,
		},
		{
			[]byte{0x20},
			-1,
		},
		{
			[]byte{0x29},
			-10,
		},
		{
			[]byte{0x38, 0x63},
			-100,
		},
		{
			[]byte{0x39, 0x03, 0xe7},
			-1000,
		},
	}
	for i := range testPatterns {
		cborDecoderHarness(t, testPatterns[i].cbor, testPatterns[i].value)
	}
}

// We do not support 16-bit floats at the moment. Test for expected functionality.
func TestReadFloatUnsupported(t *testing.T) {
	testPatterns := []struct {
		cbor []byte
		err  error
	}{
		{
			// 0.0
			[]byte{0xf9, 0x00, 0x00},
			UnsupportedTypeReadError,
		},
		{
			// -0.0
			[]byte{0xf9, 0x80, 0x00},
			UnsupportedTypeReadError,
		},
		{
			// 65504.0
			[]byte{0xf9, 0x7b, 0xff},
			UnsupportedTypeReadError,
		},
	}
	for i := range testPatterns {
		cborDecoderHarnessExpectErr(t, testPatterns[i].cbor, testPatterns[i].err)
	}
}

// We only support IEEE754 single (32bit) or double (64bit) precision floats.
func TestReadFloatSupported(t *testing.T) {
	testPatterns := []struct {
		cbor  []byte
		value float64
	}{
		{
			[]byte{0xfb, 0x3f, 0xf1, 0x99, 0x99, 0x99, 0x99, 0x99, 0x9a},
			1.1,
		},
		{
			[]byte{0xfa, 0x47, 0xc3, 0x50, 0x00},
			100000.0,
		},
		{
			[]byte{0xfa, 0x7f, 0x7f, 0xff, 0xff},
			3.4028234663852886e+38,
		},
		{
			[]byte{0xfb, 0x7e, 0x37, 0xe4, 0x3c, 0x88, 0x00, 0x75, 0x9c},
			1.0e+300,
		},
		{
			[]byte{0xfa, 0x7f, 0x80, 0x00, 0x00},
			math.Inf(1),
		},
	}
	for i := range testPatterns {
		cborDecoderHarness(t, testPatterns[i].cbor, testPatterns[i].value)
	}
	// Manually test the NaN's since NaN != NaN.
	nans := [][]byte{
		[]byte{0xfa, 0x7f, 0xc0, 0x00, 0x00},
		[]byte{0xfb, 0x7f, 0xf8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
	for _, b := range nans {
		r := NewCBORReader(bytes.NewReader(b))
		if res, err := r.Read(); err != nil {
			t.Errorf("expected no error decoding NaN but got: %v", err)
		} else if !math.IsNaN(res.(float64)) {
			t.Errorf("expected NaN but got %f decoding %v", res, b)
		}
	}
}

func TestReadString(t *testing.T) {
	testPatterns := []struct {
		cbor  []byte
		value string
	}{
		{
			[]byte{0x60},
			"",
		},
		{
			[]byte{0x61, 0x61},
			"a",
		},
		{
			[]byte{0x64, 0x49, 0x45, 0x54, 0x46},
			"IETF",
		},
		{
			[]byte{0x67, 0x5A, 0xC3, 0xBC, 0x72, 0x69, 0x63, 0x68},
			"Zürich",
		},
	}
	for i := range testPatterns {
		cborDecoderHarness(t, testPatterns[i].cbor, testPatterns[i].value)
	}
}

func TestReadSlice(t *testing.T) {
	testPatterns := []struct {
		cbor  []byte
		value []interface{}
	}{
		{
			[]byte{0x83, 0x01, 0x02, 0x03},
			[]interface{}{1, 2, 3},
		},
		{
			[]byte{0x83, 0x61, 0x61, 0x61, 0x62, 0x61, 0x63},
			[]interface{}{"a", "b", "c"},
		},
		{
			[]byte{0x82, 0x83, 0x61, 0x61, 0x61, 0x62, 0x61, 0x63, 0x81, 0x61, 0x61},
			[]interface{}{[]interface{}{"a", "b", "c"}, []interface{}{"a"}},
		},
	}
	for i := range testPatterns {
		cborDecoderUntagHarness(t, testPatterns[i].cbor, testPatterns[i].value)
	}
}

func TestReadStringMap(t *testing.T) {
	testPatterns := []struct {
		cbor  []byte
		value map[string]interface{}
	}{
		{
			[]byte{0xA1, 0x61, 0x31, 0x01},
			map[string]interface{}{
				"1": 1,
			},
		},
		{
			[]byte{0xA2, 0x61, 0x31, 0x0A, 0x61, 0x32, 0x19, 0x0C, 0x45},
			map[string]interface{}{
				"1": 10,
				"2": 3141,
			},
		},
		{
			[]byte{0xA3, 0x61, 0x31, 0x0A, 0x61, 0x32, 0x62, 0x68, 0x69, 0x61,
				0x33, 0x83, 0x01, 0x02, 0x62, 0xC3, 0x9C},
			map[string]interface{}{
				"1": 10,
				"2": "hi",
				"3": []interface{}{1, 2, "Ü"},
			},
		},
	}
	for i := range testPatterns {
		r := NewCBORReader(bytes.NewReader(testPatterns[i].cbor))
		result, err := r.ReadStringMap()
		if err != nil {
			t.Errorf("failed to decode %v: input % x, got error %v", testPatterns[i].value, testPatterns[i].cbor, err)
			return
		}
		untagged := r.UntagStringMap(result)
		if diff, equal := messagediff.PrettyDiff(untagged, testPatterns[i].value); !equal {
			t.Errorf("decoder returned unexpected result: %#v diff=%s", result, diff)
		}
	}
}

func TestReadToStruct(t *testing.T) {
	data := []byte{0xA3, 0x61, 0x41, 0x65, 0x68, 0x65, 0x6C, 0x6C, 0x6F,
		0x61, 0x42, 0x19, 0x04, 0xD2, 0x61, 0x43, 0xF5}
	type A struct {
		A string
		B int
		C bool
	}
	want := &A{
		A: "hello",
		B: 1234,
		C: true,
	}
	got := &A{}
	r := NewCBORReader(bytes.NewReader(data))
	if err := r.Unmarshal(got); err != nil {
		t.Errorf("expected nil error from unmarshal but got: %v", err)
	}
	if ok := reflect.DeepEqual(want, got); !ok {
		t.Errorf("failed unmarshaling struct: want %+v, got %+v", want, got)
	}
	data2 := []byte{
		0xa3, 0x61, 0x41, 0x82, 0x01, 0x02, 0x61, 0x42,
		0x82, 0x82, 0x65, 0x48, 0x65, 0x6c, 0x6c, 0x6f,
		0x67, 0x47, 0x72, 0xc3, 0xbc, 0x65, 0x7a, 0x69,
		0x81, 0x63, 0xe2, 0x84, 0xa2, 0x61, 0x43, 0xa3,
		0x61, 0x41, 0x65, 0x68, 0x65, 0x6c, 0x6c, 0x6f,
		0x61, 0x42, 0x19, 0x04, 0xd2, 0x61, 0x43, 0xf5,
	}
	type B struct {
		A []int
		B [][]string
		C A
	}
	want2 := &B{
		A: []int{1, 2},
		B: [][]string{[]string{"Hello", "Grüezi"}, []string{"™"}},
		C: *want,
	}
	got2 := &B{}
	r = NewCBORReader(bytes.NewReader(data2))
	if err := r.Unmarshal(got2); err != nil {
		t.Errorf("expected nil error from unmarshal but got: %v", err)
	}
	if diff, equal := messagediff.PrettyDiff(want2, got2); !equal {
		t.Errorf("failed unmarshaling struct, got=%+v, diff=%s", got, diff)
	}
}
