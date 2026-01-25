package tracker

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/vsariola/sointu"
	"gopkg.in/yaml.v3"
)

// Note returns the Note view of the model, containing methods to manipulate
// the note data.
func (m *Model) Note() *NoteModel { return (*NoteModel)(m) }

type NoteModel Model

// Step returns an Int controlling how many note rows the cursor advances every
// time the user inputs a note.
func (m *NoteModel) Step() Int { return MakeInt((*noteStep)(m)) }

type noteStep NoteModel

func (v *noteStep) Value() int { return v.d.Step }
func (v *noteStep) SetValue(value int) bool {
	defer (*Model)(v).change("StepInt", NoChange, MinorChange)()
	v.d.Step = value
	return true
}
func (v *noteStep) Range() RangeInclusive { return RangeInclusive{0, 8} }

// UniquePatterns returns a Bool controlling whether patterns are made unique
// when editing notes.
func (m *NoteModel) UniquePatterns() Bool { return MakeBoolFromPtr(&m.uniquePatterns) }

// Octave returns an Int controlling the current octave for note input.
func (m *NoteModel) Octave() Int { return MakeInt((*noteOctave)(m)) }

type noteOctave NoteModel

func (v *noteOctave) Value() int              { return v.d.Octave }
func (v *noteOctave) SetValue(value int) bool { v.d.Octave = value; return true }
func (v *noteOctave) Range() RangeInclusive   { return RangeInclusive{0, 9} }

// AddSemiTone returns an Action for adding a semitone to the selected notes.
func (m *NoteModel) AddSemitone() Action { return MakeAction((*addSemitone)(m)) }

type addSemitone NoteModel

func (m *addSemitone) Do() { Table{(*NoteModel)(m)}.Add(1, false) }

// SubtractSemitone returns an Action for subtracting a semitone from the
// selected notes.
func (m *NoteModel) SubtractSemitone() Action { return MakeAction((*subtractSemitone)(m)) }

type subtractSemitone NoteModel

func (m *subtractSemitone) Do() { Table{(*NoteModel)(m)}.Add(-1, false) }

// AddOctave returns an Action for adding an octave to the selected notes.
func (m *NoteModel) AddOctave() Action { return MakeAction((*addOctave)(m)) }

type addOctave NoteModel

func (m *addOctave) Do() { Table{(*NoteModel)(m)}.Add(1, true) }

// SubtractOctave returns an Action for subtracting an octave from the selected
// notes.
func (m *NoteModel) SubtractOctave() Action { return MakeAction((*subtractOctave)(m)) }

type subtractOctave NoteModel

func (m *subtractOctave) Do() { Table{(*NoteModel)(m)}.Add(-1, true) }

// NoteOff returns an Action to set the selected notes to Note Off (0).
func (m *NoteModel) NoteOff() Action { return MakeAction((*editNoteOff)(m)) }

type editNoteOff NoteModel

func (m *editNoteOff) Do() { Table{(*NoteModel)(m)}.Fill(0) }

// RowList is a list of all the note rows, implementing ListData & MutableListData
// interfaces
func (m *NoteModel) RowList() List { return List{(*noteRows)(m)} }

type noteRows NoteModel

func (n *noteRows) Count() int         { return n.d.Song.Score.Length * n.d.Song.Score.RowsPerPattern }
func (n *noteRows) Selected() int      { return n.d.Song.Score.SongRow(n.d.Cursor.SongPos) }
func (n *noteRows) Selected2() int     { return n.d.Song.Score.SongRow(n.d.Cursor2.SongPos) }
func (n *noteRows) SetSelected2(v int) { n.d.Cursor2.SongPos = n.d.Song.Score.SongPos(v) }
func (n *noteRows) SetSelected(value int) {
	if value != n.d.Song.Score.SongRow(n.d.Cursor.SongPos) {
		n.follow = false
	}
	n.d.Cursor.SongPos = n.d.Song.Score.Clamp(n.d.Song.Score.SongPos(value))
}

