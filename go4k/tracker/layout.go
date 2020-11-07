package tracker

import "gioui.org/layout"

func (t *Tracker) Layout(gtx layout.Context) {
	t.QuitButton.Layout(gtx)
}
