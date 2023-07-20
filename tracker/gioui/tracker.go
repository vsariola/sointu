package gioui

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
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
	RecordBtn             *widget.Clickable
	AddUnitBtn            *widget.Clickable
	TrackHexCheckBox      *widget.Bool
	TopHorizontalSplit    *Split
	BottomHorizontalSplit *Split
	VerticalSplit         *Split
	KeyPlaying            map[string]tracker.NoteID
	Alert                 Alert
	ConfirmSongDialog     *Dialog
	WaveTypeDialog        *Dialog
	ConfirmSongActionType int
	ModalDialog           layout.Widget
	InstrumentEditor      *InstrumentEditor
	OrderEditor           *OrderEditor
	TrackEditor           *TrackEditor
	Explorer              *explorer.Explorer

	lastVolume tracker.Volume

	wavFilePath  string
	refresh      chan struct{}
	errorChannel chan error
	quitted      bool
	synthService sointu.SynthService

	*tracker.Model
}

func (t *Tracker) UnmarshalContent(bytes []byte) error {
	var units []sointu.Unit
	if errJSON := json.Unmarshal(bytes, &units); errJSON == nil {
		t.PasteUnits(units)
		return nil
	}
	if errYaml := yaml.Unmarshal(bytes, &units); errYaml == nil {
		t.PasteUnits(units)
		return nil
	}
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

func NewTracker(model *tracker.Model, synthService sointu.SynthService) *Tracker {
	t := &Tracker{
		Theme:             material.NewTheme(gofont.Collection()),
		BPM:               new(NumberInput),
		OctaveNumberInput: &NumberInput{Value: 4},
		SongLength:        new(NumberInput),
		RowsPerPattern:    new(NumberInput),
		RowsPerBeat:       new(NumberInput),
		Step:              &NumberInput{Value: 1},
		InstrumentVoices:  new(NumberInput),

		PanicBtn:         new(widget.Clickable),
		RecordBtn:        new(widget.Clickable),
		TrackHexCheckBox: new(widget.Bool),
		Menus:            make([]Menu, 2),
		MenuBar:          make([]widget.Clickable, 2),
		refresh:          make(chan struct{}, 1), // use non-blocking sends; no need to queue extra ticks if one is queued already

		TopHorizontalSplit:    &Split{Ratio: -.6},
		BottomHorizontalSplit: &Split{Ratio: -.6},
		VerticalSplit:         &Split{Axis: layout.Vertical},

		KeyPlaying:        make(map[string]tracker.NoteID),
		ConfirmSongDialog: new(Dialog),
		WaveTypeDialog:    new(Dialog),
		InstrumentEditor:  NewInstrumentEditor(),
		OrderEditor:       NewOrderEditor(),
		TrackEditor:       NewTrackEditor(),

		errorChannel: make(chan error, 32),
		synthService: synthService,
		Model:        model,
	}
	t.Theme.Palette.Fg = primaryColor
	t.Theme.Palette.ContrastFg = black
	t.TrackEditor.Focus()
	t.SetOctave(4)
	t.ResetSong()
	return t
}

func (t *Tracker) Main() {
	titleFooter := ""
	w := app.NewWindow(
		app.Size(unit.Dp(800), unit.Dp(600)),
		app.Title("Sointu Tracker"),
	)
	t.Explorer = explorer.NewExplorer(w)
	var ops op.Ops
mainloop:
	for {
		if pos, playing := t.PlayPosition(), t.Playing(); t.NoteTracking() && playing {
			cursor := t.Cursor()
			cursor.SongRow = pos
			t.SetCursor(cursor)
			t.SetSelectionCorner(cursor)
		}
		if titleFooter != t.FilePath() {
			titleFooter = t.FilePath()
			if titleFooter != "" {
				w.Option(app.Title(fmt.Sprintf("Sointu Tracker - %v", titleFooter)))
			} else {
				w.Option(app.Title(fmt.Sprintf("Sointu Tracker")))
			}
		}
		select {
		case <-t.refresh:
			w.Invalidate()
		case e := <-t.errorChannel:
			t.Alert.Update(e.Error(), Error, time.Second*5)
			w.Invalidate()
		case e := <-t.PlayerMessages:
			if err, ok := e.Inner.(tracker.PlayerCrashMessage); ok {
				t.Alert.Update(err.Error(), Error, time.Second*3)
			}
			if err, ok := e.Inner.(tracker.PlayerVolumeErrorMessage); ok {
				t.Alert.Update(err.Error(), Warning, time.Second*3)
			}
			t.lastVolume = e.Volume
			t.InstrumentEditor.voiceStates = e.VoiceStates
			t.ProcessPlayerMessage(e)
			w.Invalidate()
		case e := <-w.Events():
			switch e := e.(type) {
			case system.DestroyEvent:
				if !t.Quit(false) {
					// TODO: uh oh, there's no way of canceling the destroyevent in gioui? so we create a new window just to show the dialog
					w = app.NewWindow(
						app.Size(unit.Dp(800), unit.Dp(600)),
						app.Title("Sointu Tracker"),
					)
					t.Explorer = explorer.NewExplorer(w)
				}
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)
				t.Layout(gtx, w)
				e.Frame(gtx.Ops)
			}
		}
		if t.quitted {
			break mainloop
		}
	}
	w.Perform(system.ActionClose)
}
