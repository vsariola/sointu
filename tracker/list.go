package tracker

import (
	"errors"
	"fmt"
	"iter"
	"math"
	"math/bits"
	"strings"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
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

	// Range is used to represent a range [Start,End) of integers
	Range struct {
		Start, End int
	}

	UnitYieldFunc       func(index int, item UnitListItem) (ok bool)
	UnitSearchYieldFunc func(index int, item string) (ok bool)

	Instruments   Model // Instruments is a list of instruments, implementing ListData & MutableListData interfaces
	Units         Model // Units is a list of all the units in the selected instrument, implementing ListData & MutableListData interfaces
	Tracks        Model // Tracks is a list of all the tracks, implementing ListData & MutableListData interfaces
	OrderRows     Model // OrderRows is a list of all the order rows, implementing ListData & MutableListData interfaces
	NoteRows      Model // NoteRows is a list of all the note rows, implementing ListData & MutableListData interfaces
	SearchResults Model // SearchResults is a unmutable list of all the search results, implementing ListData interface
	Presets       Model // Presets is a unmutable list of all the presets, implementing ListData interface
)

// Model methods

func (m *Model) Instruments() *Instruments     { return (*Instruments)(m) }
func (m *Model) Units() *Units                 { return (*Units)(m) }
func (m *Model) Tracks() *Tracks               { return (*Tracks)(m) }
func (m *Model) OrderRows() *OrderRows         { return (*OrderRows)(m) }
func (m *Model) NoteRows() *NoteRows           { return (*NoteRows)(m) }
func (m *Model) SearchResults() *SearchResults { return (*SearchResults)(m) }

// MoveElements moves the selected elements in a list by delta. The list must
// implement the MutableListData interface.
func (v List) MoveElements(delta int) bool {
	s, ok := v.ListData.(MutableListData)
	if !ok {
		return false
	}
	r := v.listRange()
	if delta == 0 || r.Start+delta < 0 || r.End+delta > v.Count() {
		return false
	}
	defer s.change("MoveElements", MajorChange)()
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
func (v List) DeleteElements(backwards bool) bool {
	d, ok := v.ListData.(MutableListData)
	if !ok {
		return false
	}
	r := v.listRange()
	if r.Len() == 0 {
		return false
	}
	defer d.change("DeleteElements", MajorChange)()
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
	r := v.listRange()
	if r.Len() == 0 {
		return nil, false
	}
	ret, err := m.marshal(r)
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
	v.SetSelected2(r.End - 1)
	return true
}

func (v *List) listRange() (r Range) {
	r.Start = max(min(v.Selected(), v.Selected2()), 0)
	r.End = min(max(v.Selected(), v.Selected2())+1, v.Count())
	return
}

// Instruments methods

func (v *Instruments) List() List {
	return List{v}
}

func (v *Instruments) Item(i int) (name string, maxLevel float32, mute bool, ok bool) {
	if i < 0 || i >= len(v.d.Song.Patch) {
		return "", 0, false, false
	}
	name = v.d.Song.Patch[i].Name
	mute = v.d.Song.Patch[i].Mute
	start := v.d.Song.Patch.FirstVoiceForInstrument(i)
	end := start + v.d.Song.Patch[i].NumVoices
	if end >= vm.MAX_VOICES {
		end = vm.MAX_VOICES
	}
	if start < end {
		for _, level := range v.voiceLevels[start:end] {
			if maxLevel < level {
				maxLevel = level
			}
		}
	}
	ok = true
	return
}
func (v *Instruments) FirstID(i int) (id int, ok bool) {
	if i < 0 || i >= len(v.d.Song.Patch) {
		return 0, false
	}
	if len(v.d.Song.Patch[i].Units) == 0 {
		return 0, false
	}
	return v.d.Song.Patch[i].Units[0].ID, true
}

func (v *Instruments) Selected() int {
	return max(min(v.d.InstrIndex, v.Count()-1), 0)
}

func (v *Instruments) Selected2() int {
	return max(min(v.d.InstrIndex2, v.Count()-1), 0)
}

func (v *Instruments) SetSelected(value int) {
	v.d.InstrIndex = max(min(value, v.Count()-1), 0)
	v.d.UnitIndex = 0
	v.d.UnitIndex2 = 0
	v.d.UnitSearching = false
	v.d.UnitSearchString = ""
}

func (v *Instruments) SetSelected2(value int) {
	v.d.InstrIndex2 = max(min(value, v.Count()-1), 0)
}

func (v *Instruments) move(r Range, delta int) (ok bool) {
	voiceDelta := 0
	if delta < 0 {
		voiceDelta = -VoiceRange(v.d.Song.Patch, Range{r.Start + delta, r.Start}).Len()
	} else if delta > 0 {
		voiceDelta = VoiceRange(v.d.Song.Patch, Range{r.End, r.End + delta}).Len()
	}
	if voiceDelta == 0 {
		return false
	}
	ranges := MakeMoveRanges(VoiceRange(v.d.Song.Patch, r), voiceDelta)
	return (*Model)(v).sliceInstrumentsTracks(true, v.linkInstrTrack, ranges[:]...)
}

func (v *Instruments) delete(r Range) (ok bool) {
	ranges := Complement(VoiceRange(v.d.Song.Patch, r))
	return (*Model)(v).sliceInstrumentsTracks(true, v.linkInstrTrack, ranges[:]...)
}

func (v *Instruments) change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("Instruments."+n, SongChange, severity)
}

