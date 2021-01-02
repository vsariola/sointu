package compiler

import (
	"errors"
	"fmt"

	"github.com/vsariola/sointu"
)

// EncodedSong has a single global pattern table and all track sequences are
// indices to this table. This is in contrast with sointu. Song, which has one
// pattern table per track.
type EncodedSong struct {
	Patterns  [][]byte
	Sequences [][]byte
}

// flattenPatterns returns the track sequences flattened into linear arrays.
// Additionally, after first release (value 0), it replaces every release or
// hold with -1, denoting "don't care if it's either release (0) or hold (1)".
// As we reconstruct the pattern table, we may use any pattern that has either 0
// or hold in place of don't cares.
func flattenPatterns(song *sointu.Song) [][]int {
	ret := make([][]int, 0, len(song.Tracks))
	for _, t := range song.Tracks {
		flatSequence := make([]int, 0, song.TotalRows())
		dontCare := false
		for _, s := range t.Sequence {
			for _, note := range t.Patterns[s] {
				if !dontCare || note > song.Hold {
					if note == song.Hold {
						flatSequence = append(flatSequence, 1) // replace holds with 1s, we'll get rid of song.Hold soon and do the hold replacement at the last minute
					} else {
						flatSequence = append(flatSequence, int(note))
					}
					dontCare = note == 0 // after 0 aka release, we don't care if further releases come along
				} else {
					flatSequence = append(flatSequence, -1)
				}
			}
		}
		ret = append(ret, flatSequence)
	}
	return ret
}

// constructPatterns finds the smallest global pattern table for a given list of
// flattened patterns. If the patterns are not divisible with the patternLength,
// then: a) if the last note of a track is release (0) or don't care (-1), the
// track is extended with don't cares (-1) until the total length of the song is
// divisible with the patternLength. b) Otherwise, the track is extended with a
// single release (0), followed by don't care about release & hold (-1).
//
// In otherwords: any playing notes are released when the original song ends.
func constructPatterns(tracks [][]int, patternLength int) ([][]byte, [][]byte, error) {
	patternTable := make([][]int, 0)
	sequences := make([][]byte, 0, len(tracks))
	for _, t := range tracks {
		var sequence []byte
		for s := 0; s < len(t); s += patternLength {
			pat := t[s : s+patternLength]
			if len(pat) < patternLength {
				extension := make([]int, patternLength-len(pat))
				for i := range extension {
					if pat[len(pat)-1] > 0 && i == 0 {
						extension[i] = 0
					} else {
						extension[i] = -1
					}
				}
				pat = append(pat, extension...)
			}
			// go through the current pattern table to see if there's already a
			// pattern that could be used
			patternIndex := -1
			for j, p := range patternTable {
				match := true
				for k, n := range p {
					if (n > -1 && pat[k] > -1 && n != pat[k]) ||
						(n == -1 && pat[k] > 1) ||
						(n > 1 && pat[k] == -1) {
						match = false
						break
					}
				}
				if match {
					// if there was any don't cares in the pattern table where
					// the new pattern has non don't cares, copy them to the
					// patterns that was already in the pattern table
					for k, n := range pat {
						if n != -1 {
							patternTable[j][k] = n
						}
					}
					patternIndex = j
					break
				}
			}
			if patternIndex == -1 {
				patternIndex = len(patternTable)
				patternTable = append(patternTable, pat)
			}
			if patternIndex > 255 {
				return nil, nil, errors.New("encoding the song would result more than 256 different unique patterns")
			}
			sequence = append(sequence, byte(patternIndex))
		}
		sequences = append(sequences, sequence)
	}
	// finally, if there are still some don't cares in the table, just replace them with zeros
	byteTable := make([][]byte, 0, len(patternTable))
	for _, pat := range patternTable {
		bytePat := make([]byte, 0, patternLength)
		for _, n := range pat {
			if n >= 0 {
				bytePat = append(bytePat, byte(n))
			} else {
				bytePat = append(bytePat, 0)
			}
		}
		byteTable = append(byteTable, bytePat)
	}
	return byteTable, sequences, nil
}

func (e *EncodedSong) PatternLength() int {
	return len(e.Patterns[0])
}

func (e *EncodedSong) SequenceLength() int {
	return len(e.Sequences[0])
}

func (e *EncodedSong) TotalRows() int {
	return e.SequenceLength() * e.PatternLength()
}

func EncodeSong(song *sointu.Song) (*EncodedSong, error) {
	// TODO: we could give the user the possibility to encode the patterns with a different length here also
	patLength := song.PatternRows()
	patterns, sequences, err := constructPatterns(flattenPatterns(song), patLength)
	if err != nil {
		return nil, fmt.Errorf("error during constructPatterns: %v", err)
	}
	return &EncodedSong{Patterns: patterns, Sequences: sequences}, nil
}
