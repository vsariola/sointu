package tracker

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
	"gopkg.in/yaml.v2"
)

//go:generate go run generate/gmdls_entries.go
//go:generate go run generate/clean_presets.go

type (
	// GmDlsEntry is a single sample entry from the gm.dls file
	GmDlsEntry struct {
		Start              int    // sample start offset in words
		LoopStart          int    // loop start offset in words
		LoopLength         int    // loop length in words
		SuggestedTranspose int    // suggested transpose in semitones, so that all samples play at same pitch
		Name               string // sample Name
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

	derivedPresetSearch struct {
		dirIndex int
		noGmDls  bool
		kind     PresetKindEnum
		dirs     []string
		results  []int
	}

	PresetKindEnum int
)

const (
	BuiltinPresets PresetKindEnum = -1
	AllPresets     PresetKindEnum = 0
	UserPresets    PresetKindEnum = 1
)

func (m *Model) updateDerivedPresetSearch() {
	// parse filters from the search string. in: dir, gmdls: yes/no, kind: builtin/user/all
	search := strings.TrimSpace(m.d.PresetSearchString)
	lower := strings.ToLower(search)
	parts := strings.Fields(lower)
	// parse parts to see if they contain :
	m.derived.presetSearch.dirs = []string{"All"}
	m.derived.presetSearch.noGmDls = false
	m.derived.presetSearch.kind = AllPresets
	for _, part := range parts {
		if strings.HasPrefix(part, "d:") && len(part) > 3 {
			dir := strings.TrimSpace(part[3:])
			ind := slices.IndexFunc(m.derived.presetSearch.dirs, func(c string) bool { return c == dir })
			m.derived.presetSearch.dirIndex = max(ind, 0)
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
		}
	}
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

func (m *Model) PresetKind() Int { return MakeInt((*PresetKind)(m)) }
func (m *PresetKind) Value() int { return int(m.derived.presetSearch.kind) }
func (m *PresetKind) SetValue(val int) bool {
	if int(m.derived.presetSearch.kind) == val {
		return false
	}
	m.d.PresetSearchString = removeFilters(m.d.PresetSearchString, "kind:")
	switch PresetKindEnum(val) {
	case BuiltinPresets:
		m.d.PresetSearchString = "kind:builtin " + m.d.PresetSearchString
	case UserPresets:
		m.d.PresetSearchString = "kind:user " + m.d.PresetSearchString
	}
	(*Model)(m).updateDerivedPresetSearch()
	return true
}
func (m *PresetKind) Enabled() bool   { return true }
func (m *PresetKind) Range() IntRange { return IntRange{Min: -1, Max: 1} }

func (m *Model) ClearPresetSearch() Action { return MakeAction((*ClearPresetSearch)(m)) }
func (m *ClearPresetSearch) Enabled() bool { return len(m.d.PresetSearchString) > 0 }
func (m *ClearPresetSearch) Do() {
	m.d.PresetSearchString = ""
	(*Model)(m).updateDerivedPresetSearch()
}

func (m *Model) PresetDirList() *PresetDirList { return (*PresetDirList)(m) }
func (v *PresetDirList) List() List            { return List{v} }
func (m *PresetDirList) Count() int            { return len(m.derived.presetSearch.dirs) }
func (m *PresetDirList) Selected() int         { return m.derived.presetSearch.dirIndex }
func (m *PresetDirList) Selected2() int        { return m.derived.presetSearch.dirIndex }
func (m *PresetDirList) SetSelected2(i int)    {}
func (m *PresetDirList) Value(i int) string {
	if i < 0 || i >= len(m.derived.presetSearch.dirs) {
		return ""
	}
	return m.derived.presetSearch.dirs[i]
}
func (m *PresetDirList) SetSelected(i int) {
	i = min(max(i, 0), len(m.derived.presetSearch.dirs)-1)
	if i < 0 || i >= len(m.derived.presetSearch.dirs) {
		return
	}
	m.d.PresetSearchString = removeFilters(m.d.PresetSearchString, "d:")
	if i > 0 {
		m.d.PresetSearchString = "d: " + m.derived.presetSearch.dirs[i] + " " + m.d.PresetSearchString
	}
	(*Model)(m).updateDerivedPresetSearch()
}

func removeFilters(str string, prefix string) string {
	parts := strings.Fields(str)
	newParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if !strings.HasPrefix(strings.ToLower(part), prefix) {
			newParts = append(newParts, part)
		}
	}
	return strings.Join(newParts, " ")
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

func (m *Model) IterateInstrumentPresets(yield InstrumentPresetYieldFunc) {
	for index, instr := range instrumentPresets {
		if !yield(index, instr.Name) {
			return
		}
	}
}

func NumPresets() int {
	return len(instrumentPresets)
}

// LoadPreset loads a preset from the list of instrument presets. The index
// should be within the range of 0 to NumPresets()-1.

func (m *Model) LoadPreset(index int) Action {
	return MakeEnabledAction(LoadPreset{Index: index, Model: m})
}
func (m LoadPreset) Do() {
	defer m.change("LoadPreset", PatchChange, MajorChange)()
	if m.d.InstrIndex < 0 {
		m.d.InstrIndex = 0
	}
	m.d.InstrIndex2 = m.d.InstrIndex
	for m.d.InstrIndex >= len(m.d.Song.Patch) {
		m.d.Song.Patch = append(m.d.Song.Patch, defaultInstrument.Copy())
	}
	newInstr := instrumentPresets[m.Index].Copy()
	newInstr.NumVoices = clamp(m.d.Song.Patch[m.d.InstrIndex].NumVoices, 1, vm.MAX_VOICES)
	m.Model.assignUnitIDs(newInstr.Units)
	m.d.Song.Patch[m.d.InstrIndex] = newInstr
}

type instrumentPresetsSlice []sointu.Instrument

//go:embed presets/*
var instrumentPresetFS embed.FS
var instrumentPresets instrumentPresetsSlice

func init() {
	fs.WalkDir(instrumentPresetFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(instrumentPresetFS, path)
		if err != nil {
			return nil
		}
		var instr sointu.Instrument
		if yaml.UnmarshalStrict(data, &instr) == nil {
			noExt := path[:len(path)-len(filepath.Ext(path))]
			splitted := splitPath(noExt)
			splitted = splitted[1:] // remove "presets" from the path
			instr.Name = strings.Join(splitted, " ")
			instrumentPresets = append(instrumentPresets, instr)
		}
		return nil
	})
	if configDir, err := os.UserConfigDir(); err == nil {
		userPresets := filepath.Join(configDir, "sointu", "presets")
		filepath.WalkDir(userPresets, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			var instr sointu.Instrument
			if yaml.Unmarshal(data, &instr) == nil {
				if len(userPresets)+1 > len(path) {
					return nil
				}
				subPath := path[len(userPresets)+1:]
				noExt := subPath[:len(subPath)-len(filepath.Ext(subPath))]
				splitted := splitPath(noExt)
				instr.Name = strings.Join(splitted, " ")
				instrumentPresets = append(instrumentPresets, instr)
			}
			return nil
		})
	}
	sort.Sort(instrumentPresets)
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

func (p instrumentPresetsSlice) Len() int           { return len(p) }
func (p instrumentPresetsSlice) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p instrumentPresetsSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
