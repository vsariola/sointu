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

// FilePathString
type filePath Model

func (m *Model) FilePath() String              { return MakeString((*filePath)(m)) }
func (v *filePath) Value() string              { return v.d.FilePath }
func (v *filePath) SetValue(value string) bool { v.d.FilePath = value; return true }

// UnitSearchString
type unitSearch Model

func (m *Model) UnitSearch() String { return MakeString((*unitSearch)(m)) }
func (v *unitSearch) Value() string {
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
func (v *unitSearch) SetValue(value string) bool {
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
type instrumentName Model

func (m *Model) InstrumentName() String { return MakeString((*instrumentName)(m)) }
func (v *instrumentName) Value() string {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return ""
	}
	return v.d.Song.Patch[v.d.InstrIndex].Name
}
func (v *instrumentName) SetValue(value string) bool {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return false
	}
	defer (*Model)(v).change("InstrumentNameString", PatchChange, MinorChange)()
	v.d.Song.Patch[v.d.InstrIndex].Name = value
	return true
}

// InstrumentComment
type instrumentComment Model

func (m *Model) InstrumentComment() String { return MakeString((*instrumentComment)(m)) }
func (v *instrumentComment) Value() string {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return ""
	}
	return v.d.Song.Patch[v.d.InstrIndex].Comment
}
func (v *instrumentComment) SetValue(value string) bool {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return false
	}
	defer (*Model)(v).change("InstrumentComment", PatchChange, MinorChange)()
	v.d.Song.Patch[v.d.InstrIndex].Comment = value
	return true
}

// UnitComment
type unitComment Model

func (m *Model) UnitComment() String { return MakeString((*unitComment)(m)) }
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
