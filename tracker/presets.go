package tracker

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
	"gopkg.in/yaml.v3"
)

//go:generate go run generate/gmdls_entries.go
//go:generate go run generate/clean_presets.go

// Preset returns a PresetModel, a view of the model used to manipulate
// instrument presets.
func (m *Model) Preset() *PresetModel { return (*PresetModel)(m) }

type PresetModel Model

// SearchTerm returns a String containing the search terms for finding the
// presets.
func (m *PresetModel) SearchTerm() String { return MakeString((*presetSearchTerm)(m)) }

type presetSearchTerm PresetModel

func (m *presetSearchTerm) Value() string { return m.d.PresetSearchString }
func (m *presetSearchTerm) SetValue(value string) bool {
	if m.d.PresetSearchString == value {
		return false
	}
	m.d.PresetSearchString = value
	(*PresetModel)(m).updateCache()
	return true
}

// NoGmDls returns a Bool toggling whether to show presets relying on gm.dls
// samples.
func (m *PresetModel) NoGmDls() Bool { return MakeBool((*presetNoGmDls)(m)) }

type presetNoGmDls PresetModel

func (m *presetNoGmDls) Value() bool { return m.presetData.cache.noGmDls }
func (m *presetNoGmDls) SetValue(val bool) {
	if m.presetData.cache.noGmDls == val {
		return
	}
	m.d.PresetSearchString = removeFilters(m.d.PresetSearchString, "g:")
	if val {
		m.d.PresetSearchString = "g:n " + m.d.PresetSearchString
	}
	(*PresetModel)(m).updateCache()
}

// UserPresetsFilter returns a Bool toggling whether to show the user defined
// presets.
func (m *PresetModel) UserFilter() Bool { return MakeBool((*userPresetsFilter)(m)) }

type userPresetsFilter PresetModel

func (m *userPresetsFilter) Value() bool { return m.presetData.cache.kind == UserPresets }
func (m *userPresetsFilter) SetValue(val bool) {
	if (m.presetData.cache.kind == UserPresets) == val {
		return
	}
	m.d.PresetSearchString = removeFilters(m.d.PresetSearchString, "t:")
	if val {
		m.d.PresetSearchString = "t:u " + m.d.PresetSearchString
	}
	(*PresetModel)(m).updateCache()
}
func (m *userPresetsFilter) Enabled() bool { return true }

// BuiltinFilter return a Bool toggling whether to show the built-in
// presets in the preset search results.
func (m *PresetModel) BuiltinFilter() Bool { return MakeBool((*builtinPresetsFilter)(m)) }

type builtinPresetsFilter PresetModel

func (m *builtinPresetsFilter) Value() bool { return m.presetData.cache.kind == BuiltinPresets }
func (m *builtinPresetsFilter) SetValue(val bool) {
	if (m.presetData.cache.kind == BuiltinPresets) == val {
		return
	}
	m.d.PresetSearchString = removeFilters(m.d.PresetSearchString, "t:")
	if val {
		m.d.PresetSearchString = "t:b " + m.d.PresetSearchString
	}
	(*PresetModel)(m).updateCache()
}

// ClearSearch returns an Action to clear the current preset search
// term(s).
func (m *PresetModel) ClearSearch() Action { return MakeAction((*clearPresetSearch)(m)) }

type clearPresetSearch PresetModel

func (m *clearPresetSearch) Enabled() bool { return len(m.d.PresetSearchString) > 0 }
func (m *clearPresetSearch) Do() {
	m.d.PresetSearchString = ""
	(*PresetModel)(m).updateCache()
}

// PresetDirList return a List of all the different preset directories.
func (m *PresetModel) DirList() List { return MakeList((*presetDirList)(m)) }

type presetDirList PresetModel

