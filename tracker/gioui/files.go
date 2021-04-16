// +build !js

package gioui

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"gioui.org/app"
	"gopkg.in/yaml.v3"

	"github.com/sqweek/dialog"
	"github.com/vsariola/sointu"
)

func (t *Tracker) LoadSongFile() {
	if t.ChangedSinceSave() {
		t.ConfirmSongActionType = ConfirmLoad
		t.ConfirmSongDialog.Visible = true
		return
	}
	t.loadSong()
}

func (t *Tracker) SaveSongFile() bool {
	if p := t.FilePath(); p != "" {
		return t.saveSong(p)
	}
	return t.SaveSongAsFile()
}

func (t *Tracker) SaveSongAsFile() bool {
	filename, err := dialog.File().Filter("Sointu YAML song", "yml").Filter("Sointu JSON song", "json").Title("Save song").Save()
	if err != nil {
		return false
	}
	return t.saveSong(filename)
}

func (t *Tracker) loadSong() {
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
	t.SetFilePath(filename)
	t.window.Option(app.Title(fmt.Sprintf("Sointu Tracker - %v", filename)))
	t.ClearUndoHistory()
	t.SetChangedSinceSave(false)
}

func (t *Tracker) saveSong(filename string) bool {
	var extension = filepath.Ext(filename)
	var contents []byte
	var err error
	if extension == "json" {
		contents, err = json.Marshal(t.Song())
	} else {
		contents, err = yaml.Marshal(t.Song())
	}
	if err != nil {
		return false
	}
	if extension == "" {
		filename = filename + ".yml"
	}
	ioutil.WriteFile(filename, contents, 0644)
	t.SetFilePath(filename)
	t.window.Option(app.Title(fmt.Sprintf("Sointu Tracker - %v", filename)))
	t.SetChangedSinceSave(false)
	return true
}
