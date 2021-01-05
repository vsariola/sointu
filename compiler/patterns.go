package compiler

import (
	"errors"
	"fmt"

	"github.com/vsariola/sointu"
)

// fixPatternLength makes sure that every pattern is the same length. During
// composing. Patterns shorter than the given length are padded with 1 / "hold";
// patterns longer than the given length are cropped.
func fixPatternLength(patterns [][]byte, fixedLength int) [][]int {
	patternData := make([]int, len(patterns)*fixedLength)
	ret := make([][]int, len(patterns))
	for i, pat := range patterns {
		for j, note := range pat {
			patternData[j] = int(note)
		}
		for j := len(pat); j < fixedLength; j++ {
			patternData[j] = 1 // pad with hold
		}
		ret[i], patternData = patternData[:fixedLength], patternData[fixedLength:]
	}
	return ret
}

// flattenSequence looks up a sequence of patterns and concatenates them into a
// single linear array of notes. Note that variable length patterns are
// concatenated as such; call fixPatternLength first if you want every pattern
// to be constant length.
func flattenSequence(patterns [][]int, sequence []int) []int {
	sumLen := 0
	for _, patIndex := range sequence {
		sumLen += len(patterns[patIndex])
	}
	notes := make([]int, sumLen)
	window := notes
	for _, patIndex := range sequence {
		elementsCopied := copy(window, patterns[patIndex])
		window = window[elementsCopied:]
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
				if n != pat[i] {
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

func bytesToInts(array []byte) []int {
	ret := make([]int, len(array))
	for i, v := range array {
		ret[i] = int(v)
	}
	return ret
}

func ConstructPatterns(song *sointu.Song) ([][]byte, [][]byte, error) {
	patLength := song.PatternRows()
	sequences := make([][]byte, len(song.Tracks))
	var patterns [][]int
	for i, t := range song.Tracks {
		fixed := fixPatternLength(t.Patterns, patLength)
		flat := flattenSequence(fixed, bytesToInts(t.Sequence))
		dontCares := markDontCares(flat)
		// TODO: we could give the user the possibility to use another length during encoding that during composing
		chunks := splitSequence(dontCares, patLength)
		var sequence []int
		sequence, patterns = addPatternsToTable(chunks, patterns)
		var err error
		sequences[i], err = intsToBytes(sequence)
		if err != nil {
			return nil, nil, errors.New("the constructed pattern table would result in > 256 unique patterns; only 256 unique patterns are supported")
		}
	}
	bytePatterns := make([][]byte, len(patterns))
	for i, pat := range patterns {
		var err error
		replaceInts(pat, -1, 0) // replace don't cares with releases
		bytePatterns[i], err = intsToBytes(pat)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid note in pattern, notes should be 0 .. 255: %v", err)
		}
	}
	return bytePatterns, sequences, nil
}
