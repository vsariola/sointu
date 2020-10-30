package go4k

import "errors"

type Song struct {
	BPM        int
	Patterns   [][]byte
	Tracks     []Track
	SongLength int // in samples, 0 means calculate automatically from BPM and Track lengths, but can also set manually
	Patch      Patch
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
	samples := song.SongLength
	if samples <= 0 {
		samples = song.TotalRows() * song.SamplesPerRow()
	}
	buffer := make([]float32, samples*2)
	totaln := 0
	rowtime := song.SamplesPerRow()
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
		samples, _, _ := synth.Render(buffer[2*totaln:], rowtime)
		totaln += samples
	}
	return buffer, nil
}
