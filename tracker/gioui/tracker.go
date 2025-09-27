package gioui

import (
	"fmt"
	"image"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
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
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/tracker"
)

var canQuit = true // set to false in init() if plugin tag is enabled

type (
	Tracker struct {
		Theme                 *Theme
		OctaveNumberInput     *NumericUpDownState
		InstrumentVoices      *NumericUpDownState
		TopHorizontalSplit    *SplitState
		BottomHorizontalSplit *SplitState
		VerticalSplit         *SplitState
		KeyNoteMap            Keyboard[key.Name]
		PopupAlert            *AlertsState
		Zoom                  int

		DialogState *DialogState

		ModalDialog layout.Widget
		PatchPanel  *PatchPanel
		OrderEditor *OrderEditor
		TrackEditor *NoteEditor
		Explorer    *explorer.Explorer
		Exploring   bool
		SongPanel   *SongPanel

		filePathString tracker.String
		noteEvents     []tracker.NoteEvent

		preferences Preferences

		*tracker.Model
	}

	ShowManual Tracker
	AskHelp    Tracker
	ReportBug  Tracker

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
		OctaveNumberInput: NewNumericUpDownState(),
		InstrumentVoices:  NewNumericUpDownState(),

		TopHorizontalSplit:    &SplitState{Ratio: -.5},
		BottomHorizontalSplit: &SplitState{Ratio: -.6},
		VerticalSplit:         &SplitState{Axis: layout.Vertical},

		DialogState: new(DialogState),
		PatchPanel:  NewPatchPanel(model),
		OrderEditor: NewOrderEditor(model),
		TrackEditor: NewNoteEditor(model),

		Zoom: 6,

		Model: model,

		filePathString: model.FilePath(),
	}
	t.SongPanel = NewSongPanel(t)
	t.KeyNoteMap = MakeKeyboard[key.Name](model.Broker())
	t.PopupAlert = NewAlertsState()
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
	return t
}

func (t *Tracker) Main() {
	recoveryTicker := time.NewTicker(time.Second * 30)
	var ops op.Ops
	titlePath := ""
	globals := make(map[string]any, 1)
	globals["Tracker"] = t
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
					gtx.Values = globals
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

func TrackerFromContext(gtx C) *Tracker {
	t, ok := gtx.Values["Tracker"]
	if !ok {
		panic("Tracker not found in context values")
	}
	return t.(*Tracker)
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

	if t.InstrEnlarged().Value() {
		t.layoutTop(gtx)
	} else {
		t.VerticalSplit.Layout(gtx,
			&t.Theme.Split,
			t.layoutTop,
			t.layoutBottom)
	}
	alerts := Alerts(t.Alerts(), t.Theme, t.PopupAlert)
	alerts.Layout(gtx)
	t.showDialog(gtx)
	// this is the top level input handler for the whole app
	// it handles all the global key events and clipboard events
	// we need to tell gio that we handle tabs too; otherwise
	// it will steal them for focus switching
	for {
		ev, ok := gtx.Event(
			key.Filter{Name: "", Optional: key.ModAlt | key.ModCommand | key.ModShift | key.ModShortcut | key.ModSuper},
			key.Filter{Name: key.NameTab, Optional: key.ModShift | key.ModShortcut},
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
		dialog := MakeDialog(t.Theme, t.DialogState, "Save changes to song?", "Your changes will be lost if you don't save them.",
			DialogBtn("Save", t.SaveSong()),
			DialogBtn("Don't save", t.DiscardSong()),
			DialogBtn("Cancel", t.Cancel()),
		)
		dialog.Layout(gtx)
	case tracker.Export:
		dialog := MakeDialog(t.Theme, t.DialogState, "Export format", "Choose the sample format for the exported .wav file.",
			DialogBtn("Int16", t.ExportInt16()),
			DialogBtn("Float32", t.ExportFloat()),
			DialogBtn("Cancel", t.Cancel()),
		)
		dialog.Layout(gtx)
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
	case tracker.License:
		dialog := MakeDialog(t.Theme, t.DialogState, "License", sointu.License,
			DialogBtn("Close", t.Cancel()),
		)
		dialog.Layout(gtx)
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
				if err != explorer.ErrUserDecline {
					t.Alerts().Add(err.Error(), tracker.Error)
				}
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
				if err != explorer.ErrUserDecline {
					t.Alerts().Add(err.Error(), tracker.Error)
				}
			}
		}}
	}()
}

func (t *Tracker) layoutBottom(gtx layout.Context) layout.Dimensions {
	return t.BottomHorizontalSplit.Layout(gtx,
		&t.Theme.Split,
		t.OrderEditor.Layout,
		t.TrackEditor.Layout,
	)
}

func (t *Tracker) layoutTop(gtx layout.Context) layout.Dimensions {
	return t.TopHorizontalSplit.Layout(gtx,
		&t.Theme.Split,
		t.SongPanel.Layout,
		t.PatchPanel.Layout,
	)
}

func (t *Tracker) ShowManual() tracker.Action { return tracker.MakeEnabledAction((*ShowManual)(t)) }
func (t *ShowManual) Do()                     { (*Tracker)(t).openUrl("https://github.com/vsariola/sointu/wiki") }

func (t *Tracker) AskHelp() tracker.Action { return tracker.MakeEnabledAction((*AskHelp)(t)) }
func (t *AskHelp) Do() {
	(*Tracker)(t).openUrl("https://github.com/vsariola/sointu/discussions/categories/help-needed")
}

func (t *Tracker) ReportBug() tracker.Action { return tracker.MakeEnabledAction((*ReportBug)(t)) }
func (t *ReportBug) Do()                     { (*Tracker)(t).openUrl("https://github.com/vsariola/sointu/issues") }

func (t *Tracker) openUrl(url string) {
	var err error
	// following https://gist.github.com/hyg/9c4afcd91fe24316cbf0
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform for opening urls %s", runtime.GOOS)
	}
	if err != nil {
		t.Alerts().Add(err.Error(), tracker.Error)
	}
}

func (t *Tracker) Tags(curLevel int, yield TagYieldFunc) bool {
	ret := t.PatchPanel.Tags(curLevel+1, yield)
	if !t.InstrEnlarged().Value() {
		ret = ret && t.OrderEditor.Tags(curLevel+1, yield) &&
			t.TrackEditor.Tags(curLevel+1, yield)
	}
	return ret
}
