package tracker

import (
	"image"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

type Split struct {
	// Ratio keeps the current layout.
	// 0 is center, -1 completely to the left, 1 completely to the right.
	Ratio float32
	// Bar is the width for resizing the layout
	Bar unit.Value
	// Axis is the split direction: layout.Horizontal splits the view in left
	// and right, layout.Vertical splits the view in top and bottom
	Axis layout.Axis

	drag      bool
	dragID    pointer.ID
	dragCoord float32
}

var defaultBarWidth = unit.Dp(10)

func (s *Split) Layout(gtx layout.Context, first, second layout.Widget) layout.Dimensions {
	bar := gtx.Px(s.Bar)
	if bar <= 1 {
		bar = gtx.Px(defaultBarWidth)
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
		stack := op.Save(gtx.Ops)

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

		// register for input
		var barRect image.Rectangle
		if s.Axis == layout.Horizontal {
			barRect = image.Rect(firstSize, 0, secondOffset, gtx.Constraints.Max.Y)
		} else {
			barRect = image.Rect(0, firstSize, gtx.Constraints.Max.X, secondOffset)
		}
		pointer.Rect(barRect).Add(gtx.Ops)
		pointer.InputOp{Tag: s,
			Types: pointer.Press | pointer.Drag | pointer.Release,
			Grab:  s.drag,
		}.Add(gtx.Ops)

		stack.Load()
	}

	{
		gtx := gtx
		stack := op.Save(gtx.Ops)

		if s.Axis == layout.Horizontal {
			gtx.Constraints = layout.Exact(image.Pt(firstSize, gtx.Constraints.Max.Y))
		} else {
			gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, firstSize))
		}
		first(gtx)

		stack.Load()
	}

	{
		gtx := gtx
		stack := op.Save(gtx.Ops)
		if s.Axis == layout.Horizontal {
			op.Offset(f32.Pt(float32(secondOffset), 0)).Add(gtx.Ops)
			gtx.Constraints = layout.Exact(image.Pt(secondSize, gtx.Constraints.Max.Y))
		} else {
			op.Offset(f32.Pt(0, float32(secondOffset))).Add(gtx.Ops)
			gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, secondSize))
		}

		second(gtx)

		stack.Load()
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
