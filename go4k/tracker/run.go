package tracker

import (
	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"os"
)

func (t *Tracker) Run(w *app.Window) error {
	var ops op.Ops
	for {
		select {
		case <-t.ticked:
			w.Invalidate()
		case e := <-w.Events():
			switch e := e.(type) {
			case system.DestroyEvent:
				return e.Err
			case key.Event:
				if t.KeyEvent(e) {
					w.Invalidate()
				}
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)
				if t.QuitButton.Clicked() {
					os.Exit(0)
				}
				t.Layout(gtx)
				e.Frame(gtx.Ops)
			}
		}
	}

}
