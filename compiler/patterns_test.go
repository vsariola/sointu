package compiler_test

import (
	"reflect"
	"testing"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/compiler"
)

func TestPatternReusing(t *testing.T) {
	song := sointu.Song{
		RowsPerPattern: 8,
		Tracks: []sointu.Track{{
			Patterns: [][]byte{{64, 1, 1, 1, 0, 0, 0, 0}, {72, 0, 0, 0, 0, 0, 0, 0}},
			Sequence: []byte{0, 1},
		}, {
			Patterns: [][]byte{{64, 1, 1, 1, 0, 0, 0, 0}, {84, 0, 0, 0, 0, 0, 0, 0}},
			Sequence: []byte{0, 1},
		}},
	}
	patterns, sequences, err := compiler.ConstructPatterns(&song)
	if err != nil {
		t.Fatalf("erorr constructing patterns: %v", err)
	}
	expectedSequences := [][]byte{{0, 1}, {0, 2}}
	expectedPatterns := [][]byte{{64, 1, 1, 1, 0, 0, 0, 0}, {72, 0, 0, 0, 0, 0, 0, 0}, {84, 0, 0, 0, 0, 0, 0, 0}}
	if !reflect.DeepEqual(patterns, expectedPatterns) {
		t.Fatalf("got different patterns than expected. got: %v expected: %v", patterns, expectedPatterns)
	}
	if !reflect.DeepEqual(sequences, expectedSequences) {
		t.Fatalf("got different patterns than expected. got: %v expected: %v", patterns, expectedPatterns)
	}
}

func TestUnnecessaryHolds(t *testing.T) {
	song := sointu.Song{
		RowsPerPattern: 8,
		Tracks: []sointu.Track{{
			Patterns: [][]byte{{64, 1, 1, 1, 0, 1, 0, 0}, {72, 0, 1, 0, 1, 0, 0, 0}},
			Sequence: []byte{0, 1},
		}, {
			Patterns: [][]byte{{64, 1, 1, 1, 0, 0, 1, 0}, {84, 0, 0, 0, 1, 1, 0, 0}},
			Sequence: []byte{0, 1},
		}},
	}
	patterns, sequences, err := compiler.ConstructPatterns(&song)
	if err != nil {
		t.Fatalf("erorr constructing patterns: %v", err)
	}
	expectedSequences := [][]byte{{0, 1}, {0, 2}}
	expectedPatterns := [][]byte{{64, 1, 1, 1, 0, 0, 0, 0}, {72, 0, 0, 0, 0, 0, 0, 0}, {84, 0, 0, 0, 0, 0, 0, 0}}
	if !reflect.DeepEqual(patterns, expectedPatterns) {
		t.Fatalf("got different patterns than expected. got: %v expected: %v", patterns, expectedPatterns)
	}
	if !reflect.DeepEqual(sequences, expectedSequences) {
		t.Fatalf("got different patterns than expected. got: %v expected: %v", patterns, expectedPatterns)
	}
}

func TestDontCares(t *testing.T) {
	song := sointu.Song{
		RowsPerPattern: 8,
		Tracks: []sointu.Track{{
			Patterns: [][]byte{{64, 1, 1, 1, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0, 0, 0}},
			Sequence: []byte{0, 1},
		}, {
			Patterns: [][]byte{{64, 1, 1, 1, 1, 1, 1, 1}, {1, 1, 1, 0, 0, 0, 0, 0}},
			Sequence: []byte{0, 1},
		}},
	}
	patterns, sequences, err := compiler.ConstructPatterns(&song)
	if err != nil {
		t.Fatalf("erorr constructing patterns: %v", err)
	}
	expectedSequences := [][]byte{{0, 1}, {2, 1}}
	expectedPatterns := [][]byte{{64, 1, 1, 1, 0, 0, 0, 0}, {1, 1, 1, 0, 0, 0, 0, 0}, {64, 1, 1, 1, 1, 1, 1, 1}}
	if !reflect.DeepEqual(patterns, expectedPatterns) {
		t.Fatalf("got different patterns than expected. got: %v expected: %v", patterns, expectedPatterns)
	}
	if !reflect.DeepEqual(sequences, expectedSequences) {
		t.Fatalf("got different patterns than expected. got: %v expected: %v", patterns, expectedPatterns)
	}
}
