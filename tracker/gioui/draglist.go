package gioui

import (
	"bytes"
	"image"
	"image/color"
	"io"

	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/vsariola/sointu/tracker"
)

type DragList struct {
	TrackerList  tracker.List
	HoverItem    int
	List         *layout.List
	ScrollBar    *ScrollBar
	drag         bool
	dragID       pointer.ID
	tags         []bool
	swapped      bool
	requestFocus bool
}

type FilledDragListStyle struct {
	dragList   *DragList
	HoverColor color.NRGBA
	Cursor     CursorStyle
	Selection  CursorStyle
	ScrollBar  ScrollBarStyle
}

func NewDragList(model tracker.List, axis layout.Axis) *DragList {
	return &DragList{TrackerList: model, List: &layout.List{Axis: axis}, HoverItem: -1, ScrollBar: &ScrollBar{Axis: axis}}
}

func FilledDragList(th *Theme, dragList *DragList) FilledDragListStyle {
	return FilledDragListStyle{
		dragList:   dragList,
		HoverColor: hoveredColor(th.Selection.Active),
		Cursor:     th.Cursor,
		Selection:  th.Selection,
		ScrollBar:  th.ScrollBar,
	}
}

func (d *DragList) Focus() {
	d.requestFocus = true
}

func (d *DragList) Focused(gtx C) bool {
	return gtx.Focused(d)
}

func (s FilledDragListStyle) LayoutScrollBar(gtx C) D {
	return s.dragList.ScrollBar.Layout(gtx, &s.ScrollBar, s.dragList.TrackerList.Count(), &s.dragList.List.Position)
}

func (s FilledDragListStyle) Layout(gtx C, element, bg func(gtx C, i int) D) D {
	swap := 0

	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, s.dragList)

	if s.dragList.List.Axis == layout.Horizontal {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
	} else {
		gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	}

	if s.dragList.requestFocus {
		s.dragList.requestFocus = false
		gtx.Execute(key.FocusCmd{Tag: s.dragList})
	}

	prevKey := key.NameUpArrow
	nextKey := key.NameDownArrow
	firstKey := key.NamePageUp
	lastKey := key.NamePageDown
	if s.dragList.List.Axis == layout.Horizontal {
		prevKey = key.NameLeftArrow
		nextKey = key.NameRightArrow
		firstKey = key.NameHome
		lastKey = key.NameEnd
	}

	for {
		event, ok := gtx.Event(
			key.FocusFilter{Target: s.dragList},
			transfer.TargetFilter{Target: s.dragList, Type: "application/text"},
			key.Filter{Focus: s.dragList, Name: prevKey, Optional: key.ModShift | key.ModShortcut},
			key.Filter{Focus: s.dragList, Name: nextKey, Optional: key.ModShift | key.ModShortcut},
			key.Filter{Focus: s.dragList, Name: firstKey, Optional: key.ModShift | key.ModShortcut},
			key.Filter{Focus: s.dragList, Name: lastKey, Optional: key.ModShift | key.ModShortcut},
			key.Filter{Focus: s.dragList, Name: "A", Required: key.ModShortcut},
			key.Filter{Focus: s.dragList, Name: "C", Required: key.ModShortcut},
			key.Filter{Focus: s.dragList, Name: "X", Required: key.ModShortcut},
			key.Filter{Focus: s.dragList, Name: "V", Required: key.ModShortcut},
			key.Filter{Focus: s.dragList, Name: key.NameDeleteBackward, Required: key.ModShortcut},
			key.Filter{Focus: s.dragList, Name: key.NameDeleteForward},
		)
		if !ok {
			break
		}
		switch ke := event.(type) {
		case key.FocusEvent:
			if !ke.Focus {
				s.dragList.TrackerList.SetSelected2(s.dragList.TrackerList.Selected())
			}
		case key.Event:
			if !s.dragList.Focused(gtx) || ke.State != key.Press {
				break
			}
			s.dragList.command(gtx, ke)
		case transfer.DataEvent:
			if b, err := io.ReadAll(ke.Open()); err == nil {
				s.dragList.TrackerList.PasteElements([]byte(b))
			}

		}
		gtx.Execute(op.InvalidateCmd{})
	}

	_, isMutable := s.dragList.TrackerList.ListData.(tracker.MutableListData)

	listElem := func(gtx C, index int) D {
		for len(s.dragList.tags) <= index {
			s.dragList.tags = append(s.dragList.tags, false)
		}
		cursorBg := func(gtx C) D {
			var color color.NRGBA
			if s.dragList.TrackerList.Selected() == index {
				if gtx.Focused(s.dragList) {
					color = s.Cursor.Active
				} else {
					color = s.Cursor.Inactive
				}
			} else if between(s.dragList.TrackerList.Selected(), index, s.dragList.TrackerList.Selected2()) {
				if gtx.Focused(s.dragList) {
					color = s.Selection.Active
				} else {
					color = s.Selection.Inactive
				}
			} else if s.dragList.HoverItem == index {
				color = s.HoverColor
			}
			paint.FillShape(gtx.Ops, color, clip.Rect{Max: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}.Op())

			for {
				ev, ok := gtx.Event(pointer.Filter{
					Target: &s.dragList.tags[index],
					Kinds:  pointer.Press | pointer.Enter | pointer.Leave,
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
					s.dragList.HoverItem = index
				case pointer.Leave:
					if s.dragList.HoverItem == index {
						s.dragList.HoverItem = -1
					}
				case pointer.Press:
					if s.dragList.drag {
						break
					}
					s.dragList.TrackerList.SetSelected(index)
					if !e.Modifiers.Contain(key.ModShift) {
						s.dragList.TrackerList.SetSelected2(index)
					}
					gtx.Execute(key.FocusCmd{Tag: s.dragList})
				}
			}
			rect := image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
			area := clip.Rect(rect).Push(gtx.Ops)
			event.Op(gtx.Ops, &s.dragList.tags[index])
			area.Pop()
			if index == s.dragList.TrackerList.Selected() && isMutable {
				for {
					target := &s.dragList.drag
					if s.dragList.drag {
						target = nil
					}
					ev, ok := gtx.Event(pointer.Filter{Target: target, Kinds: pointer.Drag | pointer.Press | pointer.Release | pointer.Cancel})
					if !ok {
						break
					}
					e, ok := ev.(pointer.Event)
					if !ok {
						continue
					}
					switch e.Kind {
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
					case pointer.Release, pointer.Cancel:
						s.dragList.drag = false
					}
				}
				area := clip.Rect(rect).Push(gtx.Ops)
				event.Op(gtx.Ops, &s.dragList.drag)
				pointer.CursorGrab.Add(gtx.Ops)
				area.Pop()
			}
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}
		macro := op.Record(gtx.Ops)
		dims := element(gtx, index)
		call := macro.Stop()
		gtx.Constraints.Min = dims.Size
		if bg != nil {
			bg(gtx, index)
		}
		cursorBg(gtx)
		call.Add(gtx.Ops)
		if s.dragList.List.Axis == layout.Horizontal {
			dims.Size.Y = gtx.Constraints.Max.Y
		} else {
			dims.Size.X = gtx.Constraints.Max.X
		}
		return dims
	}
	count := s.dragList.TrackerList.Count()
	if count < 1 {
		count = 1 // draw at least one empty element to get the correct size
	}
	dims := s.dragList.List.Layout(gtx, count, listElem)
	if !s.dragList.swapped && swap != 0 {
		if s.dragList.TrackerList.MoveElements(swap) {
			gtx.Execute(op.InvalidateCmd{})
		}
		s.dragList.swapped = true
	} else {
		s.dragList.swapped = false
	}
	return dims
}

