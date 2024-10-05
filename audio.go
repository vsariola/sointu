package sointu

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

type (
	// AudioBuffer is a buffer of stereo audio samples of variable length, each
	// sample represented by [2]float32. [0] is left channel, [1] is right
	AudioBuffer [][2]float32

	CloserWaiter interface {
		io.Closer
		Wait()
	}

	// AudioContext represents the low-level audio drivers. There should be at
	// most one AudioContext at a time. The interface is implemented at least by
	// oto.OtoContext, but in future we could also mock it.
	//
	// AudioContext is used to play one or more AudioSources. Playing can be
	// stopped by closing the returned io.Closer.
	AudioContext interface {
		Play(r AudioSource) CloserWaiter
	}

	// AudioSource is an interface for reading audio samples into an
	// AudioBuffer. Returns error if the buffer is not filled.
	AudioSource interface {
		ReadAudio(buf AudioBuffer) error
	}

	BufferSource struct {
		buffer AudioBuffer
		pos    int
	}

	// Synth represents a state of a synthesizer, compiled from a Patch.
	Synth interface {
		// Render tries to fill a stereo signal buffer with sound from the
		// synthesizer, until either the buffer is full or a given number of
		// timesteps is advanced. Normally, 1 sample = 1 unit of time, but speed
		// modulations may change this. It returns the number of samples filled (in
		// stereo samples i.e. number of elements of AudioBuffer filled), the
		// number of sync outputs written, the number of time steps time advanced,
		// and a possible error.
		Render(buffer AudioBuffer, maxtime int) (sample int, time int, err error)

		// Update recompiles a patch, but should maintain as much as possible of its
		// state as reasonable. For example, filters should keep their state and
		// delaylines should keep their content. Every change in the Patch triggers
		// an Update and if the Patch would be started fresh every time, it would
		// lead to very choppy audio.
		Update(patch Patch, bpm int) error

		// Trigger triggers a note for a given voice. Called between synth.Renders.
		Trigger(voice int, note byte)

		// Release releases the currently playing note for a given voice. Called
		// between synth.Renders.
		Release(voice int)
	}

	// Synther compiles a given Patch into a Synth, throwing errors if the
	// Patch is malformed.
	Synther interface {
		Synth(patch Patch, bpm int) (Synth, error)
	}
)

// Play plays the Song by first compiling the patch with the given Synther,
// returning the stereo audio buffer as a result (and possible errors).
func Play(synther Synther, song Song, progress func(float32)) (AudioBuffer, error) {
	err := song.Validate()
	if err != nil {
		return nil, err
	}
	synth, err := synther.Synth(song.Patch, song.BPM)
	if err != nil {
		return nil, fmt.Errorf("sointu.Play failed: %v", err)
	}
	curVoices := make([]int, len(song.Score.Tracks))
	for i := range curVoices {
		curVoices[i] = song.Score.FirstVoiceForTrack(i)
	}
	initialCapacity := song.Score.LengthInRows() * song.SamplesPerRow()
	buffer := make(AudioBuffer, 0, initialCapacity)
	rowbuffer := make(AudioBuffer, song.SamplesPerRow())
	for row := 0; row < song.Score.LengthInRows(); row++ {
		patternRow := row % song.Score.RowsPerPattern
		pattern := row / song.Score.RowsPerPattern
		for t := range song.Score.Tracks {
			order := song.Score.Tracks[t].Order
			if pattern < 0 || pattern >= len(order) {
				continue
			}
			patternIndex := song.Score.Tracks[t].Order[pattern]
			patterns := song.Score.Tracks[t].Patterns
			if patternIndex < 0 || int(patternIndex) >= len(patterns) {
				continue
			}
			pattern := patterns[patternIndex]
			if patternRow < 0 || patternRow >= len(pattern) {
				continue
			}
			note := pattern[patternRow]
			if note > 0 && note <= 1 { // anything but hold causes an action.
				continue
			}
			synth.Release(curVoices[t])
			if note > 1 {
				curVoices[t]++
				first := song.Score.FirstVoiceForTrack(t)
				if curVoices[t] >= first+song.Score.Tracks[t].NumVoices {
					curVoices[t] = first
				}
				synth.Trigger(curVoices[t], note)
			}
		}
		tries := 0
		for rowtime := 0; rowtime < song.SamplesPerRow(); {
			samples, time, err := synth.Render(rowbuffer, song.SamplesPerRow()-rowtime)
			if err != nil {
				return buffer, fmt.Errorf("render failed: %v", err)
			}
			rowtime += time
			buffer = append(buffer, rowbuffer[:samples]...)
			if tries > 100 {
				return nil, fmt.Errorf("Song speed modulation likely so slow that row never advances; error at pattern %v, row %v", pattern, patternRow)
			}
		}
		if progress != nil {
			progress(float32(row+1) / float32(song.Score.LengthInRows()))
		}
	}
	return buffer, nil
}

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

func (b AudioBuffer) Source() *BufferSource {
	return &BufferSource{buffer: b}
}

// ReadAudio reads audio samples from an AudioSource into an AudioBuffer.
// Returns an error when the buffer is fully consumed.
func (a *BufferSource) ReadAudio(buf AudioBuffer) error {
	n := copy(buf, a.buffer[a.pos:])
	a.pos += n
	if a.pos >= len(a.buffer) {
		return io.EOF
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
