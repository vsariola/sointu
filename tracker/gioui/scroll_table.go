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
	"gioui.org/widget/material"
	"github.com/vsariola/sointu/tracker"
)

type ScrollTable struct {
	ColTitleList *DragList
	RowTitleList *DragList
	Table        tracker.Table
	focused      bool
	requestFocus bool
	colTag       bool
	rowTag       bool
	cursorMoved  bool
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
	return &ScrollTable{
		Table:        table,
		ColTitleList: NewDragList(vertList, layout.Horizontal),
		RowTitleList: NewDragList(horizList, layout.Vertical),
	}
}

func FilledScrollTable(th *material.Theme, scrollTable *ScrollTable, element func(gtx C, x, y int) D, colTitle, rowTitle, colTitleBg, rowTitleBg func(gtx C, i int) D) ScrollTableStyle {
	return ScrollTableStyle{
		RowTitleStyle:     FilledDragList(th, scrollTable.RowTitleList, rowTitle, rowTitleBg),
		ColTitleStyle:     FilledDragList(th, scrollTable.ColTitleList, colTitle, colTitleBg),
		ScrollTable:       scrollTable,
		element:           element,
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

func (st *ScrollTable) Focused() bool {
	return st.focused
}

func (st *ScrollTable) EnsureCursorVisible() {
	st.ColTitleList.EnsureVisible(st.Table.Cursor().X)
	st.RowTitleList.EnsureVisible(st.Table.Cursor().Y)
}

func (st *ScrollTable) ChildFocused() bool {
	return st.ColTitleList.Focused() || st.RowTitleList.Focused()
}

func (s ScrollTableStyle) Layout(gtx C) D {
	p := image.Pt(gtx.Dp(s.RowTitleWidth), gtx.Dp(s.ColumnTitleHeight))
	s.handleEvents(gtx)

	return Surface{Gray: 24, Focus: s.ScrollTable.Focused() || s.ScrollTable.ChildFocused()}.Layout(gtx, func(gtx C) D {
		defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
		dims := gtx.Constraints.Max
		s.layoutColTitles(gtx, p)
		s.layoutRowTitles(gtx, p)
		defer op.Offset(p).Push(gtx.Ops).Pop()
		gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X-p.X, gtx.Constraints.Max.Y-p.Y))
		s.layoutTable(gtx, p)
		s.RowTitleStyle.LayoutScrollBar(gtx)
		s.ColTitleStyle.LayoutScrollBar(gtx)
		return D{Size: dims}
	})
}

func (s *ScrollTableStyle) handleEvents(gtx layout.Context) {
	for {
		e, ok := gtx.Event(
			key.FocusFilter{Target: s.ScrollTable},
			transfer.TargetFilter{Target: s.ScrollTable, Type: "application/text"},
			pointer.Filter{Target: s.ScrollTable, Kinds: pointer.Press},
			key.Filter{Focus: s.ScrollTable, Name: key.NameLeftArrow, Optional: key.ModShift | key.ModCtrl | key.ModAlt},
			key.Filter{Focus: s.ScrollTable, Name: key.NameUpArrow, Optional: key.ModShift | key.ModCtrl | key.ModAlt},
			key.Filter{Focus: s.ScrollTable, Name: key.NameRightArrow, Optional: key.ModShift | key.ModCtrl | key.ModAlt},
			key.Filter{Focus: s.ScrollTable, Name: key.NameDownArrow, Optional: key.ModShift | key.ModCtrl | key.ModAlt},
			key.Filter{Focus: s.ScrollTable, Name: key.NamePageUp, Optional: key.ModShift},
			key.Filter{Focus: s.ScrollTable, Name: key.NamePageDown, Optional: key.ModShift},
			key.Filter{Focus: s.ScrollTable, Name: key.NameHome, Optional: key.ModShift},
			key.Filter{Focus: s.ScrollTable, Name: key.NameEnd, Optional: key.ModShift},
			key.Filter{Focus: s.ScrollTable, Name: key.NameDeleteBackward},
			key.Filter{Focus: s.ScrollTable, Name: key.NameDeleteForward},
			key.Filter{Focus: s.ScrollTable, Name: "C", Required: key.ModShortcut},
			key.Filter{Focus: s.ScrollTable, Name: "V", Required: key.ModShortcut},
			key.Filter{Focus: s.ScrollTable, Name: "X", Required: key.ModShortcut},
			key.Filter{Focus: s.ScrollTable, Name: "+"},
			key.Filter{Focus: s.ScrollTable, Name: "-"},
		)
		if !ok {
			break
		}
		switch e := e.(type) {
		case key.FocusEvent:
			s.ScrollTable.focused = e.Focus
		case pointer.Event:
			if e.Kind == pointer.Press {
				gtx.Execute(key.FocusCmd{Tag: s.ScrollTable})
			}
			dx := (int(e.Position.X) + s.ScrollTable.ColTitleList.List.Position.Offset) / gtx.Dp(s.CellWidth)
			dy := (int(e.Position.Y) + s.ScrollTable.RowTitleList.List.Position.Offset) / gtx.Dp(s.CellHeight)
			x := dx + s.ScrollTable.ColTitleList.List.Position.First
			y := dy + s.ScrollTable.RowTitleList.List.Position.First
			s.ScrollTable.Table.SetCursor(
				tracker.Point{X: x, Y: y},
			)
			if !e.Modifiers.Contain(key.ModShift) {
				s.ScrollTable.Table.SetCursor2(s.ScrollTable.Table.Cursor())
			}
			s.ScrollTable.cursorMoved = true
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
			key.FocusFilter{
				Target: &s.ScrollTable.rowTag,
			},
			key.Filter{
				Focus: &s.ScrollTable.rowTag,
				Name:  "→",
			},
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
			key.FocusFilter{
				Target: &s.ScrollTable.colTag,
			},
			key.Filter{
				Focus: &s.ScrollTable.colTag,
				Name:  "↓",
			},
		)
		if !ok {
			break
		}
		if e, ok := e.(key.Event); ok && e.State == key.Press {
			s.ScrollTable.Focus()
		}
	}
}

