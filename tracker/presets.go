package tracker

import (
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
	"gopkg.in/yaml.v2"
)

//go:generate go run generate/gmdls_entries.go
//go:generate go run generate/clean_presets.go

//go:embed presets/*
var instrumentPresetFS embed.FS

type (
	// GmDlsEntry is a single sample entry from the gm.dls file
	GmDlsEntry struct {
		Start              int    // sample start offset in words
		LoopStart          int    // loop start offset in words
		LoopLength         int    // loop length in words
		SuggestedTranspose int    // suggested transpose in semitones, so that all samples play at same pitch
		Name               string // sample Name
	}

	Preset struct {
		Directory  string
		User       bool
		NeedsGmDls bool
		Instr      sointu.Instrument
	}

	Presets struct {
		Presets []Preset
		Dirs    []string
	}

	InstrumentPresetYieldFunc func(index int, item string) (ok bool)
	LoadPreset                struct {
		Index int
		*Model
	}

	PresetSearchString   Model
	NoGmDlsFilter        Model
	BuiltinPresetsFilter Model
	UserPresetsFilter    Model
	PresetDirectory      Model
	PresetKind           Model
	ClearPresetSearch    Model
	PresetDirList        Model
	PresetResultList     Model
	SaveUserPreset       Model
	TryDeleteUserPreset  Model
	DeleteUserPreset     Model

	ConfirmDeleteUserPresetAction Model
	OverwriteUserPreset           Model

	derivedPresetSearch struct {
		dir           string
		dirIndex      int
		noGmDls       bool
		kind          PresetKindEnum
		searchStrings []string
		results       []Preset
	}

	PresetKindEnum int
)

const (
	BuiltinPresets PresetKindEnum = -1
	AllPresets     PresetKindEnum = 0
	UserPresets    PresetKindEnum = 1
)

func (m *Model) updateDerivedPresetSearch() {
	// reset derived data, keeping the
	str := m.derived.presetSearch.searchStrings[:0]
	m.derived.presetSearch = derivedPresetSearch{searchStrings: str, dirIndex: -1}
	// parse filters from the search string. in: dir, gmdls: yes/no, kind: builtin/user/all
	search := strings.TrimSpace(m.d.PresetSearchString)
	parts := strings.Fields(search)
	// parse parts to see if they contain :
	for _, part := range parts {
		if strings.HasPrefix(part, "d:") && len(part) > 2 {
			dir := strings.TrimSpace(part[2:])
			m.derived.presetSearch.dir = dir
			ind := slices.IndexFunc(m.presets.Dirs, func(c string) bool { return c == dir })
			m.derived.presetSearch.dirIndex = ind
		} else if strings.HasPrefix(part, "g:n") {
			m.derived.presetSearch.noGmDls = true
		} else if strings.HasPrefix(part, "t:") && len(part) > 2 {
			val := strings.TrimSpace(part[2:3])
			switch val {
			case "b":
				m.derived.presetSearch.kind = BuiltinPresets
			case "u":
				m.derived.presetSearch.kind = UserPresets
			}
		} else {
			m.derived.presetSearch.searchStrings = append(m.derived.presetSearch.searchStrings, strings.ToLower(part))
		}
	}
	// update results
	m.derived.presetSearch.results = m.derived.presetSearch.results[:0]
	for _, p := range m.presets.Presets {
		if m.derived.presetSearch.kind == BuiltinPresets && p.User {
			continue
		}
		if m.derived.presetSearch.kind == UserPresets && !p.User {
			continue
		}
		if m.derived.presetSearch.dir != "" && p.Directory != m.derived.presetSearch.dir {
			continue
		}
		if m.derived.presetSearch.noGmDls && p.NeedsGmDls {
			continue
		}
		if len(m.derived.presetSearch.searchStrings) == 0 {
			goto found
		}
		for _, s := range m.derived.presetSearch.searchStrings {
			if strings.Contains(strings.ToLower(p.Instr.Name), s) {
				goto found
			}
		}
		continue
	found:
		m.derived.presetSearch.results = append(m.derived.presetSearch.results, p)
	}
}

