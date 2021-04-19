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

	"github.com/vsariola/sointu"
)

func (t *Tracker) OpenSongFile(forced bool) {
	if !forced && t.ChangedSinceSave() {
		t.ConfirmSongActionType = ConfirmLoad
		t.ConfirmSongDialog.Visible = true
		return
	}
	if p := t.FilePath(); p != "" {
		d, _ := filepath.Split(p)
		d = filepath.Clean(d)
		t.OpenSongDialog.Directory.SetText(d)
		t.OpenSongDialog.FileName.SetText("")
	}
	t.OpenSongDialog.Visible = true
}

func (t *Tracker) SaveSongFile() bool {
	if p := t.FilePath(); p != "" {
		return t.saveSong(p)
	}
	t.SaveSongAsFile()
	return false
}

func (t *Tracker) SaveSongAsFile() {
	t.SaveSongDialog.Visible = true
	if p := t.FilePath(); p != "" {
		d, f := filepath.Split(p)
		d = filepath.Clean(d)
		t.SaveSongDialog.Directory.SetText(d)
		t.SaveSongDialog.FileName.SetText(f)
	}
}

func (t *Tracker) ExportWav() {
	t.ExportWavDialog.Visible = true
	if p := t.FilePath(); p != "" {
		d, _ := filepath.Split(p)
		d = filepath.Clean(d)
		t.ExportWavDialog.Directory.SetText(d)
	}
}

func (t *Tracker) LoadInstrument() {
	t.OpenInstrumentDialog.Visible = true
}

func (t *Tracker) SaveInstrument() {
	t.SaveInstrumentDialog.Visible = true
}

func (t *Tracker) loadSong(filename string) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	var song sointu.Song
	if errJSON := json.Unmarshal(bytes, &song); errJSON != nil {
		if errYaml := yaml.Unmarshal(bytes, &song); errYaml != nil {
			t.Alert.Update(fmt.Sprintf("Error unmarshaling a song file: %v / %v", errYaml, errJSON), Error, time.Second*3)
			return
		}
	}
	if song.Score.Length <= 0 || len(song.Score.Tracks) == 0 || len(song.Patch) == 0 {
		t.Alert.Update("The song file is malformed", Error, time.Second*3)
		return
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
	if extension == ".json" {
		contents, err = json.Marshal(t.Song())
	} else {
		contents, err = yaml.Marshal(t.Song())
	}
	if err != nil {
		t.Alert.Update(fmt.Sprintf("Error marshaling a song file: %v", err), Error, time.Second*3)
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

func (t *Tracker) exportWav(filename string, pcm16 bool) {
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

func (t *Tracker) saveInstrument(filename string) bool {
	var extension = filepath.Ext(filename)
	var contents []byte
	var err error
	if extension == ".json" {
		contents, err = json.Marshal(t.Instrument())
	} else {
		contents, err = yaml.Marshal(t.Instrument())
	}
	if err != nil {
		t.Alert.Update(fmt.Sprintf("Error marshaling a ínstrument file: %v", err), Error, time.Second*3)
		return false
	}
	if extension == "" {
		filename = filename + ".yml"
	}
	ioutil.WriteFile(filename, contents, 0644)
	return true
}

func (t *Tracker) loadInstrument(filename string) bool {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return false
	}
	var instrument sointu.Instrument
	if errJSON := json.Unmarshal(bytes, &instrument); errJSON != nil {
		if errYaml := yaml.Unmarshal(bytes, &instrument); errYaml != nil {
			t.Alert.Update(fmt.Sprintf("Error unmarshaling an instrument file: %v / %v", errYaml, errJSON), Error, time.Second*3)
			return false
		}
	}
	if len(instrument.Units) == 0 {
		t.Alert.Update("The instrument file is malformed", Error, time.Second*3)
		return false
	}
	t.SetInstrument(instrument)
	if t.Instrument().Comment != "" {
		t.InstrumentExpanded = true
	}
	return true
}
