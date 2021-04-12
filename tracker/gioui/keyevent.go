package gioui

import (
	"strconv"
	"strings"
	"time"

	"gioui.org/app"
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

var unitKeyMap = map[string]string{
	"e": "envelope",
	"o": "oscillator",
	"m": "mulp",
	"M": "mul",
	"a": "addp",
	"A": "add",
	"p": "pan",
	"S": "push",
	"P": "pop",
	"O": "out",
	"l": "loadnote",
	"L": "loadval",
	"h": "xch",
	"d": "delay",
	"D": "distort",
	"H": "hold",
	"b": "crush",
	"g": "gain",
	"i": "invgain",
	"f": "filter",
	"I": "clip",
	"E": "speed",
	"r": "compressor",
	"u": "outaux",
	"U": "aux",
	"s": "send",
	"n": "noise",
	"N": "in",
	"R": "receive",
}

// KeyEvent handles incoming key events and returns true if repaint is needed.
func (t *Tracker) KeyEvent(w *app.Window, e key.Event) bool {
	if e.State == key.Press {
		if t.InstrumentNameEditor.Focused() {
			return false
		}
		switch e.Name {
		case "C":
			if e.Modifiers.Contain(key.ModShortcut) {
				contents, err := yaml.Marshal(t.Song())
				if err == nil {
					w.WriteClipboard(string(contents))
					t.Alert.Update("Song copied to clipboard", Notify, time.Second*3)
				}
				return true
			}
		case "V":
			if e.Modifiers.Contain(key.ModShortcut) {
				w.ReadClipboard()
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
				t.ResetSong()
				return true
			}
		case "S":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.SaveSongFile()
				return false
			}
		case "O":
			if e.Modifiers.Contain(key.ModShortcut) {
				t.LoadSongFile()
				return true
			}
		case "F1":
			t.SetEditMode(tracker.EditPatterns)
			return true
		case "F2":
			t.SetEditMode(tracker.EditTracks)
			return true
		case "F3":
			t.SetEditMode(tracker.EditUnits)
			return true
		case "F4":
			t.SetEditMode(tracker.EditParameters)
			return true
		case "F5":
			t.SetNoteTracking(true)
			startRow := t.Cursor().SongRow
			if t.EditMode() == tracker.EditPatterns {
				startRow.Row = 0
			}
			t.player.Play(startRow)
			return true
		case "F6":
			t.SetNoteTracking(false)
			startRow := t.Cursor().SongRow
			if t.EditMode() == tracker.EditPatterns {
				startRow.Row = 0
			}
			t.player.Play(startRow)
			return true
		case "F8":
			t.player.Stop()
			return true
		case key.NameDeleteForward, key.NameDeleteBackward:
			switch t.EditMode() {
			case tracker.EditPatterns:
				if e.Modifiers.Contain(key.ModShortcut) {
					t.DeleteOrderRow(e.Name == key.NameDeleteForward)
				} else {
					t.DeletePatternSelection()
					if !(t.NoteTracking() && t.player.Playing()) && t.Step.Value > 0 {
						t.SetCursor(t.Cursor().AddPatterns(1))
						t.SetSelectionCorner(t.Cursor())
					}
				}
				return true
			case tracker.EditTracks:
				t.DeleteSelection()
				if !(t.NoteTracking() && t.player.Playing()) && t.Step.Value > 0 {
					t.SetCursor(t.Cursor().AddRows(t.Step.Value))
					t.SetSelectionCorner(t.Cursor())
				}
				return true
			case tracker.EditUnits:
				t.DeleteUnit(e.Name == key.NameDeleteForward)
				return true
			}
		case "Space":
			_, playing := t.player.Position()
			if !playing {
				t.SetNoteTracking(!e.Modifiers.Contain(key.ModShortcut))
				startRow := t.Cursor().SongRow
				if t.EditMode() == tracker.EditPatterns {
					startRow.Row = 0
				}
				t.player.Play(startRow)
			} else {
				t.player.Stop()
			}
			return true
		case `\`, `<`, `>`:
			if e.Modifiers.Contain(key.ModShift) {
				return t.SetOctave(t.Octave() + 1)
			}
			return t.SetOctave(t.Octave() - 1)
		case key.NameTab:
			if e.Modifiers.Contain(key.ModShift) {
				t.SetEditMode((t.EditMode() - 1 + 4) % 4)
			} else {
				t.SetEditMode((t.EditMode() + 1) % 4)
			}
			return true
		case key.NameReturn:
			switch t.EditMode() {
			case tracker.EditPatterns:
				t.AddOrderRow(!e.Modifiers.Contain(key.ModShortcut))
			case tracker.EditUnits:
				t.AddUnit(!e.Modifiers.Contain(key.ModShortcut))
			}
		case key.NameUpArrow:
			cursor := t.Cursor()
			switch t.EditMode() {
			case tracker.EditPatterns:
				if e.Modifiers.Contain(key.ModShortcut) {
					cursor.SongRow = tracker.SongRow{}
				} else {
					cursor.Row -= t.Song().Score.RowsPerPattern
				}
				t.SetNoteTracking(false)
			case tracker.EditTracks:
				if e.Modifiers.Contain(key.ModShortcut) {
					cursor.Row -= t.Song().Score.RowsPerPattern
				} else {
					if t.Step.Value > 0 {
						cursor.Row -= t.Step.Value
					} else {
						cursor.Row--
					}
				}
				t.SetNoteTracking(false)
			case tracker.EditUnits:
				t.SetUnitIndex(t.UnitIndex() - 1)
			case tracker.EditParameters:
				t.SetParamIndex(t.ParamIndex() - 1)
			}
			t.SetCursor(cursor)
			if !e.Modifiers.Contain(key.ModShift) {
				t.SetSelectionCorner(t.Cursor())
			}
			scrollToView(t.PatternOrderList, t.Cursor().Pattern, t.Song().Score.Length)
			return true
		case key.NameDownArrow:
			cursor := t.Cursor()
			switch t.EditMode() {
			case tracker.EditPatterns:
				if e.Modifiers.Contain(key.ModShortcut) {
					cursor.Row = t.Song().Score.LengthInRows() - 1
				} else {
					cursor.Row += t.Song().Score.RowsPerPattern
				}
				t.SetNoteTracking(false)
			case tracker.EditTracks:
				if e.Modifiers.Contain(key.ModShortcut) {
					cursor.Row += t.Song().Score.RowsPerPattern
				} else {
					if t.Step.Value > 0 {
						cursor.Row += t.Step.Value
					} else {
						cursor.Row++
					}
				}
				t.SetNoteTracking(false)
			case tracker.EditUnits:
				t.SetUnitIndex(t.UnitIndex() + 1)
			case tracker.EditParameters:
				t.SetParamIndex(t.ParamIndex() + 1)
			}
			t.SetCursor(cursor)
			if !e.Modifiers.Contain(key.ModShift) {
				t.SetSelectionCorner(t.Cursor())
			}
			scrollToView(t.PatternOrderList, t.Cursor().Pattern, t.Song().Score.Length)
			return true
		case key.NameLeftArrow:
			cursor := t.Cursor()
			switch t.EditMode() {
			case tracker.EditPatterns:
				if e.Modifiers.Contain(key.ModShortcut) {
					cursor.Track = 0
				} else {
					cursor.Track--
				}
			case tracker.EditTracks:
				if !t.LowNibble() || !t.Song().Score.Tracks[t.Cursor().Track].Effect || e.Modifiers.Contain(key.ModShortcut) {
					cursor.Track--
					t.SetLowNibble(true)
				} else {
					t.SetLowNibble(false)
				}
			case tracker.EditUnits:
				t.SetInstrIndex(t.InstrIndex() - 1)
			case tracker.EditParameters:
				param, _ := t.Param(t.ParamIndex())
				if e.Modifiers.Contain(key.ModShift) {
					p, err := t.Param(t.ParamIndex())
					if err == nil {
						t.SetParam(param.Value - p.LargeStep)
					}
				} else {
					t.SetParam(param.Value - 1)
				}
			}
			t.SetCursor(cursor)
			if !e.Modifiers.Contain(key.ModShift) {
				t.SetSelectionCorner(t.Cursor())
			}
			return true
		case key.NameRightArrow:
			switch t.EditMode() {
			case tracker.EditPatterns:
				cursor := t.Cursor()
				if e.Modifiers.Contain(key.ModShortcut) {
					cursor.Track = len(t.Song().Score.Tracks) - 1
				} else {
					cursor.Track++
				}
				t.SetCursor(cursor)
			case tracker.EditTracks:
				if t.LowNibble() || !t.Song().Score.Tracks[t.Cursor().Track].Effect || e.Modifiers.Contain(key.ModShortcut) {
					cursor := t.Cursor()
					cursor.Track++
					t.SetCursor(cursor)
					t.SetLowNibble(false)
				} else {
					t.SetLowNibble(true)
				}
			case tracker.EditUnits:
				t.SetInstrIndex(t.InstrIndex() + 1)
			case tracker.EditParameters:
				param, _ := t.Param(t.ParamIndex())
				if e.Modifiers.Contain(key.ModShift) {
					p, err := t.Param(t.ParamIndex())
					if err == nil {
						t.SetParam(param.Value + p.LargeStep)
					}
				} else {
					t.SetParam(param.Value + 1)
				}
			}
			if !e.Modifiers.Contain(key.ModShift) {
				t.SetSelectionCorner(t.Cursor())
			}
			return true
		case "+":
			switch t.EditMode() {
			case tracker.EditTracks:
				if e.Modifiers.Contain(key.ModShortcut) {
					t.AdjustSelectionPitch(12)
				} else {
					t.AdjustSelectionPitch(1)
				}
				return true
			}
		case "-":
			switch t.EditMode() {
			case tracker.EditTracks:
				if e.Modifiers.Contain(key.ModShortcut) {
					t.AdjustSelectionPitch(-12)
				} else {
					t.AdjustSelectionPitch(-1)
				}
				return true
			}
		}
		switch t.EditMode() {
		case tracker.EditPatterns:
			if iv, err := strconv.Atoi(e.Name); err == nil {
				t.SetCurrentPattern(iv)
				if !(t.NoteTracking() && t.player.Playing()) && t.Step.Value > 0 {
					t.SetCursor(t.Cursor().AddPatterns(1))
					t.SetSelectionCorner(t.Cursor())
				}
				return true
			}
			if b := int(e.Name[0]) - 'A'; len(e.Name) == 1 && b >= 0 && b < 26 {
				t.SetCurrentPattern(b + 10)
				if !(t.NoteTracking() && t.player.Playing()) && t.Step.Value > 0 {
					t.SetCursor(t.Cursor().AddPatterns(1))
					t.SetSelectionCorner(t.Cursor())
				}
				return true
			}
		case tracker.EditTracks:
			step := false
			if t.Song().Score.Tracks[t.Cursor().Track].Effect {
				if iv, err := strconv.ParseInt(e.Name, 16, 8); err == nil {
					t.NumberPressed(byte(iv))
					step = true
				}
			} else {
				if e.Name == "A" || e.Name == "1" {
					t.SetNote(0)
					step = true
				} else {
					if val, ok := noteMap[e.Name]; ok {
						if _, ok := t.KeyPlaying[e.Name]; !ok {
							n := tracker.NoteAsValue(t.OctaveNumberInput.Value, val)
							t.SetNote(n)
							step = true
							trk := t.Cursor().Track
							start := t.Song().Score.FirstVoiceForTrack(trk)
							end := start + t.Song().Score.Tracks[trk].NumVoices
							t.KeyPlaying[e.Name] = t.player.Trigger(start, end, n)
						}
					}
				}
			}
			if step && !(t.NoteTracking() && t.player.Playing()) && t.Step.Value > 0 {
				t.SetCursor(t.Cursor().AddRows(t.Step.Value))
				t.SetSelectionCorner(t.Cursor())
			}
			return true
		case tracker.EditUnits:
			name := e.Name
			if !e.Modifiers.Contain(key.ModShift) {
				name = strings.ToLower(name)
			}
			if val, ok := unitKeyMap[name]; ok {
				if e.Modifiers.Contain(key.ModShortcut) {
					t.SetUnitType(val)
					return true
				}
			}
			fallthrough
		case tracker.EditParameters:
			if val, ok := noteMap[e.Name]; ok {
				if _, ok := t.KeyPlaying[e.Name]; !ok {
					n := tracker.NoteAsValue(t.OctaveNumberInput.Value, val)
					instr := t.InstrIndex()
					start := t.Song().Patch.FirstVoiceForInstrument(instr)
					end := start + t.Instrument().NumVoices
					t.KeyPlaying[e.Name] = t.player.Trigger(start, end, n)
					return false
				}
			}
		}
	}
	if e.State == key.Release {
		if ID, ok := t.KeyPlaying[e.Name]; ok {
			t.player.Release(ID)
			delete(t.KeyPlaying, e.Name)
			if _, playing := t.player.Position(); t.EditMode() == tracker.EditTracks && playing && t.Note() == 1 && t.NoteTracking() {
				t.SetNote(0)
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