func (m *Presets) load() {
	*m = Presets{}
	seenDir := make(map[string]bool)
	m.loadPresetsFromFs(instrumentPresetFS, false, seenDir)
	if configDir, err := os.UserConfigDir(); err == nil {
		userPresets := filepath.Join(configDir, "sointu")
		m.loadPresetsFromFs(os.DirFS(userPresets), true, seenDir)
	}
	sort.Sort(m)
	m.Dirs = make([]string, 0, len(seenDir))
	for k := range seenDir {
		m.Dirs = append(m.Dirs, k)
	}
	sort.Strings(m.Dirs)
}

func (m *Presets) loadPresetsFromFs(fsys fs.FS, userDefined bool, seenDir map[string]bool) {
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
		if yaml.UnmarshalStrict(data, &instr) == nil {
			noExt := path[:len(path)-len(filepath.Ext(path))]
			splitted := splitPath(noExt)
			splitted = splitted[1:] // remove "presets" from the path
			instr.Name = filenameToInstrumentName(splitted[len(splitted)-1])
			dir := strings.Join(splitted[:len(splitted)-1], "/")
			preset := Preset{
				Directory:  dir,
				User:       userDefined,
				Instr:      instr,
				NeedsGmDls: checkNeedsGmDls(instr),
			}
			if dir != "" {
				seenDir[dir] = true
			}
			m.Presets = append(m.Presets, preset)
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

func (m *Model) PresetSearchString() String { return MakeString((*PresetSearchString)(m)) }
func (m *PresetSearchString) Value() string { return m.d.PresetSearchString }
func (m *PresetSearchString) SetValue(value string) bool {
	if m.d.PresetSearchString == value {
		return false
	}
	m.d.PresetSearchString = value
	(*Model)(m).updateDerivedPresetSearch()
	return true
}

func (m *Model) NoGmDls() Bool       { return MakeBool((*NoGmDlsFilter)(m)) }
func (m *NoGmDlsFilter) Value() bool { return m.derived.presetSearch.noGmDls }
func (m *NoGmDlsFilter) SetValue(val bool) {
	if m.derived.presetSearch.noGmDls == val {
		return
	}
	m.d.PresetSearchString = removeFilters(m.d.PresetSearchString, "g:")
	if val {
		m.d.PresetSearchString = "g:n " + m.d.PresetSearchString
	}
	(*Model)(m).updateDerivedPresetSearch()
}
func (m *NoGmDlsFilter) Enabled() bool { return true }

func (m *Model) UserPresetFilter() Bool  { return MakeBool((*UserPresetsFilter)(m)) }
func (m *UserPresetsFilter) Value() bool { return m.derived.presetSearch.kind == UserPresets }
func (m *UserPresetsFilter) SetValue(val bool) {
	if (m.derived.presetSearch.kind == UserPresets) == val {
		return
	}
	m.d.PresetSearchString = removeFilters(m.d.PresetSearchString, "t:")
	if val {
		m.d.PresetSearchString = "t:u " + m.d.PresetSearchString
	}
	(*Model)(m).updateDerivedPresetSearch()
}
func (m *UserPresetsFilter) Enabled() bool { return true }

func (m *Model) BuiltinPresetsFilter() Bool { return MakeBool((*BuiltinPresetsFilter)(m)) }
func (m *BuiltinPresetsFilter) Value() bool { return m.derived.presetSearch.kind == BuiltinPresets }
func (m *BuiltinPresetsFilter) SetValue(val bool) {
	if (m.derived.presetSearch.kind == BuiltinPresets) == val {
		return
	}
	m.d.PresetSearchString = removeFilters(m.d.PresetSearchString, "t:")
	if val {
		m.d.PresetSearchString = "t:b " + m.d.PresetSearchString
	}
	(*Model)(m).updateDerivedPresetSearch()
}
func (m *BuiltinPresetsFilter) Enabled() bool { return true }

func (m *Model) ClearPresetSearch() Action { return MakeAction((*ClearPresetSearch)(m)) }
func (m *ClearPresetSearch) Enabled() bool { return len(m.d.PresetSearchString) > 0 }
func (m *ClearPresetSearch) Do() {
	m.d.PresetSearchString = ""
	(*Model)(m).updateDerivedPresetSearch()
}

func (m *Model) PresetDirList() *PresetDirList { return (*PresetDirList)(m) }
func (v *PresetDirList) List() List            { return List{v} }
func (m *PresetDirList) Count() int            { return len(m.presets.Dirs) + 1 }
func (m *PresetDirList) Selected() int         { return m.derived.presetSearch.dirIndex + 1 }
func (m *PresetDirList) Selected2() int        { return m.derived.presetSearch.dirIndex + 1 }
func (m *PresetDirList) SetSelected2(i int)    {}
func (m *PresetDirList) Value(i int) string {
	if i < 1 || i > len(m.presets.Dirs) {
		return "---"
	}
	return m.presets.Dirs[i-1]
}
func (m *PresetDirList) SetSelected(i int) {
	i = min(max(i, 0), len(m.presets.Dirs))
	if i < 0 || i > len(m.presets.Dirs) {
		return
	}
	m.d.PresetSearchString = removeFilters(m.d.PresetSearchString, "d:")
	if i > 0 {
		m.d.PresetSearchString = "d:" + m.presets.Dirs[i-1] + " " + m.d.PresetSearchString
	}
	(*Model)(m).updateDerivedPresetSearch()
}

func (m *Model) PresetResultList() *PresetResultList { return (*PresetResultList)(m) }
func (v *PresetResultList) List() List               { return List{v} }
func (m *PresetResultList) Count() int               { return len(m.derived.presetSearch.results) }
func (m *PresetResultList) Selected() int {
	return min(max(m.presetIndex, 0), len(m.derived.presetSearch.results)-1)
}
func (m *PresetResultList) Selected2() int     { return m.Selected() }
func (m *PresetResultList) SetSelected2(i int) {}
func (m *PresetResultList) Value(i int) (name string, dir string, user bool) {
	if i < 0 || i >= len(m.derived.presetSearch.results) {
		return "", "", false
	}
	p := m.derived.presetSearch.results[i]
	return p.Instr.Name, p.Directory, p.User
}
func (m *PresetResultList) SetSelected(i int) {
	i = min(max(i, 0), len(m.derived.presetSearch.results)-1)
	if i < 0 || i >= len(m.derived.presetSearch.results) {
		return
	}
	m.presetIndex = i
	defer (*Model)(m).change("LoadPreset", PatchChange, MinorChange)()
	if m.d.InstrIndex < 0 {
		m.d.InstrIndex = 0
	}
	m.d.InstrIndex2 = m.d.InstrIndex
	for m.d.InstrIndex >= len(m.d.Song.Patch) {
		m.d.Song.Patch = append(m.d.Song.Patch, defaultInstrument.Copy())
	}
	newInstr := m.derived.presetSearch.results[i].Instr.Copy()
	newInstr.NumVoices = clamp(m.d.Song.Patch[m.d.InstrIndex].NumVoices, 1, vm.MAX_VOICES)
	(*Model)(m).assignUnitIDs(newInstr.Units)
	m.d.Song.Patch[m.d.InstrIndex] = newInstr
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

func (m *Model) SaveAsUserPreset() Action { return MakeAction((*SaveUserPreset)(m)) }
func (m *SaveUserPreset) Enabled() bool {
	return m.d.InstrIndex >= 0 && m.d.InstrIndex < len(m.d.Song.Patch)
}
func (m *SaveUserPreset) Do() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	userPresetsDir := filepath.Join(configDir, "sointu", "presets", m.derived.presetSearch.dir)
	instr := m.d.Song.Patch[m.d.InstrIndex]
	name := instrumentNameToFilename(instr.Name)
	fileName := filepath.Join(userPresetsDir, name+".yml")
	// if exists, do not overwrite
	if _, err := os.Stat(fileName); err == nil {
		m.dialog = OverwriteUserPresetDialog
		return
	}
	(*Model)(m).OverwriteUserPreset().Do()
}

func (m *Model) OverwriteUserPreset() Action { return MakeAction((*OverwriteUserPreset)(m)) }
func (m *OverwriteUserPreset) Enabled() bool { return true }
func (m *OverwriteUserPreset) Do() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	userPresetsDir := filepath.Join(configDir, "sointu", "presets", m.derived.presetSearch.dir)
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
	(*Model)(m).presets.load()
	(*Model)(m).updateDerivedPresetSearch()
}

func (m *Model) TryDeleteUserPreset() Action { return MakeAction((*TryDeleteUserPreset)(m)) }
func (m *TryDeleteUserPreset) Do()           { m.dialog = DeleteUserPresetDialog }
func (m *TryDeleteUserPreset) Enabled() bool {
	if m.presetIndex < 0 || m.presetIndex >= len(m.derived.presetSearch.results) {
		return false
	}
	return m.derived.presetSearch.results[m.presetIndex].User
}

func (m *Model) DeleteUserPreset() Action { return MakeAction((*DeleteUserPreset)(m)) }
func (m *DeleteUserPreset) Enabled() bool { return (*Model)(m).TryDeleteUserPreset().Enabled() }
func (m *DeleteUserPreset) Do() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	p := m.derived.presetSearch.results[m.presetIndex]
	userPresetsDir := filepath.Join(configDir, "sointu", "presets")
	if p.Directory != "" {
		userPresetsDir = filepath.Join(userPresetsDir, p.Directory)
	}
	name := instrumentNameToFilename(p.Instr.Name)
	fileName := filepath.Join(userPresetsDir, name+".yml")
	os.Remove(fileName)
	m.dialog = NoDialog
	(*Model)(m).presets.load()
	(*Model)(m).updateDerivedPresetSearch()
}

