package gioui

import (
	"image"
	"image/color"

	"gioui.org/io/key"
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
	focused      bool
	requestFocus bool
	mainTag      bool
}

type FilledDragListStyle struct {
	dragList      *DragList
	HoverColor    color.NRGBA
	SelectedColor color.NRGBA
	CursorColor   color.NRGBA
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
		HoverColor:    dragListHoverColor,
		SelectedColor: dragListSelectedColor,
		CursorColor:   cursorColor,
	}
}

func (d *DragList) Focus() {
	d.requestFocus = true
}

func (d *DragList) Focused() bool {
	return d.focused
}

func (s *FilledDragListStyle) Layout(gtx C) D {
	swap := 0

	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
	keys := key.Set("↑|↓|Ctrl-↑|Ctrl-↓")
	if s.dragList.List.Axis == layout.Horizontal {
		keys = key.Set("←|→|Ctrl-←|Ctrl-→")
	}
	key.InputOp{Tag: &s.dragList.mainTag, Keys: keys}.Add(gtx.Ops)

	if s.dragList.List.Axis == layout.Horizontal {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
	} else {
		gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	}

	if s.dragList.requestFocus {
		s.dragList.requestFocus = false
		key.FocusOp{Tag: &s.dragList.mainTag}.Add(gtx.Ops)
	}

	for _, ke := range gtx.Events(&s.dragList.mainTag) {
		switch ke := ke.(type) {
		case key.FocusEvent:
			s.dragList.focused = ke.Focus
		case key.Event:
			if !s.dragList.focused || ke.State != key.Press {
				break
			}
			delta := 0
			switch {
			case s.dragList.List.Axis == layout.Horizontal && ke.Name == key.NameLeftArrow && s.dragList.SelectedItem > 0:
				delta = -1
			case s.dragList.List.Axis == layout.Horizontal && ke.Name == key.NameRightArrow && s.dragList.SelectedItem < s.Count-1:
				delta = 1
			case s.dragList.List.Axis == layout.Vertical && ke.Name == key.NameUpArrow && s.dragList.SelectedItem > 0:
				delta = -1
			case s.dragList.List.Axis == layout.Vertical && ke.Name == key.NameDownArrow && s.dragList.SelectedItem < s.Count-1:
				delta = 1
			}
			if delta != 0 {
				if ke.Modifiers.Contain(key.ModShortcut) {
					swap = delta
				} else {
					s.dragList.SelectedItem += delta
				}
			}
		}
	}

	listElem := func(gtx C, index int) D {
		for len(s.dragList.tags) <= index {
			s.dragList.tags = append(s.dragList.tags, false)
		}
		bg := func(gtx C) D {
			var color color.NRGBA
			if s.dragList.SelectedItem == index {
				if s.dragList.focused {
					color = s.CursorColor
				} else {
					color = s.SelectedColor
				}
			} else if s.dragList.HoverItem == index {
				color = s.HoverColor
			}
			paint.FillShape(gtx.Ops, color, clip.Rect{Max: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}.Op())
			return D{Size: gtx.Constraints.Min}
		}

		inputFg := func(gtx C) D {
			//defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
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
					key.FocusOp{Tag: &s.dragList.mainTag}.Add(gtx.Ops)
				}
			}
			rect := image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
			area := clip.Rect(rect).Push(gtx.Ops)
			pointer.InputOp{Tag: &s.dragList.tags[index],
				Types: pointer.Press | pointer.Enter | pointer.Leave,
			}.Add(gtx.Ops)
			area.Pop()
			if index == s.dragList.SelectedItem {
				for _, ev := range gtx.Events(&s.dragList.focused) {
					e, ok := ev.(pointer.Event)
					if !ok {
						continue
					}
					switch e.Type {
					case pointer.Press:
						s.dragList.dragID = e.PointerID
						s.dragList.drag = true
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
				area := clip.Rect(rect).Push(gtx.Ops)
				pointer.InputOp{Tag: &s.dragList.focused,
					Types: pointer.Drag | pointer.Press | pointer.Release,
					Grab:  s.dragList.drag,
				}.Add(gtx.Ops)
				pointer.CursorGrab.Add(gtx.Ops)
				area.Pop()
			}
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}
		return layout.Stack{Alignment: layout.W}.Layout(gtx,
			layout.Expanded(bg),
			layout.Expanded(inputFg),
			layout.Stacked(func(gtx C) D {
				return s.element(gtx, index)
			}),
		)
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
