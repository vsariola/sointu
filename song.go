package sointu

import (
	"errors"
)

type (
	// Song includes a Score (the arrangement of notes in the song in one or more
	// tracks) and a Patch (the list of one or more instruments). Additionally,
	// BPM and RowsPerBeat fields set how fast the song should be played.
	// Currently, BPM is an integer as it offers already quite much granularity
	// for controlling the playback speed, but this could be changed to a
	// floating point in future if finer adjustments are necessary.
	Song struct {
		BPM         int
		RowsPerBeat int
		Score       Score
		Patch       Patch
	}

	// Score represents the arrangement of notes in a song; just a list of
	// tracks and RowsPerPattern and Length (in patterns) to know the desired
	// length of a song in rows. If any of the tracks is too short, all the
	// notes outside the range should be just considered as holding the last
	// note.
	Score struct {
		Tracks         []Track
		RowsPerPattern int // number of rows in each pattern
		Length         int // length of the song, in number of patterns
	}

	// Track represents the patterns and orderlist for each track. Note that
	// each track has its own patterns, so one track cannot use another tracks
	// patterns. This makes the data more intuitive to humans, as the reusing of
	// patterns over tracks is a rather rare occurence. However, the compiler
	// will put all the patterns in one global table (identical patterns only
	// appearing once), to optimize the runtime code.
	Track struct {
		// NumVoices is the number of voices this track triggers, cycling through
		// the voices. When this track triggers a new voice, the previous should be
		// released.
		NumVoices int

		// Effect hints the GUI if this is more of an effect track than a note
		// track: if true, e.g. the GUI can display the values as hexadecimals
		// instead of note values.
		Effect bool `yaml:",omitempty"`

		// Order is a list telling which pattern comes in which order in the song in
		// this track.
		Order Order `yaml:",flow"`

		// Patterns is a list of Patterns for this track.
		Patterns []Pattern `yaml:",flow"`
	}

	// Pattern represents a single pattern of note, in practice just a slice of
	// bytes, but provides convenience functions that return 1 values (hold) for
	// indices out of bounds of the array, and functions to increase the size of
	// the slice only by necessary amount when a new item is added, filling the
	// unused slots with 1s.
	Pattern []byte

	// Order is the pattern order for a track, in practice just a slice of
	// integers, but provides convenience functions that return -1 values for
	// indices out of bounds of the array, and functions to increase the size of
	// the slice only by necessary amount when a new item is added, filling the
	// unused slots with -1s.
	Order []int

	// SongPos represents a position in a song, in terms of order row and
	// pattern row. The order row is the index of the pattern in the order list,
	// and the pattern row is the index of the row in the pattern.
	SongPos struct {
		OrderRow   int
		PatternRow int
	}
)

func (s *Score) SongPos(songRow int) SongPos {
	if s.RowsPerPattern == 0 {
		return SongPos{OrderRow: 0, PatternRow: 0}
	}
	orderRow := songRow / s.RowsPerPattern
	patternRow := songRow % s.RowsPerPattern
	return SongPos{OrderRow: orderRow, PatternRow: patternRow}
}

func (s *Score) SongRow(songPos SongPos) int {
	return songPos.OrderRow*s.RowsPerPattern + songPos.PatternRow
}

func (s *Score) Wrap(songPos SongPos) SongPos {
	ret := s.SongPos(s.SongRow(songPos))
	ret.OrderRow %= s.Length
	return ret
}

func (s *Score) Clamp(songPos SongPos) SongPos {
	r := s.SongRow(songPos)
	if l := s.LengthInRows(); r >= l {
		r = l - 1
	}
	if r < 0 {
		r = 0
	}
	return s.SongPos(r)
}

// Get returns the value at index; or -1 is the index is out of range
func (s Order) Get(index int) int {
	if index < 0 || index >= len(s) {
		return -1
	}
	return s[index]
}

// Set sets the value at index; appending -1s until the slice is long enough.
func (s *Order) Set(index, value int) {
	for len(*s) <= index {
		*s = append(*s, -1)
	}
	(*s)[index] = value
}

