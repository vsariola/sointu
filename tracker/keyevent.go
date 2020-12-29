package tracker

import (
	"os"
	"strconv"

	"gioui.org/io/key"
)

var noteMap = map[string]byte{
	"Z": 0,
	"S": 1,
	"X": 2,
	"D": 3,
	"C": 4,
	"V": 5,
	"G": 6,
	"B": 7,
	"H": 8,
	"N": 9,
	"J": 10,
	"M": 11,
	"Q": 12,
	"2": 13,
	"W": 14,
	"3": 15,
	"E": 16,
	"R": 17,
	"5": 18,
	"T": 19,
	"6": 20,
	"Y": 21,
	"7": 22,
	"U": 23,
}

// KeyEvent handles incoming key events and returns true if repaint is needed.
func (t *Tracker) KeyEvent(e key.Event) bool {
	if e.State == key.Press {
		if t.CursorColumn == 0 {
			if val, ok := noteMap[e.Name]; ok {
				t.NotePressed(val)
				return true
			}
		} else {
			if iv, err := strconv.ParseInt(e.Name, 16, 8); err == nil {
				t.NumberPressed(byte(iv))
				return true
			}
		}
		switch e.Name {
		case "A":
			t.setCurrent(0)
		case key.NameDeleteForward:
			t.setCurrent(1)
		case key.NameEscape:
			os.Exit(0)
		case "Space":
			t.TogglePlay()
			return true
		case key.NameUpArrow:
			t.CursorRow = (t.CursorRow + t.song.PatternRows() - 1) % t.song.PatternRows()
			return true
		case key.NameDownArrow:
			t.CursorRow = (t.CursorRow + 1) % t.song.PatternRows()
			return true
		case key.NameLeftArrow:
			if t.CursorColumn == 0 {
				t.ActiveTrack = (t.ActiveTrack + len(t.song.Tracks) - 1) % len(t.song.Tracks)
				t.CursorColumn = 2
			} else {
				t.CursorColumn--
			}
			return true
		case key.NameRightArrow:
			if t.CursorColumn == 2 {
				t.ActiveTrack = (t.ActiveTrack + 1) % len(t.song.Tracks)
				t.CursorColumn = 0
			} else {
				t.CursorColumn++
			}
			return true
		case key.NameTab:
			if e.Modifiers.Contain(key.ModShift) {
				t.ActiveTrack = (t.ActiveTrack + len(t.song.Tracks) - 1) % len(t.song.Tracks)
			} else {
				t.ActiveTrack = (t.ActiveTrack + 1) % len(t.song.Tracks)
			}
			t.CursorColumn = 0
			return true
		}
	}
	return false
}

// setCurrent sets the (note) value in current pattern under cursor to iv
func (t *Tracker) setCurrent(iv byte) {
	t.song.Patterns[t.song.Tracks[t.ActiveTrack].Sequence[t.DisplayPattern]][t.CursorRow] = iv
}

// getCurrent returns the current (note) value in current pattern under the cursor
func (t *Tracker) getCurrent() byte {
	return t.song.Patterns[t.song.Tracks[t.ActiveTrack].Sequence[t.DisplayPattern]][t.CursorRow]
}

// NotePressed handles incoming key presses while in the note column
func (t *Tracker) NotePressed(val byte) {
	t.setCurrent(getNoteValue(t.CurrentOctave, val))
}

// NumberPressed handles incoming presses while in either of the hex number columns
func (t *Tracker) NumberPressed(iv byte) {
	val := t.getCurrent()
	if t.CursorColumn == 1 {
		val = ((iv & 0xF) << 4) | (val & 0xF)
	} else if t.CursorColumn == 2 {
		val = (val & 0xF0) | (iv & 0xF)
	}
	t.setCurrent(val)
}
