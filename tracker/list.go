package tracker

import (
	"errors"
	"fmt"
	"iter"
	"math"
	"math/bits"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
	"gopkg.in/yaml.v3"
)

type (
	List struct {
		data ListData
	}

	ListData interface {
		Selected() int
		Selected2() int
		SetSelected(int)
		SetSelected2(int)
		Count() int
	}

	MutableListData interface {
		Change(kind string, severity ChangeSeverity) func()
		Cancel()
		Move(r Range, delta int) (ok bool)
		Delete(r Range) (ok bool)
		Marshal(r Range) ([]byte, error)
		Unmarshal([]byte) (r Range, err error)
	}

	// Range is used to represent a range [Start,End) of integers
	Range struct {
		Start, End int
	}
)

func MakeList(data ListData) List { return List{data} }

func (l List) Selected() int          { return max(min(l.data.Selected(), l.data.Count()-1), 0) }
func (l List) Selected2() int         { return max(min(l.data.Selected2(), l.data.Count()-1), 0) }
func (l List) SetSelected(value int)  { l.data.SetSelected(max(min(value, l.data.Count()-1), 0)) }
func (l List) SetSelected2(value int) { l.data.SetSelected2(max(min(value, l.data.Count()-1), 0)) }
func (l List) Count() int             { return l.data.Count() }

// MoveElements moves the selected elements in a list by delta. The list must
// implement the MutableListData interface.
func (v List) MoveElements(delta int) bool {
	s, ok := v.data.(MutableListData)
	if !ok {
		return false
	}
	r := v.listRange()
	if delta == 0 || r.Start+delta < 0 || r.End+delta > v.Count() {
		return false
	}
	defer s.Change("MoveElements", MajorChange)()
	if !s.Move(r, delta) {
		s.Cancel()
		return false
	}
	v.SetSelected(v.Selected() + delta)
	v.SetSelected2(v.Selected2() + delta)
	return true
}

// DeleteElements deletes the selected elements in a list. The list must
// implement the MutableListData interface.
func (v List) DeleteElements(backwards bool) bool {
	d, ok := v.data.(MutableListData)
	if !ok {
		return false
	}
	r := v.listRange()
	if r.Len() == 0 {
		return false
	}
	defer d.Change("DeleteElements", MajorChange)()
	if !d.Delete(r) {
		d.Cancel()
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
	m, ok := v.data.(MutableListData)
	if !ok {
		return nil, false
	}
	r := v.listRange()
	if r.Len() == 0 {
		return nil, false
	}
	ret, err := m.Marshal(r)
	if err != nil {
		return nil, false
	}
	return ret, true
}

// PasteElements pastes the data into the list. The data is unmarshaled from the
// byte slice. The list must implement the MutableListData interface. Returns
// true if successful.
func (v List) PasteElements(data []byte) (ok bool) {
	m, ok := v.data.(MutableListData)
	if !ok {
		return false
	}
	defer m.Change("PasteElements", MajorChange)()
	r, err := m.Unmarshal(data)
	if err != nil {
		m.Cancel()
		return false
	}
	v.SetSelected(r.Start)
	v.SetSelected2(r.End - 1)
	return true
}

func (v List) Mutable() bool {
	_, ok := v.data.(MutableListData)
	return ok
}

func (v *List) listRange() (r Range) {
	r.Start = max(min(v.Selected(), v.Selected2()), 0)
	r.End = min(max(v.Selected(), v.Selected2())+1, v.Count())
	return
}

// instruments is a list of instruments, implementing ListData & MutableListData interfaces
type instruments Model

func (m *Model) Instruments() List { return List{(*instruments)(m)} }

func (v *Model) Instrument(i int) (name string, maxLevel float32, mute bool, ok bool) {
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
		for _, level := range v.playerStatus.VoiceLevels[start:end] {
			if maxLevel < level {
				maxLevel = level
			}
		}
	}
	ok = true
	return
}

func (v *instruments) Count() int             { return len(v.d.Song.Patch) }
func (v *instruments) Selected() int          { return v.d.InstrIndex }
func (v *instruments) Selected2() int         { return v.d.InstrIndex2 }
func (v *instruments) SetSelected2(value int) { v.d.InstrIndex2 = value }
func (v *instruments) SetSelected(value int) {
	v.d.InstrIndex = value
	v.d.UnitIndex = 0
	v.d.UnitIndex2 = 0
	v.d.UnitSearching = false
	v.d.UnitSearchString = ""
}

