package tracker

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

type C = layout.Context
type D = layout.Dimensions

func (t *Tracker) Layout(gtx layout.Context) {
	paint.FillShape(gtx.Ops, backgroundColor, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())
	t.VerticalSplit.Layout(gtx,
		t.layoutTop,
		t.layoutBottom)
	t.updateInstrumentScroll()
}

func (t *Tracker) layoutBottom(gtx layout.Context) layout.Dimensions {
	return t.BottomHorizontalSplit.Layout(gtx,
		func(gtx C) D {
			return Surface{Gray: 24, Focus: t.EditMode == 0}.Layout(gtx, t.layoutPatterns)
		},
		func(gtx C) D {
			return Surface{Gray: 24, Focus: t.EditMode == 1}.Layout(gtx, t.layoutTracker)
		},
	)
}

func (t *Tracker) layoutTop(gtx layout.Context) layout.Dimensions {
	for t.NewInstrumentBtn.Clicked() {
		t.AddInstrument()
	}

	return t.TopHorizontalSplit.Layout(gtx,
		t.layoutSongPanel,
		t.layoutInstruments,
	)

}
