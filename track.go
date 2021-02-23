package sointu

type Track struct {
	NumVoices int
	Effect    bool     `yaml:",omitempty"`
	Order     []int    `yaml:",flow"`
	Patterns  [][]byte `yaml:",flow"`
}

func (t *Track) Copy() Track {
	order := make([]int, len(t.Order))
	copy(order, t.Order)
	patterns := make([][]byte, len(t.Patterns))
	for i, oldPat := range t.Patterns {
		newPat := make([]byte, len(oldPat))
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
