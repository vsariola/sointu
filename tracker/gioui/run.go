package gioui

import (
	"fmt"
	"os"

	"gioui.org/app"
	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/vsariola/sointu"
)

func (t *Tracker) Run(w *app.Window) error {
	var ops op.Ops
	for {
		if pos, playing := t.player.Position(); t.NoteTracking() && playing {
			cursor := t.Cursor()
			cursor.SongRow = pos
			t.SetCursor(cursor)
			t.SetSelectionCorner(cursor)
		}
		select {
		case <-t.refresh:
			w.Invalidate()
		case v := <-t.volumeChan:
			t.lastVolume = v
			w.Invalidate()
		case e := <-w.Events():
			switch e := e.(type) {
			case system.DestroyEvent:
				return e.Err
			case key.Event:
				if t.KeyEvent(w, e) {
					w.Invalidate()
				}
			case clipboard.Event:
				err := t.UnmarshalContent([]byte(e.Text))
				if err == nil {
					w.Invalidate()
				}
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)
				t.Layout(gtx)
				e.Frame(gtx.Ops)
			}
		}
	}
}

func Main(audioContext sointu.AudioContext, synthService sointu.SynthService) {
	go func() {
		w := app.NewWindow(
			app.Size(unit.Dp(800), unit.Dp(600)),
			app.Title("Sointu Tracker"),
		)
		t := New(audioContext, synthService)
		defer t.Close()
		if err := t.Run(w); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}
