package tracker

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vsariola/sointu"
	"gopkg.in/yaml.v2"
)

type (
	List struct {
		ListData
	}

	ListData interface {
		Selected() int
		Selected2() int
		SetSelected(int)
		SetSelected2(int)
		Count() int
	}

	MutableListData interface {
		change(kind string, severity ChangeSeverity) func()
		cancel()
		move(r Range, delta int) (ok bool)
		delete(r Range) (ok bool)
		marshal(r Range) ([]byte, error)
		unmarshal([]byte) (r Range, err error)
	}

	UnitListItem struct {
		Type, Comment                      string
		Disabled                           bool
		StackNeed, StackBefore, StackAfter int
	}

	UnitYieldFunc       func(index int, item UnitListItem) (ok bool)
	UnitSearchYieldFunc func(index int, item string) (ok bool)

	Units         Model // Units is a list of all the units in the selected instrument, implementing ListData & MutableListData interfaces
	OrderRows     Model // OrderRows is a list of all the order rows, implementing ListData & MutableListData interfaces
	NoteRows      Model // NoteRows is a list of all the note rows, implementing ListData & MutableListData interfaces
	SearchResults Model // SearchResults is a unmutable list of all the search results, implementing ListData interface
	Presets       Model // Presets is a unmutable list of all the presets, implementing ListData interface
)

// Model methods

func (m *Model) Units() *Units                 { return (*Units)(m) }
func (m *Model) OrderRows() *OrderRows         { return (*OrderRows)(m) }
func (m *Model) NoteRows() *NoteRows           { return (*NoteRows)(m) }
func (m *Model) SearchResults() *SearchResults { return (*SearchResults)(m) }

// MoveElements moves the selected elements in a list by delta. If delta is
// negative, the elements move up, otherwise down. The list must implement the
// MutableListData interface.
func (v List) MoveElements(delta int) (ok bool) {
	if delta == 0 {
		return false
	}
	s, ok := v.ListData.(MutableListData)
	if !ok {
		return
	}
	defer s.change("MoveElements", MajorChange)()
	r := v.listRange()
	if !s.move(r, delta) {
		s.cancel()
		return false
	}
	v.SetSelected(v.Selected() + delta)
	v.SetSelected2(v.Selected2() + delta)
	return true
}

// DeleteElements deletes the selected elements in a list. The list must
// implement the MutableListData interface.
func (v List) DeleteElements(backwards bool) (ok bool) {
	d, ok := v.ListData.(MutableListData)
	if !ok {
		return
	}
	defer d.change("DeleteElements", MajorChange)()
	r := v.listRange()
	if !d.delete(r) {
		d.cancel()
		return false
	}
	if backwards && r.Start > 0 {
		r.Start--
	}
	v.SetSelected(r.Start)
	v.SetSelected2(r.Start)
	return true
}

// CopyElements copies the selected elements in a list. The list must implement
// the MutableListData interface. Returns the copied data, marshaled into byte
// slice, and true if successful.
func (v List) CopyElements() ([]byte, bool) {
	m, ok := v.ListData.(MutableListData)
	if !ok {
		return nil, false
	}
	ret, err := m.marshal(v.listRange())
	if err != nil {
		return nil, false
	}
	return ret, true
}

// PasteElements pastes the data into the list. The data is unmarshaled from the
// byte slice. The list must implement the MutableListData interface. Returns
// true if successful.
func (v List) PasteElements(data []byte) (ok bool) {
	m, ok := v.ListData.(MutableListData)
	if !ok {
		return false
	}
	defer m.change("PasteElements", MajorChange)()
	r, err := m.unmarshal(data)
	if err != nil {
		m.cancel()
		return false
	}
	v.SetSelected(r.Start)
	v.SetSelected2(r.End + 1)
	return true
}

func (v *List) listRange() (r Range) {
	return Inclusive(v.Selected(), v.Selected2())
}

// Units methods

func (v *Units) List() List {
	return List{v}
}

func (m *Units) SelectedType() string {
	if m.d.InstrIndex < 0 ||
		m.d.InstrIndex >= len(m.d.Song.Patch) ||
		m.d.UnitIndex < 0 ||
		m.d.UnitIndex >= len(m.d.Song.Patch[m.d.InstrIndex].Units) {
		return ""
	}
	return m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex].Type
}

