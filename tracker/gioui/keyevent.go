package gioui

import (
	"time"

	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/op"
	"github.com/vsariola/sointu/tracker"
	"gopkg.in/yaml.v3"
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
func (t *Tracker) KeyEvent(e key.Event, o *op.Ops) {
	if e.State == key.Press {
		switch e.Name {
		case "C":
			if e.Modifiers.Contain(key.ModShortcut) {
				contents, err := yaml.Marshal(t.Song())
				if err == nil {
					clipboard.WriteOp{Text: string(contents)}.Add(o)
					t.Alert.Update("Song copied to clipboard", Notify, time.Second*3)
				}
				return
			}
		case "V":
			if e.Modifiers.Contain(key.ModShortcut) {
				clipboard.ReadOp{Tag: t}.Add(o)
				return
			}
		case "Z":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.Undo()
				return
			}
		case "Y":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.Redo()
				return
			}
		case "N":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.NewSong(false)
				return
			}
		case "S":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.SaveSongFile()
				return
			}
		case "O":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.OpenSongFile(false)
				return
			}
		case "F1":
			t.OrderEditor.Focus()
			return
		case "F2":
			t.TrackEditor.Focus()
			return
		case "F3":
			t.InstrumentEditor.Focus()
			return
		case "F4":
			t.TrackEditor.Focus()
			return
		case "F5":
			t.SetNoteTracking(true)
			startRow := t.Cursor().ScoreRow
			t.PlayFromPosition(startRow)
			return
		case "F6":
			t.SetNoteTracking(false)
			startRow := t.Cursor().ScoreRow
			t.PlayFromPosition(startRow)
			return
		case "F8":
			t.SetPlaying(false)
			return
		case "Space":
			if !t.Playing() && !t.InstrEnlarged() {
				t.SetNoteTracking(!e.Modifiers.Contain(key.ModShortcut))
				startRow := t.Cursor().ScoreRow
				t.PlayFromPosition(startRow)
			} else {
				t.SetPlaying(false)
			}
		case `\`, `<`, `>`:
			if e.Modifiers.Contain(key.ModShift) {
				t.SetOctave(t.Octave() + 1)
			} else {
				t.SetOctave(t.Octave() - 1)
			}
		case key.NameTab:
			if e.Modifiers.Contain(key.ModShift) {
				switch {
				case t.OrderEditor.Focused():
					t.InstrumentEditor.paramEditor.Focus()
				case t.TrackEditor.Focused():
					t.OrderEditor.Focus()
				case t.InstrumentEditor.Focused():
					if t.InstrEnlarged() {
						t.InstrumentEditor.paramEditor.Focus()
					} else {
						t.TrackEditor.Focus()
					}
				default:
					t.InstrumentEditor.Focus()
				}
			} else {
				switch {
				case t.OrderEditor.Focused():
					t.TrackEditor.Focus()
				case t.TrackEditor.Focused():
					t.InstrumentEditor.Focus()
				case t.InstrumentEditor.Focused():
					t.InstrumentEditor.paramEditor.Focus()
				default:
					if t.InstrEnlarged() {
						t.InstrumentEditor.Focus()
					} else {
						t.OrderEditor.Focus()
					}
				}
			}
		}
		t.JammingPressed(e)
	} else { // e.State == key.Release
		t.JammingReleased(e)
	}
}

// NumberPressed handles incoming presses while in either of the hex number columns
func (t *Tracker) NumberPressed(iv byte) {
	val := t.Note()
	if val == 1 {
		val = 0
	}
	if t.LowNibble() {
		val = (val & 0xF0) | (iv & 0xF)
	} else {
		val = ((iv & 0xF) << 4) | (val & 0xF)
	}
	t.SetNote(val)
}

func (t *Tracker) JammingPressed(e key.Event) byte {
	if val, ok := noteMap[e.Name]; ok {
		if _, ok := t.KeyPlaying[e.Name]; !ok {
			n := tracker.NoteAsValue(t.OctaveNumberInput.Value, val)
			instr := t.InstrIndex()
			noteID := tracker.NoteIDInstr(instr, n)
			t.NoteOn(noteID)
			t.KeyPlaying[e.Name] = noteID
			return n
		}
	}
	return 0
}

func (t *Tracker) JammingReleased(e key.Event) bool {
	if noteID, ok := t.KeyPlaying[e.Name]; ok {
		t.NoteOff(noteID)
		delete(t.KeyPlaying, e.Name)
		return true
	}
	return false
}
