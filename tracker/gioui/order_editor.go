package gioui

import (
	"image"
	"math"
	"strconv"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/vsariola/sointu/tracker"
)

const patternCellHeight = unit.Dp(16)
const patternCellWidth = unit.Dp(16)
const orderTitleHeight = unit.Dp(52)

type OrderEditor struct {
	scrollTable *ScrollTable
	tag         struct{}
}

var patternIndexStrings [36]string

func init() {
	for i := 0; i < 10; i++ {
		patternIndexStrings[i] = string('0' + byte(i))
	}
	for i := 10; i < 36; i++ {
		patternIndexStrings[i] = string('A' + byte(i-10))
	}
}

func NewOrderEditor(m *tracker.Model) *OrderEditor {
	return &OrderEditor{
		scrollTable: NewScrollTable(
			m.Order().Table(),
			m.Track().List(),
			m.Order().RowList(),
		),
	}
}

func (oe *OrderEditor) Layout(gtx C) D {
	t := TrackerFromContext(gtx)
	if oe.scrollTable.CursorMoved() {
		cursor := t.TrackEditor.scrollTable.Table.Cursor()
		t.TrackEditor.scrollTable.ColTitleList.CenterOn(cursor.X)
		t.TrackEditor.scrollTable.RowTitleList.CenterOn(cursor.Y)
	}

	oe.handleEvents(gtx, t)

	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, &oe.tag)

	colTitle := func(gtx C, i int) D {
		h := gtx.Dp(orderTitleHeight)
		defer op.Offset(image.Pt(0, -2)).Push(gtx.Ops).Pop()
		defer op.Affine(f32.Affine2D{}.Rotate(f32.Pt(0, 0), -90*math.Pi/180).Offset(f32.Point{X: 0, Y: float32(h)})).Push(gtx.Ops).Pop()
		gtx.Constraints = layout.Exact(image.Pt(1e6, 1e6))
		Label(t.Theme, &t.Theme.OrderEditor.TrackTitle, t.Model.Track().Item(i).Title).Layout(gtx)
		return D{Size: image.Pt(gtx.Dp(patternCellWidth), h)}
	}

	rowTitleBg := func(gtx C, j int) D {
		if t.Model.Play().Started().Value() && j == t.Play().Position().OrderRow {
			paint.FillShape(gtx.Ops, t.Theme.OrderEditor.Play, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(patternCellHeight))}.Op())
		}
		return D{}
	}

	rowMarkerPatternTextColorOp := colorOp(gtx, t.Theme.OrderEditor.RowTitle.Color)
	loopMarkerColorOp := colorOp(gtx, t.Theme.OrderEditor.Loop)

	rowTitle := func(gtx C, j int) D {
		w := gtx.Dp(unit.Dp(30))
		callOp := rowMarkerPatternTextColorOp
		if l := t.Play().Loop(); j >= l.Start && j < l.Start+l.Length {
			callOp = loopMarkerColorOp
		}
		defer op.Offset(image.Pt(0, -2)).Push(gtx.Ops).Pop()
		widget.Label{}.Layout(gtx, t.Theme.Material.Shaper, t.Theme.OrderEditor.RowTitle.Font, t.Theme.OrderEditor.RowTitle.TextSize, hexStr[j&255], callOp)
		return D{Size: image.Pt(w, gtx.Dp(patternCellHeight))}
	}

	selection := oe.scrollTable.Table.Range()
	cellColorOp := colorOp(gtx, t.Theme.OrderEditor.Cell.Color)

	cell := func(gtx C, x, y int) D {
		val := patternIndexToString(t.Model.Order().Value(tracker.Point{X: x, Y: y}))
		color := t.Theme.OrderEditor.CellBg
		point := tracker.Point{X: x, Y: y}
		if selection.Contains(point) {
			color = t.Theme.Selection.Inactive
			if gtx.Focused(oe.scrollTable) {
				color = t.Theme.Selection.Active
			}
			if point == oe.scrollTable.Table.Cursor() {
				color = t.Theme.Cursor.Inactive
				if gtx.Focused(oe.scrollTable) {
					color = t.Theme.Cursor.Active
				}
			}
		}
		paint.FillShape(gtx.Ops, color, clip.Rect{Min: image.Pt(1, 1), Max: image.Pt(gtx.Constraints.Min.X-1, gtx.Constraints.Min.X-1)}.Op())
		defer op.Offset(image.Pt(0, -2)).Push(gtx.Ops).Pop()
		widget.Label{Alignment: text.Middle}.Layout(gtx, t.Theme.Material.Shaper, t.Theme.OrderEditor.Cell.Font, t.Theme.OrderEditor.Cell.TextSize, val, cellColorOp)
		return D{Size: image.Pt(gtx.Dp(patternCellWidth), gtx.Dp(patternCellHeight))}
	}

	table := FilledScrollTable(t.Theme, oe.scrollTable)
	table.ColumnTitleHeight = orderTitleHeight

	return Surface{Height: 3, Focus: oe.scrollTable.TreeFocused(gtx)}.Layout(gtx, func(gtx C) D {
		return table.Layout(gtx, cell, colTitle, rowTitle, nil, rowTitleBg)
	})
}