func (m *Units) SetSelectedType(t string) {
	if m.d.InstrIndex < 0 ||
		m.d.InstrIndex >= len(m.d.Song.Patch) {
		return
	}
	if m.d.UnitIndex < 0 {
		m.d.UnitIndex = 0
	}
	for len(m.d.Song.Patch[m.d.InstrIndex].Units) <= m.d.UnitIndex {
		m.d.Song.Patch[m.d.InstrIndex].Units = append(m.d.Song.Patch[m.d.InstrIndex].Units, sointu.Unit{})
	}
	unit, ok := defaultUnits[t]
	if !ok { // if the type is invalid, we just set it to empty unit
		unit = sointu.Unit{Parameters: make(map[string]int)}
	} else {
		unit = unit.Copy()
	}
	oldUnit := m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex]
	if oldUnit.Type == unit.Type {
		return
	}
	defer m.change("SetSelectedType", MajorChange)()
	m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex] = unit
	m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex].ID = oldUnit.ID // keep the ID of the replaced unit
}

func (v *Units) Iterate(yield UnitYieldFunc) {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return
	}
	stackBefore := 0
	for i, unit := range v.d.Song.Patch[v.d.InstrIndex].Units {
		stackAfter := stackBefore + unit.StackChange()
		if !yield(i, UnitListItem{
			Type:        unit.Type,
			Comment:     unit.Comment,
			Disabled:    unit.Disabled,
			StackNeed:   unit.StackNeed(),
			StackBefore: stackBefore,
			StackAfter:  stackAfter,
		}) {
			break
		}
		stackBefore = stackAfter
	}
}

func (v *Units) Selected() int {
	return max(min(v.d.UnitIndex, v.Count()-1), 0)
}

func (v *Units) Selected2() int {
	return max(min(v.d.UnitIndex2, v.Count()-1), 0)
}

func (v *Units) SetSelected(value int) {
	m := (*Model)(v)
	m.d.UnitIndex = max(min(value, v.Count()-1), 0)
	m.d.ParamIndex = 0
	m.d.UnitSearching = false
	m.d.UnitSearchString = ""
}

func (v *Units) SetSelected2(value int) {
	(*Model)(v).d.UnitIndex2 = max(min(value, v.Count()-1), 0)
}

func (v *Units) Count() int {
	m := (*Model)(v)
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return 0
	}
	return len(m.d.Song.Patch[(*Model)(v).d.InstrIndex].Units)
}

func (v *Units) move(from, to, delta int) (ok bool) {
	m := (*Model)(v)
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	r := MakeMoveRanges(Range{from, to + 1}, delta)
	m.d.Song.Patch[m.d.InstrIndex].Units, ok = Slice(m.d.Song.Patch[m.d.InstrIndex].Units, r[:]...)
	return ok
}

func (v *Units) delete(from, to int) (ok bool) {
	m := (*Model)(v)
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	r := MakeComplementaryRanges(Range{from, to + 1})
	m.d.Song.Patch[m.d.InstrIndex].Units, ok = Slice(m.d.Song.Patch[m.d.InstrIndex].Units, r[:]...)
	return ok
}

func (v *Units) change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("UnitListView."+n, PatchChange, severity)
}

func (v *Units) cancel() {
	(*Model)(v).changeCancel = true
}

func (v *Units) marshal(from, to int) ([]byte, error) {
	m := (*Model)(v)
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return nil, errors.New("UnitListView.marshal: no instruments")
	}
	if from < 0 || to >= len(m.d.Song.Patch[m.d.InstrIndex].Units) || from > to {
		return nil, fmt.Errorf("UnitListView.marshal: index out of range: %d, %d", from, to)
	}
	ret, err := yaml.Marshal(struct{ Units []sointu.Unit }{m.d.Song.Patch[m.d.InstrIndex].Units[from : to+1]})
	if err != nil {
		return nil, fmt.Errorf("UnitListView.marshal: %v", err)
	}
	return ret, nil
}

