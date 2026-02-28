//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/tracker"
	"gopkg.in/yaml.v3"
)

func main() {
	filepath.WalkDir("presets", func(path string, d fs.DirEntry, err error) error {
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
		if yaml.Unmarshal(data, &instr) != nil {
			fmt.Fprintf(os.Stderr, "could not unmarshal the preset file %v: %v\n", path, err)
			return nil
		}
		tracker.RemoveUnusedUnitParameters(&instr) // remove invalid parameters
		instr2 := sointu.Instrument{               // keep only the relevant fields
			Comment: instr.Comment,
			Units:   instr.Units,
		}
		outData, err := yaml.Marshal(instr2)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not marshal the preset file %v: %v\n", path, err)
			return nil
		}
		if err := os.WriteFile(path, outData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "could not write the preset file %v: %v\n", path, err)
			return nil
		}
		return nil
	})
}
