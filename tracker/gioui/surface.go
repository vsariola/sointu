package gioui

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

type Surface struct {
	Gray  int
	Inset layout.Inset
	Focus bool
}

func (s Surface) Layout(gtx C, widget layout.Widget) D {
	return layout.Background{}.Layout(gtx,
		func(gtx C) D {
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
		},
		func(gtx C) D {
			return s.Inset.Layout(gtx, widget)
		},
	)
}
