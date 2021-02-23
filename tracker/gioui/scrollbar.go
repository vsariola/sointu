package gioui

import (
	"image"

	"gioui.org/f32"
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
}

func (s *ScrollBar) Layout(gtx C, width unit.Value, numItems int, pos *layout.Position) D {
	defer op.Save(gtx.Ops).Load()
	clip.Rect{Max: gtx.Constraints.Min}.Add(gtx.Ops)
	gradientSize := gtx.Px(unit.Dp(4))
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

	scrollBarRelStart := (float32(pos.First)*totalPixelsEstimate/float32(numItems) + float32(pos.Offset)) / totalPixelsEstimate
	scrWidth := gtx.Px(width)

	stack := op.Save(gtx.Ops)
	switch s.Axis {
	case layout.Vertical:
		if scrollBarRelLength < 1 && (s.dragging || s.hovering) {
			y1 := int(scrollBarRelStart * float32(gtx.Constraints.Min.Y))
			y2 := int((scrollBarRelStart + scrollBarRelLength) * float32(gtx.Constraints.Min.Y))
			paint.FillShape(gtx.Ops, scrollBarColor, clip.Rect{Min: image.Pt(gtx.Constraints.Min.X-scrWidth, y1), Max: image.Pt(gtx.Constraints.Min.X, y2)}.Op())
		}
		rect := image.Rect(gtx.Constraints.Min.X-scrWidth, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
		pointer.Rect(rect).Add(gtx.Ops)
	case layout.Horizontal:
		if scrollBarRelLength < 1 && (s.dragging || s.hovering) {
			x1 := int(scrollBarRelStart * float32(gtx.Constraints.Min.X))
			x2 := int((scrollBarRelStart + scrollBarRelLength) * float32(gtx.Constraints.Min.X))
			paint.FillShape(gtx.Ops, scrollBarColor, clip.Rect{Min: image.Pt(x1, gtx.Constraints.Min.Y-scrWidth), Max: image.Pt(x2, gtx.Constraints.Min.Y)}.Op())
		}
		rect := image.Rect(0, gtx.Constraints.Min.Y-scrWidth, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
		pointer.Rect(rect).Add(gtx.Ops)
	}
	pointer.InputOp{Tag: &s.dragStart,
		Types: pointer.Drag | pointer.Press | pointer.Cancel | pointer.Release,
	}.Add(gtx.Ops)
	stack.Load()

	for _, ev := range gtx.Events(&s.dragStart) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Type {
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
				pos.Offset += int(e.Position.X - s.dragStart + 0.5)
				s.dragStart = e.Position.X
			} else {
				pos.Offset += int(e.Position.Y - s.dragStart + 0.5)
				s.dragStart = e.Position.Y
			}
		case pointer.Release, pointer.Cancel:
			s.dragging = false
		}
	}

	pointer.PassOp{Pass: true}.Add(gtx.Ops)
	rect := image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
	pointer.Rect(rect).Add(gtx.Ops)
	pointer.InputOp{Tag: s,
		Types: pointer.Enter | pointer.Leave,
	}.Add(gtx.Ops)

	for _, ev := range gtx.Events(s) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Type {
		case pointer.Enter:
			s.hovering = true
		case pointer.Leave:
			s.hovering = false
		}
	}

	return D{Size: gtx.Constraints.Min}
}
