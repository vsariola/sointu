package gioui

import (
	"encoding/json"
	"errors"
	"fmt"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/tracker"
	"gopkg.in/yaml.v3"
)

const (
	ConfirmQuit = iota
	ConfirmLoad
	ConfirmNew
)

type Tracker struct {
	Theme                 *material.Theme
	MenuBar               []widget.Clickable
	Menus                 []Menu
	OctaveNumberInput     *NumberInput
	BPM                   *NumberInput
	RowsPerPattern        *NumberInput
	RowsPerBeat           *NumberInput
	Step                  *NumberInput
	InstrumentVoices      *NumberInput
	SongLength            *NumberInput
	PanicBtn              *widget.Clickable
	AddUnitBtn            *widget.Clickable
	TrackHexCheckBox      *widget.Bool
	TopHorizontalSplit    *Split
	BottomHorizontalSplit *Split
	VerticalSplit         *Split
	KeyPlaying            map[string]uint32
	Alert                 Alert
	ConfirmSongDialog     *Dialog
	WaveTypeDialog        *Dialog
	OpenSongDialog        *FileDialog
	SaveSongDialog        *FileDialog
	OpenInstrumentDialog  *FileDialog
	SaveInstrumentDialog  *FileDialog
	ExportWavDialog       *FileDialog
	ConfirmSongActionType int
	window                *app.Window
	ModalDialog           layout.Widget
	InstrumentEditor      *InstrumentEditor
	OrderEditor           *OrderEditor
	TrackEditor           *TrackEditor

	lastVolume tracker.Volume
	volumeChan chan tracker.Volume

	wavFilePath  string
	player       *tracker.Player
	refresh      chan struct{}
	playerCloser chan struct{}
	errorChannel chan error
	quitted      bool
	audioContext sointu.AudioContext
	synthService sointu.SynthService

	*tracker.Model
}

func (t *Tracker) UnmarshalContent(bytes []byte) error {
	var instr sointu.Instrument
	if errJSON := json.Unmarshal(bytes, &instr); errJSON == nil {
		if t.SetInstrument(instr) {
			return nil
		}
	}
	if errYaml := yaml.Unmarshal(bytes, &instr); errYaml == nil {
		if t.SetInstrument(instr) {
			return nil
		}
	}
	var song sointu.Song
	if errJSON := json.Unmarshal(bytes, &song); errJSON != nil {
		if errYaml := yaml.Unmarshal(bytes, &song); errYaml != nil {
			return fmt.Errorf("the song could not be parsed as .json (%v) or .yml (%v)", errJSON, errYaml)
		}
	}
	if song.BPM > 0 {
		t.SetSong(song)
		return nil
	}
	return errors.New("was able to unmarshal a song, but the bpm was 0")
}

func (t *Tracker) Close() {
	t.playerCloser <- struct{}{}
	t.audioContext.Close()
}

func New(audioContext sointu.AudioContext, synthService sointu.SynthService, syncChannel chan<- []float32, window *app.Window) *Tracker {
	t := &Tracker{
		Theme:             material.NewTheme(gofont.Collection()),
		audioContext:      audioContext,
		BPM:               new(NumberInput),
		OctaveNumberInput: &NumberInput{Value: 4},
		SongLength:        new(NumberInput),
		RowsPerPattern:    new(NumberInput),
		RowsPerBeat:       new(NumberInput),
		Step:              &NumberInput{Value: 1},
		InstrumentVoices:  new(NumberInput),

		PanicBtn:         new(widget.Clickable),
		TrackHexCheckBox: new(widget.Bool),
		Menus:            make([]Menu, 2),
		MenuBar:          make([]widget.Clickable, 2),
		refresh:          make(chan struct{}, 1), // use non-blocking sends; no need to queue extra ticks if one is queued already

		TopHorizontalSplit:    &Split{Ratio: -.6},
		BottomHorizontalSplit: &Split{Ratio: -.6},
		VerticalSplit:         &Split{Axis: layout.Vertical},

		KeyPlaying:           make(map[string]uint32),
		volumeChan:           make(chan tracker.Volume, 1),
		playerCloser:         make(chan struct{}),
		ConfirmSongDialog:    new(Dialog),
		WaveTypeDialog:       new(Dialog),
		OpenSongDialog:       NewFileDialog(),
		SaveSongDialog:       NewFileDialog(),
		OpenInstrumentDialog: NewFileDialog(),
		SaveInstrumentDialog: NewFileDialog(),
		InstrumentEditor:     NewInstrumentEditor(),
		OrderEditor:          NewOrderEditor(),
		TrackEditor:          NewTrackEditor(),

		ExportWavDialog: NewFileDialog(),
		errorChannel:    make(chan error, 32),
		window:          window,
		synthService:    synthService,
	}
	t.Model = tracker.NewModel()
	vuBufferObserver := make(chan []float32)
	go tracker.VuAnalyzer(0.3, 1e-4, 1, -100, 20, vuBufferObserver, t.volumeChan, t.errorChannel)
	t.Theme.Palette.Fg = primaryColor
	t.Theme.Palette.ContrastFg = black
	t.TrackEditor.Focus()
	t.SetOctave(4)
	patchObserver := make(chan sointu.Patch, 16)
	t.AddPatchObserver(patchObserver)
	scoreObserver := make(chan sointu.Score, 16)
	t.AddScoreObserver(scoreObserver)
	sprObserver := make(chan int, 16)
	t.AddSamplesPerRowObserver(sprObserver)
	audioChannel := make(chan []float32)
	t.player = tracker.NewPlayer(synthService, t.playerCloser, patchObserver, scoreObserver, sprObserver, t.refresh, syncChannel, audioChannel, vuBufferObserver)
	audioOut := audioContext.Output()
	go func() {
		for buf := range audioChannel {
			audioOut.WriteAudio(buf)
		}
	}()
	t.ResetSong()
	return t
}

func (t *Tracker) Quit(forced bool) bool {
	if !forced && t.ChangedSinceSave() {
		t.ConfirmSongActionType = ConfirmQuit
		t.ConfirmSongDialog.Visible = true
		return false
	}
	t.quitted = true
	return true
}
