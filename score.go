package sointu

// Score represents the arrangement of notes in a song; just a list of tracks
// and RowsPerPattern and Length (in patterns) to know the desired length of a
// song in rows. If any of the tracks is too short, all the notes outside the
// range should be just considered as holding the last note.
type Score struct {
	Tracks         []Track
	RowsPerPattern int // number of rows in each pattern
	Length         int // length of the song, in number of patterns
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
