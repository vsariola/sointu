package tracker

import "fmt"

const baseNote = 20

var notes = []string{
	"C-",
	"C#",
	"D-",
	"D#",
	"E-",
	"F-",
	"F#",
	"G-",
	"G#",
	"A-",
	"A#",
	"B-",
}

// valueAsNote returns the textual representation of a note value
func valueAsNote(val byte) string {
	if val == 1 {
		return "..." // hold
	}
	if val == 0 {
		return "---" // release
	}
	octave := (val - baseNote) / 12
	oNote := (val - baseNote) % 12
	if octave < 0 || oNote < 0 || octave > 10 {
		return "???"
	}
	return fmt.Sprintf("%s%d", notes[oNote], octave)
}

// noteValue return the note value for a particular note and octave combination
func getNoteValue(octave, note byte) byte {
	return baseNote + (octave * 12) + note
}
