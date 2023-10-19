package sointu

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

type (
	// AudioBuffer is a buffer of stereo audio samples of variable length, each
	// sample represented by [2]float32. [0] is left channel, [1] is right
	AudioBuffer [][2]float32

	// AudioOutput represents something where we can send audio e.g. audio output.
	// WriteAudio should block if not ready to accept audio e.g. buffer full.
	AudioOutput interface {
		WriteAudio(buffer AudioBuffer) error
		Close() error
	}

	// AudioContext represents the low-level audio drivers. There should be at most
	// one AudioContext at a time. The interface is implemented at least by
	// oto.OtoContext, but in future we could also mock it.
	//
	// AudioContext is used to create one or more AudioOutputs with Output(); each
	// can be used to output separate sound & closed when done.
	AudioContext interface {
		Output() AudioOutput
		Close() error
	}
)

// Fill fills the AudioBuffer using a Synth, disregarding all syncs and time
// limits. Note that this will change the state of the Synth.
func (buffer AudioBuffer) Fill(synth Synth) error {
	s, _, err := synth.Render(buffer, math.MaxInt32)
	if err != nil {
		return fmt.Errorf("synth.Render failed: %v", err)
	}
	if s != len(buffer) {
		return errors.New("in AudioBuffer.Fill, synth.Render should have filled the whole buffer but did not")
	}
	return nil
}

// Wav converts an AudioBuffer into a valid WAV-file, returned as a []byte
// array.
//
// If pcm16 is set to true, the samples in the WAV-file will be 16-bit signed
// integers; otherwise the samples will be 32-bit floats
func (buffer AudioBuffer) Wav(pcm16 bool) ([]byte, error) {
	buf := new(bytes.Buffer)
	wavHeader(len(buffer)*2, pcm16, buf)
	err := buffer.rawToBuffer(pcm16, buf)
	if err != nil {
		return nil, fmt.Errorf("Wav failed: %v", err)
	}
	return buf.Bytes(), nil
}

// Raw converts an AudioBuffer into a raw audio file, returned as a []byte
// array.
//
// If pcm16 is set to true, the samples will be 16-bit signed integers;
// otherwise the samples will be 32-bit floats
func (buffer AudioBuffer) Raw(pcm16 bool) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := buffer.rawToBuffer(pcm16, buf)
	if err != nil {
		return nil, fmt.Errorf("Raw failed: %v", err)
	}
	return buf.Bytes(), nil
}

func (data AudioBuffer) rawToBuffer(pcm16 bool, buf *bytes.Buffer) error {
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
