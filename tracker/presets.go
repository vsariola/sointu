package tracker

import (
	"embed"
	"io/fs"
	"sort"

	"github.com/vsariola/sointu"
	"gopkg.in/yaml.v3"
)

type PresetList []sointu.Instrument

//go:embed presets/*
var presetFS embed.FS

var Presets PresetList

func init() {
	fs.WalkDir(presetFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(presetFS, path)
		if err != nil {
			return nil
		}
		var instr sointu.Instrument
		if yaml.Unmarshal(data, &instr) != nil {
			return nil
		}
		Presets = append(Presets, instr)
		return nil
	})
	sort.Sort(Presets)
}

func (p PresetList) Len() int {
	return len(p)
}

func (p PresetList) Less(i, j int) bool {
	return p[i].Name < p[j].Name
}

func (p PresetList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
