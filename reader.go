package borat

import "io"

type CBORReader struct {
	in io.Reader
}

func NewCBORReader(in io.Reader) *CBORReader {
	r := new(CBORReader)
	r.in = in
	return r
}
