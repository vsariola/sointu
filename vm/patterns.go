package vm

import (
	"errors"
	"fmt"

	"github.com/vsariola/sointu"
)

// flattenSequence returns the notes of a track in a single linear array of
// integer notes.
func flattenSequence(t sointu.Track, songLength int, rowsPerPattern int, releaseFirst bool) []int {
	sumLen := rowsPerPattern * songLength
	notes := make([]int, sumLen)
	k := 0
	for i := 0; i < songLength; i++ {
		patIndex := t.Order.Get(i)
		var pattern sointu.Pattern = nil
		if patIndex >= 0 && patIndex < len(t.Patterns) {
			pattern = t.Patterns[patIndex]
		}
		for j := 0; j < rowsPerPattern; j++ {
			note := int(pattern.Get(j))
			if releaseFirst && i == 0 && j == 0 && note == 1 {
				note = 0
			}
			notes[k] = note
			k++
		}
	}
	return notes
}

// markDontCares goes through a linear array of notes and marks every hold (1)
// or release (0) after the first release (0) as -1 or "don't care". This means
// that for -1:s, we don't care if it's a hold or release; it does not affect
// the sound as the note has been already released.
func markDontCares(notes []int) []int {
	notesWithDontCares := make([]int, len(notes))
	dontCare := false
	for i, n := range notes {
		if dontCare && n <= 1 {
			notesWithDontCares[i] = -1
		} else {
			notesWithDontCares[i] = n
			dontCare = n == 0
		}
	}
	return notesWithDontCares
}

// replaceInt replaces all occurrences of needle in the haystack with the value
// "with"
func replaceInts(haystack []int, needle int, with int) {
	for i, v := range haystack {
		if v == needle {
			haystack[i] = with
		}
	}
}

// splitSequence splits a linear sequence of notes into patternLength size
// chunks. If the last chunk is shorter than the patternLength, then it is
// padded with dontCares (-1).
func splitSequence(sequence []int, patternLength int) [][]int {
	numChunksRoundedUp := (len(sequence) + patternLength - 1) / patternLength
	chunks := make([][]int, numChunksRoundedUp)
	for i := range chunks {
		if len(sequence) >= patternLength {
			chunks[i], sequence = sequence[:patternLength], sequence[patternLength:]
		} else {
			padded := make([]int, patternLength)
			j := copy(padded, sequence)
			for ; j < patternLength; j++ {
				padded[j] = -1
			}
			chunks[i] = padded
		}
	}
	return chunks
}

// addPatternsToTable adds given patterns to the table, checking if existing
// pattern could be used. DontCares are taken into account so a pattern that has
// don't care where another has a hold or release is ok. It returns a 1D
// sequence of indices of each added pattern in the updated pattern table & the
// updated pattern table.
func addPatternsToTable(patterns [][]int, table [][]int) ([]int, [][]int) {
	updatedTable := make([][]int, len(table))
	copy(updatedTable, table) // avoid updating the underlying slices for concurrency safety
	sequence := make([]int, len(patterns))
	for i, pat := range patterns {
		// go through the current pattern table to see if there's already a
		// pattern that could be used
		patternIndex := -1
		for j, p := range updatedTable {
			match := true
			identical := true
			for k, n := range p {
				if (n > -1 && pat[k] > -1 && n != pat[k]) ||
					(n == -1 && pat[k] > 1) ||
					(n > 1 && pat[k] == -1) {
					match = false
					break
				}
				if (i < len(pat) && n != pat[i]) || n != 1 {
					identical = false
				}
			}
			if match {
				if !identical {
					// the patterns were not identical; one of them had don't
					// cares where another had hold or release so we make a new
					// copy with merged data, that essentially is a max of the
					// two patterns
					mergedPat := make([]int, len(p))
					copy(mergedPat, p) // make a copy instead of updating existing, for concurrency safety
					for k, n := range pat {
						if n != -1 {
							mergedPat[k] = n
						}
					}
					updatedTable[j] = mergedPat
				}
				patternIndex = j
				break
			}
		}
		if patternIndex == -1 {
			patternIndex = len(updatedTable)
			updatedTable = append(updatedTable, pat)
		}
		sequence[i] = patternIndex
	}
	return sequence, updatedTable
}

func intsToBytes(array []int) ([]byte, error) {
	ret := make([]byte, len(array))
	for i, v := range array {
		if v < 0 || v > 255 {
			return nil, fmt.Errorf("when converting intsToBytes, all values should be 0 .. 255 (was: %v)", v)
		}
		ret[i] = byte(v)
	}
	return ret, nil
}

func ConstructPatterns(song *sointu.Song) ([][]byte, [][]byte, error) {
	sequences := make([][]byte, len(song.Score.Tracks))
	var patterns [][]int
	for i, t := range song.Score.Tracks {
		flat := flattenSequence(t, song.Score.Length, song.Score.RowsPerPattern, true)
		dontCares := markDontCares(flat)
		// TODO: we could give the user the possibility to use another length during encoding that during composing
		chunks := splitSequence(dontCares, song.Score.RowsPerPattern)
		var sequence []int
		sequence, patterns = addPatternsToTable(chunks, patterns)
		var err error
		sequences[i], err = intsToBytes(sequence)
		if err != nil {
			return nil, nil, errors.New("the constructed pattern table would result in > 256 unique patterns; only 256 unique patterns are supported")
		}
	}

	// Determine the length of bytePatterns
	bytePatternsLength := len(patterns)
	if song.CreateEmptyPatterns {
		bytePatternsLength = 256
	}

	bytePatterns := make([][]byte, bytePatternsLength) // Initialize bytePatterns with the specified length
	for i := 0; i < bytePatternsLength; i++ {
		bytePatterns[i] = make([]byte, song.Score.RowsPerPattern) // Initialize each element with an array of zeros with length RowsPerPattern
	}

	for i, pat := range patterns {
		var err error
		replaceInts(pat, -1, 0) // replace don't cares with releases
		bytePatterns[i], err = intsToBytes(pat)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid note in pattern, notes should be 0 .. 255: %v", err)
		}
	}
	allZeroIndex := -1
	for i, pat := range patterns {
		match := true
		for _, note := range pat {
			if note != 0 {
				match = false
				break
			}
		}
		if match {
			allZeroIndex = i
			break
		}
	}
	// if there's a pattern full of zeros...
	if allZeroIndex > -1 {
		// ...swap that into position 0, as it is likely the most common pattern
		bytePatterns[allZeroIndex], bytePatterns[0] = bytePatterns[0], bytePatterns[allZeroIndex]
		for _, s := range sequences {
			for j, n := range s {
				if n == 0 {
					s[j] = byte(allZeroIndex)
				} else if n == byte(allZeroIndex) {
					s[j] = 0
				}
			}
		}
	}
	return bytePatterns, sequences, nil
}
