package gioui

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

type Surface struct {
	Gray    int
	Inset   layout.Inset
	FitSize bool
	Focus   bool
}

func (s Surface) Layout(gtx C, widget layout.Widget) D {
	bg := func(gtx C) D {
		grayInt := s.Gray
		if s.Focus {
			grayInt += 8
		}
		var grayUint8 uint8
		if grayInt < 0 {
			grayUint8 = 0
		} else if grayInt > 255 {
			grayUint8 = 255
		} else {
			grayUint8 = uint8(grayInt)
		}
		color := color.NRGBA{R: grayUint8, G: grayUint8, B: grayUint8, A: 255}
		paint.FillShape(gtx.Ops, color, clip.Rect{
			Max: gtx.Constraints.Min,
		}.Op())
		return D{Size: gtx.Constraints.Min}
	}
	fg := func(gtx C) D {
		return s.Inset.Layout(gtx, widget)
	}
	if s.FitSize {
		macro := op.Record(gtx.Ops)
		dims := fg(gtx)
		call := macro.Stop()
		gtx.Constraints = layout.Exact(dims.Size)
		bg(gtx)
		call.Add(gtx.Ops)
		return dims
	}
	gtxbg := gtx
	gtxbg.Constraints.Min = gtxbg.Constraints.Max
	bg(gtxbg)
	return fg(gtx)
}
