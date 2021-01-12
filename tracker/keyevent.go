package tracker

import (
	"os"
	"strconv"

	"gioui.org/io/key"
)

var noteMap = map[string]int{
	"Z": -12,
	"S": -11,
	"X": -10,
	"D": -9,
	"C": -8,
	"V": -7,
	"G": -6,
	"B": -5,
	"H": -4,
	"N": -3,
	"J": -2,
	"M": -1,
	",": 0,
	"L": 1,
	".": 2,
	"Q": 0,
	"2": 1,
	"W": 2,
	"3": 3,
	"E": 4,
	"R": 5,
	"5": 6,
	"T": 7,
	"6": 8,
	"Y": 9,
	"7": 10,
	"U": 11,
	"I": 12,
	"9": 13,
	"O": 14,
	"0": 15,
	"P": 16,
}

// KeyEvent handles incoming key events and returns true if repaint is needed.
func (t *Tracker) KeyEvent(e key.Event) bool {
	if e.State == key.Press {
		switch e.Name {
		case "Z":
			if e.Modifiers.Contain(key.ModCtrl) {
				t.Undo()
				return true
			}
		case "Y":
			if e.Modifiers.Contain(key.ModCtrl) {
				t.Redo()
				return true
			}
		case "A":
			t.SetCurrentNote(0)
			return true
		case key.NameDeleteForward:
			t.SetCurrentNote(1)
			return true
		case key.NameEscape:
			os.Exit(0)
		case "Space":
			t.TogglePlay()
			return true
		case `\`:
			if e.Modifiers.Contain(key.ModShift) {
				return t.ChangeBPM(1)
			}
			return t.ChangeBPM(-1)
		case key.NameUpArrow:
			delta := -1
			if e.Modifiers.Contain(key.ModCtrl) {
				delta = -t.song.PatternRows()
			}
			t.moveCursor(delta)
			t.NoteTracking = false
			return true
		case key.NameDownArrow:
			delta := 1
			if e.Modifiers.Contain(key.ModCtrl) {
				delta = t.song.PatternRows()
			}
			t.moveCursor(delta)
			t.NoteTracking = false
			return true
		case key.NameLeftArrow:
			if t.CursorColumn == 0 || !t.TrackShowHex[t.ActiveTrack] || e.Modifiers.Contain(key.ModCtrl) {
				t.ActiveTrack = (t.ActiveTrack + len(t.song.Tracks) - 1) % len(t.song.Tracks)
				if t.TrackShowHex[t.ActiveTrack] {
					t.CursorColumn = 1
				} else {
					t.CursorColumn = 0
				}
			} else {
				t.CursorColumn--
			}
			return true
		case key.NameRightArrow:
			if t.CursorColumn == 1 || !t.TrackShowHex[t.ActiveTrack] || e.Modifiers.Contain(key.ModCtrl) {
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
		if e.Modifiers.Contain(key.ModCtrl) {
			if iv, err := strconv.ParseInt(e.Name, 16, 8); err == nil {
				t.SetCurrentPattern(byte(iv))
				return true
			}
		} else {
			if !t.TrackShowHex[t.ActiveTrack] {
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
		}
	}
	return false
}

func (t *Tracker) moveCursor(delta int) {
	newRow := t.CursorRow + delta
	remainder := (newRow + t.song.PatternRows()) % t.song.PatternRows()
	t.DisplayPattern += (newRow - remainder) / t.song.PatternRows()
	if t.DisplayPattern < 0 {
		t.CursorRow = 0
		t.DisplayPattern = 0
	} else if t.DisplayPattern >= t.song.SequenceLength() {
		t.CursorRow = t.song.PatternRows() - 1
		t.DisplayPattern = t.song.SequenceLength() - 1
	} else {
		t.CursorRow = remainder
	}
}

// getCurrent returns the current (note) value in current pattern under the cursor
func (t *Tracker) getCurrent() byte {
	return t.song.Tracks[t.ActiveTrack].Patterns[t.song.Tracks[t.ActiveTrack].Sequence[t.DisplayPattern]][t.CursorRow]
}

// NotePressed handles incoming key presses while in the note column
func (t *Tracker) NotePressed(val int) {
	t.SetCurrentNote(getNoteValue(int(t.CurrentOctave), val))
}

// NumberPressed handles incoming presses while in either of the hex number columns
func (t *Tracker) NumberPressed(iv byte) {
	val := t.getCurrent()
	if t.CursorColumn == 0 {
		val = ((iv & 0xF) << 4) | (val & 0xF)
	} else if t.CursorColumn == 1 {
		val = (val & 0xF0) | (iv & 0xF)
	}
	t.SetCurrentNote(val)
}
