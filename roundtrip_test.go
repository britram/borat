package borat

import (
	"bytes"
	"reflect"
	"testing"
)

type ConvolutedIndirectable interface {
	ConvolutedIndirection() int
}

type Indirector struct {
	I int
}

func (i Indirector) ConvolutedIndirection() int { return i.I }

type One struct {
	A uint64
	B string
	C [4]byte
	D []string
	E []Two
	F []*Two
	G ConvolutedIndirectable
}

type Two struct {
	A string
}

func TestDirectInterface(t *testing.T) {
	var x ConvolutedIndirectable
	x = &Indirector{
		I: 1,
	}
	buf := bytes.NewBuffer([]byte{})
	writer := NewCBORWriter(buf)
	writer.RegisterCBORTag(1, Indirector{})
	reader := NewCBORReader(buf)
	reader.RegisterCBORTag(1, Indirector{})
	if err := writer.Marshal(x); err != nil {
		t.Fatalf("failed to marshal interface type: %v", err)
	}
	var r ConvolutedIndirectable
	if err := reader.Unmarshal(&r); err != nil {
		t.Fatalf("failed to unmarshal interface type: %v", err)
	}
	if !reflect.DeepEqual(x, r) {
		t.Errorf("got: %v, want %v", r, x)
	}
}

func TestRoundtripStructs(t *testing.T) {
	s := One{
		A: 1234,
		B: "Hello",
		C: [4]byte{0xC, 0xA, 0xF, 0xE},
		D: []string{"Lorem", "Ipsum"},
		E: []Two{Two{"First"}, Two{"Second"}, Two{"Third"}},
		F: []*Two{&Two{"Stuff"}},
		G: &Indirector{1},
	}
	buf := bytes.NewBuffer([]byte{})
	writer := NewCBORWriter(buf)
	writer.RegisterCBORTag(1, Indirector{})
	reader := NewCBORReader(buf)
	reader.RegisterCBORTag(1, Indirector{})
	if err := writer.Marshal(s); err != nil {
		t.Errorf("Marshal failed: %v", err)
	}
	var e One
	if err := reader.Unmarshal(&e); err != nil {
		t.Errorf("Unmarshal failed: %v", err)
	}
	if !reflect.DeepEqual(e, s) {
		t.Errorf("structs differ: got %v, want %v", e, s)
	}
}
