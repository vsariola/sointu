package oto

import (
	"math"
)

// FloatBufferTo16BitLE is a naive helper method to convert []float32 buffers to
// 16-bit little-endian, but encoded in byte buffer
//
// Appends the encoded bytes into "to" slice, allowing you to preallocate the
// capacity or just use nil
func FloatBufferTo16BitLE(from []float32, to []byte) []byte {
	for _, v := range from {
		var uv int16
		if v < -1.0 {
			uv = -math.MaxInt16 // we are a bit lazy: -1.0 is encoded as -32767, as this makes math easier, and -32768 is unused
		} else if v > 1.0 {
			uv = math.MaxInt16
		} else {
			uv = int16(v * math.MaxInt16)
		}
		to = append(to, byte(uv&255), byte(uv>>8))
	}
	return to
}
