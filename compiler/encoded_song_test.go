package compiler_test

import (
	"reflect"
	"testing"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/compiler"
)

func TestPatternReusing(t *testing.T) {
	song := sointu.Song{
		Hold: 1,
		Tracks: []sointu.Track{{
			Patterns: [][]byte{{64, 1, 1, 1, 0, 0, 0, 0}, {72, 0, 0, 0, 0, 0, 0, 0}},
			Sequence: []byte{0, 1},
		}, {
			Patterns: [][]byte{{64, 1, 1, 1, 0, 0, 0, 0}, {84, 0, 0, 0, 0, 0, 0, 0}},
			Sequence: []byte{0, 1},
		}},
	}
	encodedSong, err := compiler.EncodeSong(&song)
	if err != nil {
		t.Fatalf("song encoding error: %v", err)
	}
	expected := compiler.EncodedSong{
		Sequences: [][]byte{{0, 1}, {0, 2}},
		Patterns:  [][]byte{{64, 1, 1, 1, 0, 0, 0, 0}, {72, 0, 0, 0, 0, 0, 0, 0}, {84, 0, 0, 0, 0, 0, 0, 0}},
	}
	if !reflect.DeepEqual(*encodedSong, expected) {
		t.Fatalf("got different EncodedSong than expected. got: %v expected: %v", *encodedSong, expected)
	}
}

func TestUnnecessaryHolds(t *testing.T) {
	song := sointu.Song{
		Hold: 1,
		Tracks: []sointu.Track{{
			Patterns: [][]byte{{64, 1, 1, 1, 0, 1, 0, 0}, {72, 0, 1, 0, 1, 0, 0, 0}},
			Sequence: []byte{0, 1},
		}, {
			Patterns: [][]byte{{64, 1, 1, 1, 0, 0, 1, 0}, {84, 0, 0, 0, 1, 1, 0, 0}},
			Sequence: []byte{0, 1},
		}},
	}
	encodedSong, err := compiler.EncodeSong(&song)
	if err != nil {
		t.Fatalf("song encoding error: %v", err)
	}
	expected := compiler.EncodedSong{
		Sequences: [][]byte{{0, 1}, {0, 2}},
		Patterns:  [][]byte{{64, 1, 1, 1, 0, 0, 0, 0}, {72, 0, 0, 0, 0, 0, 0, 0}, {84, 0, 0, 0, 0, 0, 0, 0}},
	}
	if !reflect.DeepEqual(*encodedSong, expected) {
		t.Fatalf("got different EncodedSong than expected. got: %v expected: %v", *encodedSong, expected)
	}
}

func TestDontCares(t *testing.T) {
	song := sointu.Song{
		Hold: 1,
		Tracks: []sointu.Track{{
			Patterns: [][]byte{{64, 1, 1, 1, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0, 0, 0}},
			Sequence: []byte{0, 1},
		}, {
			Patterns: [][]byte{{64, 1, 1, 1, 1, 1, 1, 1}, {1, 1, 1, 0, 0, 0, 0, 0}},
			Sequence: []byte{0, 1},
		}},
	}
	encodedSong, err := compiler.EncodeSong(&song)
	if err != nil {
		t.Fatalf("song encoding error: %v", err)
	}
	expected := compiler.EncodedSong{
		Sequences: [][]byte{{0, 1}, {2, 1}},
		Patterns:  [][]byte{{64, 1, 1, 1, 0, 0, 0, 0}, {1, 1, 1, 0, 0, 0, 0, 0}, {64, 1, 1, 1, 1, 1, 1, 1}},
	}
	if !reflect.DeepEqual(*encodedSong, expected) {
		t.Fatalf("got different EncodedSong than expected. got: %v expected: %v", *encodedSong, expected)
	}
}
