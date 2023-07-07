package gioui

import (
	"image"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
)

type Split struct {
	// Ratio keeps the current layout.
	// 0 is center, -1 completely to the left, 1 completely to the right.
	Ratio float32
	// Bar is the width for resizing the layout
	Bar unit.Dp
	// Axis is the split direction: layout.Horizontal splits the view in left
	// and right, layout.Vertical splits the view in top and bottom
	Axis layout.Axis

	drag      bool
	dragID    pointer.ID
	dragCoord float32
}

var defaultBarWidth = unit.Dp(10)

func (s *Split) Layout(gtx layout.Context, first, second layout.Widget) layout.Dimensions {
	bar := gtx.Dp(s.Bar)
	if bar <= 1 {
		bar = gtx.Dp(defaultBarWidth)
	}

	var coord int
	if s.Axis == layout.Horizontal {
		coord = gtx.Constraints.Max.X
	} else {
		coord = gtx.Constraints.Max.Y
	}

	proportion := (s.Ratio + 1) / 2
	firstSize := int(proportion*float32(coord) - float32(bar))

	secondOffset := firstSize + bar
	secondSize := coord - secondOffset

	{ // handle input
		// Avoid affecting the input tree with pointer events.

		for _, ev := range gtx.Events(s) {
			e, ok := ev.(pointer.Event)
			if !ok {
				continue
			}

			switch e.Type {
			case pointer.Press:
				if s.drag {
					break
				}

				s.dragID = e.PointerID
				if s.Axis == layout.Horizontal {
					s.dragCoord = e.Position.X
				} else {
					s.dragCoord = e.Position.Y
				}
				s.drag = true

			case pointer.Drag:
				if s.dragID != e.PointerID {
					break
				}

				var deltaCoord, deltaRatio float32
				if s.Axis == layout.Horizontal {
					deltaCoord = e.Position.X - s.dragCoord
					s.dragCoord = e.Position.X
					deltaRatio = deltaCoord * 2 / float32(gtx.Constraints.Max.X)
				} else {
					deltaCoord = e.Position.Y - s.dragCoord
					s.dragCoord = e.Position.Y
					deltaRatio = deltaCoord * 2 / float32(gtx.Constraints.Max.Y)
				}

				s.Ratio += deltaRatio

			case pointer.Release:
				fallthrough
			case pointer.Cancel:
				s.drag = false
			}
		}

		low := -1 + float32(bar)/float32(coord)*2
		const snapMargin = 0.1

		if s.Ratio < low {
			s.Ratio = low
		}

		if s.Ratio > 1 {
			s.Ratio = 1
		}

		if s.Ratio < low+snapMargin {
			firstSize = 0
			secondOffset = bar
			secondSize = coord - bar
		} else if s.Ratio > 1-snapMargin {
			firstSize = coord - bar
			secondOffset = coord
			secondSize = 0
		}

		// register for input
		var barRect image.Rectangle
		if s.Axis == layout.Horizontal {
			barRect = image.Rect(firstSize, 0, secondOffset, gtx.Constraints.Max.Y)
		} else {
			barRect = image.Rect(0, firstSize, gtx.Constraints.Max.X, secondOffset)
		}
		area := clip.Rect(barRect).Push(gtx.Ops)
		pointer.InputOp{Tag: s,
			Types: pointer.Press | pointer.Drag | pointer.Release,
			Grab:  s.drag,
		}.Add(gtx.Ops)
		area.Pop()
	}

	{
		gtx := gtx

		if s.Axis == layout.Horizontal {
			gtx.Constraints = layout.Exact(image.Pt(firstSize, gtx.Constraints.Max.Y))
		} else {
			gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, firstSize))
		}
		area := clip.Rect(image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)).Push(gtx.Ops)
		first(gtx)
		area.Pop()
	}

	{
		gtx := gtx

		var transform op.TransformStack
		if s.Axis == layout.Horizontal {
			transform = op.Offset(image.Pt(secondOffset, 0)).Push(gtx.Ops)
			gtx.Constraints = layout.Exact(image.Pt(secondSize, gtx.Constraints.Max.Y))
		} else {
			transform = op.Offset(image.Pt(0, secondOffset)).Push(gtx.Ops)
			gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, secondSize))
		}

		area := clip.Rect(image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)).Push(gtx.Ops)
		second(gtx)
		area.Pop()
		transform.Pop()
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
