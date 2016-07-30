package ebml

import "io"

// Marshaler is the interface implemented by objects that can marshal themselves into valid EBML.
type Marshaler interface {
	MarshalEBML(enc *Encoder) error
}

// An Encoder writes EBML elements to an output stream.
type Encoder struct {
	w io.Writer
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	enc := new(Encoder)
	enc.w = w
	return enc
}

// Encode writes the EBML encoding of v to the stream, followed by a newline character.
func (enc *Encoder) Encode(v interface{}) error {
	return nil
}
