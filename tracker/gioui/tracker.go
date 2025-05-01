package gioui

import (
	"fmt"
	"image"
	"io"
	"path/filepath"
	"time"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
	"github.com/vsariola/sointu/tracker"
)

var canQuit = true // set to false in init() if plugin tag is enabled

type (
	Tracker struct {
		Theme                 *material.Theme
		OctaveNumberInput     *NumberInput
		InstrumentVoices      *NumberInput
		TopHorizontalSplit    *Split
		BottomHorizontalSplit *Split
		VerticalSplit         *Split
		KeyPlaying            map[key.Name]tracker.NoteID
		MidiNotePlaying       []byte
		PopupAlert            *PopupAlert
		Zoom                  int

		SaveChangesDialog *Dialog
		WaveTypeDialog    *Dialog

		ModalDialog      layout.Widget
		InstrumentEditor *InstrumentEditor
		OrderEditor      *OrderEditor
		TrackEditor      *NoteEditor
		Explorer         *explorer.Explorer
		Exploring        bool
		SongPanel        *SongPanel

		filePathString tracker.String

		execChan    chan func()
		preferences Preferences

		*tracker.Model
	}

	C = layout.Context
	D = layout.Dimensions
)

const (
	ConfirmQuit = iota
	ConfirmLoad
	ConfirmNew
)

var ZoomFactors = []float32{.25, 1. / 3, .5, 2. / 3, .75, .8, 1, 1.1, 1.25, 1.5, 1.75, 2, 2.5, 3, 4, 5}

func NewTracker(model *tracker.Model) *Tracker {
	t := &Tracker{
		Theme:             material.NewTheme(),
		OctaveNumberInput: NewNumberInput(model.Octave().Int()),
		InstrumentVoices:  NewNumberInput(model.InstrumentVoices().Int()),

		TopHorizontalSplit:    &Split{Ratio: -.5, MinSize1: 180, MinSize2: 180},
		BottomHorizontalSplit: &Split{Ratio: -.6, MinSize1: 180, MinSize2: 180},
		VerticalSplit:         &Split{Axis: layout.Vertical, MinSize1: 180, MinSize2: 180},

		KeyPlaying:        make(map[key.Name]tracker.NoteID),
		MidiNotePlaying:   make([]byte, 0, 32),
		SaveChangesDialog: NewDialog(model.SaveSong(), model.DiscardSong(), model.Cancel()),
		WaveTypeDialog:    NewDialog(model.ExportInt16(), model.ExportFloat(), model.Cancel()),
		InstrumentEditor:  NewInstrumentEditor(model),
		OrderEditor:       NewOrderEditor(model),
		TrackEditor:       NewNoteEditor(model),
		SongPanel:         NewSongPanel(model),

		Zoom: 6,

		Model: model,

		filePathString: model.FilePath().String(),
		preferences:    MakePreferences(),
	}
	t.Theme.Shaper = text.NewShaper(text.WithCollection(fontCollection))
	t.PopupAlert = NewPopupAlert(model.Alerts(), t.Theme.Shaper)
	if t.preferences.YmlError != nil {
		model.Alerts().Add(
			fmt.Sprintf("Preferences YML Error: %s", t.preferences.YmlError),
			tracker.Warning,
		)
	}
	t.Theme.Palette.Fg = primaryColor
	t.Theme.Palette.ContrastFg = black
	t.TrackEditor.scrollTable.Focus()
	return t
}

