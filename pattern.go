package sointu

// Pattern represents a single pattern of note, in practice just a slice of bytes,
// but provides convenience functions that return 1 values (hold) for indices out of
// bounds of the array, and functions to increase the size of the slice only by
// necessary amount when a new item is added, filling the unused slots with 1s.
type Pattern []byte

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
