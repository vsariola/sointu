package tracker

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/vsariola/sointu"
	"gopkg.in/yaml.v3"
)

// Song returns the Song view of the model, containing methods to manipulate the
// song.
func (m *Model) Song() *SongModel { return (*SongModel)(m) }

type SongModel Model

// FilePath returns a String representing the file path of the current song.
func (m *SongModel) FilePath() String { return MakeString((*songFilePath)(m)) }

type songFilePath SongModel

func (v *songFilePath) Value() string              { return v.d.FilePath }
func (v *songFilePath) SetValue(value string) bool { v.d.FilePath = value; return true }

// BPM returns an Int representing the BPM of the current song.
func (m *SongModel) BPM() Int { return MakeInt((*songBpm)(m)) }

type songBpm SongModel

func (v *songBpm) Value() int { return v.d.Song.BPM }
func (v *songBpm) SetValue(value int) bool {
	defer (*Model)(v).change("BPMInt", SongChange, MinorChange)()
	v.d.Song.BPM = value
	return true
}
func (v *songBpm) Range() RangeInclusive { return RangeInclusive{1, 999} }

// RowsPerPattern returns an Int representing the number of rows per pattern of
// the current song.
func (m *SongModel) RowsPerPattern() Int { return MakeInt((*songRowsPerPattern)(m)) }

type songRowsPerPattern SongModel

func (v *songRowsPerPattern) Value() int { return v.d.Song.Score.RowsPerPattern }
func (v *songRowsPerPattern) SetValue(value int) bool {
	defer (*Model)(v).change("RowsPerPatternInt", SongChange, MinorChange)()
	v.d.Song.Score.RowsPerPattern = value
	return true
}
func (v *songRowsPerPattern) Range() RangeInclusive { return RangeInclusive{1, 256} }

// Length returns an Int representing the length of the current song, in number
// of order rows.
func (m *SongModel) Length() Int { return MakeInt((*songLength)(m)) }

type songLength SongModel

func (v *songLength) Value() int { return v.d.Song.Score.Length }
func (v *songLength) SetValue(value int) bool {
	defer (*Model)(v).change("SongLengthInt", SongChange, MinorChange)()
	v.d.Song.Score.Length = value
	return true
}
func (v *songLength) Range() RangeInclusive { return RangeInclusive{1, math.MaxInt32} }

// RowsPerBeat returns an Int representing the number of rows per beat of the
// current song.
func (m *SongModel) RowsPerBeat() Int { return MakeInt((*songRowsPerBeat)(m)) }

type songRowsPerBeat SongModel

func (v *songRowsPerBeat) Value() int { return v.d.Song.RowsPerBeat }
func (v *songRowsPerBeat) SetValue(value int) bool {
	defer (*Model)(v).change("RowsPerBeatInt", SongChange, MinorChange)()
	v.d.Song.RowsPerBeat = value
	return true
}
func (v *songRowsPerBeat) Range() RangeInclusive { return RangeInclusive{1, 32} }

// Save returns an Action to initiate saving the current song to disk.
func (m *SongModel) Save() Action { return MakeAction((*saveSong)(m)) }

type saveSong Model

func (m *saveSong) Do() {
	if m.d.FilePath == "" {
		switch m.dialog {
		case NoDialog:
			m.dialog = SaveAsExplorer
		case NewSongChanges:
			m.dialog = NewSongSaveExplorer
		case OpenSongChanges:
			m.dialog = OpenSongSaveExplorer
		case QuitChanges:
			m.dialog = QuitSaveExplorer
		}
		return
	}
	f, err := os.Create(m.d.FilePath)
	if err != nil {
		(*Model)(m).Alerts().Add("Error creating file: "+err.Error(), Error)
		return
	}
	(*Model)(m).Song().Write(f)
	m.d.ChangedSinceSave = false
}

// New returns an Action to create a new song.
func (m *SongModel) New() Action { return MakeAction((*newSong)(m)) }

type newSong SongModel

func (m *newSong) Do() {
	m.dialog = NewSongChanges
	(*SongModel)(m).completeAction(true)
}

func (m *SongModel) completeAction(checkSave bool) {
	if checkSave && m.d.ChangedSinceSave {
		return
	}
	switch m.dialog {
	case NewSongChanges, NewSongSaveExplorer:
		c := (*Model)(m).change("NewSong", SongChange, MajorChange)
		m.reset()
		(*Model)(m).setLoop(Loop{})
		c()
		m.d.ChangedSinceSave = false
		m.dialog = NoDialog
	case OpenSongChanges, OpenSongSaveExplorer:
		m.dialog = OpenSongOpenExplorer
	case QuitChanges, QuitSaveExplorer:
		m.quitted = true
		m.dialog = NoDialog
	default:
		m.dialog = NoDialog
	}
}

