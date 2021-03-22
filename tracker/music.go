package tracker

import "fmt"

// note 81 = A4
// note 72 = C4
// note 24 = C0
const baseNote = 24

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

func NoteStr(val byte) string {
	if val == 1 {
		return "..." // hold
	}
	if val == 0 {
		return "---" // release
	}
	oNote := mod(int(val-baseNote), 12)
	octave := (int(val) - oNote - baseNote) / 12
	if octave < 0 {
		return fmt.Sprintf("%s%s", notes[oNote], string(byte('Z'+1+octave)))
	}
	if octave >= 10 {
		return fmt.Sprintf("%s%s", notes[oNote], string(byte('A'+octave-10)))
	}
	return fmt.Sprintf("%s%d", notes[oNote], octave)
}

func NoteAsValue(octave, note int) byte {
	return byte(baseNote + (octave * 12) + note)
}
