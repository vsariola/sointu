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

func valueAsNote(val byte) string {
	octave := (val - baseNote) / 12
	oNote := (val - baseNote) % 12
	if octave < 0 || oNote < 0 || octave > 10 {
		return "..."
	}
	return fmt.Sprintf("%s%d", notes[oNote], octave)
}
