package gioui

import (
	"time"

	"gioui.org/io/key"
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
func (t *Tracker) KeyEvent(e key.Event) bool {
	if e.State == key.Press {
		if t.OpenSongDialog.Visible ||
			t.SaveSongDialog.Visible ||
			t.SaveInstrumentDialog.Visible ||
			t.OpenInstrumentDialog.Visible ||
			t.ExportWavDialog.Visible {
			return false
		}
		switch e.Name {
		case "C":
			if e.Modifiers.Contain(key.ModShortcut) {
				contents, err := yaml.Marshal(t.Song())
				if err == nil {
					t.window.WriteClipboard(string(contents))
					t.Alert.Update("Song copied to clipboard", Notify, time.Second*3)
				}
				return true
			}
		case "V":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.window.ReadClipboard()
				return true
			}
		case "Z":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.Undo()
				return true
			}
		case "Y":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.Redo()
				return true
			}
		case "N":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.NewSong(false)
				return true
			}
		case "S":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.SaveSongFile()
				return false
			}
		case "O":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.OpenSongFile(false)
				return true
			}
		case "F1":
			t.OrderEditor.Focus()
			return true
		case "F2":
			t.TrackEditor.Focus()
			return true
		case "F3":
			t.InstrumentEditor.Focus()
			return true
		case "F4":
			t.TrackEditor.Focus()
			return true
		case "F5":
			t.SetNoteTracking(true)
			startRow := t.Cursor().SongRow
			if t.OrderEditor.Focused() {
				startRow.Row = 0
			}
			t.player.Play(startRow)
			return true
		case "F6":
			t.SetNoteTracking(false)
			startRow := t.Cursor().SongRow
			if t.OrderEditor.Focused() {
				startRow.Row = 0
			}
			t.player.Play(startRow)
			return true
		case "F8":
			t.player.Stop()
			return true
		case "Space":
			_, playing := t.player.Position()
			if !playing {
				t.SetNoteTracking(!e.Modifiers.Contain(key.ModShortcut))
				startRow := t.Cursor().SongRow
				t.player.Play(startRow)
			} else {
				t.player.Stop()
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
					t.TrackEditor.Focus()
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
					t.OrderEditor.Focus()
				}
			}
		}
	}

	return false
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

func (t *Tracker) JammingPressed(e key.Event) {
	if val, ok := noteMap[e.Name]; ok {
		if _, ok := t.KeyPlaying[e.Name]; !ok {
			n := tracker.NoteAsValue(t.OctaveNumberInput.Value, val)
			instr := t.InstrIndex()
			start := t.Song().Patch.FirstVoiceForInstrument(instr)
			end := start + t.Instrument().NumVoices
			t.KeyPlaying[e.Name] = t.player.Trigger(start, end, n)
		}
	}
}

func (t *Tracker) JammingReleased(e key.Event) {
	if ID, ok := t.KeyPlaying[e.Name]; ok {
		t.player.Release(ID)
		delete(t.KeyPlaying, e.Name)
		if _, playing := t.player.Position(); t.TrackEditor.focused && playing && t.Note() == 1 && t.NoteTracking() {
			t.SetNote(0)
		}
	}
}