func (e *DragList) command(gtx layout.Context, k key.Event) {
	if k.Modifiers.Contain(key.ModShortcut) {
		switch k.Name {
		case "V":
			gtx.Execute(clipboard.ReadCmd{Tag: e})
			return
		case "C", "X":
			data, ok := e.TrackerList.CopyElements()
			if ok && (k.Name == "C" || e.TrackerList.DeleteElements(false)) {
				gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(bytes.NewReader(data))})
			}
			return
		case "A":
			e.TrackerList.SetSelected(0)
			e.TrackerList.SetSelected2(e.TrackerList.Count() - 1)
			return
		}
	}
	delta := 0
	switch k.Name {
	case key.NameDeleteBackward:
		if k.Modifiers.Contain(key.ModShortcut) {
			e.TrackerList.DeleteElements(true)
		}
		return
	case key.NameDeleteForward:
		e.TrackerList.DeleteElements(false)
		return
	case key.NameLeftArrow:
		delta = -1
	case key.NameRightArrow:
		delta = 1
	case key.NameHome:
		delta = -1e6
	case key.NameEnd:
		delta = 1e6
	case key.NameUpArrow:
		delta = -1
	case key.NameDownArrow:
		delta = 1
	case key.NamePageUp:
		delta = -1e6
	case key.NamePageDown:
		delta = 1e6
	}
	if k.Modifiers.Contain(key.ModShortcut) {
		e.TrackerList.MoveElements(delta)
	} else {
		e.TrackerList.SetSelected(e.TrackerList.Selected() + delta)
		if !k.Modifiers.Contain(key.ModShift) {
			e.TrackerList.SetSelected2(e.TrackerList.Selected())
		}
	}
	e.EnsureVisible(e.TrackerList.Selected())
}

func (l *DragList) EnsureVisible(item int) {
	first := l.List.Position.First
	last := l.List.Position.First + l.List.Position.Count - 1
	if item < first || (item == first && l.List.Position.Offset > 0) {
		l.List.ScrollTo(item)
	}
	if item > last || (item == last && l.List.Position.OffsetLast < 0) {
		o := -l.List.Position.OffsetLast + l.List.Position.Offset
		l.List.ScrollTo(item - l.List.Position.Count + 1)
		l.List.Position.Offset = o
	}
}

func (l *DragList) CenterOn(item int) {
	lenPerChildPx := l.List.Position.Length / l.TrackerList.Count()
	if lenPerChildPx == 0 {
		return
	}
	listLengthPx := l.List.Position.Count*l.List.Position.Length/l.TrackerList.Count() + l.List.Position.OffsetLast - l.List.Position.Offset
	lenBeforeItem := (listLengthPx - lenPerChildPx) / 2
	quot := lenBeforeItem / lenPerChildPx
	rem := lenBeforeItem % lenPerChildPx
	l.List.ScrollTo(item - quot - 1)
	l.List.Position.Offset = lenPerChildPx - rem
}

func between(a, b, c int) bool {
	return (a <= b && b <= c) || (c <= b && b <= a)
}
