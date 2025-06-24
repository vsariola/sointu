package gioui

import (
	"image"
	"image/color"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

type (
	PopupStyle struct {
		Color       color.NRGBA
		CornerRadii struct {
			SE, SW, NW, NE unit.Dp
		}
		Shadow struct {
			Color      color.NRGBA
			N, E, W, S unit.Dp
		}
	}

	PopupWidget struct {
		Style   *PopupStyle
		Visible *bool
	}
)

func Popup(th *Theme, visible *bool) PopupWidget {
	return PopupWidget{
		Style:   &th.Popup.Dialog,
		Visible: visible,
	}
}

func (s PopupWidget) Layout(gtx C, contents layout.Widget) D {
	s.update(gtx)

	if !*s.Visible {
		return D{}
	}

	bg := func(gtx C) D {
		rrect := clip.RRect{
			Rect: image.Rectangle{Max: gtx.Constraints.Min},
			SE:   gtx.Dp(s.Style.CornerRadii.SE),
			SW:   gtx.Dp(s.Style.CornerRadii.SW),
			NW:   gtx.Dp(s.Style.CornerRadii.NW),
			NE:   gtx.Dp(s.Style.CornerRadii.NE),
		}
		rrect2 := rrect
		rrect2.Rect.Min = rrect2.Rect.Min.Sub(image.Pt(gtx.Dp(s.Style.Shadow.W), gtx.Dp(s.Style.Shadow.N)))
		rrect2.Rect.Max = rrect2.Rect.Max.Add(image.Pt(gtx.Dp(s.Style.Shadow.E), gtx.Dp(s.Style.Shadow.S)))
		paint.FillShape(gtx.Ops, s.Style.Shadow.Color, rrect2.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, s.Style.Color, rrect.Op(gtx.Ops))
		area := clip.Rect(image.Rect(-1e6, -1e6, 1e6, 1e6)).Push(gtx.Ops)
		event.Op(gtx.Ops, s.Visible)
		area.Pop()
		area = clip.Rect(rrect2.Rect).Push(gtx.Ops)
		event.Op(gtx.Ops, &dummyTag)
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

func (s *PopupWidget) update(gtx C) {
	for {
		event, ok := gtx.Event(pointer.Filter{
			Target: s.Visible,
			Kinds:  pointer.Press,
		})
		if !ok {
			break
		}
		e, ok := event.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Press:
			*s.Visible = false
		}
	}
}

var dummyTag bool
