package gioui

import (
	"bytes"
	"image"
	"io"

	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"github.com/vsariola/sointu/tracker"
)

type ScrollTable struct {
	ColTitleList *DragList
	RowTitleList *DragList
	Table        tracker.Table
	requestFocus bool
	cursorMoved  bool
	eventFilters []event.Filter
	drag         bool
	dragID       pointer.ID
}

type ScrollTableStyle struct {
	RowTitleStyle     FilledDragListStyle
	ColTitleStyle     FilledDragListStyle
	ScrollTable       *ScrollTable
	ScrollBarWidth    unit.Dp
	RowTitleWidth     unit.Dp
	ColumnTitleHeight unit.Dp
	CellWidth         unit.Dp
	CellHeight        unit.Dp
	element           func(gtx C, x, y int) D
}

func NewScrollTable(table tracker.Table, vertList, horizList tracker.List) *ScrollTable {
	ret := &ScrollTable{
		Table:        table,
		ColTitleList: NewDragList(vertList, layout.Horizontal),
		RowTitleList: NewDragList(horizList, layout.Vertical),
	}
	ret.eventFilters = []event.Filter{
		key.FocusFilter{Target: ret},
		transfer.TargetFilter{Target: ret, Type: "application/text"},
		pointer.Filter{Target: ret, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel},
		key.Filter{Focus: ret, Name: key.NameLeftArrow, Optional: key.ModShift | key.ModCtrl | key.ModAlt},
		key.Filter{Focus: ret, Name: key.NameUpArrow, Optional: key.ModShift | key.ModCtrl | key.ModAlt},
		key.Filter{Focus: ret, Name: key.NameRightArrow, Optional: key.ModShift | key.ModCtrl | key.ModAlt},
		key.Filter{Focus: ret, Name: key.NameDownArrow, Optional: key.ModShift | key.ModCtrl | key.ModAlt},
		key.Filter{Focus: ret, Name: key.NamePageUp, Optional: key.ModShift},
		key.Filter{Focus: ret, Name: key.NamePageDown, Optional: key.ModShift},
		key.Filter{Focus: ret, Name: key.NameHome, Optional: key.ModShift},
		key.Filter{Focus: ret, Name: key.NameEnd, Optional: key.ModShift},
		key.Filter{Focus: ret, Name: key.NameDeleteBackward},
		key.Filter{Focus: ret, Name: key.NameDeleteForward},
	}
	for k, a := range keyBindingMap {
		switch a {
		case "Copy", "Paste", "Cut", "Increase", "Decrease":
			ret.eventFilters = append(ret.eventFilters, key.Filter{Focus: ret, Name: k.Name, Required: k.Modifiers})
		}
	}
	return ret
}

func FilledScrollTable(th *Theme, scrollTable *ScrollTable) ScrollTableStyle {
	return ScrollTableStyle{
		RowTitleStyle:     FilledDragList(th, scrollTable.RowTitleList),
		ColTitleStyle:     FilledDragList(th, scrollTable.ColTitleList),
		ScrollTable:       scrollTable,
		ScrollBarWidth:    unit.Dp(14),
		RowTitleWidth:     unit.Dp(30),
		ColumnTitleHeight: unit.Dp(16),
		CellWidth:         unit.Dp(16),
		CellHeight:        unit.Dp(16),
	}
}

func (st *ScrollTable) CursorMoved() bool {
	ret := st.cursorMoved
	st.cursorMoved = false
	return ret
}

func (st *ScrollTable) Focus() {
	st.requestFocus = true
}

func (st *ScrollTable) Focused(gtx C) bool {
	return gtx.Source.Focused(st)
}

func (st *ScrollTable) EnsureCursorVisible() {
	st.ColTitleList.EnsureVisible(st.Table.Cursor().X)
	st.RowTitleList.EnsureVisible(st.Table.Cursor().Y)
}

func (st *ScrollTable) ChildFocused(gtx C) bool {
	return st.ColTitleList.Focused(gtx) || st.RowTitleList.Focused(gtx)
}

