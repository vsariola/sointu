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
	if s.Score.NumVoices() > s.Patch.NumVoices() {
		return errors.New("Tracks use too many voices")
	}
	return nil
}
