// +build !js

package gioui

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

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

func (t *Tracker) exportWav(pcm16 bool) {
	filename, err := dialog.File().Filter(".wav file", "wav").Title("Export .wav").Save()
	if err != nil {
		return
	}
	var extension = filepath.Ext(filename)
	if extension == "" {
		filename = filename + ".wav"
	}
	synth, err := t.synthService.Compile(t.Song().Patch)
	if err != nil {
		t.Alert.Update(fmt.Sprintf("Error compiling the patch during export: %v", err), Error, time.Second*3)
		return
	}
	for i := 0; i < 32; i++ {
		synth.Release(i)
	}
	data, _, err := sointu.Play(synth, t.Song()) // render the song to calculate its length
	if err != nil {
		t.Alert.Update(fmt.Sprintf("Error rendering the song during export: %v", err), Error, time.Second*3)
		return
	}
	buffer, err := sointu.Wav(data, pcm16)
	if err != nil {
		t.Alert.Update(fmt.Sprintf("Error converting to .wav: %v", err), Error, time.Second*3)
		return
	}
	ioutil.WriteFile(filename, buffer, 0644)
}
