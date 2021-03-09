package sointu

import (
	"errors"
	"fmt"
	"math"
)

type Synth interface {
	Render(buffer []float32, syncBuffer []float32, maxtime int) (sample int, syncs int, time int, err error)
	Update(patch Patch) error
	Trigger(voice int, note byte)
	Release(voice int)
}

type SynthService interface {
	Compile(patch Patch) (Synth, error)
}

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

func Play(synth Synth, song Song) ([]float32, []float32, error) {
	err := song.Validate()
	if err != nil {
		return nil, nil, err
	}
	curVoices := make([]int, len(song.Score.Tracks))
	for i := range curVoices {
		curVoices[i] = song.Score.FirstVoiceForTrack(i)
	}
	initialCapacity := song.Score.LengthInRows() * song.SamplesPerRow() * 2
	buffer := make([]float32, 0, initialCapacity)
	syncBuffer := make([]float32, 0, initialCapacity)
	rowbuffer := make([]float32, song.SamplesPerRow()*2)
	numSyncs := song.Patch.NumSyncs()
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
			syncBuffer = append(syncBuffer, syncRowBuffer[:syncs]...)
			if tries > 100 {
				return nil, nil, fmt.Errorf("Song speed modulation likely so slow that row never advances; error at pattern %v, row %v", pattern, patternRow)
			}
		}
	}
	return buffer, syncBuffer, nil
}
