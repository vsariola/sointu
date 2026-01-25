package tracker

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vsariola/sointu"
	"gopkg.in/yaml.v3"
)

// Unit returns the Unit view of the model, containing methods to manipulate the
// units.
func (m *Model) Unit() *UnitModel { return (*UnitModel)(m) }

type UnitModel Model

// Add returns an Action to add a new unit. If the before parameter is true,
// then the new unit is added before the currently selected unit; otherwise,
// after.
func (m *UnitModel) Add(before bool) Action {
	return MakeAction(addUnit{Before: before, Model: (*Model)(m)})
}

type addUnit struct {
	Before bool
	*Model
}

func (a addUnit) Do() {
	m := (*Model)(a.Model)
	defer m.change("AddUnitAction", PatchChange, MajorChange)()
	if len(m.d.Song.Patch) == 0 { // no instruments, add one
		instr := sointu.Instrument{NumVoices: 1}
		instr.Units = make([]sointu.Unit, 0, 1)
		m.d.Song.Patch = append(m.d.Song.Patch, instr)
		m.d.UnitIndex = 0
	} else {
		if !a.Before {
			m.d.UnitIndex++
		}
	}
	m.d.InstrIndex = max(min(m.d.InstrIndex, len(m.d.Song.Patch)-1), 0)
	instr := m.d.Song.Patch[m.d.InstrIndex]
	newUnits := make([]sointu.Unit, len(instr.Units)+1)
	m.d.UnitIndex = clamp(m.d.UnitIndex, 0, len(newUnits)-1)
	m.d.UnitIndex2 = m.d.UnitIndex
	copy(newUnits, instr.Units[:m.d.UnitIndex])
	copy(newUnits[m.d.UnitIndex+1:], instr.Units[m.d.UnitIndex:])
	m.assignUnitIDs(newUnits[m.d.UnitIndex : m.d.UnitIndex+1])
	m.d.Song.Patch[m.d.InstrIndex].Units = newUnits
	m.d.ParamIndex = 0
}

// Delete returns an Action to delete the currently selected unit(s).
func (m *UnitModel) Delete() Action { return MakeAction((*deleteUnit)(m)) }

type deleteUnit UnitModel

func (m *deleteUnit) Enabled() bool {
	i := (*Model)(m).d.InstrIndex
	return i >= 0 && i < len((*Model)(m).d.Song.Patch) && len((*Model)(m).d.Song.Patch[i].Units) > 1
}
func (m *deleteUnit) Do() {
	defer (*Model)(m).change("DeleteUnitAction", PatchChange, MajorChange)()
	(*UnitModel)(m).List().DeleteElements(true)
}

// Clear returns an Action to clear the currently selected unit(s) i.e. they are
// set as empty units, but are kept in the unit list.
func (m *UnitModel) Clear() Action { return MakeAction((*clearUnit)(m)) }

type clearUnit UnitModel

func (m *clearUnit) Enabled() bool {
	i := (*Model)(m).d.InstrIndex
	return i >= 0 && i < len(m.d.Song.Patch) && len(m.d.Song.Patch[i].Units) > 0
}
func (m *clearUnit) Do() {
	defer (*Model)(m).change("DeleteUnitAction", PatchChange, MajorChange)()
	l := ((*UnitModel)(m)).List()
	r := l.listRange()
	for i := r.Start; i < r.End; i++ {
		m.d.Song.Patch[m.d.InstrIndex].Units[i] = sointu.Unit{}
		m.d.Song.Patch[m.d.InstrIndex].Units[i].ID = (*Model)(m).maxID() + 1
	}
}

// Searching returns a Bool telling whether the user is currently searching for
// a unit (should the search resultsbe displayed).
func (m *UnitModel) Searching() Bool { return MakeBool((*unitSearching)(m)) }

type unitSearching UnitModel

func (m *unitSearching) Value() bool { return m.d.UnitSearching }
func (m *unitSearching) SetValue(val bool) {
	m.d.UnitSearching = val
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		m.d.UnitSearchString = ""
		return
	}
	if m.d.UnitIndex < 0 || m.d.UnitIndex >= len(m.d.Song.Patch[m.d.InstrIndex].Units) {
		m.d.UnitSearchString = ""
		return
	}
	m.d.UnitSearchString = m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex].Type
	(*UnitModel)(m).updateDerivedUnitSearch()
}

// SearchTerm returns a String which is the search term user has typed when
// searching for units.
func (m *UnitModel) SearchTerm() String { return MakeString((*unitSearchTerm)(m)) }

type unitSearchTerm UnitModel

func (v *unitSearchTerm) Value() string {
	// return current unit type string if not searching
	if !v.d.UnitSearching {
		if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
			return ""
		}
		if v.d.UnitIndex < 0 || v.d.UnitIndex >= len(v.d.Song.Patch[v.d.InstrIndex].Units) {
			return ""
		}
		return v.d.Song.Patch[v.d.InstrIndex].Units[v.d.UnitIndex].Type
	} else {
		return v.d.UnitSearchString
	}
}
func (v *unitSearchTerm) SetValue(value string) bool {
	v.d.UnitSearchString = value
	v.d.UnitSearching = true
	(*UnitModel)(v).updateDerivedUnitSearch()
	return true
}

func (v *UnitModel) updateDerivedUnitSearch() {
	// update search results based on current search string
	v.derived.searchResults = v.derived.searchResults[:0]
	for _, name := range sointu.UnitNames {
		if strings.HasPrefix(name, v.SearchTerm().Value()) {
			v.derived.searchResults = append(v.derived.searchResults, name)
		}
	}
}

