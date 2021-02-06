package tracker

import (
	"image"
	"image/color"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget/material"
)

type DragList struct {
	SelectedItem int
	HoverItem    int
	List         *layout.List
	drag         bool
	dragID       pointer.ID
	tags         []bool
	swapped      bool
}

type FilledDragListStyle struct {
	dragList      *DragList
	SurfaceColor  color.NRGBA
	HoverColor    color.NRGBA
	SelectedColor color.NRGBA
	Count         int
	element       func(gtx C, i int) D
	swap          func(i, j int)
}

func FilledDragList(th *material.Theme, dragList *DragList, count int, element func(gtx C, i int) D, swap func(i, j int)) FilledDragListStyle {
	return FilledDragListStyle{
		dragList:      dragList,
		element:       element,
		swap:          swap,
		Count:         count,
		SurfaceColor:  dragListSurfaceColor,
		HoverColor:    dragListHoverColor,
		SelectedColor: dragListSelectedColor,
	}
}

func (s *FilledDragListStyle) Layout(gtx C) D {
	swap := 0

	paint.FillShape(gtx.Ops, s.SurfaceColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}.Op())
	defer op.Save(gtx.Ops).Load()

	if s.dragList.List.Axis == layout.Horizontal {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
	} else {
		gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	}

	listElem := func(gtx C, index int) D {
		for len(s.dragList.tags) <= index {
			s.dragList.tags = append(s.dragList.tags, false)
		}
		bg := func(gtx C) D {
			gtx.Constraints = layout.Exact(image.Pt(120, 20))
			var color color.NRGBA
			if s.dragList.SelectedItem == index {
				color = s.SelectedColor
			} else if s.dragList.HoverItem == index {
				color = s.HoverColor
			}
			paint.FillShape(gtx.Ops, color, clip.Rect{Max: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}.Op())
			return D{Size: gtx.Constraints.Min}
		}
		inputFg := func(gtx C) D {
			defer op.Save(gtx.Ops).Load()
			for _, ev := range gtx.Events(&s.dragList.tags[index]) {
				e, ok := ev.(pointer.Event)
				if !ok {
					continue
				}
				switch e.Type {
				case pointer.Enter:
					s.dragList.HoverItem = index
				case pointer.Leave:
					if s.dragList.HoverItem == index {
						s.dragList.HoverItem = -1
					}
				case pointer.Press:
					if s.dragList.drag {
						break
					}
					s.dragList.SelectedItem = index
				}
			}
			rect := image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
			pointer.Rect(rect).Add(gtx.Ops)
			pointer.InputOp{Tag: &s.dragList.tags[index],
				Types: pointer.Press | pointer.Enter | pointer.Leave,
			}.Add(gtx.Ops)
			if index == s.dragList.SelectedItem {
				for _, ev := range gtx.Events(s.dragList) {
					e, ok := ev.(pointer.Event)
					if !ok {
						continue
					}
					switch e.Type {
					case pointer.Press:
						s.dragList.dragID = e.PointerID
					case pointer.Drag:
						if s.dragList.dragID != e.PointerID {
							break
						}
						if s.dragList.List.Axis == layout.Horizontal {
							if e.Position.X < 0 {
								swap = -1
							}
							if e.Position.X > float32(gtx.Constraints.Min.X) {
								swap = 1
							}
						} else {
							if e.Position.Y < 0 {
								swap = -1
							}
							if e.Position.Y > float32(gtx.Constraints.Min.Y) {
								swap = 1
							}
						}
					case pointer.Release:
						fallthrough
					case pointer.Cancel:
						s.dragList.drag = false
					}
				}
				pointer.InputOp{Tag: s.dragList,
					Types: pointer.Drag | pointer.Press | pointer.Release,
					Grab:  s.dragList.drag,
				}.Add(gtx.Ops)
			}
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}
		return layout.Stack{Alignment: layout.W}.Layout(gtx,
			layout.Expanded(bg),
			layout.Stacked(func(gtx C) D {
				return s.element(gtx, index)
			}),
			layout.Expanded(inputFg))
	}
	dims := s.dragList.List.Layout(gtx, s.Count, listElem)
	if !s.dragList.swapped && swap != 0 && s.dragList.SelectedItem+swap >= 0 && s.dragList.SelectedItem+swap < s.Count {
		s.swap(s.dragList.SelectedItem, s.dragList.SelectedItem+swap)
		s.dragList.SelectedItem += swap
		s.dragList.swapped = true
	} else {
		s.dragList.swapped = false
	}
	return dims
}
