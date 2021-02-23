package gioui

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/sqweek/dialog"
	"github.com/vsariola/sointu"
)

func (t *Tracker) LoadSongFile() {
	filename, err := dialog.File().Filter("Sointu YAML song", "yml").Filter("Sointu JSON song", "json").Title("Load song").Load()
	if err != nil {
		return
	}
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	var song sointu.Song
	if errJSON := json.Unmarshal(bytes, &song); errJSON != nil {
		if errYaml := yaml.Unmarshal(bytes, &song); errYaml != nil {
			return
		}
	}
	t.SetSong(song)
}

func (t *Tracker) SaveSongFile() {
	filename, err := dialog.File().Filter("Sointu YAML song", "yml").Filter("Sointu JSON song", "json").Title("Save song").Save()
	if err != nil {
		return
	}
	var extension = filepath.Ext(filename)
	var contents []byte
	if extension == "json" {
		contents, err = json.Marshal(t.Song())
	} else {
		contents, err = yaml.Marshal(t.Song())
	}
	if err != nil {
		return
	}
	if extension == "" {
		filename = filename + ".yml"
	}
	ioutil.WriteFile(filename, contents, 0644)
}
