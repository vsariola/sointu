//go:build !js
// +build !js

package gioui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/vsariola/sointu"
)

func (t *Tracker) OpenSongFile(forced bool) {
	if !forced && t.ChangedSinceSave() {
		t.ConfirmSongActionType = ConfirmLoad
		t.ConfirmSongDialog.Visible = true
		return
	}
	reader, err := t.Explorer.ChooseFile(".yml", ".json")
	if err != nil {
		return
	}
	t.loadSong(reader)
}

func (t *Tracker) SaveSongFile() bool {
	if p := t.FilePath(); p != "" {
		if f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE, 0644); err == nil {
			return t.saveSong(f)
		}
	}
	t.SaveSongAsFile()
	return false
}

func (t *Tracker) SaveSongAsFile() {
	p := t.FilePath()
	if p == "" {
		p = "song.yml"
	}
	writer, err := t.Explorer.CreateFile(p)
	if err != nil {
		return
	}
	t.saveSong(writer)
}

func (t *Tracker) ExportWav(pcm16 bool) {
	filename := "song.wav"
	if p := t.FilePath(); p != "" {
		filename = p[:len(p)-len(filepath.Ext(p))] + ".wav"
	}
	writer, err := t.Explorer.CreateFile(filename)
	if err != nil {
		return
	}
	t.exportWav(writer, pcm16)
}

func (t *Tracker) LoadInstrument() {
	reader, err := t.Explorer.ChooseFile(".yml", ".json", ".4ki", ".4kp")
	if err != nil {
		return
	}
	t.loadInstrument(reader)
}

func (t *Tracker) SaveInstrument() {
	writer, err := t.Explorer.CreateFile(t.Instrument().Name + ".yml")
	if err != nil {
		return
	}
	t.saveInstrument(writer)
}

func (t *Tracker) loadSong(r io.ReadCloser) {
	b, err := io.ReadAll(r)
	if err != nil {
		return
	}
	err = r.Close()
	if err != nil {
		return
	}
	var song sointu.Song
	if errJSON := json.Unmarshal(b, &song); errJSON != nil {
		if errYaml := yaml.Unmarshal(b, &song); errYaml != nil {
			t.Alert.Update(fmt.Sprintf("Error unmarshaling a song file: %v / %v", errYaml, errJSON), Error, time.Second*3)
		}
	}
	if song.Score.Length <= 0 || len(song.Score.Tracks) == 0 || len(song.Patch) == 0 {
		t.Alert.Update("The song file is malformed", Error, time.Second*3)
		return
	}
	t.SetSong(song)
	path := ""
	if f, ok := r.(*os.File); ok {
		path = f.Name()
	}
	t.SetFilePath(path)
	t.ClearUndoHistory()
	t.SetChangedSinceSave(false)
}

func (t *Tracker) saveSong(w io.WriteCloser) bool {
	path := ""
	if f, ok := w.(*os.File); ok {
		path = f.Name()
	}
	var extension = filepath.Ext(path)
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
	if _, err := w.Write(contents); err != nil {
		t.Alert.Update(fmt.Sprintf("Error writing to file: %v", err), Error, time.Second*3)
		return false
	}
	if err := w.Close(); err != nil {
		t.Alert.Update(fmt.Sprintf("Error closing file: %v", err), Error, time.Second*3)
		return false
	}
	t.SetFilePath(path)
	t.SetChangedSinceSave(false)
	return true
}

func (t *Tracker) exportWav(w io.WriteCloser, pcm16 bool) {
	data, err := sointu.Play(t.synthService, t.Song(), true) // render the song to calculate its length
	if err != nil {
		t.Alert.Update(fmt.Sprintf("Error rendering the song during export: %v", err), Error, time.Second*3)
		return
	}
	buffer, err := sointu.Wav(data, pcm16)
	if err != nil {
		t.Alert.Update(fmt.Sprintf("Error converting to .wav: %v", err), Error, time.Second*3)
		return
	}
	w.Write(buffer)
	w.Close()
}

func (t *Tracker) saveInstrument(w io.WriteCloser) bool {
	path := ""
	if f, ok := w.(*os.File); ok {
		path = f.Name()
	}
	var extension = filepath.Ext(path)
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
	w.Write(contents)
	w.Close()
	return true
}

func (t *Tracker) loadInstrument(r io.ReadCloser) bool {
	b, err := io.ReadAll(r)
	if err != nil {
		return false
	}
	var instrument sointu.Instrument
	var errJSON, errYaml, err4ki, err4kp error
	var patch sointu.Patch
	errJSON = json.Unmarshal(b, &instrument)
	if errJSON == nil {
		goto success
	}
	errYaml = yaml.Unmarshal(b, &instrument)
	if errYaml == nil {
		goto success
	}
	patch, err4kp = sointu.Read4klangPatch(bytes.NewReader(b))
	if err4kp == nil {
		song := t.Song()
		song.Score = t.Song().Score.Copy()
		song.Patch = patch
		t.SetSong(song)
		return true
	}
	instrument, err4ki = sointu.Read4klangInstrument(bytes.NewReader(b))
	if err4ki == nil {
		goto success
	}
	t.Alert.Update(fmt.Sprintf("Error unmarshaling an instrument file: %v / %v / %v / %v", errYaml, errJSON, err4ki, err4kp), Error, time.Second*3)
	return false
success:
	if f, ok := r.(*os.File); ok {
		filename := f.Name()
		// the 4klang instrument names are junk, replace them with the filename without extension
		instrument.Name = filepath.Base(filename[:len(filename)-len(filepath.Ext(filename))])
	}
	if len(instrument.Units) == 0 {
		t.Alert.Update("The instrument file is malformed", Error, time.Second*3)
		return false
	}
	t.SetInstrument(instrument)
	if t.Instrument().Comment != "" {
		t.InstrumentEditor.ExpandComment()
	}
	return true
}