func (v *noteRows) Move(r Range, delta int) (ok bool) {
	for a, b := range r.Swaps(delta) {
		apos := v.d.Song.Score.SongPos(a)
		bpos := v.d.Song.Score.SongPos(b)
		for _, t := range v.d.Song.Score.Tracks {
			n1 := t.Note(apos)
			n2 := t.Note(bpos)
			t.SetNote(apos, n2, v.uniquePatterns)
			t.SetNote(bpos, n1, v.uniquePatterns)
		}
	}
	return true
}

func (v *noteRows) Delete(r Range) (ok bool) {
	for _, track := range v.d.Song.Score.Tracks {
		for i := r.Start; i < r.End; i++ {
			pos := v.d.Song.Score.SongPos(i)
			track.SetNote(pos, 1, v.uniquePatterns)
		}
	}
	return true
}

func (v *noteRows) Change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("NoteRowList."+n, ScoreChange, severity)
}

func (v *noteRows) Cancel() {
	(*Model)(v).changeCancel = true
}

type marshalNoteRows struct {
	NoteRows [][]byte `yaml:",flow"`
}

func (v *noteRows) Marshal(r Range) ([]byte, error) {
	var table marshalNoteRows
	for i, track := range v.d.Song.Score.Tracks {
		table.NoteRows = append(table.NoteRows, make([]byte, r.Len()))
		for j := 0; j < r.Len(); j++ {
			row := r.Start + j
			pos := v.d.Song.Score.SongPos(row)
			table.NoteRows[i][j] = track.Note(pos)
		}
	}
	return yaml.Marshal(table)
}

func (v *noteRows) Unmarshal(data []byte) (r Range, err error) {
	var table marshalNoteRows
	if err := yaml.Unmarshal(data, &table); err != nil {
		return Range{}, fmt.Errorf("NoteRowList.unmarshal: %v", err)
	}
	if len(table.NoteRows) < 1 {
		return Range{}, errors.New("NoteRowList.unmarshal: no tracks")
	}
	r.Start = v.d.Song.Score.SongRow(v.d.Cursor.SongPos)
	for i, arr := range table.NoteRows {
		if i >= len(v.d.Song.Score.Tracks) {
			continue
		}
		r.End = r.Start + len(arr)
		for j, note := range arr {
			y := j + r.Start
			pos := v.d.Song.Score.SongPos(y)
			v.d.Song.Score.Tracks[i].SetNote(pos, note, v.uniquePatterns)
		}
	}
	return
}

// Table returns a Table of all the note data.
func (v *NoteModel) Table() Table { return Table{v} }

func (m *NoteModel) Cursor() Point {
	t := max(min(m.d.Cursor.Track, len(m.d.Song.Score.Tracks)-1), 0)
	p := max(min(m.d.Song.Score.SongRow(m.d.Cursor.SongPos), m.d.Song.Score.LengthInRows()-1), 0)
	return Point{t, p}
}

func (m *NoteModel) Cursor2() Point {
	t := max(min(m.d.Cursor2.Track, len(m.d.Song.Score.Tracks)-1), 0)
	p := max(min(m.d.Song.Score.SongRow(m.d.Cursor2.SongPos), m.d.Song.Score.LengthInRows()-1), 0)
	return Point{t, p}
}

func (v *NoteModel) SetCursor(p Point) {
	v.d.Cursor.Track = max(min(p.X, len(v.d.Song.Score.Tracks)-1), 0)
	newPos := v.d.Song.Score.Clamp(sointu.SongPos{PatternRow: p.Y})
	if newPos != v.d.Cursor.SongPos {
		v.follow = false
	}
	v.d.Cursor.SongPos = newPos
}

func (v *NoteModel) SetCursor2(p Point) {
	v.d.Cursor2.Track = max(min(p.X, len(v.d.Song.Score.Tracks)-1), 0)
	v.d.Cursor2.SongPos = v.d.Song.Score.Clamp(sointu.SongPos{PatternRow: p.Y})
}

func (m *NoteModel) SetCursorFloat(x, y float32) {
	m.SetCursor(Point{int(x), int(y)})
	m.d.LowNibble = math.Mod(float64(x), 1.0) > 0.5
}

