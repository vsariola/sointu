package tracker

import (
	"image/color"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

type PopupStyle struct {
	Visible        *bool
	Contents       layout.Widget
	SurfaceColor   color.NRGBA
	SE, SW, NW, NE unit.Value
}

func Popup(visible *bool, contents layout.Widget) PopupStyle {
	return PopupStyle{
		Visible:      visible,
		Contents:     contents,
		SurfaceColor: popupSurfaceColor,
		SE:           unit.Dp(6),
		SW:           unit.Dp(6),
		NW:           unit.Dp(6),
		NE:           unit.Dp(6),
	}
}

func (s PopupStyle) Layout(gtx C) D {
	if !*s.Visible {
		return D{}
	}
	for _, ev := range gtx.Events(s.Visible) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}

		switch e.Type {
		case pointer.Press:
			*s.Visible = false
		}
	}

	bg := func(gtx C) D {
		pointer.InputOp{Tag: s.Visible,
			Types: pointer.Press,
		}.Add(gtx.Ops)
		rrect := clip.RRect{
			Rect: f32.Rectangle{Max: f32.Pt(float32(gtx.Constraints.Min.X), float32(gtx.Constraints.Min.Y))},
			SE:   float32(gtx.Px(s.SE)),
			SW:   float32(gtx.Px(s.SW)),
			NW:   float32(gtx.Px(s.NW)),
			NE:   float32(gtx.Px(s.NE)),
		}
		paint.FillShape(gtx.Ops, s.SurfaceColor, rrect.Op(gtx.Ops))
		return D{Size: gtx.Constraints.Min}
	}
	macro := op.Record(gtx.Ops)
	dims := layout.Stack{}.Layout(gtx,
		layout.Expanded(bg),
		layout.Stacked(s.Contents),
	)
	callop := macro.Stop()
	op.Defer(gtx.Ops, callop)
	return dims
}
