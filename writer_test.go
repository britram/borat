package borat_test

import (
	"bytes"
	"testing"

	"github.com/britram/borat"
)

func TestWriteIntegers(t *testing.T) {
	testPatterns := []struct {
		value int
		cbor  []byte
	}{
		{0, []byte{0x00}},
		{1, []byte{0x01}},
		{-1, []byte{0x20}},
		{33, []byte{0x18, 0x21}},
		{444, []byte{0x19, 0x01, 0xbc}},
		{-6666, []byte{0x39, 0x1a, 0x09}},
		{99999, []byte{0x1a, 0x00, 0x01, 0x86, 0x9f}},
		{123123123123, []byte{0x1b, 0x00, 00, 00, 0x1c, 0xaa, 0xb5, 0xc3, 0xb3}},
	}

	for i := range testPatterns {

		var buf bytes.Buffer

		w := borat.NewCBORWriter(&buf)

		w.WriteInt(testPatterns[i].value)

		if bytes.Compare(buf.Bytes(), testPatterns[i].cbor) != 0 {
			t.Errorf("error writing %d: expected %v, got %v",
				testPatterns[i].value, testPatterns[i].cbor, buf.Bytes())
		}
	}
}

func TestWriteStrings(t *testing.T) {
	testPatterns := []struct {
		value string
		cbor  []byte
	}{
		{"hello", []byte{0x65, 0x68, 0x65, 0x6c, 0x6c, 0x6f}},
		{"höi", []byte{0x64, 0x68, 0xc3, 0xb6, 0x69}},
	}

	for i := range testPatterns {

		var buf bytes.Buffer

		w := borat.NewCBORWriter(&buf)

		w.WriteString(testPatterns[i].value)

		if bytes.Compare(buf.Bytes(), testPatterns[i].cbor) != 0 {
			t.Errorf("error writing %s: expected %v, got %v",
				testPatterns[i].value, testPatterns[i].cbor, buf.Bytes())
		}
	}
}

func TestWriteArray(t *testing.T) {
	testPatterns := []struct {
		value []interface{}
		cbor  []byte
	}{
		{
			[]interface{}{"hello", "höi", "ciao"},
			[]byte{0x83, 0x65, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x64, 0x68, 0xc3, 0xb6, 0x69, 0x64, 0x63, 0x69, 0x61, 0x6f},
		},
	}

	for i := range testPatterns {

		var buf bytes.Buffer

		w := borat.NewCBORWriter(&buf)

		w.WriteArray(testPatterns[i].value)

		if bytes.Compare(buf.Bytes(), testPatterns[i].cbor) != 0 {
			t.Errorf("error writing %v: expected %v, got %v",
				testPatterns[i].value, testPatterns[i].cbor, buf.Bytes())
		}
	}
}

type untaggedTestStruct struct {
	NumericValue int
	StringValue  string
	BooleanValue bool
}

type stringTaggedTestStruct struct {
	NumericValue int    `cbor:"number"`
	StringValue  string `cbor:"string"`
	BooleanValue bool   `cbor:"truth"`
}

type intTaggedTestStruct struct {
	NumericValue int    `cbor:"#1"`
	StringValue  string `cbor:"#2"`
	BooleanValue bool   `cbor:"#3"`
}

func TestWriteStructs(t *testing.T) {
	testPatterns := []struct {
		value interface{}
		cbor  []byte
	}{
		{
			untaggedTestStruct{33, "møøse", false},
			[]byte{
				0xa3, 0x6c, 0x42, 0x6f, 0x6f, 0x6c, 0x65, 0x61,
				0x6e, 0x56, 0x61, 0x6c, 0x75, 0x65, 0xf4, 0x6c,
				0x4e, 0x75, 0x6d, 0x65, 0x72, 0x69, 0x63, 0x56,
				0x61, 0x6c, 0x75, 0x65, 0x18, 0x21, 0x6b, 0x53,
				0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c,
				0x75, 0x65, 0x67, 0x6d, 0xc3, 0xb8, 0xc3, 0xb8,
				0x73, 0x65,
			},
		},
		{
			stringTaggedTestStruct{7171, "spåm", true},
			[]byte{
				0xa3, 0x66, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72,
				0x19, 0x1c, 0x03, 0x66, 0x73, 0x74, 0x72, 0x69,
				0x6e, 0x67, 0x65, 0x73, 0x70, 0xc3, 0xa5, 0x6d,
				0x65, 0x74, 0x72, 0x75, 0x74, 0x68, 0xf5,
			},
		},
		{
			intTaggedTestStruct{998877, "surewhynot", false},
			[]byte{
				0xa3, 0x01, 0x1a, 0x00, 0x0f, 0x3d, 0xdd, 0x02,
				0x6a, 0x73, 0x75, 0x72, 0x65, 0x77, 0x68, 0x79,
				0x6e, 0x6f, 0x74, 0x03, 0xf4,
			},
		},
	}

	for i := range testPatterns {

		var buf bytes.Buffer

		w := borat.NewCBORWriter(&buf)

		w.Marshal(testPatterns[i].value)

		if bytes.Compare(buf.Bytes(), testPatterns[i].cbor) != 0 {
			t.Errorf("error writing %v: expected %v, got %v",
				testPatterns[i].value, testPatterns[i].cbor, buf.Bytes())
		}
	}
}
