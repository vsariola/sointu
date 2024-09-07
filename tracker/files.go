package tracker

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
)

func (m *Model) ReadSong(r io.ReadCloser) {
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
			m.Alerts().Add(fmt.Sprintf("Error unmarshaling a song file: %v / %v", errYaml, errJSON), Error)
			return
		}
	}
	f := m.change("LoadSong", SongChange, MajorChange)
	m.d.Song = song
	if f, ok := r.(*os.File); ok {
		m.d.FilePath = f.Name()
		// when the song is loaded from a file, we are quite confident that the file is persisted and thus
		// we can close sointu without worrying about losing changes
		m.d.ChangedSinceSave = false
	}
	f()
	m.completeAction(false)
}

func (m *Model) WriteSong(w io.WriteCloser) {
	path := ""
	var extension = filepath.Ext(path)
	var contents []byte
	var err error
	if extension == ".json" {
		contents, err = json.Marshal(m.d.Song)
	} else {
		contents, err = yaml.Marshal(m.d.Song)
	}
	if err != nil {
		m.Alerts().Add(fmt.Sprintf("Error marshaling a song file: %v", err), Error)
		return
	}
	if _, err := w.Write(contents); err != nil {
		m.Alerts().Add(fmt.Sprintf("Error writing to file: %v", err), Error)
		return
	}
	if f, ok := w.(*os.File); ok {
		path = f.Name()
		// when the song is saved to a file, we are quite confident that the file is persisted and thus
		// we can close sointu without worrying about losing changes
		m.d.ChangedSinceSave = false
	}
	if err := w.Close(); err != nil {
		m.Alerts().Add(fmt.Sprintf("Error rendering the song during export: %v", err), Error)
		return
	}
	m.d.FilePath = path
	m.completeAction(false)
}

func (m *Model) WriteWav(w io.WriteCloser, pcm16 bool, execChan chan<- func()) {
	m.dialog = NoDialog
	song := m.d.Song.Copy()
	go func() {
		b := make([]byte, 32+2)
		rand.Read(b)
		name := fmt.Sprintf("%x", b)[2 : 32+2]
		data, err := sointu.Play(m.synther, song, func(p float32) {
			execChan <- func() {
				m.Alerts().AddNamed(name, fmt.Sprintf("Exporting song: %.0f%%", p*100), Info)
			}
		}) // render the song to calculate its length
		if err != nil {
			execChan <- func() {
				m.Alerts().Add(fmt.Sprintf("Error rendering the song during export: %v", err), Error)
			}
			return
		}
		buffer, err := data.Wav(pcm16)
		if err != nil {
			execChan <- func() {
				m.Alerts().Add(fmt.Sprintf("Error converting to .wav: %v", err), Error)
			}
			return
		}
		w.Write(buffer)
		w.Close()
	}()
}

func (m *Model) SaveInstrument(w io.WriteCloser) bool {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		m.Alerts().Add("No instrument selected", Error)
		return false
	}
	path := ""
	if f, ok := w.(*os.File); ok {
		path = f.Name()
	}
	var extension = filepath.Ext(path)
	var contents []byte
	var err error
	if extension == ".json" {
		contents, err = json.Marshal(m.d.Song.Patch[m.d.InstrIndex])
	} else {
		contents, err = yaml.Marshal(m.d.Song.Patch[m.d.InstrIndex])
	}
	if err != nil {
		m.Alerts().Add(fmt.Sprintf("Error marshaling a ínstrument file: %v", err), Error)
		return false
	}
	w.Write(contents)
	w.Close()
	return true
}

func (m *Model) LoadInstrument(r io.ReadCloser) bool {
	if m.d.InstrIndex < 0 {
		return false
	}
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
		defer m.change("LoadInstrument", PatchChange, MajorChange)()
		m.d.Song.Patch = patch
		return true
	}
	instrument, err4ki = sointu.Read4klangInstrument(bytes.NewReader(b))
	if err4ki == nil {
		goto success
	}
	m.Alerts().Add(fmt.Sprintf("Error unmarshaling an instrument file: %v / %v / %v / %v", errYaml, errJSON, err4ki, err4kp), Error)
	return false
success:
	if f, ok := r.(*os.File); ok {
		filename := f.Name()
		// the 4klang instrument names are junk, replace them with the filename without extension
		instrument.Name = filepath.Base(filename[:len(filename)-len(filepath.Ext(filename))])
	}
	defer m.change("LoadInstrument", PatchChange, MajorChange)()
	for len(m.d.Song.Patch) <= m.d.InstrIndex {
		m.d.Song.Patch = append(m.d.Song.Patch, defaultInstrument.Copy())
	}
	m.d.Song.Patch[m.d.InstrIndex] = sointu.Instrument{}
	numVoices := m.d.Song.Patch.NumVoices()
	if numVoices >= vm.MAX_VOICES {
		// this really shouldn't happen, as we have already cleared the
		// instrument and assuming each instrument has at least 1 voice, it
		// should have freed up some voices
		m.Alerts().Add(fmt.Sprintf("The patch has already %d voices", vm.MAX_VOICES), Error)
		return false
	}
	instrument.NumVoices = clamp(instrument.NumVoices, 1, 32-numVoices)
	m.assignUnitIDs(instrument.Units)
	m.d.Song.Patch[m.d.InstrIndex] = instrument
	if m.d.Song.Patch[m.d.InstrIndex].Comment != "" {
		m.commentExpanded = true
	}
	return true
}