func (s ScrollTableStyle) Layout(gtx C, element func(gtx C, x, y int) D, colTitle, rowTitle, colTitleBg, rowTitleBg func(gtx C, i int) D) D {
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, s.ScrollTable)

	p := image.Pt(gtx.Dp(s.RowTitleWidth), gtx.Dp(s.ColumnTitleHeight))
	s.handleEvents(gtx, p)

	return Surface{Gray: 24, Focus: s.ScrollTable.Focused(gtx) || s.ScrollTable.ChildFocused(gtx)}.Layout(gtx, func(gtx C) D {
		defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
		dims := gtx.Constraints.Max
		s.layoutColTitles(gtx, p, colTitle, colTitleBg)
		s.layoutRowTitles(gtx, p, rowTitle, rowTitleBg)
		defer op.Offset(p).Push(gtx.Ops).Pop()
		gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X-p.X, gtx.Constraints.Max.Y-p.Y))
		s.layoutTable(gtx, element)
		s.RowTitleStyle.LayoutScrollBar(gtx)
		s.ColTitleStyle.LayoutScrollBar(gtx)
		return D{Size: dims}
	})
}

func (s *ScrollTableStyle) handleEvents(gtx layout.Context, p image.Point) {
	for {
		e, ok := gtx.Event(s.ScrollTable.eventFilters...)
		if !ok {
			break
		}
		switch e := e.(type) {
		case pointer.Event:
			switch e.Kind {
			case pointer.Press:
				if s.ScrollTable.drag {
					break
				}
				s.ScrollTable.dragID = e.PointerID
				s.ScrollTable.drag = true
				fallthrough
			case pointer.Drag:
				if s.ScrollTable.dragID != e.PointerID {
					break
				}
				if int(e.Position.X) < p.X || int(e.Position.Y) < p.Y {
					break
				}
				e.Position.X -= float32(p.X)
				e.Position.Y -= float32(p.Y)
				if e.Kind == pointer.Press {
					gtx.Execute(key.FocusCmd{Tag: s.ScrollTable})
				}
				dx := (e.Position.X + float32(s.ScrollTable.ColTitleList.List.Position.Offset)) / float32(gtx.Dp(s.CellWidth))
				dy := (e.Position.Y + float32(s.ScrollTable.RowTitleList.List.Position.Offset)) / float32(gtx.Dp(s.CellHeight))
				x := dx + float32(s.ScrollTable.ColTitleList.List.Position.First)
				y := dy + float32(s.ScrollTable.RowTitleList.List.Position.First)
				cursorPoint := tracker.Point{X: int(x), Y: int(y)}
				s.ScrollTable.Table.SetCursor2(cursorPoint)
				if e.Kind == pointer.Press && !e.Modifiers.Contain(key.ModShift) {
					s.ScrollTable.Table.SetCursor(cursorPoint)
				}
				s.ScrollTable.cursorMoved = true
			case pointer.Release:
				fallthrough
			case pointer.Cancel:
				s.ScrollTable.drag = false
			}
		case key.Event:
			if e.State == key.Press {
				s.ScrollTable.command(gtx, e)
			}
		case transfer.DataEvent:
			if b, err := io.ReadAll(e.Open()); err == nil {
				s.ScrollTable.Table.Paste(b)
			}
		}
	}

	for {
		e, ok := gtx.Event(
			key.Filter{Focus: s.ScrollTable.RowTitleList, Name: "→"},
		)
		if !ok {
			break
		}
		if e, ok := e.(key.Event); ok && e.State == key.Press {
			s.ScrollTable.Focus()
		}
	}

	for {
		e, ok := gtx.Event(
			key.Filter{Focus: s.ScrollTable.ColTitleList, Name: "↓"},
		)
		if !ok {
			break
		}
		if e, ok := e.(key.Event); ok && e.State == key.Press {
			s.ScrollTable.Focus()
		}
	}
}