func (v *NoteModel) Width() int {
	return len((*Model)(v).d.Song.Score.Tracks)
}

func (v *NoteModel) Height() int {
	return (*Model)(v).d.Song.Score.Length * (*Model)(v).d.Song.Score.RowsPerPattern
}

func (v *NoteModel) MoveCursor(dx, dy int) (ok bool) {
	p := v.Cursor()
	for dx < 0 {
		if (*TrackModel)(v).Item(p.X).Effect && v.d.LowNibble {
			v.d.LowNibble = false
		} else {
			p.X--
			v.d.LowNibble = true
		}
		dx++
	}
	for dx > 0 {
		if (*TrackModel)(v).Item(p.X).Effect && !v.d.LowNibble {
			v.d.LowNibble = true
		} else {
			p.X++
			v.d.LowNibble = false
		}
		dx--
	}
	p.Y += dy
	v.SetCursor(p)
	return p == v.Cursor()
}

func (v *NoteModel) clear(p Point) {
	v.Input(1)
}

func (v *NoteModel) set(p Point, value int) {
	v.SetValue(p, byte(value))
}

func (v *NoteModel) add(rect Rect, delta int, largeStep bool) (ok bool) {
	if largeStep {
		delta *= 12
	}
	for x := rect.BottomRight.X; x >= rect.TopLeft.X; x-- {
		for y := rect.BottomRight.Y; y >= rect.TopLeft.Y; y-- {
			if x < 0 || x >= len(v.d.Song.Score.Tracks) || y < 0 || y >= v.d.Song.Score.LengthInRows() {
				continue
			}
			pos := v.d.Song.Score.SongPos(y)
			note := v.d.Song.Score.Tracks[x].Note(pos)
			if note <= 1 {
				continue
			}
			newVal := int(note) + delta
			if newVal < 2 {
				newVal = 2
			} else if newVal > 255 {
				newVal = 255
			}
			// only do all sets after all gets, so we don't accidentally adjust single note multiple times
			defer v.d.Song.Score.Tracks[x].SetNote(pos, byte(newVal), v.uniquePatterns)
		}
	}
	return true
}

type noteTable struct {
	Notes [][]byte `yaml:",flow"`
}

func (m *NoteModel) marshal(rect Rect) (data []byte, ok bool) {
	width := rect.BottomRight.X - rect.TopLeft.X + 1
	height := rect.BottomRight.Y - rect.TopLeft.Y + 1
	var table = noteTable{Notes: make([][]byte, 0, width)}
	for x := 0; x < width; x++ {
		table.Notes = append(table.Notes, make([]byte, 0, rect.BottomRight.Y-rect.TopLeft.Y+1))
		for y := 0; y < height; y++ {
			pos := m.d.Song.Score.SongPos(y + rect.TopLeft.Y)
			ax := x + rect.TopLeft.X
			if ax < 0 || ax >= len(m.d.Song.Score.Tracks) {
				continue
			}
			table.Notes[x] = append(table.Notes[x], m.d.Song.Score.Tracks[ax].Note(pos))
		}
	}
	ret, err := yaml.Marshal(table)
	if err != nil {
		return nil, false
	}
	return ret, true
}

func (v *NoteModel) unmarshal(data []byte) (noteTable, bool) {
	var table noteTable
	yaml.Unmarshal(data, &table)
	if len(table.Notes) == 0 {
		return noteTable{}, false
	}
	for i := 0; i < len(table.Notes); i++ {
		if len(table.Notes[i]) > 0 {
			return table, true
		}
	}
	return noteTable{}, false
}

func (v *NoteModel) unmarshalAtCursor(data []byte) bool {
	table, ok := v.unmarshal(data)
	if !ok {
		return false
	}
	for i := 0; i < len(table.Notes); i++ {
		for j, q := range table.Notes[i] {
			x := i + v.Cursor().X
			y := j + v.Cursor().Y
			if x < 0 || x >= len(v.d.Song.Score.Tracks) || y < 0 || y >= v.d.Song.Score.LengthInRows() {
				continue
			}
			pos := v.d.Song.Score.SongPos(y)
			v.d.Song.Score.Tracks[x].SetNote(pos, q, v.uniquePatterns)
		}
	}
	return true
}

