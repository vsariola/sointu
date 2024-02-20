package tracker

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
	"gopkg.in/yaml.v3"
)

//go:generate go run generate/main.go

type (
	// GmDlsEntry is a single sample entry from the gm.dls file
	GmDlsEntry struct {
		Start              int    // sample start offset in words
		LoopStart          int    // loop start offset in words
		LoopLength         int    // loop length in words
		SuggestedTranspose int    // suggested transpose in semitones, so that all samples play at same pitch
		Name               string // sample name
	}

	InstrumentPresetYieldFunc func(index int, item string) (ok bool)
	LoadPreset                struct {
		Index int
		*Model
	}
)

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
	"filter":     {Type: "filter", Parameters: map[string]int{"stereo": 0, "frequency": 64, "resonance": 64, "lowpass": 1, "bandpass": 0, "highpass": 0, "negbandpass": 0, "neghighpass": 0}},
	"out":        {Type: "out", Parameters: map[string]int{"stereo": 1, "gain": 64}},
	"outaux":     {Type: "outaux", Parameters: map[string]int{"stereo": 1, "outgain": 64, "auxgain": 64}},
	"aux":        {Type: "aux", Parameters: map[string]int{"stereo": 1, "gain": 64, "channel": 2}},
	"delay": {Type: "delay",
		Parameters: map[string]int{"damp": 0, "dry": 128, "feedback": 96, "notetracking": 2, "pregain": 40, "stereo": 0},
		VarArgs:    []int{48}},
	"in":         {Type: "in", Parameters: map[string]int{"stereo": 1, "channel": 2}},
	"speed":      {Type: "speed", Parameters: map[string]int{}},
	"compressor": {Type: "compressor", Parameters: map[string]int{"stereo": 0, "attack": 64, "release": 64, "invgain": 64, "threshold": 64, "ratio": 64}},
	"send":       {Type: "send", Parameters: map[string]int{"stereo": 0, "amount": 128, "voice": 0, "unit": 0, "port": 0, "sendpop": 1}},
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

func (m *Model) LoadPreset(index int) Action {
	return Action{do: func() {
		defer m.change("LoadPreset", PatchChange, MajorChange)()
		if m.d.InstrIndex < 0 {
			m.d.InstrIndex = 0
		}
		m.d.InstrIndex2 = m.d.InstrIndex
		for m.d.InstrIndex >= len(m.d.Song.Patch) {
			m.d.Song.Patch = append(m.d.Song.Patch, defaultInstrument.Copy())
		}
		m.d.Song.Patch[m.d.InstrIndex] = instrumentPresets[index].Copy()
	}, allowed: func() bool {
		return true
	}}
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
		if yaml.Unmarshal(data, &instr) == nil {
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
				instrumentPresets = append(instrumentPresets, instr)
			}
			return nil
		})
	}
	sort.Sort(instrumentPresets)
}

func (p instrumentPresetsSlice) Len() int           { return len(p) }
func (p instrumentPresetsSlice) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p instrumentPresetsSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