func (v *Instruments) cancel() {
	v.changeCancel = true
}

func (v *Instruments) Count() int {
	return len(v.d.Song.Patch)
}

func (v *Instruments) marshal(r Range) ([]byte, error) {
	return (*Model)(v).marshalVoices(VoiceRange(v.d.Song.Patch, r))
}

func (m *Instruments) unmarshal(data []byte) (r Range, err error) {
	voiceIndex := m.d.Song.Patch.FirstVoiceForInstrument(m.d.InstrIndex)
	r, _, ok := (*Model)(m).unmarshalVoices(voiceIndex, data, true, m.linkInstrTrack)
	if !ok {
		return Range{}, fmt.Errorf("unmarshal: unmarshalVoices failed")
	}
	return r, nil
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

func (v *Units) move(r Range, delta int) (ok bool) {
	m := (*Model)(v)
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	units := m.d.Song.Patch[m.d.InstrIndex].Units
	for i, j := range r.Swaps(delta) {
		units[i], units[j] = units[j], units[i]
	}
	return true
}

func (v *Units) delete(r Range) (ok bool) {
	m := (*Model)(v)
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	u := m.d.Song.Patch[m.d.InstrIndex].Units
	m.d.Song.Patch[m.d.InstrIndex].Units = append(u[:r.Start], u[r.End:]...)
	return true
}

func (v *Units) change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("UnitListView."+n, PatchChange, severity)
}

func (v *Units) cancel() {
	(*Model)(v).changeCancel = true
}

func (v *Units) marshal(r Range) ([]byte, error) {
	m := (*Model)(v)
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return nil, errors.New("UnitListView.marshal: no instruments")
	}
	units := m.d.Song.Patch[m.d.InstrIndex].Units[r.Start:r.End]
	ret, err := yaml.Marshal(struct{ Units []sointu.Unit }{units})
	if err != nil {
		return nil, fmt.Errorf("UnitListView.marshal: %v", err)
	}
	return ret, nil
}

func (v *Units) unmarshal(data []byte) (r Range, err error) {
	m := (*Model)(v)
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return Range{}, errors.New("UnitListView.unmarshal: no instruments")
	}
	var pastedUnits struct{ Units []sointu.Unit }
	if err := yaml.Unmarshal(data, &pastedUnits); err != nil {
		return Range{}, fmt.Errorf("UnitListView.unmarshal: %v", err)
	}
	if len(pastedUnits.Units) == 0 {
		return Range{}, errors.New("UnitListView.unmarshal: no units")
	}
	m.assignUnitIDs(pastedUnits.Units)
	sel := v.Selected()
	var ok bool
	m.d.Song.Patch[m.d.InstrIndex].Units, ok = Insert(m.d.Song.Patch[m.d.InstrIndex].Units, sel, pastedUnits.Units...)
	if !ok {
		return Range{}, errors.New("UnitListView.unmarshal: insert failed")
	}
	return Range{sel, sel + len(pastedUnits.Units)}, nil
}

