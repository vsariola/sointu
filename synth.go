package sointu

import (
	"errors"
	"fmt"
	"math"
)

type (
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

// Render fills an stereo audio buffer using a Synth, disregarding all syncs and
// time limits.
func Render(synth Synth, buffer AudioBuffer) error {
	s, _, err := synth.Render(buffer, math.MaxInt32)
	if err != nil {
		return fmt.Errorf("sointu.Render failed: %v", err)
	}
	if s != len(buffer) {
		return errors.New("in sointu.Render, synth.Render should have filled the whole buffer but did not")
	}
	return nil
}

// Play plays the Song by first compiling the patch with the given Synther,
// returning the stereo audio buffer as a result (and possible errors).
func Play(synther Synther, song Song) (AudioBuffer, error) {
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
	}
	return buffer, nil
}
