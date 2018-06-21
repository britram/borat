package borat

import (
	"bytes"
	"reflect"
	"testing"

	"gopkg.in/d4l3k/messagediff.v1"
)

type IntTypedef uint8

type ConvolutedIndirectable interface {
	ConvolutedIndirection() int
}

type Indirector struct {
	I int
}

type Indirector2 struct {
	Something int
}

func (i Indirector) ConvolutedIndirection() int { return i.I }

func (i Indirector2) ConvolutedIndirection() int { return i.Something }

type One struct {
	A uint64
	B string
	C [4]byte
	D []string
	E []Two
	F []*Two
	G ConvolutedIndirectable
	H []ConvolutedIndirectable
	I IntTypedef
}

type Two struct {
	A string
}

func TestDirectInterface(t *testing.T) {
	var x ConvolutedIndirectable
	x = &Indirector{
		I: 1,
	}
	var y ConvolutedIndirectable
	y = &Indirector2{
		Something: 123,
	}
	buf := bytes.NewBuffer([]byte{})
	writer := NewCBORWriter(buf)
	writer.RegisterCBORTag(CBORTag(1), Indirector{})
	writer.RegisterCBORTag(CBORTag(2), Indirector2{})
	reader := NewCBORReader(buf)
	reader.RegisterCBORTag(CBORTag(1), Indirector{})
	reader.RegisterCBORTag(CBORTag(2), Indirector2{})
	var r ConvolutedIndirectable
	if err := writer.Marshal(x); err != nil {
		t.Errorf("failed to marshal interface type %T: %v", x, err)
		goto testY
	}
	if err := reader.Unmarshal(&r); err != nil {
		t.Errorf("failed to unmarshal interface type: %v", err)
		goto testY
	}
	if !reflect.DeepEqual(x, r) {
		t.Errorf("got: %v, want %v", r, x)
	}
testY:
	if err := writer.Marshal(y); err != nil {
		t.Fatalf("failed to marshal interface type %T: %v", y, err)
	}
	if err := reader.Unmarshal(&r); err != nil {
		t.Fatalf("failed to unmarshal interface type 2: %v", err)
	}
	if !reflect.DeepEqual(y, r) {
		t.Errorf("got %v, want %v", r, y)
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
		G: Indirector{1},
		H: []ConvolutedIndirectable{&Indirector{21}, &Indirector{31}},
		I: IntTypedef(32),
	}
	buf := bytes.NewBuffer([]byte{})
	writer := NewCBORWriter(buf)
	writer.RegisterCBORTag(0xaa, Two{})
	writer.RegisterCBORTag(0xbb, &Indirector{})
	writer.RegisterCBORTag(0xcc, uint8(0))
	writer.RegisterCBORTag(0xdd, Indirector{})
	reader := NewCBORReader(buf)
	reader.RegisterCBORTag(0xaa, Two{})
	reader.RegisterCBORTag(0xbb, &Indirector{})
	reader.RegisterCBORTag(0xcc, uint8(0))
	reader.RegisterCBORTag(0xdd, Indirector{})
	if err := writer.Marshal(s); err != nil {
		t.Errorf("Marshal failed: %v", err)
	}
	var e One
	if err := reader.Unmarshal(&e); err != nil {
		t.Errorf("Unmarshal failed: %v", err)
	}
	if diff, ok := messagediff.PrettyDiff(e, s); !ok {
		t.Errorf("structs differ, diff: %v", diff)
	}
}