// Tracks methods

func (v *Tracks) List() List {
	return List{v}
}

func (v *Tracks) Selected() int {
	return max(min(v.d.Cursor.Track, v.Count()-1), 0)
}

func (v *Tracks) Selected2() int {
	return max(min(v.d.Cursor2.Track, v.Count()-1), 0)
}

func (v *Tracks) SetSelected(value int) {
	(*Model)(v).ChangeTrack(value)
}

func (v *Tracks) SetSelected2(value int) {
	v.d.Cursor2.Track = max(min(value, v.Count()-1), 0)
}

func (v *Tracks) move(r Range, delta int) (ok bool) {
	voiceDelta := 0
	if delta < 0 {
		voiceDelta = -VoiceRange(v.d.Song.Score.Tracks, Range{r.Start + delta, r.Start}).Len()
	} else if delta > 0 {
		voiceDelta = VoiceRange(v.d.Song.Score.Tracks, Range{r.End, r.End + delta}).Len()
	}
	if voiceDelta == 0 {
		return false
	}
	ranges := MakeMoveRanges(VoiceRange(v.d.Song.Score.Tracks, r), voiceDelta)
	return (*Model)(v).sliceInstrumentsTracks(v.linkInstrTrack, true, ranges[:]...)
}

func (v *Tracks) delete(r Range) (ok bool) {
	ranges := Complement(VoiceRange(v.d.Song.Score.Tracks, r))
	return (*Model)(v).sliceInstrumentsTracks(v.linkInstrTrack, true, ranges[:]...)
}

func (v *Tracks) change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("TrackList."+n, SongChange, severity)
}

func (v *Tracks) cancel() {
	v.changeCancel = true
}

func (v *Tracks) Count() int {
	return len((*Model)(v).d.Song.Score.Tracks)
}

func (v *Tracks) marshal(r Range) ([]byte, error) {
	return (*Model)(v).marshalVoices(VoiceRange(v.d.Song.Score.Tracks, r))
}

