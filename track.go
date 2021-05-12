package sointu

type Track struct {
	NumVoices int
	Effect    bool      `yaml:",omitempty"`
	Order     Order     `yaml:",flow"`
	Patterns  []Pattern `yaml:",flow"`
}

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