func (v *NoteModel) unmarshalRange(rect Rect, data []byte) bool {
	table, ok := v.unmarshal(data)
	if !ok {
		return false
	}
	for i := 0; i < rect.Width(); i++ {
		for j := 0; j < rect.Height(); j++ {
			k := i % len(table.Notes)
			l := j % len(table.Notes[k])
			a := table.Notes[k][l]
			x := i + rect.TopLeft.X
			y := j + rect.TopLeft.Y
			if x < 0 || x >= len(v.d.Song.Score.Tracks) || y < 0 || y >= v.d.Song.Score.LengthInRows() {
				continue
			}
			pos := v.d.Song.Score.SongPos(y)
			v.d.Song.Score.Tracks[x].SetNote(pos, a, v.uniquePatterns)
		}
	}
	return true
}

func (v *NoteModel) change(kind string, severity ChangeSeverity) func() {
	return (*Model)(v).change("OrderTableView."+kind, ScoreChange, severity)
}

func (v *NoteModel) cancel() {
	v.changeCancel = true
}

// At returns the note value at the given point.
func (m *NoteModel) At(p Point) byte {
	if p.Y < 0 || p.X < 0 || p.X >= len(m.d.Song.Score.Tracks) {
		return 1
	}
	pos := m.d.Song.Score.SongPos(p.Y)
	return m.d.Song.Score.Tracks[p.X].Note(pos)
}

// LowNibble returns whether the user is currently editing the low nibble of the
// note value when editing an effect track.
func (m *NoteModel) LowNibble() bool { return m.d.LowNibble }

// SetValue sets the note value at the given point.
func (m *NoteModel) SetValue(p Point, val byte) {
	defer m.change("SetValue", MinorChange)()
	if p.Y < 0 || p.X < 0 || p.X >= len(m.d.Song.Score.Tracks) {
		return
	}
	track := &(m.d.Song.Score.Tracks[p.X])
	pos := m.d.Song.Score.SongPos(p.Y)
	(*track).SetNote(pos, val, m.uniquePatterns)
}

// Input fills the current selection of the note table with a given note value,
// returning a NoteEvent telling which note should be played.
func (v *NoteModel) Input(note byte) NoteEvent {
	v.Table().Fill(int(note))
	return v.finishInput(note)
}

// InputNibble fills the nibbles of current selection of the note table with a
// given nibble value. LowNibble tells whether the user is currently editing the
// low or high nibbles. It returns a NoteEvent telling which note should be
// played.
func (v *NoteModel) InputNibble(nibble byte) NoteEvent {
	defer v.change("FillNibble", MajorChange)()
	rect := Table{v}.Range()
	for y := rect.TopLeft.Y; y <= rect.BottomRight.Y; y++ {
		for x := rect.TopLeft.X; x <= rect.BottomRight.X; x++ {
			val := v.At(Point{x, y})
			if val == 1 {
				val = 0 // treat hold also as 0
			}
			if v.d.LowNibble {
				val = (val & 0xf0) | byte(nibble&15)
			} else {
				val = (val & 0x0f) | byte((nibble&15)<<4)
			}
			v.SetValue(Point{x, y}, val)
		}
	}
	return v.finishInput(v.At(v.Cursor()))
}

func (v *NoteModel) finishInput(note byte) NoteEvent {
	if step := v.d.Step; step > 0 {
		v.Table().MoveCursor(0, step)
		v.Table().SetCursor2(v.Table().Cursor())
	}
	TrySend(v.broker.ToGUI, any(MsgToGUI{Kind: GUIMessageEnsureCursorVisible, Param: v.Table().Cursor().Y}))
	track := v.Cursor().X
	ts := time.Now().UnixMilli() * 441 / 10 // convert to 44100Hz frames
	return NoteEvent{IsTrack: true, Channel: track, Note: note, On: true, Timestamp: ts}
}
