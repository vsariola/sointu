package tracker

import (
	"strings"

	"github.com/vsariola/sointu"
)

type (
	String struct {
		value StringValue
	}

	StringValue interface {
		Value() string
		SetValue(string) bool
	}

	FilePath          Model
	InstrumentName    Model
	InstrumentComment Model
	UnitSearch        Model
	UnitComment       Model
)

func MakeString(value StringValue) String {
	return String{value: value}
}

func (v String) SetValue(value string) bool {
	if v.value == nil || v.value.Value() == value {
		return false
	}
	return v.value.SetValue(value)
}

func (v String) Value() string {
	if v.value == nil {
		return ""
	}
	return v.value.Value()
}

// Model methods

func (m *Model) FilePath() String          { return MakeString((*FilePath)(m)) }
func (m *Model) InstrumentName() String    { return MakeString((*InstrumentName)(m)) }
func (m *Model) InstrumentComment() String { return MakeString((*InstrumentComment)(m)) }
func (m *Model) UnitSearch() String        { return MakeString((*UnitSearch)(m)) }
func (m *Model) UnitComment() String       { return MakeString((*UnitComment)(m)) }

// FilePathString

func (v *FilePath) Value() string              { return v.d.FilePath }
func (v *FilePath) SetValue(value string) bool { v.d.FilePath = value; return true }

// UnitSearchString

func (v *UnitSearch) Value() string {
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
func (v *UnitSearch) SetValue(value string) bool {
	v.d.UnitSearchString = value
	v.d.UnitSearching = true
	(*Model)(v).updateDerivedUnitSearch()
	return true
}
func (v *Model) updateDerivedUnitSearch() {
	// update search results based on current search string
	v.derived.searchResults = v.derived.searchResults[:0]
	for _, name := range sointu.UnitNames {
		if strings.HasPrefix(name, v.UnitSearch().Value()) {
			v.derived.searchResults = append(v.derived.searchResults, name)
		}
	}
}

// InstrumentNameString

func (v *InstrumentName) Value() string {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return ""
	}
	return v.d.Song.Patch[v.d.InstrIndex].Name
}

func (v *InstrumentName) SetValue(value string) bool {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return false
	}
	defer (*Model)(v).change("InstrumentNameString", PatchChange, MinorChange)()
	v.d.Song.Patch[v.d.InstrIndex].Name = value
	return true
}

// InstrumentComment

func (v *InstrumentComment) Value() string {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return ""
	}
	return v.d.Song.Patch[v.d.InstrIndex].Comment
}

func (v *InstrumentComment) SetValue(value string) bool {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return false
	}
	defer (*Model)(v).change("InstrumentComment", PatchChange, MinorChange)()
	v.d.Song.Patch[v.d.InstrIndex].Comment = value
	return true
}

// UnitComment

func (v *UnitComment) Value() string {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) ||
		v.d.UnitIndex < 0 || v.d.UnitIndex >= len(v.d.Song.Patch[v.d.InstrIndex].Units) {
		return ""
	}
	return v.d.Song.Patch[v.d.InstrIndex].Units[v.d.UnitIndex].Comment
}

func (v *UnitComment) SetValue(value string) bool {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) ||
		v.d.UnitIndex < 0 || v.d.UnitIndex >= len(v.d.Song.Patch[v.d.InstrIndex].Units) {
		return false
	}
	defer (*Model)(v).change("UnitComment", PatchChange, MinorChange)()
	v.d.Song.Patch[v.d.InstrIndex].Units[v.d.UnitIndex].Comment = value
	return true
}
