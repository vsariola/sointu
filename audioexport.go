package sointu

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// Wav converts a stereo signal of 32-bit floats (L R L R..., length should be
// divisible by 2) into a valid WAV-file, returned as a []byte array.
//
// If pcm16 is set to true, the samples in the WAV-file will be 16-bit signed
// integers; otherwise the samples will be 32-bit floats
func Wav(buffer AudioBuffer, pcm16 bool) ([]byte, error) {
	buf := new(bytes.Buffer)
	wavHeader(len(buffer)*2, pcm16, buf)
	err := rawToBuffer(buffer, pcm16, buf)
	if err != nil {
		return nil, fmt.Errorf("Wav failed: %v", err)
	}
	return buf.Bytes(), nil
}

// Raw converts a stereo signal of 32-bit floats (L R L R..., length should be
// divisible by 2) into a raw audio file, returned as a []byte array.
//
// If pcm16 is set to true, the samples will be 16-bit signed integers;
// otherwise the samples will be 32-bit floats
func Raw(buffer AudioBuffer, pcm16 bool) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := rawToBuffer(buffer, pcm16, buf)
	if err != nil {
		return nil, fmt.Errorf("Raw failed: %v", err)
	}
	return buf.Bytes(), nil
}

func rawToBuffer(data AudioBuffer, pcm16 bool, buf *bytes.Buffer) error {
	var err error
	if pcm16 {
		int16data := make([][2]int16, len(data))
		for i, v := range data {
			int16data[i][0] = int16(clamp(int(v[0]*math.MaxInt16), math.MinInt16, math.MaxInt16))
			int16data[i][1] = int16(clamp(int(v[1]*math.MaxInt16), math.MinInt16, math.MaxInt16))
		}
		err = binary.Write(buf, binary.LittleEndian, int16data)
	} else {
		err = binary.Write(buf, binary.LittleEndian, data)
	}
	if err != nil {
		return fmt.Errorf("could not binary write data to binary buffer: %v", err)
	}
	return nil
}

// wavHeader writes a wave header for either float32 or int16 .wav file into the
// bytes.buffer. It needs to know the length of the buffer and assumes stereo
// sound, so the length in stereo samples (L + R) is bufferlength / 2. If pcm16
// = true, then the header is for int16 audio; pcm16 = false means the header is
// for float32 audio. Assumes 44100 Hz sample rate.
func wavHeader(bufferLength int, pcm16 bool, buf *bytes.Buffer) {
	// Refer to: http://www-mmsp.ece.mcgill.ca/Documents/AudioFormats/WAVE/WAVE.html
	numChannels := 2
	sampleRate := 44100
	var bytesPerSample, chunkSize, fmtChunkSize, waveFormat int
	var factChunk bool
	if pcm16 {
		bytesPerSample = 2
		chunkSize = 36 + bytesPerSample*bufferLength
		fmtChunkSize = 16
		waveFormat = 1 // PCM
		factChunk = false
	} else {
		bytesPerSample = 4
		chunkSize = 50 + bytesPerSample*bufferLength
		fmtChunkSize = 18
		waveFormat = 3 // IEEE float
		factChunk = true
	}
	buf.Write([]byte("RIFF"))
	binary.Write(buf, binary.LittleEndian, uint32(chunkSize))
	buf.Write([]byte("WAVE"))
	buf.Write([]byte("fmt "))
	binary.Write(buf, binary.LittleEndian, uint32(fmtChunkSize))
	binary.Write(buf, binary.LittleEndian, uint16(waveFormat))
	binary.Write(buf, binary.LittleEndian, uint16(numChannels))
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate*numChannels*bytesPerSample)) // avgBytesPerSec
	binary.Write(buf, binary.LittleEndian, uint16(numChannels*bytesPerSample))            // blockAlign
	binary.Write(buf, binary.LittleEndian, uint16(8*bytesPerSample))                      // bits per sample
	if fmtChunkSize > 16 {
		binary.Write(buf, binary.LittleEndian, uint16(0)) // size of extension
	}
	if factChunk {
		buf.Write([]byte("fact"))
		binary.Write(buf, binary.LittleEndian, uint32(4))            // fact chunk size
		binary.Write(buf, binary.LittleEndian, uint32(bufferLength)) // sample length
	}
	buf.Write([]byte("data"))
	binary.Write(buf, binary.LittleEndian, uint32(bytesPerSample*bufferLength))
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