// gmDlsEntryMap is a reverse map, to find the index of the GmDlsEntry in the
// GmDlsEntries list based on the sample offset. Do not modify during runtime.
var gmDlsEntryMap = make(map[vm.SampleOffset]int)

func init() {
	for i, e := range GmDlsEntries {
		key := vm.SampleOffset{Start: uint32(e.Start), LoopStart: uint16(e.LoopStart), LoopLength: uint16(e.LoopLength)}
		gmDlsEntryMap[key] = i
	}
}

var defaultUnits = map[string]sointu.Unit{
	"envelope":   {Type: "envelope", Parameters: map[string]int{"stereo": 0, "attack": 64, "decay": 64, "sustain": 64, "release": 64, "gain": 64}},
	"oscillator": {Type: "oscillator", Parameters: map[string]int{"stereo": 0, "transpose": 64, "detune": 64, "phase": 0, "color": 64, "shape": 64, "gain": 64, "type": sointu.Sine}},
	"noise":      {Type: "noise", Parameters: map[string]int{"stereo": 0, "shape": 64, "gain": 64}},
	"mulp":       {Type: "mulp", Parameters: map[string]int{"stereo": 0}},
	"mul":        {Type: "mul", Parameters: map[string]int{"stereo": 0}},
	"add":        {Type: "add", Parameters: map[string]int{"stereo": 0}},
	"addp":       {Type: "addp", Parameters: map[string]int{"stereo": 0}},
	"push":       {Type: "push", Parameters: map[string]int{"stereo": 0}},
	"pop":        {Type: "pop", Parameters: map[string]int{"stereo": 0}},
	"xch":        {Type: "xch", Parameters: map[string]int{"stereo": 0}},
	"receive":    {Type: "receive", Parameters: map[string]int{"stereo": 0}},
	"loadnote":   {Type: "loadnote", Parameters: map[string]int{"stereo": 0}},
	"loadval":    {Type: "loadval", Parameters: map[string]int{"stereo": 0, "value": 64}},
	"pan":        {Type: "pan", Parameters: map[string]int{"stereo": 0, "panning": 64}},
	"gain":       {Type: "gain", Parameters: map[string]int{"stereo": 0, "gain": 64}},
	"invgain":    {Type: "invgain", Parameters: map[string]int{"stereo": 0, "invgain": 64}},
	"dbgain":     {Type: "dbgain", Parameters: map[string]int{"stereo": 0, "decibels": 64}},
	"crush":      {Type: "crush", Parameters: map[string]int{"stereo": 0, "resolution": 64}},
	"clip":       {Type: "clip", Parameters: map[string]int{"stereo": 0}},
	"hold":       {Type: "hold", Parameters: map[string]int{"stereo": 0, "holdfreq": 64}},
	"distort":    {Type: "distort", Parameters: map[string]int{"stereo": 0, "drive": 64}},
	"filter":     {Type: "filter", Parameters: map[string]int{"stereo": 0, "frequency": 64, "resonance": 64, "lowpass": 1, "bandpass": 0, "highpass": 0}},
	"out":        {Type: "out", Parameters: map[string]int{"stereo": 1, "gain": 64}},
	"outaux":     {Type: "outaux", Parameters: map[string]int{"stereo": 1, "outgain": 64, "auxgain": 64}},
	"aux":        {Type: "aux", Parameters: map[string]int{"stereo": 1, "gain": 64, "channel": 2}},
	"delay": {Type: "delay",
		Parameters: map[string]int{"damp": 0, "dry": 128, "feedback": 96, "notetracking": 2, "pregain": 40, "stereo": 0},
		VarArgs:    []int{48}},
	"in":         {Type: "in", Parameters: map[string]int{"stereo": 1, "channel": 2}},
	"speed":      {Type: "speed", Parameters: map[string]int{}},
	"compressor": {Type: "compressor", Parameters: map[string]int{"stereo": 0, "attack": 64, "release": 64, "invgain": 64, "threshold": 64, "ratio": 64}},
	"send":       {Type: "send", Parameters: map[string]int{"stereo": 0, "amount": 64, "voice": 0, "unit": 0, "port": 0, "sendpop": 1}},
	"sync":       {Type: "sync", Parameters: map[string]int{}},
}

