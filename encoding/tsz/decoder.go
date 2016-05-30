package tsz

import (
	"io"
	"time"

	"code.uber.internal/infra/memtsdb/encoding"
)

// decoder implements the decoding scheme in the facebook paper
// "Gorilla: A Fast, Scalable, In-Memory Time Series Database".
type decoder struct {
	tu time.Duration // time unit
}

// NewDecoder creates a decoder.
func NewDecoder(timeUnit time.Duration) encoding.Decoder {
	return &decoder{timeUnit}
}

// Decode decodes the encoded data captured by the reader.
func (dec *decoder) Decode(r io.Reader) encoding.Iterator {
	return newIterator(r, dec.tu)
}