func (m *Tracks) unmarshal(data []byte) (r Range, err error) {
	voiceIndex := m.d.Song.Score.FirstVoiceForTrack(m.d.Cursor.Track)
	_, r, ok := (*Model)(m).unmarshalVoices(voiceIndex, data, m.linkInstrTrack, true)
	if !ok {
		return Range{}, fmt.Errorf("unmarshal: unmarshalVoices failed")
	}
	return r, nil
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

func (v *OrderRows) move(r Range, delta int) (ok bool) {
	swaps := r.Swaps(delta)
	for i, t := range v.d.Song.Score.Tracks {
		for a, b := range swaps {
			ea, eb := t.Order.Get(a), t.Order.Get(b)
			v.d.Song.Score.Tracks[i].Order.Set(a, eb)
			v.d.Song.Score.Tracks[i].Order.Set(b, ea)
		}
	}
	return true
}

func (v *OrderRows) delete(r Range) (ok bool) {
	for i, t := range v.d.Song.Score.Tracks {
		r2 := r.Intersect(Range{0, len(t.Order)})
		v.d.Song.Score.Tracks[i].Order = append(t.Order[:r2.Start], t.Order[r2.End:]...)
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

func (v *OrderRows) marshal(r Range) ([]byte, error) {
	var table marshalOrderRows
	for i := range v.d.Song.Score.Tracks {
		table.Columns = append(table.Columns, make([]int, r.Len()))
		for j := 0; j < r.Len(); j++ {
			table.Columns[i][j] = v.d.Song.Score.Tracks[i].Order.Get(r.Start + j)
		}
	}
	return yaml.Marshal(table)
}

func (v *OrderRows) unmarshal(data []byte) (r Range, err error) {
	var table marshalOrderRows
	err = yaml.Unmarshal(data, &table)
	if err != nil {
		return
	}
	if len(table.Columns) == 0 {
		err = errors.New("OrderRowList.unmarshal: no rows")
		return
	}
	r.Start = v.d.Cursor.OrderRow
	r.End = v.d.Cursor.OrderRow + len(table.Columns[0])
	for i := range v.d.Song.Score.Tracks {
		if i >= len(table.Columns) {
			break
		}
		order := &v.d.Song.Score.Tracks[i].Order
		for j := 0; j < r.Start-len(*order); j++ {
			*order = append(*order, -1)
		}
		if len(*order) > r.Start {
			table.Columns[i] = append(table.Columns[i], (*order)[r.Start:]...)
			*order = (*order)[:r.Start]
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

func (v *NoteRows) move(r Range, delta int) (ok bool) {
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

func (v *NoteRows) delete(r Range) (ok bool) {
	for _, track := range v.d.Song.Score.Tracks {
		for i := r.Start; i < r.End; i++ {
			pos := v.d.Song.Score.SongPos(i)
			track.SetNote(pos, 1, v.uniquePatterns)
		}
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

func (v *NoteRows) marshal(r Range) ([]byte, error) {
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

func (v *NoteRows) unmarshal(data []byte) (r Range, err error) {
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

func (r Range) Len() int { return r.End - r.Start }

func (r Range) Swaps(delta int) iter.Seq2[int, int] {
	if delta > 0 {
		return func(yield func(int, int) bool) {
			for i := r.End - 1; i >= r.Start; i-- {
				if !yield(i, i+delta) {
					return
				}
			}
		}
	}
	return func(yield func(int, int) bool) {
		for i := r.Start; i < r.End; i++ {
			if !yield(i, i+delta) {
				return
			}
		}
	}
}

func (r Range) Intersect(s Range) (ret Range) {
	ret.Start = max(r.Start, s.Start)
	ret.End = max(min(r.End, s.End), ret.Start)
	if ret.Len() == 0 {
		return Range{}
	}
	return
}

func MakeMoveRanges(a Range, delta int) [4]Range {
	if delta < 0 {
		return [4]Range{
			{math.MinInt, a.Start + delta},
			{a.Start, a.End},
			{a.Start + delta, a.Start},
			{a.End, math.MaxInt},
		}
	}
	return [4]Range{
		{math.MinInt, a.Start},
		{a.End, a.End + delta},
		{a.Start, a.End},
		{a.End + delta, math.MaxInt},
	}
}

// MakeSetLength takes a range and a length, and returns a slice of ranges that
// can be used with VoiceSlice to expand or shrink the range to the given
// length, by either duplicating or removing elements. The function tries to
// duplicate elements so all elements are equally spaced, and tries to remove
// elements from the middle of the range.
func MakeSetLength(a Range, length int) []Range {
	if length <= 0 || a.Len() <= 0 {
		return []Range{{a.Start, a.Start}}
	}
	ret := make([]Range, a.Len(), max(a.Len(), length)+2)
	for i := 0; i < a.Len(); i++ {
		ret[i] = Range{a.Start + i, a.Start + i + 1}
	}
	for x := len(ret); x < length; x++ {
		e := (x << 1) ^ (1 << bits.Len((uint)(x)))
		ret = append(ret[0:e+1], ret[e:]...)
	}
	for x := len(ret); x > length; x-- {
		e := (((x << 1) ^ (1 << bits.Len((uint)(x)))) + x - 1) % x
		ret = append(ret[0:e], ret[e+1:]...)
	}
	ret = append([]Range{{math.MinInt, a.Start}}, ret...)
	ret = append(ret, Range{a.End, math.MaxInt})
	return ret
}

func Complement(a Range) [2]Range {
	return [2]Range{
		{math.MinInt, a.Start},
		{a.End, math.MaxInt},
	}
}

// Insert inserts elements into a slice at the given index. If the index is out
// of bounds, the function returns false.
func Insert[T any, S ~[]T](slice S, index int, inserted ...T) (ret S, ok bool) {
	if index < 0 || index > len(slice) {
		return nil, false
	}
	ret = make(S, 0, len(slice)+len(inserted))
	ret = append(ret, slice[:index]...)
	ret = append(ret, inserted...)
	ret = append(ret, slice[index:]...)
	return ret, true
}

// VoiceSlice works similar to the Slice function, but takes a slice of
// NumVoicer:s and treats it as a "virtual slice", with element repeated by the
// number of voices it has. NumVoicer interface is implemented at least by
// sointu.Tracks and sointu.Instruments. For example, if parameter "slice" has
// three elements, returning GetNumVoices 2, 1, and 3, the VoiceSlice thinks of
// this as a virtual slice of 6 elements [0,0,1,2,2,2]. Then, the "ranges"
// parameter are slicing ranges to this virtual slice. Continuing with the
// example, if "ranges" was [2,5), the virtual slice would be [1,2,2], and the
// function would return a slice with two elements: first with NumVoices 1 and
// second with NumVoices 2. If multiple ranges are given, multiple virtual
// slices are concatenated. However, when doing so, splitting an element is not
// allowed. In the previous example, if the ranges were [1,3) and [0,1), the
// resulting concatenated virtual slice would be [0,1,0], and here the 0 element
// would be split. This is to avoid accidentally making shallow copies of
// reference types.
func VoiceSlice[T any, S ~[]T, P sointu.NumVoicerPointer[T]](slice S, ranges ...Range) (ret S, ok bool) {
	ret = make(S, 0, len(slice))
	last := -1
	used := make([]bool, len(slice))
outer:
	for _, r := range ranges {
		left := 0
		for i, elem := range slice {
			right := left + (P)(&slice[i]).GetNumVoices()
			if left >= r.End {
				continue outer
			}
			if right <= r.Start {
				left = right
				continue
			}
			overlap := min(right, r.End) - max(left, r.Start)
			if last == i {
				(P)(&ret[len(ret)-1]).SetNumVoices(
					(P)(&ret[len(ret)-1]).GetNumVoices() + overlap)
			} else {
				if last == math.MaxInt || used[i] {
					return nil, false
				}
				ret = append(ret, elem)
				(P)(&ret[len(ret)-1]).SetNumVoices(overlap)
				used[i] = true
			}
			last = i
			left = right
		}
		if left >= r.End {
			continue outer
		}
		last = math.MaxInt // the list is closed, adding more elements causes it to fail
	}
	return ret, true
}

// VoiceInsert tries adding the elements "added" to the slice "orig" at the
// voice index "index". Notice that index is the index into a virtual slice
// where each element is repeated by the number of voices it has. If the index
// is between elements, the new elements are added in between the old elements.
// If the addition would cause splitting of an element, we rather increase the
// number of voices the element has, but do not split it.
func VoiceInsert[T any, S ~[]T, P sointu.NumVoicerPointer[T]](orig S, index, length int, added ...T) (ret S, retRange Range, ok bool) {
	ret = make(S, 0, len(orig)+length)
	left := 0
	for i, elem := range orig {
		right := left + (P)(&orig[i]).GetNumVoices()
		if left == index { // we are between elements and it's safe to add there
			if sointu.TotalVoices[T, S, P](added) < length {
				return nil, Range{}, false // we are missing some elements
			}
			retRange = Range{len(ret), len(ret) + len(added)}
			ret = append(ret, added...)
		} else if left < index && index < right { // we are inside an element and would split it; just increase its voices instead of splitting
			(P)(&elem).SetNumVoices((P)(&orig[i]).GetNumVoices() + sointu.TotalVoices[T, S, P](added))
			retRange = Range{len(ret), len(ret)}
		}
		ret = append(ret, elem)
		left = right
	}
	if left == index { // we are at the end and it's safe to add there, even if we are missing some elements
		retRange = Range{len(ret), len(ret) + len(added)}
		ret = append(ret, added...)
	}
	return ret, retRange, true
}

func VoiceRange[T any, S ~[]T, P sointu.NumVoicerPointer[T]](slice S, indexRange Range) (voiceRange Range) {
	indexRange.Start = max(0, indexRange.Start)
	indexRange.End = min(len(slice), indexRange.End)
	for _, e := range slice[:indexRange.Start] {
		voiceRange.Start += (P)(&e).GetNumVoices()
	}
	voiceRange.End = voiceRange.Start
	for i := indexRange.Start; i < indexRange.End; i++ {
		voiceRange.End += (P)(&slice[i]).GetNumVoices()
	}
	return
}

// helpers

func (m *Model) sliceInstrumentsTracks(instruments, tracks bool, ranges ...Range) (ok bool) {
	defer m.change("sliceInstrumentsTracks", PatchChange, MajorChange)()
	if instruments {
		m.d.Song.Patch, ok = VoiceSlice(m.d.Song.Patch, ranges...)
		if !ok {
			goto fail
		}
	}
	if tracks {
		m.d.Song.Score.Tracks, ok = VoiceSlice(m.d.Song.Score.Tracks, ranges...)
		if !ok {
			goto fail
		}
	}
	return true
fail:
	(*Model)(m).Alerts().AddNamed("slicesInstrumentsTracks", "Modify prevented by Instrument-Track linking", Warning)
	m.changeCancel = true
	return false
}

func (m *Model) marshalVoices(r Range) (data []byte, err error) {
	patch, ok := VoiceSlice(m.d.Song.Patch, r)
	if !ok {
		return nil, fmt.Errorf("marshalVoiceRange: slicing patch failed")
	}
	tracks, ok := VoiceSlice(m.d.Song.Score.Tracks, r)
	if !ok {
		return nil, fmt.Errorf("marshalVoiceRange: slicing tracks failed")
	}
	return yaml.Marshal(struct {
		Patch  sointu.Patch
		Tracks []sointu.Track
	}{patch, tracks})
}

func (m *Model) unmarshalVoices(voiceIndex int, data []byte, instruments, tracks bool) (instrRange, trackRange Range, ok bool) {
	var d struct {
		Patch  sointu.Patch
		Tracks []sointu.Track
	}
	if err := yaml.Unmarshal(data, &d); err != nil {
		return Range{}, Range{}, false
	}
	return m.addVoices(voiceIndex, d.Patch, d.Tracks, instruments, tracks)
}

func (m *Model) addVoices(voiceIndex int, p sointu.Patch, t []sointu.Track, instruments, tracks bool) (instrRange Range, trackRange Range, ok bool) {
	defer m.change("addVoices", PatchChange, MajorChange)()
	addedLength := max(p.NumVoices(), sointu.TotalVoices(t))
	if instruments {
		m.assignUnitIDsForPatch(p)
		m.d.Song.Patch, instrRange, ok = VoiceInsert(m.d.Song.Patch, voiceIndex, addedLength, p...)
		if !ok {
			goto fail
		}
	}
	if tracks {
		m.d.Song.Score.Tracks, trackRange, ok = VoiceInsert(m.d.Song.Score.Tracks, voiceIndex, addedLength, t...)
		if !ok {
			goto fail
		}
	}
	return instrRange, trackRange, true
fail:
	(*Model)(m).Alerts().AddNamed("addVoices", "Adding voices prevented by Instrument-Track linking", Warning)
	m.changeCancel = true
	return Range{}, Range{}, false
}

func (m *Model) remainingVoices(instruments, tracks bool) (ret int) {
	ret = math.MaxInt
	if instruments {
		ret = min(ret, vm.MAX_VOICES-m.d.Song.Patch.NumVoices())
	}
	if tracks {
		ret = min(ret, vm.MAX_VOICES-m.d.Song.Score.NumVoices())
	}
	return
}