func (v *Units) unmarshal(data []byte) (from, to int, err error) {
	m := (*Model)(v)
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return 0, 0, errors.New("UnitListView.unmarshal: no instruments")
	}
	var pastedUnits struct{ Units []sointu.Unit }
	if err := yaml.Unmarshal(data, &pastedUnits); err != nil {
		return 0, 0, fmt.Errorf("UnitListView.unmarshal: %v", err)
	}
	if len(pastedUnits.Units) == 0 {
		return 0, 0, errors.New("UnitListView.unmarshal: no units")
	}
	sointu.AvoidUnitIDs(pastedUnits.Units, m.d.Song.Patch)
	sel := v.Selected()
	units := append(m.d.Song.Patch[m.d.InstrIndex].Units, make([]sointu.Unit, len(pastedUnits.Units))...)
	copy(units[sel+len(pastedUnits.Units):], units[sel:])
	copy(units[sel:], pastedUnits.Units)
	m.d.Song.Patch[m.d.InstrIndex].Units = units
	from = sel
	to = sel + len(pastedUnits.Units) - 1
	return
}

// OrderRows methods

func (v *OrderRows) List() List {
	return List{v}
}

func (v *OrderRows) Selected() int {
	p := v.d.Cursor.OrderRow
	p = max(min(p, v.Count()-1), 0)
	return p
}

func (v *OrderRows) Selected2() int {
	p := v.d.Cursor2.OrderRow
	p = max(min(p, v.Count()-1), 0)
	return p
}

func (v *OrderRows) SetSelected(value int) {
	y := max(min(value, v.Count()-1), 0)
	if y != v.d.Cursor.OrderRow {
		v.follow = false
	}
	v.d.Cursor.OrderRow = y
}

func (v *OrderRows) SetSelected2(value int) {
	v.d.Cursor2.OrderRow = max(min(value, v.Count()-1), 0)
}

func (v *OrderRows) swap(x, y int) (ok bool) {
	for i := range v.d.Song.Score.Tracks {
		track := &v.d.Song.Score.Tracks[i]
		a, b := track.Order.Get(x), track.Order.Get(y)
		track.Order.Set(x, b)
		track.Order.Set(y, a)
	}
	return true
}

func (v *OrderRows) delete(from, to int) (ok bool) {
	r := MakeComplementaryRanges(Range{from, to + 1})
	for _, track := range v.d.Song.Score.Tracks {
		track.Order, ok = Slice(track.Order, r[:]...)
		if !ok {
			return false
		}
	}
	return true
}

func (v *OrderRows) change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("OrderRowList."+n, ScoreChange, severity)
}

func (v *OrderRows) cancel() {
	v.changeCancel = true
}

func (v *OrderRows) Count() int {
	return v.d.Song.Score.Length
}

type marshalOrderRows struct {
	Columns [][]int `yaml:",flow"`
}

func (v *OrderRows) marshal(from, to int) ([]byte, error) {
	var table marshalOrderRows
	for i := range v.d.Song.Score.Tracks {
		table.Columns = append(table.Columns, make([]int, to-from+1))
		for j := 0; j < to-from+1; j++ {
			table.Columns[i][j] = v.d.Song.Score.Tracks[i].Order.Get(from + j)
		}
	}
	return yaml.Marshal(table)
}

func (v *OrderRows) unmarshal(data []byte) (from, to int, err error) {
	var table marshalOrderRows
	err = yaml.Unmarshal(data, &table)
	if err != nil {
		return
	}
	if len(table.Columns) == 0 {
		err = errors.New("OrderRowList.unmarshal: no rows")
		return
	}
	from = v.d.Cursor.OrderRow
	to = v.d.Cursor.OrderRow + len(table.Columns[0]) - 1
	for i := range v.d.Song.Score.Tracks {
		if i >= len(table.Columns) {
			break
		}
		order := &v.d.Song.Score.Tracks[i].Order
		for j := 0; j < from-len(*order); j++ {
			*order = append(*order, -1)
		}
		if len(*order) > from {
			table.Columns[i] = append(table.Columns[i], (*order)[from:]...)
			*order = (*order)[:from]
		}
		*order = append(*order, table.Columns[i]...)
	}
	return
}

// NoteRows methods

func (v *NoteRows) List() List {
	return List{v}
}

func (v *NoteRows) Selected() int {
	return v.d.Song.Score.SongRow(v.d.Song.Score.Clamp(v.d.Cursor.SongPos))
}