var defaultInstrument = sointu.Instrument{
	Name:      "Instr",
	NumVoices: 1,
	Units: []sointu.Unit{
		defaultUnits["envelope"],
		defaultUnits["oscillator"],
		defaultUnits["mulp"],
		defaultUnits["delay"],
		defaultUnits["pan"],
		defaultUnits["outaux"],
	},
}

var defaultSong = sointu.Song{
	BPM:         100,
	RowsPerBeat: 4,
	Score: sointu.Score{
		RowsPerPattern: 16,
		Length:         1,
		Tracks: []sointu.Track{
			{NumVoices: 1, Order: sointu.Order{0}, Patterns: []sointu.Pattern{{72, 0}}},
		},
	},
	Patch: sointu.Patch{defaultInstrument,
		{Name: "Global", NumVoices: 1, Units: []sointu.Unit{
			defaultUnits["in"],
			{Type: "delay",
				Parameters: map[string]int{"damp": 64, "dry": 128, "feedback": 125, "notetracking": 0, "pregain": 40, "stereo": 1},
				VarArgs: []int{1116, 1188, 1276, 1356, 1422, 1492, 1556, 1618,
					1140, 1212, 1300, 1380, 1446, 1516, 1580, 1642,
				}},
			{Type: "out", Parameters: map[string]int{"stereo": 1, "gain": 128}},
		}}},
}

var reverbs = []delayPreset{
	{"stereo", 1, []int{1116, 1188, 1276, 1356, 1422, 1492, 1556, 1618,
		1140, 1212, 1300, 1380, 1446, 1516, 1580, 1642,
	}},
	{"left", 0, []int{1116, 1188, 1276, 1356, 1422, 1492, 1556, 1618}},
	{"right", 0, []int{1140, 1212, 1300, 1380, 1446, 1516, 1580, 1642}},
}

type delayPreset struct {
	name    string
	stereo  int
	varArgs []int
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

func (p Presets) Len() int { return len(p.Presets) }
func (p Presets) Less(i, j int) bool {
	if p.Presets[i].Instr.Name == p.Presets[j].Instr.Name {
		return p.Presets[i].User && !p.Presets[j].User
	}
	return p.Presets[i].Instr.Name < p.Presets[j].Instr.Name
}
func (p Presets) Swap(i, j int) { p.Presets[i], p.Presets[j] = p.Presets[j], p.Presets[i] }