func (m *presetDirList) Count() int         { return len(m.presetData.dirs) + 1 }
func (m *presetDirList) Selected() int      { return m.presetData.cache.dirIndex + 1 }
func (m *presetDirList) Selected2() int     { return m.presetData.cache.dirIndex + 1 }
func (m *presetDirList) SetSelected2(i int) {}
func (m *presetDirList) SetSelected(i int) {
	i = min(max(i, 0), len(m.presetData.dirs))
	if i < 0 || i > len(m.presetData.dirs) {
		return
	}
	m.d.PresetSearchString = removeFilters(m.d.PresetSearchString, "d:")
	if i > 0 {
		m.d.PresetSearchString = "d:" + m.presetData.dirs[i-1] + " " + m.d.PresetSearchString
	}
	(*PresetModel)(m).updateCache()
}

// Dir returns the name of the directory at the given index in the preset
// directory list.
func (m *PresetModel) Dir(i int) string {
	if i < 1 || i > len(m.presetData.dirs) {
		return "---"
	}
	return m.presetData.dirs[i-1]
}

// SearchResultList returns a List of the current preset search results.
func (m *PresetModel) SearchResultList() List { return MakeList((*presetResultList)(m)) }

type presetResultList PresetModel

func (v *presetResultList) List() List { return List{v} }
func (m *presetResultList) Count() int { return len(m.presetData.cache.results) }
func (m *presetResultList) Selected() int {
	return min(max(m.presetData.presetIndex, 0), len(m.presetData.cache.results)-1)
}
func (m *presetResultList) Selected2() int     { return m.Selected() }
func (m *presetResultList) SetSelected2(i int) {}
func (m *presetResultList) SetSelected(i int) {
	i = min(max(i, 0), len(m.presetData.cache.results)-1)
	if i < 0 || i >= len(m.presetData.cache.results) {
		return
	}
	m.presetData.presetIndex = i
	defer (*Model)(m).change("LoadPreset", PatchChange, MinorChange)()
	if m.d.InstrIndex < 0 {
		m.d.InstrIndex = 0
	}
	m.d.InstrIndex2 = m.d.InstrIndex
	for m.d.InstrIndex >= len(m.d.Song.Patch) {
		m.d.Song.Patch = append(m.d.Song.Patch, defaultInstrument.Copy())
	}
	newInstr := m.presetData.cache.results[i].instr.Copy()
	newInstr.NumVoices = clamp(m.d.Song.Patch[m.d.InstrIndex].NumVoices, 1, vm.MAX_VOICES)
	(*Model)(m).assignUnitIDs(newInstr.Units)
	m.d.Song.Patch[m.d.InstrIndex] = newInstr
}

// SearchResult returns the search result at the given index in the search
// result list.
func (m *PresetModel) SearchResult(i int) (name string, dir string, user bool) {
	if i < 0 || i >= len(m.presetData.cache.results) {
		return "", "", false
	}
	p := m.presetData.cache.results[i]
	return p.instr.Name, p.dir, p.user
}

// Save returns an Action to save the current instrument as a user-defined
// preset. It will not overwrite existing presets, but rather show a dialog to
// confirm the overwrite.
func (m *PresetModel) Save() Action { return MakeAction((*saveUserPreset)(m)) }

type saveUserPreset PresetModel

func (m *saveUserPreset) Enabled() bool {
	return m.d.InstrIndex >= 0 && m.d.InstrIndex < len(m.d.Song.Patch)
}
func (m *saveUserPreset) Do() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	userPresetsDir := filepath.Join(configDir, "sointu", "presets", m.presetData.cache.dir)
	instr := m.d.Song.Patch[m.d.InstrIndex]
	name := instrumentNameToFilename(instr.Name)
	fileName := filepath.Join(userPresetsDir, name+".yml")
	// if exists, do not overwrite
	if _, err := os.Stat(fileName); err == nil {
		m.dialog = OverwriteUserPresetDialog
		return
	}
	(*PresetModel)(m).Overwrite().Do()
}

// OverwriteUserPreset returns an Action to overwrite the current instrument
// as a user-defined preset.
func (m *PresetModel) Overwrite() Action { return MakeAction((*overwriteUserPreset)(m)) }

type overwriteUserPreset PresetModel