func (oe *OrderEditor) handleEvents(gtx C, t *Tracker) {
	for {
		e, ok := gtx.Event(
			key.Filter{Focus: oe.scrollTable, Name: key.NameDeleteBackward, Required: key.ModShortcut},
			key.Filter{Focus: oe.scrollTable, Name: key.NameDeleteForward, Required: key.ModShortcut},
			key.Filter{Focus: oe.scrollTable, Name: key.NameReturn, Optional: key.ModShortcut},
			key.Filter{Focus: oe.scrollTable, Name: "0"},
			key.Filter{Focus: oe.scrollTable, Name: "1"},
			key.Filter{Focus: oe.scrollTable, Name: "2"},
			key.Filter{Focus: oe.scrollTable, Name: "3"},
			key.Filter{Focus: oe.scrollTable, Name: "4"},
			key.Filter{Focus: oe.scrollTable, Name: "5"},
			key.Filter{Focus: oe.scrollTable, Name: "6"},
			key.Filter{Focus: oe.scrollTable, Name: "7"},
			key.Filter{Focus: oe.scrollTable, Name: "8"},
			key.Filter{Focus: oe.scrollTable, Name: "9"},
			key.Filter{Focus: oe.scrollTable, Name: "A"},
			key.Filter{Focus: oe.scrollTable, Name: "B"},
			key.Filter{Focus: oe.scrollTable, Name: "C"},
			key.Filter{Focus: oe.scrollTable, Name: "D"},
			key.Filter{Focus: oe.scrollTable, Name: "E"},
			key.Filter{Focus: oe.scrollTable, Name: "F"},
			key.Filter{Focus: oe.scrollTable, Name: "G"},
			key.Filter{Focus: oe.scrollTable, Name: "H"},
			key.Filter{Focus: oe.scrollTable, Name: "I"},
			key.Filter{Focus: oe.scrollTable, Name: "J"},
			key.Filter{Focus: oe.scrollTable, Name: "K"},
			key.Filter{Focus: oe.scrollTable, Name: "L"},
			key.Filter{Focus: oe.scrollTable, Name: "M"},
			key.Filter{Focus: oe.scrollTable, Name: "N"},
			key.Filter{Focus: oe.scrollTable, Name: "O"},
			key.Filter{Focus: oe.scrollTable, Name: "P"},
			key.Filter{Focus: oe.scrollTable, Name: "Q"},
			key.Filter{Focus: oe.scrollTable, Name: "R"},
			key.Filter{Focus: oe.scrollTable, Name: "S"},
			key.Filter{Focus: oe.scrollTable, Name: "T"},
			key.Filter{Focus: oe.scrollTable, Name: "U"},
			key.Filter{Focus: oe.scrollTable, Name: "V"},
			key.Filter{Focus: oe.scrollTable, Name: "W"},
			key.Filter{Focus: oe.scrollTable, Name: "X"},
			key.Filter{Focus: oe.scrollTable, Name: "Y"},
			key.Filter{Focus: oe.scrollTable, Name: "Z"},
		)
		if !ok {
			break
		}
		if e, ok := e.(key.Event); ok {
			if e.State != key.Press {
				continue
			}
			oe.command(t, e)
		}
	}
}

func (oe *OrderEditor) command(t *Tracker, e key.Event) {
	switch e.Name {
	case key.NameDeleteBackward:
		if e.Modifiers.Contain(key.ModShortcut) {
			t.Model.Order().DeleteRow(true).Do()
		}
	case key.NameDeleteForward:
		if e.Modifiers.Contain(key.ModShortcut) {
			t.Model.Order().DeleteRow(false).Do()
		}
	case key.NameReturn:
		t.Model.Order().AddRow(e.Modifiers.Contain(key.ModShortcut)).Do()
	}
	if iv, err := strconv.Atoi(string(e.Name)); err == nil {
		t.Model.Order().SetValue(oe.scrollTable.Table.Cursor(), iv)
		oe.scrollTable.EnsureCursorVisible()
	}
	if b := int(e.Name[0]) - 'A'; len(e.Name) == 1 && b >= 0 && b < 26 {
		t.Model.Order().SetValue(oe.scrollTable.Table.Cursor(), b+10)
		oe.scrollTable.EnsureCursorVisible()
	}
}

func (t *OrderEditor) Tags(level int, yield TagYieldFunc) bool {
	return yield(level+1, t.scrollTable.RowTitleList) && yield(level+1, t.scrollTable.ColTitleList) && yield(level, t.scrollTable)
}

func patternIndexToString(index int) string {
	if index < 0 {
		return ""
	} else if index < len(patternIndexStrings) {
		return patternIndexStrings[index]
	}
	return "?"
}