func (v *instruments) Move(r Range, delta int) (ok bool) {
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

func (v *instruments) Delete(r Range) (ok bool) {
	ranges := Complement(VoiceRange(v.d.Song.Patch, r))
	return (*Model)(v).sliceInstrumentsTracks(true, v.linkInstrTrack, ranges[:]...)
}

func (v *instruments) Change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("Instruments."+n, SongChange, severity)
}

func (v *instruments) Cancel() {
	v.changeCancel = true
}

func (v *instruments) Marshal(r Range) ([]byte, error) {
	return (*Model)(v).marshalVoices(VoiceRange(v.d.Song.Patch, r))
}

func (m *instruments) Unmarshal(data []byte) (r Range, err error) {
	voiceIndex := m.d.Song.Patch.FirstVoiceForInstrument(m.d.InstrIndex)
	r, _, ok := (*Model)(m).unmarshalVoices(voiceIndex, data, true, m.linkInstrTrack)
	if !ok {
		return Range{}, fmt.Errorf("unmarshal: unmarshalVoices failed")
	}
	return r, nil
}

// units is a list of all the units in the selected instrument, implementing ListData & MutableListData interfaces
type (
	units        Model
	UnitListItem struct {
		Type, Comment string
		Disabled      bool
		Signals       Rail
	}
)

func (m *Model) Units() List { return List{(*units)(m)} }

func (v *Model) Unit(index int) UnitListItem {
	i := v.d.InstrIndex
	if i < 0 || i >= len(v.d.Song.Patch) || index < 0 || index >= (*units)(v).Count() {
		return UnitListItem{}
	}
	unit := v.d.Song.Patch[v.d.InstrIndex].Units[index]
	signals := Rail{}
	if i >= 0 && i < len(v.derived.patch) && index >= 0 && index < len(v.derived.patch[i].rails) {
		signals = v.derived.patch[i].rails[index]
	}
	return UnitListItem{
		Type:     unit.Type,
		Comment:  unit.Comment,
		Disabled: unit.Disabled,
		Signals:  signals,
	}
}

func (m *Model) SelectedUnitType() string {
	if m.d.InstrIndex < 0 ||
		m.d.InstrIndex >= len(m.d.Song.Patch) ||
		m.d.UnitIndex < 0 ||
		m.d.UnitIndex >= len(m.d.Song.Patch[m.d.InstrIndex].Units) {
		return ""
	}
	return m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex].Type
}

func (m *Model) SetSelectedUnitType(t string) {
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
	defer (*units)(m).Change("SetSelectedType", MajorChange)()
	m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex] = unit
	m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex].ID = oldUnit.ID // keep the ID of the replaced unit
}

func (v *units) Selected() int          { return v.d.UnitIndex }
func (v *units) Selected2() int         { return v.d.UnitIndex2 }
func (v *units) SetSelected2(value int) { v.d.UnitIndex2 = value }
func (m *units) SetSelected(value int) {
	m.d.UnitIndex = value
	m.d.ParamIndex = 0
	m.d.UnitSearching = false
	m.d.UnitSearchString = ""
}
func (v *units) Count() int {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return 0
	}
	return len(v.d.Song.Patch[v.d.InstrIndex].Units)
}

func (v *units) Move(r Range, delta int) (ok bool) {
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

func (v *units) Delete(r Range) (ok bool) {
	m := (*Model)(v)
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	u := m.d.Song.Patch[m.d.InstrIndex].Units
	m.d.Song.Patch[m.d.InstrIndex].Units = append(u[:r.Start], u[r.End:]...)
	return true
}

func (v *units) Change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("UnitListView."+n, PatchChange, severity)
}

func (v *units) Cancel() {
	(*Model)(v).changeCancel = true
}