func (m *overwriteUserPreset) Enabled() bool { return true }
func (m *overwriteUserPreset) Do() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	userPresetsDir := filepath.Join(configDir, "sointu", "presets", m.presetData.cache.dir)
	instr := m.d.Song.Patch[m.d.InstrIndex]
	name := instrumentNameToFilename(instr.Name)
	fileName := filepath.Join(userPresetsDir, name+".yml")
	os.MkdirAll(userPresetsDir, 0755)
	data, err := yaml.Marshal(&instr)
	if err != nil {
		return
	}
	os.WriteFile(fileName, data, 0644)
	m.dialog = NoDialog
	(*PresetModel)(m).presetData.load()
	(*PresetModel)(m).updateCache()
}

// TryDeleteUserPreset returns an Action to display a dialog to confirm deletion
// of an user preset.
func (m *PresetModel) Delete() Action { return MakeAction((*tryDeleteUserPreset)(m)) }

type tryDeleteUserPreset PresetModel

func (m *tryDeleteUserPreset) Do() { m.dialog = DeleteUserPresetDialog }
func (m *tryDeleteUserPreset) Enabled() bool {
	if m.presetData.presetIndex < 0 || m.presetData.presetIndex >= len(m.presetData.cache.results) {
		return false
	}
	return m.presetData.cache.results[m.presetData.presetIndex].user
}

// DeleteUserPreset returns an Action to confirm the deletion of an user preset.
func (m *PresetModel) ConfirmDelete() Action { return MakeAction((*deleteUserPreset)(m)) }

type deleteUserPreset PresetModel

func (m *deleteUserPreset) Enabled() bool { return (*Model)(m).Preset().Delete().Enabled() }
func (m *deleteUserPreset) Do() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	p := m.presetData.cache.results[m.presetData.presetIndex]
	userPresetsDir := filepath.Join(configDir, "sointu", "presets")
	if p.dir != "" {
		userPresetsDir = filepath.Join(userPresetsDir, p.dir)
	}
	name := instrumentNameToFilename(p.instr.Name)
	fileName := filepath.Join(userPresetsDir, name+".yml")
	os.Remove(fileName)
	m.dialog = NoDialog
	(*PresetModel)(m).presetData.load()
	(*PresetModel)(m).updateCache()
}

type (
	presetData struct {
		presets     []preset
		dirs        []string
		presetIndex int

		cache presetCache
	}

	preset struct {
		dir        string
		user       bool
		needsGmDls bool
		instr      sointu.Instrument
	}

	presetCache struct {
		dir           string
		dirIndex      int
		noGmDls       bool
		kind          presetKindEnum
		searchStrings []string
		results       []preset
	}

	presetKindEnum int
)

const (
	BuiltinPresets presetKindEnum = -1
	AllPresets     presetKindEnum = 0
	UserPresets    presetKindEnum = 1
)

func (m *PresetModel) updateCache() {
	// reset derived data, keeping the
	str := m.presetData.cache.searchStrings[:0]
	m.presetData.cache = presetCache{searchStrings: str, dirIndex: -1}
	// parse filters from the search string. in: dir, gmdls: yes/no, kind: builtin/user/all
	search := strings.TrimSpace(m.d.PresetSearchString)
	parts := strings.Fields(search)
	// parse parts to see if they contain :
	for _, part := range parts {
		if strings.HasPrefix(part, "d:") && len(part) > 2 {
			dir := strings.TrimSpace(part[2:])
			m.presetData.cache.dir = dir
			ind := slices.IndexFunc(m.presetData.dirs, func(c string) bool { return c == dir })
			m.presetData.cache.dirIndex = ind
		} else if strings.HasPrefix(part, "g:n") {
			m.presetData.cache.noGmDls = true
		} else if strings.HasPrefix(part, "t:") && len(part) > 2 {
			val := strings.TrimSpace(part[2:3])
			switch val {
			case "b":
				m.presetData.cache.kind = BuiltinPresets
			case "u":
				m.presetData.cache.kind = UserPresets
			}
		} else {
			m.presetData.cache.searchStrings = append(m.presetData.cache.searchStrings, strings.ToLower(part))
		}
	}
	// update results
	m.presetData.cache.results = m.presetData.cache.results[:0]
	for _, p := range m.presetData.presets {
		if m.presetData.cache.kind == BuiltinPresets && p.user {
			continue
		}
		if m.presetData.cache.kind == UserPresets && !p.user {
			continue
		}
		if m.presetData.cache.dir != "" && p.dir != m.presetData.cache.dir {
			continue
		}
		if m.presetData.cache.noGmDls && p.needsGmDls {
			continue
		}
		if len(m.presetData.cache.searchStrings) == 0 {
			goto found
		}
		for _, s := range m.presetData.cache.searchStrings {
			if strings.Contains(strings.ToLower(p.instr.Name), s) {
				goto found
			}
		}
		continue
	found:
		m.presetData.cache.results = append(m.presetData.cache.results, p)
	}
}

