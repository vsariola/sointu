package gioui

import (
	"image"
	"image/color"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

type PopupStyle struct {
	Visible        *bool
	SurfaceColor   color.NRGBA
	ShadowColor    color.NRGBA
	ShadowN        unit.Dp
	ShadowE        unit.Dp
	ShadowW        unit.Dp
	ShadowS        unit.Dp
	SE, SW, NW, NE unit.Dp
}

func Popup(visible *bool) PopupStyle {
	return PopupStyle{
		Visible:      visible,
		SurfaceColor: popupSurfaceColor,
		ShadowColor:  popupShadowColor,
		ShadowN:      unit.Dp(2),
		ShadowE:      unit.Dp(2),
		ShadowS:      unit.Dp(2),
		ShadowW:      unit.Dp(2),
		SE:           unit.Dp(6),
		SW:           unit.Dp(6),
		NW:           unit.Dp(6),
		NE:           unit.Dp(6),
	}
}

func (s PopupStyle) Layout(gtx C, contents layout.Widget) D {
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
		rrect := clip.RRect{
			Rect: image.Rectangle{Max: gtx.Constraints.Min},
			SE:   gtx.Dp(s.SE),
			SW:   gtx.Dp(s.SW),
			NW:   gtx.Dp(s.NW),
			NE:   gtx.Dp(s.NE),
		}
		rrect2 := rrect
		rrect2.Rect.Min = rrect2.Rect.Min.Sub(image.Pt(gtx.Dp(s.ShadowW), gtx.Dp(s.ShadowN)))
		rrect2.Rect.Max = rrect2.Rect.Max.Add(image.Pt(gtx.Dp(s.ShadowE), gtx.Dp(s.ShadowS)))
		paint.FillShape(gtx.Ops, s.ShadowColor, rrect2.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, s.SurfaceColor, rrect.Op(gtx.Ops))
		area := clip.Rect(image.Rect(-1e6, -1e6, 1e6, 1e6)).Push(gtx.Ops)
		pointer.InputOp{Tag: s.Visible,
			Types: pointer.Press,
			Grab:  true,
		}.Add(gtx.Ops)
		area.Pop()
		area = clip.Rect(rrect2.Rect).Push(gtx.Ops)
		pointer.InputOp{Tag: &dummyTag,
			Types: pointer.Press,
			Grab:  true,
		}.Add(gtx.Ops)
		area.Pop()
		return D{Size: gtx.Constraints.Min}
	}
	macro := op.Record(gtx.Ops)
	dims := layout.Stack{}.Layout(gtx,
		layout.Expanded(bg),
		layout.Stacked(contents),
	)
	callop := macro.Stop()
	op.Defer(gtx.Ops, callop)
	return dims
}

var dummyTag bool
