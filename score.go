package sointu

type Score struct {
	Tracks         []Track
	RowsPerPattern int
	Length         int // length of the song, in number of patterns
}

func (l Score) Copy() Score {
	tracks := make([]Track, len(l.Tracks))
	for i, t := range l.Tracks {
		tracks[i] = t.Copy()
	}
	return Score{Tracks: tracks, RowsPerPattern: l.RowsPerPattern, Length: l.Length}
}

func (l Score) NumVoices() int {
	ret := 0
	for _, t := range l.Tracks {
		ret += t.NumVoices
	}
	return ret
}

func (l Score) FirstVoiceForTrack(track int) int {
	ret := 0
	for _, t := range l.Tracks[:track] {
		ret += t.NumVoices
	}
	return ret
}

func (l Score) LengthInRows() int {
	return l.RowsPerPattern * l.Length
}
