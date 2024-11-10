package tracker

import (
	"errors"
	"fmt"

	"github.com/vsariola/sointu"
	"gopkg.in/yaml.v2"
)

type (
	UnitListItem struct {
		Type, Comment                      string
		Disabled                           bool
		StackNeed, StackBefore, StackAfter int
	}

	UnitYieldFunc       func(index int, item UnitListItem) (ok bool)
	UnitSearchYieldFunc func(index int, item string) (ok bool)

	Units Model // Units is a list of all the units in the selected instrument, implementing ListData & MutableListData interfaces

)

// Model methods

func (m *Model) Units() *Units { return (*Units)(m) }

// Units methods

func (ul *Units) List() List {
	return List{ul}
}

func (ul *Units) SelectedType() string {
	if ul.d.InstrIndex < 0 ||
		ul.d.InstrIndex >= len(ul.d.Song.Patch) ||
		ul.d.UnitIndex < 0 ||
		ul.d.UnitIndex >= len(ul.d.Song.Patch[ul.d.InstrIndex].Units) {
		return ""
	}
	return ul.d.Song.Patch[ul.d.InstrIndex].Units[ul.d.UnitIndex].Type
}

func (ul *Units) SetSelectedType(t string) {
	if ul.d.InstrIndex < 0 ||
		ul.d.InstrIndex >= len(ul.d.Song.Patch) {
		return
	}
	if ul.d.UnitIndex < 0 {
		ul.d.UnitIndex = 0
	}
	for len(ul.d.Song.Patch[ul.d.InstrIndex].Units) <= ul.d.UnitIndex {
		ul.d.Song.Patch[ul.d.InstrIndex].Units = append(ul.d.Song.Patch[ul.d.InstrIndex].Units, sointu.Unit{})
	}
	unit, ok := defaultUnits[t]
	if !ok { // if the type is invalid, we just set it to empty unit
		unit = sointu.Unit{Parameters: make(map[string]int)}
	} else {
		unit = unit.Copy()
	}
	oldUnit := ul.d.Song.Patch[ul.d.InstrIndex].Units[ul.d.UnitIndex]
	if oldUnit.Type == unit.Type {
		return
	}
	defer ul.change("SetSelectedType", MajorChange)()
	ul.d.Song.Patch[ul.d.InstrIndex].Units[ul.d.UnitIndex] = unit
	ul.d.Song.Patch[ul.d.InstrIndex].Units[ul.d.UnitIndex].ID = oldUnit.ID // keep the ID of the replaced unit
}

func (ul *Units) Iterate(yield UnitYieldFunc) {
	if ul.d.InstrIndex < 0 || ul.d.InstrIndex >= len(ul.d.Song.Patch) {
		return
	}
	stackBefore := 0
	for i, unit := range ul.d.Song.Patch[ul.d.InstrIndex].Units {
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

func (ul *Units) Selected() int {
	return max(min(ul.d.UnitIndex, ul.Count()-1), 0)
}

func (ul *Units) Selected2() int {
	return max(min(ul.d.UnitIndex2, ul.Count()-1), 0)
}

func (ul *Units) SetSelected(value int) {
	m := (*Model)(ul)
	m.d.UnitIndex = max(min(value, ul.Count()-1), 0)
	m.d.ParamIndex = 0
	m.d.UnitSearching = false
	m.d.UnitSearchString = ""
}

func (ul *Units) SetSelected2(value int) {
	(*Model)(ul).d.UnitIndex2 = max(min(value, ul.Count()-1), 0)
}

func (ul *Units) Count() int {
	m := (*Model)(ul)
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return 0
	}
	return len(m.d.Song.Patch[(*Model)(ul).d.InstrIndex].Units)
}

func (ul *Units) move(r Range, delta int) (ok bool) {
	m := (*Model)(ul)
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	units := m.d.Song.Patch[m.d.InstrIndex].Units
	for i, j := range r.Swaps(delta) {
		units[i], units[j] = units[j], units[i]
	}
	return true
}

func (ul *Units) delete(r Range) (ok bool) {
	m := (*Model)(ul)
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	u := m.d.Song.Patch[m.d.InstrIndex].Units
	m.d.Song.Patch[m.d.InstrIndex].Units = append(u[:r.Start], u[r.End:]...)
	return true
}

func (ul *Units) change(n string, severity ChangeSeverity) func() {
	return (*Model)(ul).change("UnitListView."+n, PatchChange, severity)
}

func (ul *Units) cancel() {
	(*Model)(ul).changeCancel = true
}

func (ul *Units) marshal(r Range) ([]byte, error) {
	m := (*Model)(ul)
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

func (ul *Units) unmarshal(data []byte) (r Range, err error) {
	m := (*Model)(ul)
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
	sel := ul.Selected()
	var ok bool
	m.d.Song.Patch[m.d.InstrIndex].Units, ok = Insert(m.d.Song.Patch[m.d.InstrIndex].Units, sel, pastedUnits.Units...)
	if !ok {
		return Range{}, errors.New("UnitListView.unmarshal: insert failed")
	}
	return Range{sel, sel + len(pastedUnits.Units)}, nil
}

func (ul *Units) CurrentInstrumentUnitAt(index int) sointu.Unit {
	units := ul.d.Song.Patch[ul.d.InstrIndex].Units
	if index < 0 || index >= len(units) {
		return sointu.Unit{}
	}
	return units[index]
}