// SearchResult returns the unit search result at a given index.
func (l *UnitModel) SearchResult(index int) (name string, ok bool) {
	if index < 0 || index >= len(l.derived.searchResults) {
		return "", false
	}
	return l.derived.searchResults[index], true
}

// SearchResults returns a List of all the unit names matching the given search
// term.
func (m *UnitModel) SearchResults() List { return List{(*unitSearchResults)(m)} }

type unitSearchResults UnitModel

func (l *unitSearchResults) Selected() int          { return l.d.UnitSearchIndex }
func (l *unitSearchResults) Selected2() int         { return l.d.UnitSearchIndex }
func (l *unitSearchResults) SetSelected(value int)  { l.d.UnitSearchIndex = value }
func (l *unitSearchResults) SetSelected2(value int) {}
func (l *unitSearchResults) Count() (count int)     { return len(l.derived.searchResults) }

// Comment returns a String representing the comment string of the current unit.
func (m *UnitModel) Comment() String { return MakeString((*unitComment)(m)) }

type unitComment UnitModel

func (v *unitComment) Value() string {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) ||
		v.d.UnitIndex < 0 || v.d.UnitIndex >= len(v.d.Song.Patch[v.d.InstrIndex].Units) {
		return ""
	}
	return v.d.Song.Patch[v.d.InstrIndex].Units[v.d.UnitIndex].Comment
}
func (v *unitComment) SetValue(value string) bool {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) ||
		v.d.UnitIndex < 0 || v.d.UnitIndex >= len(v.d.Song.Patch[v.d.InstrIndex].Units) {
		return false
	}
	defer (*Model)(v).change("UnitComment", PatchChange, MinorChange)()
	v.d.Song.Patch[v.d.InstrIndex].Units[v.d.UnitIndex].Comment = value
	return true
}

// Disabled returns a Bool controlling whether the currently selected unit(s)
// are disabled.
func (m *UnitModel) Disabled() Bool { return MakeBool((*unitDisabled)(m)) }

type unitDisabled UnitModel

func (m *unitDisabled) Value() bool {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	if m.d.UnitIndex < 0 || m.d.UnitIndex >= len(m.d.Song.Patch[m.d.InstrIndex].Units) {
		return false
	}
	return m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex].Disabled
}
func (m *unitDisabled) SetValue(val bool) {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return
	}
	l := ((*UnitModel)(m)).List()
	r := l.listRange()
	defer (*Model)(m).change("UnitDisabledSet", PatchChange, MajorChange)()
	for i := r.Start; i < r.End; i++ {
		m.d.Song.Patch[m.d.InstrIndex].Units[i].Disabled = val
	}
}
func (m *unitDisabled) Enabled() bool {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	if len(m.d.Song.Patch[m.d.InstrIndex].Units) == 0 {
		return false
	}
	return true
}

// Item returns information about the unit at the given index.
func (v *UnitModel) Item(index int) UnitListItem {
	i := v.d.InstrIndex
	if i < 0 || i >= len(v.d.Song.Patch) || index < 0 || index >= (*unitList)(v).Count() {
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

type UnitListItem struct {
	Type, Comment string
	Disabled      bool
	Signals       Rail
}

// Type returns the type of the currently selected unit.
func (m *UnitModel) Type() string {
	if m.d.InstrIndex < 0 ||
		m.d.InstrIndex >= len(m.d.Song.Patch) ||
		m.d.UnitIndex < 0 ||
		m.d.UnitIndex >= len(m.d.Song.Patch[m.d.InstrIndex].Units) {
		return ""
	}
	return m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex].Type
}

// SetType sets the type of the currently selected unit.
func (m *UnitModel) SetType(t string) {
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
	defer (*unitList)(m).Change("SetSelectedType", MajorChange)()
	m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex] = unit
	m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex].ID = oldUnit.ID // keep the ID of the replaced unit
}

// List returns a List of all the units of the selected instrument, implementing
// ListData & MutableListData interfaces
func (m *UnitModel) List() List { return List{(*unitList)(m)} }

type unitList UnitModel

func (v *unitList) Selected() int          { return v.d.UnitIndex }
func (v *unitList) Selected2() int         { return v.d.UnitIndex2 }
func (v *unitList) SetSelected2(value int) { v.d.UnitIndex2 = value }
func (m *unitList) SetSelected(value int) {
	m.d.UnitIndex = value
	m.d.ParamIndex = 0
	m.d.UnitSearching = false
	m.d.UnitSearchString = ""
}
func (v *unitList) Count() int {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return 0
	}
	return len(v.d.Song.Patch[v.d.InstrIndex].Units)
}

func (v *unitList) Move(r Range, delta int) (ok bool) {
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

func (v *unitList) Delete(r Range) (ok bool) {
	m := (*Model)(v)
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	u := m.d.Song.Patch[m.d.InstrIndex].Units
	m.d.Song.Patch[m.d.InstrIndex].Units = append(u[:r.Start], u[r.End:]...)
	return true
}

func (v *unitList) Change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("UnitListView."+n, PatchChange, severity)
}

func (v *unitList) Cancel() {
	(*Model)(v).changeCancel = true
}

func (v *unitList) Marshal(r Range) ([]byte, error) {
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

func (v *unitList) Unmarshal(data []byte) (r Range, err error) {
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

func (s *UnitModel) RailError() RailError { return s.derived.railError }

func (s *UnitModel) RailWidth() int {
	i := s.d.InstrIndex
	if i < 0 || i >= len(s.derived.patch) {
		return 0
	}
	return s.derived.patch[i].railWidth
}

func (e *RailError) Error() string { return e.Err.Error() }

func (s *Rail) StackAfter() int { return s.PassThrough + s.StackUse.NumOutputs }
