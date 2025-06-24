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
			gray := s.Gray
			if s.Focus {
				gray += 8
			}
			gray8 := uint8(min(max(gray, 0), 255))
			color := color.NRGBA{R: gray8, G: gray8, B: gray8, A: 255}
			paint.FillShape(gtx.Ops, color, clip.Rect{Max: gtx.Constraints.Min}.Op())
			return D{Size: gtx.Constraints.Min}
		},
		func(gtx C) D {
			return s.Inset.Layout(gtx, widget)
		},
	)
}
