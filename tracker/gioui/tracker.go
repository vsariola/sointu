package gioui

import (
	"fmt"
	"image"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
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
		KeyPlaying            map[string]tracker.NoteID
		PopupAlert            *PopupAlert

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

		quitWG   sync.WaitGroup
		execChan chan func()

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

func NewTracker(model *tracker.Model) *Tracker {
	t := &Tracker{
		Theme:             material.NewTheme(),
		OctaveNumberInput: NewNumberInput(model.Octave().Int()),
		InstrumentVoices:  NewNumberInput(model.InstrumentVoices().Int()),

		TopHorizontalSplit:    &Split{Ratio: -.6},
		BottomHorizontalSplit: &Split{Ratio: -.6},
		VerticalSplit:         &Split{Axis: layout.Vertical},

		KeyPlaying:        make(map[string]tracker.NoteID),
		SaveChangesDialog: NewDialog(model.SaveSong(), model.DiscardSong(), model.Cancel()),
		WaveTypeDialog:    NewDialog(model.ExportInt16(), model.ExportFloat(), model.Cancel()),
		InstrumentEditor:  NewInstrumentEditor(model),
		OrderEditor:       NewOrderEditor(model),
		TrackEditor:       NewNoteEditor(model),
		SongPanel:         NewSongPanel(model),

		Model: model,

		filePathString: model.FilePath().String(),
		execChan:       make(chan func(), 1024),
	}
	t.Theme.Shaper = text.NewShaper(text.WithCollection(fontCollection))
	t.PopupAlert = NewPopupAlert(model.Alerts(), t.Theme.Shaper)
	t.Theme.Palette.Fg = primaryColor
	t.Theme.Palette.ContrastFg = black
	t.TrackEditor.scrollTable.Focus()
	t.quitWG.Add(1)
	return t
}

func (t *Tracker) Main() {
	titleFooter := ""
	w := app.NewWindow(
		app.Size(unit.Dp(800), unit.Dp(600)),
		app.Title("Sointu Tracker"),
	)
	t.InstrumentEditor.Focus()
	recoveryTicker := time.NewTicker(time.Second * 30)
	t.Explorer = explorer.NewExplorer(w)
	var ops op.Ops
	for {
		if titleFooter != t.filePathString.Value() {
			titleFooter = t.filePathString.Value()
			if titleFooter != "" {
				w.Option(app.Title(fmt.Sprintf("Sointu Tracker - %v", titleFooter)))
			} else {
				w.Option(app.Title(fmt.Sprintf("Sointu Tracker")))
			}
		}
		select {
		case e := <-t.PlayerMessages:
			t.ProcessPlayerMessage(e)
			w.Invalidate()
		case e := <-w.Events():
			switch e := e.(type) {
			case system.DestroyEvent:
				if canQuit {
					t.Quit().Do()
				}
				if !t.Quitted() {
					// TODO: uh oh, there's no way of canceling the destroyevent in gioui? so we create a new window just to show the dialog
					w = app.NewWindow(
						app.Size(unit.Dp(800), unit.Dp(600)),
						app.Title("Sointu Tracker"),
					)
					t.Explorer = explorer.NewExplorer(w)
				}
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)
				if t.SongPanel.PlayingBtn.Bool.Value() && t.SongPanel.NoteTracking.Bool.Value() {
					t.TrackEditor.scrollTable.RowTitleList.CenterOn(t.PlaySongRow())
				}
				t.Layout(gtx, w)
				e.Frame(gtx.Ops)
			}
		case <-recoveryTicker.C:
			t.SaveRecovery()
		case f := <-t.execChan:
			f()
		}
		if t.Quitted() {
			break
		}
	}
	recoveryTicker.Stop()
	w.Perform(system.ActionClose)
	t.SaveRecovery()
	t.quitWG.Done()
}

func (t *Tracker) Exec() chan<- func() {
	return t.execChan
}

func (t *Tracker) WaitQuitted() {
	t.quitWG.Wait()
}

func (t *Tracker) Layout(gtx layout.Context, w *app.Window) {
	// this is the top level input handler for the whole app
	// it handles all the global key events and clipboard events
	// we need to tell gio that we handle tabs too; otherwise
	// it will steal them for focus switching
	key.InputOp{Tag: t, Keys: "Tab|Shift-Tab"}.Add(gtx.Ops)
	for _, ev := range gtx.Events(t) {
		switch e := ev.(type) {
		case key.Event:
			t.KeyEvent(e, gtx.Ops)
		case clipboard.Event:
			stringReader := strings.NewReader(e.Text)
			stringReadCloser := io.NopCloser(stringReader)
			t.ReadSong(stringReadCloser)
		}
	}

	paint.FillShape(gtx.Ops, backgroundColor, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())
	if t.InstrumentEditor.enlargeBtn.Bool.Value() {
		t.layoutTop(gtx)
	} else {
		t.VerticalSplit.Layout(gtx,
			t.layoutTop,
			t.layoutBottom)
	}
	t.PopupAlert.Layout(gtx)
	t.showDialog(gtx)
}

func (t *Tracker) showDialog(gtx C) {
	if t.Exploring {
		return
	}
	switch t.Dialog() {
	case tracker.NewSongChanges, tracker.OpenSongChanges, tracker.QuitChanges:
		dstyle := ConfirmDialog(t.Theme, t.SaveChangesDialog, "Save changes to song?", "Your changes will be lost if you don't save them.")
		dstyle.OkStyle.Text = "Save"
		dstyle.AltStyle.Text = "Don't save"
		dstyle.Layout(gtx)
	case tracker.Export:
		dstyle := ConfirmDialog(t.Theme, t.WaveTypeDialog, "", "Export .wav in int16 or float32 sample format?")
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
			t.WriteWav(wc, t.Dialog() == tracker.ExportInt16Explorer, t.execChan)
		}, filename)
	}
}

func (t *Tracker) explorerChooseFile(success func(io.ReadCloser), extensions ...string) {
	t.Exploring = true
	go func() {
		file, err := t.Explorer.ChooseFile(extensions...)
		t.Exec() <- func() {
			t.Exploring = false
			if err == nil {
				success(file)
			} else {
				t.Cancel().Do()
			}
		}
	}()
}

func (t *Tracker) explorerCreateFile(success func(io.WriteCloser), filename string) {
	t.Exploring = true
	go func() {
		file, err := t.Explorer.CreateFile(filename)
		t.Exec() <- func() {
			t.Exploring = false
			if err == nil {
				success(file)
			} else {
				t.Cancel().Do()
			}
		}
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
