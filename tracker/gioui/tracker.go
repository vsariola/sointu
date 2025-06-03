package gioui

import (
	"fmt"
	"image"
	"io"
	"path/filepath"
	"time"

	"gioui.org/app"
	"gioui.org/font/gofont"
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
	"gioui.org/x/explorer"
	"github.com/vsariola/sointu/tracker"
)

var canQuit = true // set to false in init() if plugin tag is enabled

type (
	Tracker struct {
		Theme                 *Theme
		OctaveNumberInput     *NumberInput
		InstrumentVoices      *NumberInput
		TopHorizontalSplit    *Split
		BottomHorizontalSplit *Split
		VerticalSplit         *Split
		KeyNoteMap            Keyboard[key.Name]
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
		noteEvents     []tracker.NoteEvent

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
		OctaveNumberInput: NewNumberInput(model.Octave()),
		InstrumentVoices:  NewNumberInput(model.InstrumentVoices()),

		TopHorizontalSplit:    &Split{Ratio: -.5, MinSize1: 180, MinSize2: 180},
		BottomHorizontalSplit: &Split{Ratio: -.6, MinSize1: 180, MinSize2: 180},
		VerticalSplit:         &Split{Axis: layout.Vertical, MinSize1: 180, MinSize2: 180},

		SaveChangesDialog: NewDialog(model.SaveSong(), model.DiscardSong(), model.Cancel()),
		WaveTypeDialog:    NewDialog(model.ExportInt16(), model.ExportFloat(), model.Cancel()),
		InstrumentEditor:  NewInstrumentEditor(model),
		OrderEditor:       NewOrderEditor(model),
		TrackEditor:       NewNoteEditor(model),
		SongPanel:         NewSongPanel(model),

		Zoom: 6,

		Model: model,

		filePathString: model.FilePath(),
	}
	t.KeyNoteMap = MakeKeyboard[key.Name](model.Broker())
	t.PopupAlert = NewPopupAlert(model.Alerts())
	var warn error
	if t.Theme, warn = NewTheme(); warn != nil {
		model.Alerts().AddAlert(tracker.Alert{
			Priority: tracker.Warning,
			Message:  warn.Error(),
			Duration: 10 * time.Second,
		})
	}
	t.Theme.Material.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	if warn := ReadConfig(defaultPreferences, "preferences.yml", &t.preferences); warn != nil {
		model.Alerts().AddAlert(tracker.Alert{
			Priority: tracker.Warning,
			Message:  warn.Error(),
			Duration: 10 * time.Second,
		})
	}
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
			case e := <-t.Broker().ToGUI:
				switch e := e.(type) {
				case tracker.NoteEvent:
					t.noteEvents = append(t.noteEvents, e)
				case tracker.MsgToGUI:
					switch e.Kind {
					case tracker.GUIMessageCenterOnRow:
						t.TrackEditor.scrollTable.RowTitleList.CenterOn(e.Param)
					case tracker.GUIMessageEnsureCursorVisible:
						t.TrackEditor.scrollTable.EnsureCursorVisible()
					}
				}
				w.Invalidate()
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
	paint.Fill(gtx.Ops, t.Theme.Material.Bg)
	event.Op(gtx.Ops, t) // area for capturing scroll events

	if t.InstrumentEditor.enlargeBtn.Bool.Value() {
		t.layoutTop(gtx)
	} else {
		t.VerticalSplit.Layout(gtx,
			t.layoutTop,
			t.layoutBottom)
	}
	t.PopupAlert.Layout(gtx, t.Theme)
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
	// if no-one else handled the note events, we handle them here
	for len(t.noteEvents) > 0 {
		ev := t.noteEvents[0]
		ev.IsTrack = false
		ev.Channel = t.Model.Instruments().Selected()
		ev.Source = t
		copy(t.noteEvents, t.noteEvents[1:])
		t.noteEvents = t.noteEvents[:len(t.noteEvents)-1]
		tracker.TrySend(t.Broker().ToPlayer, any(ev))
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