func (v *NoteRows) Selected2() int {
	return v.d.Song.Score.SongRow(v.d.Song.Score.Clamp(v.d.Cursor2.SongPos))
}

func (v *NoteRows) SetSelected(value int) {
	if value != v.d.Song.Score.SongRow(v.d.Cursor.SongPos) {
		v.follow = false
	}
	v.d.Cursor.SongPos = v.d.Song.Score.Clamp(v.d.Song.Score.SongPos(value))
}

func (v *NoteRows) SetSelected2(value int) {
	v.d.Cursor2.SongPos = v.d.Song.Score.Clamp(v.d.Song.Score.SongPos(value))

}

func (v *NoteRows) swap(i, j int) (ok bool) {
	ipos := v.d.Song.Score.SongPos(i)
	jpos := v.d.Song.Score.SongPos(j)
	for _, track := range v.d.Song.Score.Tracks {
		n1 := track.Note(ipos)
		n2 := track.Note(jpos)
		track.SetNote(ipos, n2, v.uniquePatterns)
		track.SetNote(jpos, n1, v.uniquePatterns)
	}
	return true
}

func (v *NoteRows) delete(i int) (ok bool) {
	if i < 0 || i >= v.Count() {
		return
	}
	pos := v.d.Song.Score.SongPos(i)
	for _, track := range v.d.Song.Score.Tracks {
		track.SetNote(pos, 1, v.uniquePatterns)
	}
	return true
}

func (v *NoteRows) change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("NoteRowList."+n, ScoreChange, severity)
}

func (v *NoteRows) cancel() {
	(*Model)(v).changeCancel = true
}

func (v *NoteRows) Count() int {
	return (*Model)(v).d.Song.Score.Length * v.d.Song.Score.RowsPerPattern
}

type marshalNoteRows struct {
	NoteRows [][]byte `yaml:",flow"`
}

func (v *NoteRows) marshal(from, to int) ([]byte, error) {
	var table marshalNoteRows
	for i, track := range v.d.Song.Score.Tracks {
		table.NoteRows = append(table.NoteRows, make([]byte, to-from+1))
		for j := 0; j < to-from+1; j++ {
			row := from + j
			pos := v.d.Song.Score.SongPos(row)
			table.NoteRows[i][j] = track.Note(pos)
		}
	}
	return yaml.Marshal(table)
}

func (v *NoteRows) unmarshal(data []byte) (from, to int, err error) {
	var table marshalNoteRows
	if err := yaml.Unmarshal(data, &table); err != nil {
		return 0, 0, fmt.Errorf("NoteRowList.unmarshal: %v", err)
	}
	if len(table.NoteRows) < 1 {
		return 0, 0, errors.New("NoteRowList.unmarshal: no tracks")
	}
	from = v.d.Song.Score.SongRow(v.d.Cursor.SongPos)
	for i, arr := range table.NoteRows {
		if i >= len(v.d.Song.Score.Tracks) {
			continue
		}
		to = from + len(arr) - 1
		for j, note := range arr {
			y := j + from
			pos := v.d.Song.Score.SongPos(y)
			v.d.Song.Score.Tracks[i].SetNote(pos, note, v.uniquePatterns)
		}
	}
	return
}

// SearchResults

func (v *SearchResults) List() List {
	return List{v}
}

func (l *SearchResults) Iterate(yield UnitSearchYieldFunc) {
	index := 0
	for _, name := range sointu.UnitNames {
		if !strings.HasPrefix(name, l.d.UnitSearchString) {
			continue
		}
		if !yield(index, name) {
			break
		}
		index++
	}
}

func (l *SearchResults) Selected() int {
	return max(min(l.d.UnitSearchIndex, l.Count()-1), 0)
}

func (l *SearchResults) Selected2() int {
	return max(min(l.d.UnitSearchIndex, l.Count()-1), 0)
}

func (l *SearchResults) SetSelected(value int) {
	l.d.UnitSearchIndex = max(min(value, l.Count()-1), 0)
}

func (l *SearchResults) SetSelected2(value int) {
}

func (l *SearchResults) Count() (count int) {
	for _, n := range sointu.UnitNames {
		if strings.HasPrefix(n, l.d.UnitSearchString) {
			count++
		}
	}
	return
}
