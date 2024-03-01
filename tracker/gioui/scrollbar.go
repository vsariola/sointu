package gioui

import (
	"image"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

type ScrollBar struct {
	Axis      layout.Axis
	dragStart float32
	hovering  bool
	dragging  bool
	tag       bool
}

func (s *ScrollBar) Layout(gtx C, width unit.Dp, numItems int, pos *layout.Position) D {
	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
	gradientSize := gtx.Dp(unit.Dp(4))
	var totalPixelsEstimate, scrollBarRelLength float32
	switch s.Axis {
	case layout.Vertical:
		if pos.First > 0 || pos.Offset > 0 {
			paint.LinearGradientOp{Color1: black, Color2: transparent, Stop2: f32.Pt(0, float32(gradientSize))}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
		}
		if pos.BeforeEnd {
			paint.LinearGradientOp{Color1: black, Color2: transparent, Stop1: f32.Pt(0, float32(gtx.Constraints.Min.Y)), Stop2: f32.Pt(0, float32(gtx.Constraints.Min.Y-gradientSize))}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
		}
		totalPixelsEstimate = float32(gtx.Constraints.Min.Y+pos.Offset-pos.OffsetLast) * float32(numItems) / float32(pos.Count)
		scrollBarRelLength = float32(gtx.Constraints.Min.Y) / float32(totalPixelsEstimate)

	case layout.Horizontal:
		if pos.First > 0 || pos.Offset > 0 {
			paint.LinearGradientOp{Color1: black, Color2: transparent, Stop2: f32.Pt(float32(gradientSize), 0)}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
		}
		if pos.BeforeEnd {
			paint.LinearGradientOp{Color1: black, Color2: transparent, Stop1: f32.Pt(float32(gtx.Constraints.Min.X), 0), Stop2: f32.Pt(float32(gtx.Constraints.Min.X-gradientSize), 0)}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
		}
		totalPixelsEstimate = float32(gtx.Constraints.Min.X+pos.Offset-pos.OffsetLast) * float32(numItems) / float32(pos.Count)
		scrollBarRelLength = float32(gtx.Constraints.Min.X) / float32(totalPixelsEstimate)
	}
	if scrollBarRelLength < 1e-2 {
		scrollBarRelLength = 1e-2 // make sure it doesn't disappear completely
	}

	scrollBarRelStart := (float32(pos.First)*totalPixelsEstimate/float32(numItems) + float32(pos.Offset)) / totalPixelsEstimate
	scrWidth := gtx.Dp(width)

	stack := op.Offset(image.Point{}).Push(gtx.Ops)
	var area clip.Stack
	switch s.Axis {
	case layout.Vertical:
		if scrollBarRelLength < 1 && (s.dragging || s.hovering) {
			y1 := int(scrollBarRelStart * float32(gtx.Constraints.Min.Y))
			y2 := int((scrollBarRelStart + scrollBarRelLength) * float32(gtx.Constraints.Min.Y))
			paint.FillShape(gtx.Ops, scrollBarColor, clip.Rect{Min: image.Pt(gtx.Constraints.Min.X-scrWidth, y1), Max: image.Pt(gtx.Constraints.Min.X, y2)}.Op())
		}
		rect := image.Rect(gtx.Constraints.Min.X-scrWidth, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
		area = clip.Rect(rect).Push(gtx.Ops)
	case layout.Horizontal:
		if scrollBarRelLength < 1 && (s.dragging || s.hovering) {
			x1 := int(scrollBarRelStart * float32(gtx.Constraints.Min.X))
			x2 := int((scrollBarRelStart + scrollBarRelLength) * float32(gtx.Constraints.Min.X))
			paint.FillShape(gtx.Ops, scrollBarColor, clip.Rect{Min: image.Pt(x1, gtx.Constraints.Min.Y-scrWidth), Max: image.Pt(x2, gtx.Constraints.Min.Y)}.Op())
		}
		rect := image.Rect(0, gtx.Constraints.Min.Y-scrWidth, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
		area = clip.Rect(rect).Push(gtx.Ops)
	}
	event.Op(gtx.Ops, &s.dragStart)
	area.Pop()
	stack.Pop()

	for {
		ev, ok := gtx.Event(
			pointer.Filter{Target: &s.dragStart, Kinds: pointer.Press | pointer.Cancel | pointer.Release | pointer.Drag},
		)
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Press:
			if s.Axis == layout.Horizontal {
				s.dragStart = e.Position.X
				s.dragging = true
			} else {
				s.dragStart = e.Position.Y
				s.dragging = true
			}
		case pointer.Drag:
			if s.Axis == layout.Horizontal {
				pos.Offset += int((e.Position.X-s.dragStart)/scrollBarRelLength + 0.5)
				s.dragStart = e.Position.X
			} else {
				pos.Offset += int((e.Position.Y-s.dragStart)/scrollBarRelLength + 0.5)
				s.dragStart = e.Position.Y
			}
		case pointer.Release, pointer.Cancel:
			s.dragging = false
		}
	}

	rect := image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
	area2 := clip.Rect(rect).Push(gtx.Ops)
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, &s.tag)
	area2.Pop()

	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: &s.tag,
			Kinds:  pointer.Enter | pointer.Leave,
		})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Enter:
			s.hovering = true
		case pointer.Leave:
			s.hovering = false
		}
	}

	return D{Size: gtx.Constraints.Min}
}

func scrollToView(l *layout.List, index int, length int) {
	pmin := index + 2 - l.Position.Count
	pmax := index - 1
	if pmin < 0 {
		pmin = 0
	}
	if pmax < 0 {
		pmax = 0
	}
	m := length - 1
	if pmin > m {
		pmin = m
	}
	if pmax > m {
		pmax = m
	}
	if l.Position.First > pmax {
		l.Position.First = pmax
		l.Position.Offset = 0
	}
	if l.Position.First < pmin {
		l.Position.First = pmin
	}
}