func (t *Tracker) Main() {
	t.InstrumentEditor.Focus()
	recoveryTicker := time.NewTicker(time.Second * 30)
	var ops op.Ops
	titlePath := ""
	for !t.Quitted() {
		w := t.newWindow()
		w.Option(app.Title(titleFromPath(titlePath)))
		t.Explorer = explorer.NewExplorer(w)
		acks := make(chan struct{})
		events := make(chan event.Event)
		go func() {
			for {
				ev := w.Event()
				events <- ev
				<-acks
				if _, ok := ev.(app.DestroyEvent); ok {
					return
				}
			}
		}()
	F:
		for {
			select {
			case e := <-t.Broker().ToModel:
				t.ProcessMsg(e)
				w.Invalidate()
			case <-t.Broker().CloseGUI:
				t.ForceQuit().Do()
				w.Perform(system.ActionClose)
			case e := <-events:
				switch e := e.(type) {
				case app.DestroyEvent:
					if canQuit {
						t.RequestQuit().Do()
					}
					acks <- struct{}{}
					break F // this window is done, we need to create a new one
				case app.FrameEvent:
					if titlePath != t.filePathString.Value() {
						titlePath = t.filePathString.Value()
						w.Option(app.Title(titleFromPath(titlePath)))
					}
					gtx := app.NewContext(&ops, e)
					if t.Playing().Value() && t.Follow().Value() {
						t.TrackEditor.scrollTable.RowTitleList.CenterOn(t.PlaySongRow())
					}
					t.Layout(gtx, w)
					e.Frame(gtx.Ops)
					if t.Quitted() {
						w.Perform(system.ActionClose)
					}
				}
				acks <- struct{}{}
			case <-recoveryTicker.C:
				t.SaveRecovery()
			}
		}
	}
	recoveryTicker.Stop()
	t.SaveRecovery()
	close(t.Broker().FinishedGUI)
}

func (t *Tracker) newWindow() *app.Window {
	w := new(app.Window)
	w.Option(app.Size(t.preferences.WindowSize()))
	if t.preferences.Window.Maximized {
		w.Option(app.Maximized.Option())
	}
	return w
}

func titleFromPath(path string) string {
	if path == "" {
		return "Sointu Tracker"
	}
	return fmt.Sprintf("Sointu Tracker - %s", path)
}

func (t *Tracker) Layout(gtx layout.Context, w *app.Window) {
	zoomFactor := ZoomFactors[t.Zoom]
	gtx.Metric.PxPerDp *= zoomFactor
	gtx.Metric.PxPerSp *= zoomFactor
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, backgroundColor)
	event.Op(gtx.Ops, t) // area for capturing scroll events

	if t.InstrumentEditor.enlargeBtn.Bool.Value() {
		t.layoutTop(gtx)
	} else {
		t.VerticalSplit.Layout(gtx,
			t.layoutTop,
			t.layoutBottom)
	}
	t.PopupAlert.Layout(gtx)
	t.showDialog(gtx)
	// this is the top level input handler for the whole app
	// it handles all the global key events and clipboard events
	// we need to tell gio that we handle tabs too; otherwise
	// it will steal them for focus switching
	for {
		ev, ok := gtx.Event(
			key.Filter{Name: "", Optional: key.ModAlt | key.ModCommand | key.ModShift | key.ModShortcut | key.ModSuper},
			key.Filter{Name: key.NameTab, Optional: key.ModShift},
			transfer.TargetFilter{Target: t, Type: "application/text"},
			pointer.Filter{Target: t, Kinds: pointer.Scroll, ScrollY: pointer.ScrollRange{Min: -1, Max: 1}},
		)
		if !ok {
			break
		}
		switch e := ev.(type) {
		case pointer.Event:
			switch e.Kind {
			case pointer.Scroll:
				if e.Modifiers.Contain(key.ModShortcut) {
					t.Zoom = min(max(t.Zoom-int(e.Scroll.Y), 0), len(ZoomFactors)-1)
					t.Alerts().AddNamed("ZoomFactor", fmt.Sprintf("%.0f%%", ZoomFactors[t.Zoom]*100), tracker.Info)
				}
			}
		case key.Event:
			t.KeyEvent(e, gtx)
		case transfer.DataEvent:
			t.ReadSong(e.Open())
		}
	}

}

