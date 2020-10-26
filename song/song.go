package song

import (
	"errors"

	"github.com/vsariola/sointu/bridge"
)

type Track struct {
	NumVoices int
	Sequence  []byte
}

type Song struct {
	BPM      int
	Patterns [][]byte
	Tracks   []Track
	Patch    bridge.Patch
	Samples  int // -1 means calculate automatically, but you can also set it manually
}

func NewSong(bpm int, patterns [][]byte, tracks []Track, patch bridge.Patch) (*Song, error) {
	s := new(Song)
	s.BPM = bpm
	s.Patterns = patterns
	s.Tracks = tracks
	s.Patch = patch
	err := s.Validate()
	if err != nil {
		return nil, err
	}
	s.Samples = -1
	return s, nil
}

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

func (s *Song) Render() ([]float32, error) {
	err := s.Validate()
	if err != nil {
		return nil, err
	}
	synth := bridge.NewSynthState()
	synth.SetPatch(s.Patch)
	synth.SetSamplesPerRow(44100 * 60 / (s.BPM * 4))
	curVoices := make([]int, len(s.Tracks))
	for i := range curVoices {
		curVoices[i] = s.FirstTrackVoice(i)
	}
	samples := s.Samples
	if samples < 0 {
		samples = s.TotalRows() * s.SamplesPerRow()
	}
	buffer := make([]float32, samples*2)
	totaln := 0
	for row := 0; row < s.TotalRows(); row++ {
		patternRow := row % s.PatternRows()
		pattern := row / s.PatternRows()
		for t := range s.Tracks {
			note := s.Patterns[pattern][patternRow]
			if note == 1 { // anything but hold causes an action.
				continue // TODO: can hold be actually something else than 1?
			}
			synth.Release(curVoices[t])
			if note > 1 {
				curVoices[t]++
				first := s.FirstTrackVoice(t)
				if curVoices[t] >= first+s.Tracks[t].NumVoices {
					curVoices[t] = first
				}
				synth.Trigger(curVoices[t], note)
			}
		}
		n, _, _ := synth.Render(buffer[2*totaln:])
		totaln += n
	}
	return buffer, nil
}
