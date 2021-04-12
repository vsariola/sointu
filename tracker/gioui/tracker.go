package gioui

import (
	"encoding/json"
	"errors"
	"fmt"

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/tracker"
	"gopkg.in/yaml.v3"
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
	TrackVoices           *NumberInput
	InstrumentNameEditor  *widget.Editor
	NewTrackBtn           *widget.Clickable
	DeleteTrackBtn        *widget.Clickable
	NewInstrumentBtn      *widget.Clickable
	DeleteInstrumentBtn   *widget.Clickable
	AddSemitoneBtn        *widget.Clickable
	SubtractSemitoneBtn   *widget.Clickable
	AddOctaveBtn          *widget.Clickable
	SubtractOctaveBtn     *widget.Clickable
	NoteOffBtn            *widget.Clickable
	SongLength            *NumberInput
	PanicBtn              *widget.Clickable
	CopyInstrumentBtn     *widget.Clickable
	ParameterList         *layout.List
	ParameterScrollBar    *ScrollBar
	Parameters            []*ParameterWidget
	UnitDragList          *DragList
	UnitScrollBar         *ScrollBar
	DeleteUnitBtn         *widget.Clickable
	ClearUnitBtn          *widget.Clickable
	ChooseUnitTypeList    *layout.List
	ChooseUnitScrollBar   *ScrollBar
	ChooseUnitTypeBtns    []*widget.Clickable
	AddUnitBtn            *widget.Clickable
	InstrumentDragList    *DragList
	InstrumentScrollBar   *ScrollBar
	TrackHexCheckBox      *widget.Bool
	TopHorizontalSplit    *Split
	BottomHorizontalSplit *Split
	VerticalSplit         *Split
	StackUse              []int
	KeyPlaying            map[string]uint32
	Alert                 Alert
	PatternOrderList      *layout.List
	PatternOrderScrollBar *ScrollBar
	ConfirmInstrDelete    *Dialog

	lastVolume tracker.Volume
	volumeChan chan tracker.Volume

	player       *tracker.Player
	refresh      chan struct{}
	playerCloser chan struct{}
	errorChannel chan error
	audioContext sointu.AudioContext

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

func New(audioContext sointu.AudioContext, synthService sointu.SynthService, syncChannel chan<- []float32) *Tracker {
	t := &Tracker{
		Theme:                 material.NewTheme(gofont.Collection()),
		audioContext:          audioContext,
		BPM:                   new(NumberInput),
		OctaveNumberInput:     &NumberInput{Value: 4},
		SongLength:            new(NumberInput),
		RowsPerPattern:        new(NumberInput),
		RowsPerBeat:           new(NumberInput),
		Step:                  &NumberInput{Value: 1},
		InstrumentVoices:      new(NumberInput),
		TrackVoices:           new(NumberInput),
		InstrumentNameEditor:  &widget.Editor{SingleLine: true, Submit: true, Alignment: text.Middle},
		NewTrackBtn:           new(widget.Clickable),
		DeleteTrackBtn:        new(widget.Clickable),
		NewInstrumentBtn:      new(widget.Clickable),
		DeleteInstrumentBtn:   new(widget.Clickable),
		AddSemitoneBtn:        new(widget.Clickable),
		SubtractSemitoneBtn:   new(widget.Clickable),
		AddOctaveBtn:          new(widget.Clickable),
		SubtractOctaveBtn:     new(widget.Clickable),
		NoteOffBtn:            new(widget.Clickable),
		AddUnitBtn:            new(widget.Clickable),
		DeleteUnitBtn:         new(widget.Clickable),
		ClearUnitBtn:          new(widget.Clickable),
		PanicBtn:              new(widget.Clickable),
		CopyInstrumentBtn:     new(widget.Clickable),
		TrackHexCheckBox:      new(widget.Bool),
		Menus:                 make([]Menu, 2),
		MenuBar:               make([]widget.Clickable, 2),
		UnitDragList:          &DragList{List: &layout.List{Axis: layout.Vertical}, HoverItem: -1},
		UnitScrollBar:         &ScrollBar{Axis: layout.Vertical},
		refresh:               make(chan struct{}, 1), // use non-blocking sends; no need to queue extra ticks if one is queued already
		InstrumentDragList:    &DragList{List: &layout.List{Axis: layout.Horizontal}, HoverItem: -1},
		InstrumentScrollBar:   &ScrollBar{Axis: layout.Horizontal},
		ParameterList:         &layout.List{Axis: layout.Vertical},
		ParameterScrollBar:    &ScrollBar{Axis: layout.Vertical},
		TopHorizontalSplit:    &Split{Ratio: -.6},
		BottomHorizontalSplit: &Split{Ratio: -.6},
		VerticalSplit:         &Split{Axis: layout.Vertical},
		ChooseUnitTypeList:    &layout.List{Axis: layout.Vertical},
		ChooseUnitScrollBar:   &ScrollBar{Axis: layout.Vertical},
		KeyPlaying:            make(map[string]uint32),
		volumeChan:            make(chan tracker.Volume, 1),
		playerCloser:          make(chan struct{}),
		PatternOrderList:      &layout.List{Axis: layout.Vertical},
		PatternOrderScrollBar: &ScrollBar{Axis: layout.Vertical},
		ConfirmInstrDelete:    new(Dialog),
		errorChannel:          make(chan error, 32),
	}
	t.Model = tracker.NewModel()
	vuBufferObserver := make(chan []float32)
	go tracker.VuAnalyzer(0.3, 1e-4, 1, -100, 20, vuBufferObserver, t.volumeChan, t.errorChannel)
	t.Theme.Palette.Fg = primaryColor
	t.Theme.Palette.ContrastFg = black
	t.SetEditMode(tracker.EditTracks)
	for range tracker.UnitTypeNames {
		t.ChooseUnitTypeBtns = append(t.ChooseUnitTypeBtns, new(widget.Clickable))
	}
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
