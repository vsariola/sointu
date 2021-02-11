package tracker

import (
	"strconv"
	"strings"

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
func (t *Tracker) KeyEvent(e key.Event) bool {
	if e.State == key.Press {
		if t.InstrumentNameEditor.Focused() {
			return false
		}
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
		case key.NameDeleteForward:
			switch t.EditMode {
			case EditTracks:
				t.DeleteSelection()
				return true
			case EditUnits:
				t.DeleteUnit()
				return true
			}
		case "Space":
			t.TogglePlay()
			return true
		case `\`:
			if e.Modifiers.Contain(key.ModShift) {
				return t.ChangeOctave(1)
			}
			return t.ChangeOctave(-1)
		case key.NameTab:
			if e.Modifiers.Contain(key.ModShift) {
				t.EditMode = (t.EditMode - 1 + 4) % 4
			} else {
				t.EditMode = (t.EditMode + 1) % 4
			}
			return true
		case key.NameUpArrow:
			switch t.EditMode {
			case EditPatterns:
				if e.Modifiers.Contain(key.ModCtrl) {
					t.Cursor.SongRow = SongRow{}
				} else {
					t.Cursor.Row -= t.song.RowsPerPattern
				}
				t.NoteTracking = false
			case EditTracks:
				if e.Modifiers.Contain(key.ModCtrl) {
					t.Cursor.Row -= t.song.RowsPerPattern
				} else {
					t.Cursor.Row--
				}
				t.NoteTracking = false
			case EditUnits:
				t.CurrentUnit--
			case EditParameters:
				t.CurrentParam--
			}
			t.ClampPositions()
			if !e.Modifiers.Contain(key.ModShift) {
				t.Unselect()
			}
			return true
		case key.NameDownArrow:
			switch t.EditMode {
			case EditPatterns:
				if e.Modifiers.Contain(key.ModCtrl) {
					t.Cursor.Row = t.song.TotalRows() - 1
				} else {
					t.Cursor.Row += t.song.RowsPerPattern
				}
				t.NoteTracking = false
			case EditTracks:
				if e.Modifiers.Contain(key.ModCtrl) {
					t.Cursor.Row += t.song.RowsPerPattern
				} else {
					t.Cursor.Row++
				}
				t.NoteTracking = false
			case EditUnits:
				t.CurrentUnit++
			case EditParameters:
				t.CurrentParam++
			}
			t.ClampPositions()
			if !e.Modifiers.Contain(key.ModShift) {
				t.Unselect()
			}
			return true
		case key.NameLeftArrow:
			switch t.EditMode {
			case EditPatterns:
				if e.Modifiers.Contain(key.ModCtrl) {
					t.Cursor.Track = 0
				} else {
					t.Cursor.Track--
				}
			case EditTracks:
				if t.CursorColumn == 0 || !t.TrackShowHex[t.Cursor.Track] || e.Modifiers.Contain(key.ModCtrl) {
					t.Cursor.Track--
					t.CursorColumn = 1
				} else {
					t.CursorColumn--
				}
			case EditUnits:
				t.CurrentInstrument--
			case EditParameters:
				if e.Modifiers.Contain(key.ModShift) {
					t.SetUnitParam(t.GetUnitParam() - 16)
				} else {
					t.SetUnitParam(t.GetUnitParam() - 1)
				}
			}
			t.ClampPositions()
			if !e.Modifiers.Contain(key.ModShift) {
				t.Unselect()
			}
			return true
		case key.NameRightArrow:
			switch t.EditMode {
			case EditPatterns:
				if e.Modifiers.Contain(key.ModCtrl) {
					t.Cursor.Track = len(t.song.Tracks) - 1
				} else {
					t.Cursor.Track++
				}
			case EditTracks:
				if t.CursorColumn == 0 || !t.TrackShowHex[t.Cursor.Track] || e.Modifiers.Contain(key.ModCtrl) {
					t.Cursor.Track++
					t.CursorColumn = 0
				} else {
					t.CursorColumn++
				}
			case EditUnits:
				t.CurrentInstrument++
			case EditParameters:
				if e.Modifiers.Contain(key.ModShift) {
					t.SetUnitParam(t.GetUnitParam() + 16)
				} else {
					t.SetUnitParam(t.GetUnitParam() + 1)
				}
			}
			t.ClampPositions()
			if !e.Modifiers.Contain(key.ModShift) {
				t.Unselect()
			}
			return true
		case "+":
			switch t.EditMode {
			case EditTracks:
				if e.Modifiers.Contain(key.ModCtrl) {
					t.AdjustSelectionPitch(12)
				} else {
					t.AdjustSelectionPitch(1)
				}
				return true
			}
		case "-":
			switch t.EditMode {
			case EditTracks:
				if e.Modifiers.Contain(key.ModCtrl) {
					t.AdjustSelectionPitch(-12)
				} else {
					t.AdjustSelectionPitch(-1)
				}
				return true
			}
		}
		switch t.EditMode {
		case EditPatterns:
			if iv, err := strconv.Atoi(e.Name); err == nil {
				t.SetCurrentPattern(byte(iv))
				return true
			}
			if b := byte(e.Name[0]) - 'A'; len(e.Name) == 1 && b >= 0 && b < 26 {
				t.SetCurrentPattern(b + 10)
				return true
			}
		case EditTracks:
			if t.TrackShowHex[t.Cursor.Track] {
				if iv, err := strconv.ParseInt(e.Name, 16, 8); err == nil {
					t.NumberPressed(byte(iv))
					return true
				}
			} else {
				if e.Name == "A" {
					t.SetCurrentNote(0)
					return true
				}
				if val, ok := noteMap[e.Name]; ok {
					if _, ok := t.KeyPlaying[e.Name]; !ok {
						n := getNoteValue(int(t.Octave.Value), val)
						t.SetCurrentNote(n)
						trk := t.Cursor.Track
						start := t.song.FirstTrackVoice(trk)
						end := start + t.song.Tracks[trk].NumVoices
						t.KeyPlaying[e.Name] = t.sequencer.Trigger(start, end, n)
						return true
					}
				}
			}
		case EditUnits:
			name := e.Name
			if !e.Modifiers.Contain(key.ModShift) {
				name = strings.ToLower(name)
			}
			if val, ok := unitKeyMap[name]; ok {
				if e.Modifiers.Contain(key.ModCtrl) {
					t.SetUnit(val)
					return true
				}
			}
			fallthrough
		case EditParameters:
			if val, ok := noteMap[e.Name]; ok {
				if _, ok := t.KeyPlaying[e.Name]; !ok {
					note := getNoteValue(int(t.Octave.Value), val)
					instr := t.CurrentInstrument
					start := t.song.FirstInstrumentVoice(instr)
					end := start + t.song.Patch.Instruments[instr].NumVoices
					t.KeyPlaying[e.Name] = t.sequencer.Trigger(start, end, note)
					return false
				}
			}
		}
	}
	if e.State == key.Release {
		if f, ok := t.KeyPlaying[e.Name]; ok {
			f()
			delete(t.KeyPlaying, e.Name)
			if t.EditMode == EditTracks && t.Playing && t.getCurrent() == 1 {
				t.SetCurrentNote(0)
			}
		}
	}
	return false
}

// getCurrent returns the current (note) value in current pattern under the cursor
func (t *Tracker) getCurrent() byte {
	return t.song.Tracks[t.Cursor.Track].Patterns[t.song.Tracks[t.Cursor.Track].Sequence[t.Cursor.Pattern]][t.Cursor.Row]
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