//go:embed presets/*
var builtInPresetsFS embed.FS

func (m *presetData) load() {
	m.dirs = m.dirs[:0]
	m.presets = m.presets[:0]
	seenDir := make(map[string]bool)
	m.loadPresetsFromFs(builtInPresetsFS, false, seenDir)
	if configDir, err := os.UserConfigDir(); err == nil {
		userPresets := filepath.Join(configDir, "sointu")
		m.loadPresetsFromFs(os.DirFS(userPresets), true, seenDir)
	}
	sort.Sort(m)
	m.dirs = make([]string, 0, len(seenDir))
	for k := range seenDir {
		m.dirs = append(m.dirs, k)
	}
	sort.Strings(m.dirs)
}

func (m *presetData) loadPresetsFromFs(fsys fs.FS, userDefined bool, seenDir map[string]bool) {
	fs.WalkDir(fsys, "presets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return nil
		}
		var instr sointu.Instrument

		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if dec.Decode(&instr) == nil {
			noExt := path[:len(path)-len(filepath.Ext(path))]
			splitted := splitPath(noExt)
			splitted = splitted[1:] // remove "presets" from the path
			instr.Name = filenameToInstrumentName(splitted[len(splitted)-1])
			dir := strings.Join(splitted[:len(splitted)-1], "/")
			preset := preset{
				dir:        dir,
				user:       userDefined,
				instr:      instr,
				needsGmDls: checkNeedsGmDls(instr),
			}
			if dir != "" {
				seenDir[dir] = true
			}
			m.presets = append(m.presets, preset)
		}
		return nil
	})
}

func filenameToInstrumentName(filename string) string {
	return strings.ReplaceAll(filename, "_", " ")
}

func instrumentNameToFilename(name string) string {
	// remove all special characters
	reg, _ := regexp.Compile("[^a-zA-Z0-9 _]+")
	name = reg.ReplaceAllString(name, "")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

func checkNeedsGmDls(instr sointu.Instrument) bool {
	for _, u := range instr.Units {
		if u.Type == "oscillator" {
			if u.Parameters["type"] == sointu.Sample {
				return true
			}
		}
	}
	return false
}

func removeFilters(str string, prefix string) string {
	parts := strings.Split(str, " ")
	newParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if !strings.HasPrefix(strings.ToLower(part), prefix) {
			newParts = append(newParts, part)
		}
	}
	return strings.Join(newParts, " ")
}

func splitPath(path string) []string {
	subPath := path
	var result []string
	for {
		subPath = filepath.Clean(subPath) // Amongst others, removes trailing slashes (except for the root directory).

		dir, last := filepath.Split(subPath)
		if last == "" {
			if dir != "" { // Root directory.
				result = append(result, dir)
			}
			break
		}
		result = append(result, last)

		if dir == "" { // Nothing to split anymore.
			break
		}
		subPath = dir
	}

	slices.Reverse(result)
	return result
}

func (p presetData) Len() int { return len(p.presets) }
func (p presetData) Less(i, j int) bool {
	if p.presets[i].instr.Name == p.presets[j].instr.Name {
		return p.presets[i].user && !p.presets[j].user
	}
	return p.presets[i].instr.Name < p.presets[j].instr.Name
}
func (p presetData) Swap(i, j int) { p.presets[i], p.presets[j] = p.presets[j], p.presets[i] }
