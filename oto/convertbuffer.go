package oto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// FloatBufferTo16BitLE is a naive helper method to convert []float32 buffers to
// 16-bit little-endian integer buffers.
// TODO: optimize/refactor this, current is far from the best solution
func FloatBufferTo16BitLE(buff []float32) ([]byte, error) {
	var buf bytes.Buffer
	for i, v := range buff {
		var uv int16
		if v < -1.0 {
			uv = -math.MaxInt16
		} else if v > 1.0 {
			uv = math.MaxInt16
		} else {
			uv = int16(v * math.MaxInt16)
		}
		if err := binary.Write(&buf, binary.LittleEndian, uv); err != nil {
			return nil, fmt.Errorf("error converting buffer (@ %v, value %v) to bytes: %w", i, v, err)
		}
	}
	return buf.Bytes(), nil
}