func (m *SongModel) reset() {
	m.d.Song = defaultSong.Copy()
	for _, instr := range m.d.Song.Patch {
		(*Model)(m).assignUnitIDs(instr.Units)
	}
	m.d.FilePath = ""
	m.d.ChangedSinceSave = false
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
	"belleq":     {Type: "belleq", Parameters: map[string]int{"stereo": 0, "frequency": 64, "bandwidth": 64, "gain": 64}},
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

// Open returns an Action to open a song from the disk.
func (m *SongModel) Open() Action { return MakeAction((*openSong)(m)) }

type openSong SongModel

func (m *openSong) Do() {
	m.dialog = OpenSongChanges
	(*SongModel)(m).completeAction(true)
}

// SaveAs returns an Action to save the song to the disk with a new filename.
func (m *SongModel) SaveAs() Action { return MakeAction((*saveSongAs)(m)) }

type saveSongAs SongModel

func (m *saveSongAs) Do() { m.dialog = SaveAsExplorer }

// Discard returns an Action to discard the current changes to the song when
// opening a song from disk or creating a new one.
func (m *SongModel) Discard() Action { return MakeAction((*discardSong)(m)) }

type discardSong SongModel

func (m *discardSong) Do() { (*SongModel)(m).completeAction(false) }

// Read the song from a given io.ReadCloser, trying parsing it both as json and
// yaml.
func (m *SongModel) Read(r io.ReadCloser) {
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
			(*Model)(m).Alerts().Add(fmt.Sprintf("Error unmarshaling a song file: %v / %v", errYaml, errJSON), Error)
			return
		}
	}
	f := (*Model)(m).change("LoadSong", SongChange, MajorChange)
	m.d.Song = song
	if f, ok := r.(*os.File); ok {
		m.d.FilePath = f.Name()
		// when the song is loaded from a file, we are quite confident that the file is persisted and thus
		// we can close sointu without worrying about losing changes
		m.d.ChangedSinceSave = false
	}
	f()
	(*SongModel)(m).completeAction(false)
}

// Save the song to a given io.ReadCloser. If the given argument is an os.File
// and has the file extension ".json", the song is marshaled as json; otherwise,
// it's marshaled as yaml.
func (m *SongModel) Write(w io.WriteCloser) {
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
		(*Model)(m).Alerts().Add(fmt.Sprintf("Error marshaling a song file: %v", err), Error)
		return
	}
	if _, err := w.Write(contents); err != nil {
		(*Model)(m).Alerts().Add(fmt.Sprintf("Error writing to file: %v", err), Error)
		return
	}
	if f, ok := w.(*os.File); ok {
		path = f.Name()
		// when the song is saved to a file, we are quite confident that the file is persisted and thus
		// we can close sointu without worrying about losing changes
		m.d.ChangedSinceSave = false
	}
	if err := w.Close(); err != nil {
		(*Model)(m).Alerts().Add(fmt.Sprintf("Error closing the song file: %v", err), Error)
		return
	}
	m.d.FilePath = path
	(*SongModel)(m).completeAction(false)
}

// Export returns an Action to show the wav export dialog.
func (m *SongModel) Export() Action { return MakeAction((*exportAction)(m)) }

type exportAction SongModel

func (m *exportAction) Do() { m.dialog = Export }

// ExportFloat returns an Action to start exporting the song as a wav file with
// 32-bit float samples.
func (m *SongModel) ExportFloat() Action { return MakeAction((*exportFloat)(m)) }

type exportFloat SongModel

func (m *exportFloat) Do() { m.dialog = ExportFloatExplorer }

// ExportInt16 returns an Action to start exporting the song as a wav file with
// 16-bit integer samples.
func (m *SongModel) ExportInt16() Action { return MakeAction((*exportInt16)(m)) }

type exportInt16 SongModel

func (m *exportInt16) Do() { m.dialog = ExportInt16Explorer }

// WriteWav renders the song as a wav file and outputs it to the given
// io.WriteCloser. If the pcm16 is true, the sample format is 16-bit unsigned
// shorts, otherwise it's 32-bit floats.
func (m *SongModel) WriteWav(w io.WriteCloser, pcm16 bool) {
	m.dialog = NoDialog
	song := m.d.Song.Copy()
	go func() {
		b := make([]byte, 32+2)
		rand.Read(b)
		name := fmt.Sprintf("%x", b)[2 : 32+2]
		data, err := sointu.Play(m.curSynther, song, func(p float32) {
			txt := fmt.Sprintf("Exporting song: %.0f%%", p*100)
			TrySend(m.broker.ToModel, MsgToModel{Data: Alert{Message: txt, Priority: Info, Name: name, Duration: defaultAlertDuration}})
		}) // render the song to calculate its length
		if err != nil {
			txt := fmt.Sprintf("Error rendering the song during export: %v", err)
			TrySend(m.broker.ToModel, MsgToModel{Data: Alert{Message: txt, Priority: Error, Name: name, Duration: defaultAlertDuration}})
			return
		}
		buffer, err := data.Wav(pcm16)
		if err != nil {
			txt := fmt.Sprintf("Error converting to .wav: %v", err)
			TrySend(m.broker.ToModel, MsgToModel{Data: Alert{Message: txt, Priority: Error, Name: name, Duration: defaultAlertDuration}})
			return
		}
		w.Write(buffer)
		w.Close()
	}()
}
