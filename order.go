package sointu

// Order is the pattern order for a track, in practice just a slice of integers,
// but provides convenience functions that return -1 values for indices out of
// bounds of the array, and functions to increase the size of the slice only by
// necessary amount when a new item is added, filling the unused slots with -1s.
type Order []int

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
