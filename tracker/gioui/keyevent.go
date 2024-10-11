package gioui

import (
	"gioui.org/io/clipboard"
	"gioui.org/io/key"
)

var noteMap = map[key.Name]int{
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
func (t *Tracker) KeyEvent(e key.Event, gtx C) {
	if e.State == key.Press {
		switch e.Name {
		case "V":
			if e.Modifiers.Contain(key.ModShortcut) {
				gtx.Execute(clipboard.ReadCmd{Tag: t})
				return
			}
		case "Z":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.Model.Undo().Do()
				return
			}
		case "Y":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.Model.Redo().Do()
				return
			}
		case "D":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.Model.UnitDisabled().Bool().Toggle()
				return
			}
		case "L":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.Model.LoopToggle().Bool().Toggle()
				return
			}
		case "N":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.NewSong().Do()
				return
			}
		case "S":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.SaveSong().Do()
				return
			}
		case "O":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.OpenSong().Do()
				return
			}
		case "I":
			if e.Modifiers.Contain(key.ModShortcut) {
				if e.Modifiers.Contain(key.ModShift) {
					t.DeleteInstrument().Do()
				} else {
					t.AddInstrument().Do()
				}
				return
			}
		case "T":
			if e.Modifiers.Contain(key.ModShortcut) {
				if e.Modifiers.Contain(key.ModShift) {
					t.DeleteTrack().Do()
				} else {
					t.AddTrack().Do()
				}
				return
			}
		case "E":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.InstrEnlarged().Bool().Toggle()
				return
			}
		case "W":
			if e.Modifiers.Contain(key.ModShortcut) && canQuit {
				t.Quit().Do()
				return
			}
		case "F1":
			t.OrderEditor.scrollTable.Focus()
			return
		case "F2":
			t.TrackEditor.scrollTable.Focus()
			return
		case "F3":
			t.InstrumentEditor.Focus()
			return
		case "Space":
			t.NoteTracking().Bool().Set(e.Modifiers.Contain(key.ModShift))
			t.Playing().Bool().Toggle()
			return
		case "F5":
			t.NoteTracking().Bool().Set(e.Modifiers.Contain(key.ModShift))
			if e.Modifiers.Contain(key.ModCtrl) {
				t.Model.PlayFromSongStart().Do()
			} else {
				t.Model.PlayFromCurrentPosition().Do()
			}
			return
		case "F6":
			t.NoteTracking().Bool().Set(e.Modifiers.Contain(key.ModShift))
			if e.Modifiers.Contain(key.ModCtrl) {
				t.Model.PlayFromLoopStart().Do()
			} else {
				t.Model.PlaySelected().Do()
			}
			return
		case "F7":
			t.IsRecording().Bool().Toggle()
			return
		case "F8":
			t.StopPlaying().Do()
			return
		case "F9":
			t.NoteTracking().Bool().Toggle()
			return
		case "F12":
			t.Panic().Bool().Toggle()
			return
		case `\`, `<`, `>`:
			if e.Modifiers.Contain(key.ModShift) {
				t.OctaveNumberInput.Int.Add(1)
			} else {
				t.OctaveNumberInput.Int.Add(-1)
			}
		case key.NameTab:
			if e.Modifiers.Contain(key.ModShift) {
				switch {
				case t.OrderEditor.scrollTable.Focused():
					t.InstrumentEditor.unitEditor.sliderList.Focus()
				case t.TrackEditor.scrollTable.Focused():
					t.OrderEditor.scrollTable.Focus()
				case t.InstrumentEditor.Focused():
					if t.InstrumentEditor.enlargeBtn.Bool.Value() {
						t.InstrumentEditor.unitEditor.sliderList.Focus()
					} else {
						t.TrackEditor.scrollTable.Focus()
					}
				default:
					t.InstrumentEditor.Focus()
				}
			} else {
				switch {
				case t.OrderEditor.scrollTable.Focused():
					t.TrackEditor.scrollTable.Focus()
				case t.TrackEditor.scrollTable.Focused():
					t.InstrumentEditor.Focus()
				case t.InstrumentEditor.Focused():
					t.InstrumentEditor.unitEditor.sliderList.Focus()
				default:
					if t.InstrumentEditor.enlargeBtn.Bool.Value() {
						t.InstrumentEditor.Focus()
					} else {
						t.OrderEditor.scrollTable.Focus()
					}
				}
			}
		}
		t.JammingPressed(e)
	} else { // e.State == key.Release
		t.JammingReleased(e)
	}
}

func (t *Tracker) JammingPressed(e key.Event) byte {
	if val, ok := noteMap[e.Name]; ok {
		if _, ok := t.KeyPlaying[e.Name]; !ok {
			n := noteAsValue(t.OctaveNumberInput.Int.Value(), val)
			instr := t.InstrumentEditor.instrumentDragList.TrackerList.Selected()
			t.KeyPlaying[e.Name] = t.InstrNoteOn(instr, n)
			return n
		}
	}
	return 0
}

func (t *Tracker) JammingReleased(e key.Event) bool {
	if noteID, ok := t.KeyPlaying[e.Name]; ok {
		noteID.NoteOff()
		delete(t.KeyPlaying, e.Name)
		return true
	}
	return false
}
