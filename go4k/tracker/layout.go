package tracker

import (
	"gioui.org/layout"
	"gioui.org/op/paint"
)

func (t *Tracker) Layout(gtx layout.Context) {
	layout.Stack{Alignment: layout.NW}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paint.Fill(gtx.Ops, black)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Expanded(t.QuitButton.Layout),
		layout.Stacked(Raised(Label("Hello", white))),
	)
}