func (s Track) Note(pos SongPos) byte {
	if pos.OrderRow < 0 || pos.OrderRow >= len(s.Order) {
		return 1
	}
	pat := s.Order[pos.OrderRow]
	if pat < 0 || pat >= len(s.Patterns) {
		return 1
	}
	if pos.PatternRow < 0 || pos.PatternRow >= len(s.Patterns[pat]) {
		return 1
	}
	return s.Patterns[pat][pos.PatternRow]
}

func (s *Track) SetNote(pos SongPos, note byte) {
	if pos.OrderRow < 0 || pos.PatternRow < 0 {
		return
	}
	pat := s.Order.Get(pos.OrderRow)
	if pat < 0 {
		if note == 1 {
			return
		}
		for _, o := range s.Order {
			if pat <= o {
				pat = o
			}
		}
		pat += 1
		if pat >= 36 {
			return
		}
		s.Order.Set(pos.OrderRow, pat)
	}
	if pat >= len(s.Patterns) && note == 1 {
		return
	}
	for pat >= len(s.Patterns) {
		s.Patterns = append(s.Patterns, Pattern{})
	}
	if pos.PatternRow >= len(s.Patterns[pat]) && note == 1 {
		return
	}
	for pos.PatternRow >= len(s.Patterns[pat]) {
		s.Patterns[pat] = append(s.Patterns[pat], 1)
	}
	s.Patterns[pat][pos.PatternRow] = note
}

// Get returns the value at index; or 1 is the index is out of range
func (s Pattern) Get(index int) byte {
	if index < 0 || index >= len(s) {
		return 1
	}
	return s[index]
}

// Set sets the value at index; appending 1s until the slice is long enough.
func (s *Pattern) Set(index int, value byte) {
	for len(*s) <= index {
		*s = append(*s, 1)
	}
	(*s)[index] = value
}

// Copy makes a deep copy of a Track.
func (t *Track) Copy() Track {
	order := make([]int, len(t.Order))
	copy(order, t.Order)
	patterns := make([]Pattern, len(t.Patterns))
	for i, oldPat := range t.Patterns {
		newPat := make(Pattern, len(oldPat))
		copy(newPat, oldPat)
		patterns[i] = newPat
	}
	return Track{
		NumVoices: t.NumVoices,
		Effect:    t.Effect,
		Order:     order,
		Patterns:  patterns,
	}
}

// Copy makes a deep copy of a Score.
func (l Score) Copy() Score {
	tracks := make([]Track, len(l.Tracks))
	for i, t := range l.Tracks {
		tracks[i] = t.Copy()
	}
	return Score{Tracks: tracks, RowsPerPattern: l.RowsPerPattern, Length: l.Length}
}

// NumVoices returns the total number of voices used in the Score; summing the
// voices of every track
func (l Score) NumVoices() int {
	ret := 0
	for _, t := range l.Tracks {
		ret += t.NumVoices
	}
	return ret
}

// FirstVoiceForTrack returns the index of the first voice of given track. For
// example, if the Score has three tracks (0, 1 and 2), with 1, 3, 2 voices,
// respectively, then FirstVoiceForTrack(0) returns 0, FirstVoiceForTrack(1)
// returns 1 and FirstVoiceForTrack(2) returns 4. Essentially computes just the
// cumulative sum.
func (l Score) FirstVoiceForTrack(track int) int {
	ret := 0
	for _, t := range l.Tracks[:track] {
		ret += t.NumVoices
	}
	return ret
}

// LengthInRows returns just RowsPerPattern * Length, as the length is the
// length in the number of patterns.
func (l Score) LengthInRows() int {
	return l.RowsPerPattern * l.Length
}

// Copy makes a deep copy of a Score.
func (s *Song) Copy() Song {
	return Song{BPM: s.BPM, RowsPerBeat: s.RowsPerBeat, Score: s.Score.Copy(), Patch: s.Patch.Copy()}
}

// Assuming 44100 Hz playback speed, return the number of samples of each row of
// the song.
func (s *Song) SamplesPerRow() int {
	if divisor := s.BPM * s.RowsPerBeat; divisor > 0 {
		return 44100 * 60 / divisor
	}
	return 0
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
