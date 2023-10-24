package gioui

import (
	"image"

	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
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
	tag          bool
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

	for _, e := range gtx.Events(&s.ScrollTable.tag) {
		switch e := e.(type) {
		case key.FocusEvent:
			s.ScrollTable.focused = e.Focus
		case pointer.Event:
			if e.Position.X >= float32(p.X) && e.Position.Y >= float32(p.Y) {
				if e.Type == pointer.Press {
					key.FocusOp{Tag: &s.ScrollTable.tag}.Add(gtx.Ops)
				}
				dx := (int(e.Position.X) + s.ScrollTable.ColTitleList.List.Position.Offset - p.X) / gtx.Dp(s.CellWidth)
				dy := (int(e.Position.Y) + s.ScrollTable.RowTitleList.List.Position.Offset - p.Y) / gtx.Dp(s.CellHeight)
				x := dx + s.ScrollTable.ColTitleList.List.Position.First
				y := dy + s.ScrollTable.RowTitleList.List.Position.First
				s.ScrollTable.Table.SetCursor(
					tracker.Point{X: x, Y: y},
				)
				if !e.Modifiers.Contain(key.ModShift) {
					s.ScrollTable.Table.SetCursor2(s.ScrollTable.Table.Cursor())
				}
				s.ScrollTable.cursorMoved = true
			}
		case key.Event:
			if e.State == key.Press {
				s.ScrollTable.command(gtx, e)
			}
		case clipboard.Event:
			s.ScrollTable.Table.Paste([]byte(e.Text))
		}
	}

	for _, e := range gtx.Events(&s.ScrollTable.rowTag) {
		if e, ok := e.(key.Event); ok && e.State == key.Press {
			s.ScrollTable.Focus()
		}
	}

	for _, e := range gtx.Events(&s.ScrollTable.colTag) {
		if e, ok := e.(key.Event); ok && e.State == key.Press {
			s.ScrollTable.Focus()
		}
	}

	return Surface{Gray: 24, Focus: s.ScrollTable.Focused() || s.ScrollTable.ChildFocused()}.Layout(gtx, func(gtx C) D {
		defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
		pointer.InputOp{
			Tag:   &s.ScrollTable.tag,
			Types: pointer.Press,
		}.Add(gtx.Ops)
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

func (s ScrollTableStyle) layoutTable(gtx C, p image.Point) {
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()

	if s.ScrollTable.requestFocus {
		s.ScrollTable.requestFocus = false
		key.FocusOp{Tag: &s.ScrollTable.tag}.Add(gtx.Ops)
	}
	key.InputOp{Tag: &s.ScrollTable.tag, Keys: "←|→|↑|↓|Shift-←|Shift-→|Shift-↑|Shift-↓|Ctrl-←|Ctrl-→|Ctrl-↑|Ctrl-↓|Ctrl-Shift-←|Ctrl-Shift-→|Ctrl-Shift-↑|Ctrl-Shift-↓|Alt-←|Alt-→|Alt-↑|Alt-↓|Alt-Shift-←|Alt-Shift-→|Alt-Shift-↑|Alt-Shift-↓|⇱|⇲|Shift-⇱|Shift-⇲|⌫|⌦|⇞|⇟|Shift-⇞|Shift-⇟|Ctrl-C|Ctrl-V|Ctrl-X|Shift-,|Shift-."}.Add(gtx.Ops)
	cellWidth := gtx.Dp(s.CellWidth)
	cellHeight := gtx.Dp(s.CellHeight)

	gtx.Constraints = layout.Exact(image.Pt(cellWidth, cellHeight))

	colP := s.ColTitleStyle.dragList.List.Position
	rowP := s.RowTitleStyle.dragList.List.Position
	defer op.Offset(image.Pt(-colP.Offset, -rowP.Offset)).Push(gtx.Ops).Pop()
	for x := colP.First; x < colP.First+colP.Count; x++ {
		offs := op.Offset(image.Point{}).Push(gtx.Ops)
		for y := rowP.First; y < rowP.First+rowP.Count; y++ {
			s.element(gtx, x, y)
			op.Offset(image.Pt(0, cellHeight)).Add(gtx.Ops)
		}
		offs.Pop()
		op.Offset(image.Pt(cellWidth, 0)).Add(gtx.Ops)
	}
}

func (s *ScrollTableStyle) layoutRowTitles(gtx C, p image.Point) {
	defer op.Offset(image.Pt(0, p.Y)).Push(gtx.Ops).Pop()
	gtx.Constraints.Min.X = p.X
	gtx.Constraints.Max.Y -= p.Y
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	key.InputOp{Tag: &s.ScrollTable.rowTag, Keys: "→"}.Add(gtx.Ops)
	s.RowTitleStyle.Layout(gtx)
}

func (s *ScrollTableStyle) layoutColTitles(gtx C, p image.Point) {
	defer op.Offset(image.Pt(p.X, 0)).Push(gtx.Ops).Pop()
	gtx.Constraints.Min.Y = p.Y
	gtx.Constraints.Max.X -= p.X
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	key.InputOp{Tag: &s.ScrollTable.colTag, Keys: "↓"}.Add(gtx.Ops)
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
			clipboard.WriteOp{Text: string(contents)}.Add(gtx.Ops)
			if e.Name == "X" {
				s.Table.Clear()
			}
			return
		}
	case "V":
		if e.Modifiers.Contain(key.ModShortcut) {
			clipboard.ReadOp{Tag: &s.tag}.Add(gtx.Ops)
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
	case ".":
		s.Table.Add(1)
	case ",":
		s.Table.Add(-1)
	}
	if !e.Modifiers.Contain(key.ModShift) {
		s.Table.SetCursor2(s.Table.Cursor())
	}
	s.ColTitleList.EnsureVisible(s.Table.Cursor().X)
	s.RowTitleList.EnsureVisible(s.Table.Cursor().Y)
	s.cursorMoved = true
}