func (t *Tracker) showDialog(gtx C) {
	if t.Exploring {
		return
	}
	switch t.Dialog() {
	case tracker.NewSongChanges, tracker.OpenSongChanges, tracker.QuitChanges:
		dstyle := ConfirmDialog(gtx, t.Theme, t.SaveChangesDialog, "Save changes to song?", "Your changes will be lost if you don't save them.")
		dstyle.OkStyle.Text = "Save"
		dstyle.AltStyle.Text = "Don't save"
		dstyle.Layout(gtx)
	case tracker.Export:
		dstyle := ConfirmDialog(gtx, t.Theme, t.WaveTypeDialog, "", "Export .wav in int16 or float32 sample format?")
		dstyle.OkStyle.Text = "Int16"
		dstyle.AltStyle.Text = "Float32"
		dstyle.Layout(gtx)
	case tracker.OpenSongOpenExplorer:
		t.explorerChooseFile(t.ReadSong, ".yml", ".json")
	case tracker.NewSongSaveExplorer, tracker.OpenSongSaveExplorer, tracker.QuitSaveExplorer, tracker.SaveAsExplorer:
		filename := t.filePathString.Value()
		if filename == "" {
			filename = "song.yml"
		}
		t.explorerCreateFile(t.WriteSong, filename)
	case tracker.ExportFloatExplorer, tracker.ExportInt16Explorer:
		filename := "song.wav"
		if p := t.filePathString.Value(); p != "" {
			filename = p[:len(p)-len(filepath.Ext(p))] + ".wav"
		}
		t.explorerCreateFile(func(wc io.WriteCloser) {
			t.WriteWav(wc, t.Dialog() == tracker.ExportInt16Explorer)
		}, filename)
	}
}

func (t *Tracker) explorerChooseFile(success func(io.ReadCloser), extensions ...string) {
	t.Exploring = true
	go func() {
		file, err := t.Explorer.ChooseFile(extensions...)
		t.Broker().ToModel <- tracker.MsgToModel{Data: func() {
			t.Exploring = false
			if err == nil {
				success(file)
			} else {
				t.Cancel().Do()
			}
		}}
	}()
}

func (t *Tracker) explorerCreateFile(success func(io.WriteCloser), filename string) {
	t.Exploring = true
	go func() {
		file, err := t.Explorer.CreateFile(filename)
		t.Broker().ToModel <- tracker.MsgToModel{Data: func() {
			t.Exploring = false
			if err == nil {
				success(file)
			} else {
				t.Cancel().Do()
			}
		}}
	}()
}

func (t *Tracker) layoutBottom(gtx layout.Context) layout.Dimensions {
	return t.BottomHorizontalSplit.Layout(gtx,
		func(gtx C) D {
			return t.OrderEditor.Layout(gtx, t)
		},
		func(gtx C) D {
			return t.TrackEditor.Layout(gtx, t)
		},
	)
}

func (t *Tracker) layoutTop(gtx layout.Context) layout.Dimensions {
	return t.TopHorizontalSplit.Layout(gtx,
		func(gtx C) D {
			return t.SongPanel.Layout(gtx, t)
		},
		func(gtx C) D {
			return t.InstrumentEditor.Layout(gtx, t)
		},
	)
}

/// Event Handling (for UI updates when playing etc.)

func (t *Tracker) ProcessMessage(msg interface{}) {
	switch msg.(type) {
	case tracker.StartPlayMsg:
		fmt.Println("Tracker received StartPlayMsg")
	case tracker.RecordingMsg:
		fmt.Println("Tracker received RecordingMsg")
	default:
		break
	}
}

func (t *Tracker) ProcessEvent(event tracker.MIDINoteEvent) {
	// MIDINoteEvent can be only NoteOn / NoteOff, i.e. its On field
	if event.On {
		t.addToMidiNotePlaying(event.Note)
	} else {
		t.removeFromMidiNotePlaying(event.Note)
	}
	t.TrackEditor.HandleMidiInput(t)
}

func (t *Tracker) addToMidiNotePlaying(note byte) {
	for _, n := range t.MidiNotePlaying {
		if n == note {
			return
		}
	}
	t.MidiNotePlaying = append(t.MidiNotePlaying, note)
}

func (t *Tracker) removeFromMidiNotePlaying(note byte) {
	for i, n := range t.MidiNotePlaying {
		if n == note {
			t.MidiNotePlaying = append(
				t.MidiNotePlaying[:i],
				t.MidiNotePlaying[i+1:]...,
			)
		}
	}
}
