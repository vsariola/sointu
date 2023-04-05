package sointu

import (
	"errors"
)

// Song includes a Score(the arrangement of notes in the song in one or more
// tracks) and a Patch (the list of one or more instruments). Additionally, BPM
// and RowsPerBeat fields set how fast the song should be played. Currently, BPM
// is an integer as it offers already quite much granularity for controlling the
// playback speed, but this could be changed to a floating point in future if
// finer adjustments are necessary.
type Song struct {
	BPM                      int
	RowsPerBeat              int
	Score                    Score
	Patch                    Patch
	CreateEmptyPatterns      bool
	WasmDisableRenderOnStart bool
}

// Copy makes a deep copy of a Score.
func (s *Song) Copy() Song {
	return Song{BPM: s.BPM, RowsPerBeat: s.RowsPerBeat, Score: s.Score.Copy(), Patch: s.Patch.Copy()}
}

// Assuming 44100 Hz playback speed, return the number of samples of each row of
// the song.
func (s *Song) SamplesPerRow() int {
	return 44100 * 60 / (s.BPM * s.RowsPerBeat)
}

// Validate checks if the Song looks like a valid song: BPM > 0, one or more
// tracks, score uses less than or equal number of voices than patch. Not used
// much so we could probably get rid of this function.
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
