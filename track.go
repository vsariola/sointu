package sointu

// Track represents the patterns and orderlist for each track. Note that each
// track has its own patterns, so one track cannot use another tracks patterns.
// This makes the data more intuitive to humans, as the reusing of patterns over
// tracks is a rather rare occurence. However, the compiler will put all the
// patterns in one global table (identical patterns only appearing once), to
// optimize the runtime code.
type Track struct {
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
