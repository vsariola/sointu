package sointu

import (
	"errors"
)

type Song struct {
	BPM         int
	RowsPerBeat int
	Score       Score
	Patch       Patch
}

func (s *Song) Copy() Song {
	return Song{BPM: s.BPM, RowsPerBeat: s.RowsPerBeat, Score: s.Score.Copy(), Patch: s.Patch.Copy()}
}

func (s *Song) SamplesPerRow() int {
	return 44100 * 60 / (s.BPM * s.RowsPerBeat)
}

// TBD: Where shall we put methods that work on pure domain types and have no dependencies
// e.g. Validate here
func (s *Song) Validate() error {
	if s.BPM < 1 {
		return errors.New("BPM should be > 0")
	}
	if len(s.Score.Tracks) == 0 {
		return errors.New("song contains no tracks")
	}
	var patternLen int
	for i, t := range s.Score.Tracks {
		for j, pat := range t.Patterns {
			if i == 0 && j == 0 {
				patternLen = len(pat)
			} else {
				if len(pat) != patternLen {
					return errors.New("Every pattern should have the same length")
				}
			}
		}
	}
	for i := range s.Score.Tracks[:len(s.Score.Tracks)-1] {
		if len(s.Score.Tracks[i].Order) != len(s.Score.Tracks[i+1].Order) {
			return errors.New("Every track should have the same sequence length")
		}
	}
	totalTrackVoices := 0
	for _, track := range s.Score.Tracks {
		totalTrackVoices += track.NumVoices
		for _, p := range track.Order {
			if p < 0 || int(p) >= len(track.Patterns) {
				return errors.New("Tracks use a non-existing pattern")
			}
		}
	}
	if totalTrackVoices > s.Patch.NumVoices() {
		return errors.New("Tracks use too many voices")
	}
	return nil
}
