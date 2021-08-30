package sointu

import (
	"errors"
	"fmt"
	"math"
)

// Synth represents a state of a synthesizer, compiled from a Patch.
type Synth interface {
	// Render tries to fill a stereo signal buffer with sound from the
	// synthesizer, until either the buffer is full or a given number of
	// timesteps is advanced. In the process, it also fills the syncbuffer with
	// the values output by sync units. Normally, 1 sample = 1 unit of time, but
	// speed modulations may change this. It returns the number of samples
	// filled (! in stereo samples, so the buffer will have 2 * sample floats),
	// the number of sync outputs written, the number of time steps time
	// advanced, and a possible error.
	Render(buffer []float32, syncBuffer []float32, maxtime int) (sample int, syncs int, time int, err error)

	// Update recompiles a patch, but should maintain as much as possible of its
	// state as reasonable. For example, filters should keep their state and
	// delaylines should keep their content. Every change in the Patch triggers
	// an Update and if the Patch would be started fresh every time, it would
	// lead to very choppy audio.
	Update(patch Patch) error

	// Trigger triggers a note for a given voice. Called between synth.Renders.
	Trigger(voice int, note byte)

	// Release releases the currently playing note for a given voice. Called
	// between synth.Renders.
	Release(voice int)
}

// SynthService compiles a given Patch into a Synth, throwing errors if the
// Patch is malformed.
type SynthService interface {
	Compile(patch Patch) (Synth, error)
}

// Render fills an stereo audio buffer using a Synth, disregarding all syncs and
// time limits.
func Render(synth Synth, buffer []float32) error {
	s, _, _, err := synth.Render(buffer, nil, math.MaxInt32)
	if err != nil {
		return fmt.Errorf("sointu.Render failed: %v", err)
	}
	if s != len(buffer)/2 {
		return errors.New("in sointu.Render, synth.Render should have filled the whole buffer but did not")
	}
	return nil
}

// Play plays the Song using a given SynthService, returning the stereo audio
// buffer and the sync buffer as a result (and possible errors). Passing
// 'release' as true means that all the notes are released when the synth is
// created. The default behaviour during runtime rendering is to leave them
// playing, meaning that envelopes start attacking right away unless an explicit
// note release is put to every track.
func Play(synthService SynthService, song Song, release bool) ([]float32, []float32, error) {
	err := song.Validate()
	if err != nil {
		return nil, nil, err
	}
	synth, err := synthService.Compile(song.Patch)
	if err != nil {
		return nil, nil, fmt.Errorf("sointu.Play failed: %v", err)
	}
	if release {
		for i := 0; i < 32; i++ {
			synth.Release(i)
		}
	}
	curVoices := make([]int, len(song.Score.Tracks))
	for i := range curVoices {
		curVoices[i] = song.Score.FirstVoiceForTrack(i)
	}
	initialCapacity := song.Score.LengthInRows() * song.SamplesPerRow() * 2
	buffer := make([]float32, 0, initialCapacity)
	rowbuffer := make([]float32, song.SamplesPerRow()*2)
	numSyncs := song.Patch.NumSyncs()
	syncBuffer := make([]float32, 0, (song.Score.LengthInRows()*song.SamplesPerRow()+255)/256*(1+numSyncs))
	syncRowBuffer := make([]float32, ((song.SamplesPerRow()+255)/256)*(1+numSyncs))
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
			samples, syncs, time, err := synth.Render(rowbuffer, syncRowBuffer, song.SamplesPerRow()-rowtime)
			for i := 0; i < syncs; i++ {
				t := syncRowBuffer[i*(1+numSyncs)]
				t = (t+float32(rowtime))/(float32(song.SamplesPerRow())) + float32(row)
				syncRowBuffer[i*(1+numSyncs)] = t
			}
			if err != nil {
				return buffer, syncBuffer, fmt.Errorf("render failed: %v", err)
			}
			rowtime += time
			buffer = append(buffer, rowbuffer[:samples*2]...)
			syncBuffer = append(syncBuffer, syncRowBuffer[:syncs*(1+numSyncs)]...)
			if tries > 100 {
				return nil, nil, fmt.Errorf("Song speed modulation likely so slow that row never advances; error at pattern %v, row %v", pattern, patternRow)
			}
		}
	}
	return buffer, syncBuffer, nil
}
