package sointu

// Pattern represents a single pattern of note, in practice just a slice of bytes,
// but provides convenience functions that return 1 values (hold) for indices out of
// bounds of the array, and functions to increase the size of the slice only by
// necessary amount when a new item is added, filling the unused slots with 1s.
type Pattern []byte

func (s Pattern) Get(index int) byte {
	if index < 0 || index >= len(s) {
		return 1
	}
	return s[index]
}

func (s *Pattern) Set(index int, value byte) {
	for len(*s) <= index {
		*s = append(*s, 1)
	}
	(*s)[index] = value
}
