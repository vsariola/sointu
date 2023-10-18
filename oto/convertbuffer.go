package oto

import (
	"math"

	"github.com/vsariola/sointu"
)

// FloatBufferTo16BitLE is a naive helper method to convert []float32 buffers to
// 16-bit little-endian, but encoded in byte buffer
//
// Appends the encoded bytes into "to" slice, allowing you to preallocate the
// capacity or just use nil
func FloatBufferTo16BitLE(from sointu.AudioBuffer, to []byte) []byte {
	for _, v := range from {
		left := to16BitSample(v[0])
		right := to16BitSample(v[1])
		to = append(to, byte(left&255), byte(left>>8), byte(right&255), byte(right>>8))
	}
	return to
}

// convert float32 to int16, clamping to min and max
func to16BitSample(v float32) int16 {
	if v < -1.0 {
		return -math.MaxInt16
	}
	if v > 1.0 {
		return math.MaxInt16
	}
	return int16(v * math.MaxInt16)
}
