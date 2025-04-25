package gioui

import (
	"image"

	"gioui.org/io/event"
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
	// Minimum sizes of the first and second widget in the split, in dp
	MinSize1, MinSize2 unit.Dp

	drag      bool
	dragID    pointer.ID
	dragCoord float32
}

var defaultBarWidth = unit.Dp(10)

func (s *Split) Update(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: s,
			Kinds:  pointer.Press | pointer.Drag | pointer.Release,
			// TODO: there should be a grab; there was Grab:  s.drag,
		})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}

		switch e.Kind {
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
			// when the user start dragging, the new display ratio becomes the underlying ratio
			s.Ratio = s.calculateRatio(gtx)

		case pointer.Drag:
			if s.dragID != e.PointerID {
				break
			}

			if s.Axis == layout.Horizontal {
				s.Ratio += (e.Position.X - s.dragCoord) / float32(gtx.Constraints.Max.X) * 2
				s.dragCoord = e.Position.X
			} else {
				s.Ratio += (e.Position.Y - s.dragCoord) / float32(gtx.Constraints.Max.Y) * 2
				s.dragCoord = e.Position.Y
			}

		case pointer.Release, pointer.Cancel:
			if s.dragID == e.PointerID {
				// when the user release the grab, the new display ratio becomes the underlying ratio
				s.Ratio = s.calculateRatio(gtx)
			}
			s.drag = false
		}
	}
}

func (s *Split) Layout(gtx layout.Context, first, second layout.Widget) layout.Dimensions {
	s.Update(gtx)

	size1, size2, bar := s.calculateSplitSizes(gtx)
	secondOffset := size1 + bar

	{
		// register for input
		var barRect image.Rectangle
		if s.Axis == layout.Horizontal {
			barRect = image.Rect(size1, 0, secondOffset, gtx.Constraints.Max.Y)
		} else {
			barRect = image.Rect(0, size1, gtx.Constraints.Max.X, secondOffset)
		}
		area := clip.Rect(barRect).Push(gtx.Ops)
		event.Op(gtx.Ops, s)
		if s.Axis == layout.Horizontal {
			pointer.CursorColResize.Add(gtx.Ops)
		} else {
			pointer.CursorRowResize.Add(gtx.Ops)
		}
		area.Pop()
	}

	{
		gtx := gtx

		if s.Axis == layout.Horizontal {
			gtx.Constraints = layout.Exact(image.Pt(size1, gtx.Constraints.Max.Y))
		} else {
			gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, size1))
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
			gtx.Constraints = layout.Exact(image.Pt(size2, gtx.Constraints.Max.Y))
		} else {
			transform = op.Offset(image.Pt(0, secondOffset)).Push(gtx.Ops)
			gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, size2))
		}

		area := clip.Rect(image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)).Push(gtx.Ops)
		second(gtx)
		area.Pop()
		transform.Pop()
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func (s *Split) calculateRatio(gtx layout.Context) float32 {
	size1, size2, bar := s.calculateSplitSizes(gtx)
	total := size1 + size2 + bar
	if total <= 0 {
		return 0
	}
	return 2*float32(size1+bar/2)/float32(total) - 1
}

func (s *Split) calculateSplitSizes(gtx layout.Context) (size1, size2, bar int) {
	bar = gtx.Dp(s.Bar)
	if bar <= 1 {
		bar = gtx.Dp(defaultBarWidth)
	}

	total := gtx.Constraints.Max.Y
	if s.Axis == layout.Horizontal {
		total = gtx.Constraints.Max.X
	}
	if total < 0 {
		total = 0
	}
	if total < bar {
		return 0, 0, total
	}
	totalSize := total - bar
	size1 = int((s.Ratio+1)/2*float32(total) - float32(bar)/2)
	minSize1 := gtx.Dp(s.MinSize1)
	minSize2 := gtx.Dp(s.MinSize2)

	// we always hide the smaller split first
	if s.Ratio < 0 {
		size1 = limitSplitSize(size1, totalSize, minSize1, minSize2)
	} else {
		size1 = totalSize - limitSplitSize(totalSize-size1, totalSize, minSize2, minSize1)
	}
	size2 = totalSize - size1
	return size1, size2, bar
}

// limitSplitSize hides the first split if it is smaller than minSize1/2 or if
// the total size is smaller than minSize1+minSize2. Otherwise, it clamps the
// size so that both split get at least minSize1 and minSize2 respectively.
func limitSplitSize(size, totalPx, minSize1, minSize2 int) int {
	if size < minSize1/2 || totalPx < minSize1+minSize2 {
		return 0 // the first split is completely hidden
	}
	return min(max(size, minSize1), totalPx-minSize2)
}
