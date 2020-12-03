package go4k

import (
	"errors"
	"fmt"
)

type Song struct {
	BPM         int
	Patterns    [][]byte
	Tracks      []Track
	Patch       Patch
	Output16Bit bool
}

func (s *Song) PatternRows() int {
	return len(s.Patterns[0])
}

func (s *Song) SequenceLength() int {
	return len(s.Tracks[0].Sequence)
}

func (s *Song) TotalRows() int {
	return s.PatternRows() * s.SequenceLength()
}

func (s *Song) SamplesPerRow() int {
	return 44100 * 60 / (s.BPM * 4)
}

func (s *Song) FirstTrackVoice(track int) int {
	ret := 0
	for _, t := range s.Tracks[:track] {
		ret += t.NumVoices
	}
	return ret
}

// TBD: Where shall we put methods that work on pure domain types and have no dependencies
// e.g. Validate here
func (s *Song) Validate() error {
	if s.BPM < 1 {
		return errors.New("BPM should be > 0")
	}
	for i := range s.Patterns[:len(s.Patterns)-1] {
		if len(s.Patterns[i]) != len(s.Patterns[i+1]) {
			return errors.New("Every pattern should have the same length")
		}
	}
	for i := range s.Tracks[:len(s.Tracks)-1] {
		if len(s.Tracks[i].Sequence) != len(s.Tracks[i+1].Sequence) {
			return errors.New("Every track should have the same sequence length")
		}
	}
	totalTrackVoices := 0
	for _, track := range s.Tracks {
		totalTrackVoices += track.NumVoices
		for _, p := range track.Sequence {
			if p < 0 || int(p) >= len(s.Patterns) {
				return errors.New("Tracks use a non-existing pattern")
			}
		}
	}
	if totalTrackVoices > s.Patch.TotalVoices() {
		return errors.New("Tracks use too many voices")
	}
	return nil
}

func Play(synth Synth, song Song) ([]float32, error) {
	err := song.Validate()
	if err != nil {
		return nil, err
	}
	curVoices := make([]int, len(song.Tracks))
	for i := range curVoices {
		curVoices[i] = song.FirstTrackVoice(i)
	}
	initialCapacity := song.TotalRows() * song.SamplesPerRow() * 2
	buffer := make([]float32, 0, initialCapacity)
	rowbuffer := make([]float32, song.SamplesPerRow()*2)
	for row := 0; row < song.TotalRows(); row++ {
		patternRow := row % song.PatternRows()
		pattern := row / song.PatternRows()
		for t := range song.Tracks {
			patternIndex := song.Tracks[t].Sequence[pattern]
			note := song.Patterns[patternIndex][patternRow]
			if note == 1 { // anything but hold causes an action.
				continue // TODO: can hold be actually something else than 1?
			}
			synth.Release(curVoices[t])
			if note > 1 {
				curVoices[t]++
				first := song.FirstTrackVoice(t)
				if curVoices[t] >= first+song.Tracks[t].NumVoices {
					curVoices[t] = first
				}
				synth.Trigger(curVoices[t], note)
			}
		}
		tries := 0
		for rowtime := 0; rowtime < song.SamplesPerRow(); {
			samples, time, _ := synth.Render(rowbuffer, song.SamplesPerRow()-rowtime)
			rowtime += time
			buffer = append(buffer, rowbuffer[:samples*2]...)
			if tries > 100 {
				return nil, fmt.Errorf("Song speed modulation likely so slow that row never advances; error at pattern %v, row %v", pattern, patternRow)
			}
		}
	}
	return buffer, nil
}