func (s ScrollTableStyle) layoutTable(gtx C, p image.Point) {
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()

	if s.ScrollTable.requestFocus {
		s.ScrollTable.requestFocus = false
		gtx.Execute(key.FocusCmd{Tag: s.ScrollTable})
	}
	event.Op(gtx.Ops, s.ScrollTable)
	cellWidth := gtx.Dp(s.CellWidth)
	cellHeight := gtx.Dp(s.CellHeight)

	gtx.Constraints = layout.Exact(image.Pt(cellWidth, cellHeight))

	colP := s.ColTitleStyle.dragList.List.Position
	rowP := s.RowTitleStyle.dragList.List.Position
	defer op.Offset(image.Pt(-colP.Offset, -rowP.Offset)).Push(gtx.Ops).Pop()
	for x := 0; x < colP.Count; x++ {
		for y := 0; y < rowP.Count; y++ {
			o := op.Offset(image.Pt(cellWidth*x, cellHeight*y)).Push(gtx.Ops)
			s.element(gtx, x+colP.First, y+rowP.First)
			o.Pop()
		}
	}
}

func (s *ScrollTableStyle) layoutRowTitles(gtx C, p image.Point) {
	defer op.Offset(image.Pt(0, p.Y)).Push(gtx.Ops).Pop()
	gtx.Constraints.Min.X = p.X
	gtx.Constraints.Max.Y -= p.Y
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, &s.ScrollTable.rowTag)
	s.RowTitleStyle.Layout(gtx)
}

func (s *ScrollTableStyle) layoutColTitles(gtx C, p image.Point) {
	defer op.Offset(image.Pt(p.X, 0)).Push(gtx.Ops).Pop()
	gtx.Constraints.Min.Y = p.Y
	gtx.Constraints.Max.X -= p.X
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, &s.ScrollTable.colTag)
	s.ColTitleStyle.Layout(gtx)
}

func (s *ScrollTable) command(gtx C, e key.Event) {
	stepX := 1
	stepY := 1
	if e.Modifiers.Contain(key.ModAlt) {
		stepX = intMax(s.ColTitleList.List.Position.Count-3, 8)
		stepY = intMax(s.RowTitleList.List.Position.Count-3, 8)
	} else if e.Modifiers.Contain(key.ModCtrl) {
		stepX = 1e6
		stepY = 1e6
	}
	switch e.Name {
	case "X", "C":
		if e.Modifiers.Contain(key.ModShortcut) {
			contents, ok := s.Table.Copy()
			if !ok {
				return
			}
			gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(bytes.NewReader(contents))})
			if e.Name == "X" {
				s.Table.Clear()
			}
			return
		}
	case "V":
		if e.Modifiers.Contain(key.ModShortcut) {
			gtx.Execute(clipboard.ReadCmd{Tag: s})
		}
		return
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
		s.Table.MoveCursor(0, -intMax(s.RowTitleList.List.Position.Count-3, 8))
	case key.NamePageDown:
		s.Table.MoveCursor(0, intMax(s.RowTitleList.List.Position.Count-3, 8))
	case key.NameHome:
		s.Table.SetCursorX(0)
	case key.NameEnd:
		s.Table.SetCursorX(s.Table.Width() - 1)
	case "+":
		s.Table.Add(1)
		return
	case "-":
		s.Table.Add(-1)
		return
	}
	if !e.Modifiers.Contain(key.ModShift) {
		s.Table.SetCursor2(s.Table.Cursor())
	}
	s.ColTitleList.EnsureVisible(s.Table.Cursor().X)
	s.RowTitleList.EnsureVisible(s.Table.Cursor().Y)
	s.cursorMoved = true
}