func (v *units) Marshal(r Range) ([]byte, error) {
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

func (v *units) Unmarshal(data []byte) (r Range, err error) {
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

// tracks is a list of all the tracks, implementing ListData & MutableListData interfaces
type tracks Model

func (m *Model) Tracks() List { return List{(*tracks)(m)} }

func (v *tracks) Selected() int          { return v.d.Cursor.Track }
func (v *tracks) Selected2() int         { return v.d.Cursor2.Track }
func (v *tracks) SetSelected(value int)  { v.d.Cursor.Track = value }
func (v *tracks) SetSelected2(value int) { v.d.Cursor2.Track = value }
func (v *tracks) Count() int             { return len((*Model)(v).d.Song.Score.Tracks) }

func (v *tracks) Move(r Range, delta int) (ok bool) {
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

func (v *tracks) Delete(r Range) (ok bool) {
	ranges := Complement(VoiceRange(v.d.Song.Score.Tracks, r))
	return (*Model)(v).sliceInstrumentsTracks(v.linkInstrTrack, true, ranges[:]...)
}

func (v *tracks) Change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("TrackList."+n, SongChange, severity)
}

func (v *tracks) Cancel() {
	v.changeCancel = true
}

func (v *tracks) Marshal(r Range) ([]byte, error) {
	return (*Model)(v).marshalVoices(VoiceRange(v.d.Song.Score.Tracks, r))
}

func (m *tracks) Unmarshal(data []byte) (r Range, err error) {
	voiceIndex := m.d.Song.Score.FirstVoiceForTrack(m.d.Cursor.Track)
	_, r, ok := (*Model)(m).unmarshalVoices(voiceIndex, data, m.linkInstrTrack, true)
	if !ok {
		return Range{}, fmt.Errorf("unmarshal: unmarshalVoices failed")
	}
	return r, nil
}

// orderRows is a list of all the order rows, implementing ListData & MutableListData interfaces
type orderRows Model

func (m *Model) OrderRows() List { return List{(*orderRows)(m)} }

func (v *orderRows) Count() int             { return v.d.Song.Score.Length }
func (v *orderRows) Selected() int          { return v.d.Cursor.OrderRow }
func (v *orderRows) Selected2() int         { return v.d.Cursor2.OrderRow }
func (v *orderRows) SetSelected2(value int) { v.d.Cursor2.OrderRow = value }
func (v *orderRows) SetSelected(value int) {
	if value != v.d.Cursor.OrderRow {
		v.follow = false
	}
	v.d.Cursor.OrderRow = value
}

func (v *orderRows) Move(r Range, delta int) (ok bool) {
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

func (v *orderRows) Delete(r Range) (ok bool) {
	for i, t := range v.d.Song.Score.Tracks {
		r2 := r.Intersect(Range{0, len(t.Order)})
		v.d.Song.Score.Tracks[i].Order = append(t.Order[:r2.Start], t.Order[r2.End:]...)
	}
	return true
}

func (v *orderRows) Change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("OrderRowList."+n, ScoreChange, severity)
}

func (v *orderRows) Cancel() {
	v.changeCancel = true
}

type marshalOrderRows struct {
	Columns [][]int `yaml:",flow"`
}

func (v *orderRows) Marshal(r Range) ([]byte, error) {
	var table marshalOrderRows
	for i := range v.d.Song.Score.Tracks {
		table.Columns = append(table.Columns, make([]int, r.Len()))
		for j := 0; j < r.Len(); j++ {
			table.Columns[i][j] = v.d.Song.Score.Tracks[i].Order.Get(r.Start + j)
		}
	}
	return yaml.Marshal(table)
}

func (v *orderRows) Unmarshal(data []byte) (r Range, err error) {
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

// noteRows is a list of all the note rows, implementing ListData & MutableListData interfaces
type noteRows Model

func (m *Model) NoteRows() List { return List{(*noteRows)(m)} }

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

// searchResults is a unmutable list of all the search results, implementing ListData interface
type (
	searchResults       Model
	UnitSearchYieldFunc func(index int, item string) (ok bool)
)

func (m *Model) SearchResults() List { return List{(*searchResults)(m)} }
func (l *Model) SearchResult(i int) (name string, ok bool) {
	if i < 0 || i >= len(l.derived.searchResults) {
		return "", false
	}
	return l.derived.searchResults[i], true
}

func (l *searchResults) Selected() int          { return l.d.UnitSearchIndex }
func (l *searchResults) Selected2() int         { return l.d.UnitSearchIndex }
func (l *searchResults) SetSelected(value int)  { l.d.UnitSearchIndex = value }
func (l *searchResults) SetSelected2(value int) {}
func (l *searchResults) Count() (count int)     { return len(l.derived.searchResults) }

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
