package tracker

import (
	"gioui.org/app"
	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
)

func (t *Tracker) Run(w *app.Window) error {
	var ops op.Ops
	for {
		select {
		case <-t.refresh:
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