func (s ScrollTableStyle) layoutTable(gtx C, element func(gtx C, x, y int) D) {
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()

	if s.ScrollTable.requestFocus {
		s.ScrollTable.requestFocus = false
		gtx.Execute(key.FocusCmd{Tag: s.ScrollTable})
	}
	cellWidth := gtx.Dp(s.CellWidth)
	cellHeight := gtx.Dp(s.CellHeight)

	gtx.Constraints = layout.Exact(image.Pt(cellWidth, cellHeight))

	colP := s.ColTitleStyle.dragList.List.Position
	rowP := s.RowTitleStyle.dragList.List.Position
	defer op.Offset(image.Pt(-colP.Offset, -rowP.Offset)).Push(gtx.Ops).Pop()
	for x := 0; x < colP.Count; x++ {
		for y := 0; y < rowP.Count; y++ {
			o := op.Offset(image.Pt(cellWidth*x, cellHeight*y)).Push(gtx.Ops)
			element(gtx, x+colP.First, y+rowP.First)
			o.Pop()
		}
	}
}

func (s *ScrollTableStyle) layoutRowTitles(gtx C, p image.Point, fg, bg func(gtx C, i int) D) {
	defer op.Offset(image.Pt(0, p.Y)).Push(gtx.Ops).Pop()
	gtx.Constraints.Min.X = p.X
	gtx.Constraints.Max.Y -= p.Y
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	s.RowTitleStyle.Layout(gtx, fg, bg)
}

func (s *ScrollTableStyle) layoutColTitles(gtx C, p image.Point, fg, bg func(gtx C, i int) D) {
	defer op.Offset(image.Pt(p.X, 0)).Push(gtx.Ops).Pop()
	gtx.Constraints.Min.Y = p.Y
	gtx.Constraints.Max.X -= p.X
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	s.ColTitleStyle.Layout(gtx, fg, bg)
}

func (s *ScrollTable) command(gtx C, e key.Event) {
	stepX := 1
	stepY := 1
	if e.Modifiers.Contain(key.ModAlt) {
		stepX = max(s.ColTitleList.List.Position.Count-3, 8)
		stepY = max(s.RowTitleList.List.Position.Count-3, 8)
	} else if e.Modifiers.Contain(key.ModCtrl) {
		stepX = 1e6
		stepY = 1e6
	}
	switch e.Name {
	case key.NameDeleteBackward, key.NameDeleteForward:
		s.Table.Clear()
		return
	case key.NameUpArrow:
		if !s.Table.MoveCursor(0, -stepY) && stepY == 1 {
			s.ColTitleList.Focus()
		}
	case key.NameDownArrow:
		s.Table.MoveCursor(0, stepY)
	case key.NameLeftArrow:
		if !s.Table.MoveCursor(-stepX, 0) && stepX == 1 {
			s.RowTitleList.Focus()
		}
	case key.NameRightArrow:
		s.Table.MoveCursor(stepX, 0)
	case key.NamePageUp:
		s.Table.MoveCursor(0, -max(s.RowTitleList.List.Position.Count-3, 8))
	case key.NamePageDown:
		s.Table.MoveCursor(0, max(s.RowTitleList.List.Position.Count-3, 8))
	case key.NameHome:
		s.Table.SetCursorX(0)
	case key.NameEnd:
		s.Table.SetCursorX(s.Table.Width() - 1)
	default:
		a := keyBindingMap[e]
		switch a {
		case "Copy", "Cut":
			contents, ok := s.Table.Copy()
			if !ok {
				return
			}
			gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(bytes.NewReader(contents))})
			if a == "Cut" {
				s.Table.Clear()
			}
			return
		case "Paste":
			gtx.Execute(clipboard.ReadCmd{Tag: s})
			return
		case "Increase":
			s.Table.Add(1)
			return
		case "Decrease":
			s.Table.Add(-1)
			return
		}
	}
	if !e.Modifiers.Contain(key.ModShift) {
		s.Table.SetCursor2(s.Table.Cursor())
	}
	s.ColTitleList.EnsureVisible(s.Table.Cursor().X)
	s.RowTitleList.EnsureVisible(s.Table.Cursor().Y)
	s.cursorMoved = true
}
