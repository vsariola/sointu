package gioui

import (
	"fmt"
	"image"
	"math"
	"strconv"
	"strings"

	"gioui.org/f32"
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

const patternCellHeight = 16
const patternCellWidth = 16
const patternRowMarkerWidth = 30
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
			m.Tracks().List(),
			m.OrderRows().List(),
		),
	}
}

func (oe *OrderEditor) Layout(gtx C, t *Tracker) D {
	if oe.scrollTable.CursorMoved() {
		cursor := t.TrackEditor.scrollTable.Table.Cursor()
		t.TrackEditor.scrollTable.ColTitleList.CenterOn(cursor.X)
		t.TrackEditor.scrollTable.RowTitleList.CenterOn(cursor.Y)
	}

	for _, e := range gtx.Events(&oe.tag) {
		switch e := e.(type) {
		case key.Event:
			if e.State != key.Press {
				continue
			}
			oe.command(gtx, t, e)
		}
	}
	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
	key.InputOp{Tag: &oe.tag, Keys: "Ctrl-⌫|Ctrl-⌦|⏎|Ctrl-⏎|0|1|2|3|4|5|6|7|8|9|A|B|C|D|E|F|G|H|I|J|K|L|M|N|O|P|Q|R|S|T|U|V|W|X|Y|Z"}.Add(gtx.Ops)

	colTitle := func(gtx C, i int) D {
		h := gtx.Dp(orderTitleHeight)
		defer op.Offset(image.Pt(0, -2)).Push(gtx.Ops).Pop()
		defer op.Affine(f32.Affine2D{}.Rotate(f32.Pt(0, 0), -90*math.Pi/180).Offset(f32.Point{X: 0, Y: float32(h)})).Push(gtx.Ops).Pop()
		gtx.Constraints = layout.Exact(image.Pt(1e6, 1e6))
		title := t.Model.Order().Title(i)
		LabelStyle{Alignment: layout.NW, Text: title, FontSize: unit.Sp(12), Color: mediumEmphasisTextColor, Shaper: t.Theme.Shaper}.Layout(gtx)
		return D{Size: image.Pt(patternCellWidth, h)}
	}

	rowTitle := func(gtx C, j int) D {
		w := gtx.Dp(unit.Dp(30))
		if playPos := t.PlayPosition(); t.SongPanel.PlayingBtn.Bool.Value() && j == playPos.OrderRow {
			paint.FillShape(gtx.Ops, patternPlayColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, patternCellHeight)}.Op())
		}
		color := rowMarkerPatternTextColor
		if l := t.Loop(); j >= l.Start && j < l.Start+l.Length {
			color = loopMarkerColor
		}
		paint.ColorOp{Color: color}.Add(gtx.Ops)
		defer op.Offset(image.Pt(0, -2)).Push(gtx.Ops).Pop()
		widget.Label{}.Layout(gtx, t.Theme.Shaper, trackerFont, trackerFontSize, strings.ToUpper(fmt.Sprintf("%02x", j)), op.CallOp{})
		return D{Size: image.Pt(w, patternCellHeight)}
	}

	selection := oe.scrollTable.Table.Range()

	cell := func(gtx C, x, y int) D {
		val := patternIndexToString(t.Model.Order().Value(tracker.Point{X: x, Y: y}))
		color := patternCellColor
		point := tracker.Point{X: x, Y: y}
		if selection.Contains(point) {
			color = inactiveSelectionColor
			if oe.scrollTable.Focused() {
				color = selectionColor
				if point == oe.scrollTable.Table.Cursor() {
					color = cursorColor
				}
			}
		}
		paint.FillShape(gtx.Ops, color, clip.Rect{Min: image.Pt(1, 1), Max: image.Pt(gtx.Constraints.Min.X-1, gtx.Constraints.Min.X-1)}.Op())
		paint.ColorOp{Color: patternTextColor}.Add(gtx.Ops)
		defer op.Offset(image.Pt(0, -2)).Push(gtx.Ops).Pop()
		widget.Label{Alignment: text.Middle}.Layout(gtx, t.Theme.Shaper, trackerFont, trackerFontSize, val, op.CallOp{})
		return D{Size: image.Pt(patternCellWidth, patternCellHeight)}
	}

	table := FilledScrollTable(t.Theme, oe.scrollTable, cell, colTitle, rowTitle, nil, nil)
	table.ColumnTitleHeight = orderTitleHeight

	return table.Layout(gtx)
}

func (oe *OrderEditor) command(gtx C, t *Tracker, e key.Event) {
	switch e.Name {
	case key.NameDeleteBackward:
		if e.Modifiers.Contain(key.ModShortcut) {
			t.Model.DeleteOrderRow(true).Do()
		}
	case key.NameDeleteForward:
		if e.Modifiers.Contain(key.ModShortcut) {
			t.Model.DeleteOrderRow(false).Do()
		}
	case key.NameReturn:
		if e.Modifiers.Contain(key.ModShortcut) {
			oe.scrollTable.Table.MoveCursor(0, -1)
			oe.scrollTable.Table.SetCursor2(oe.scrollTable.Table.Cursor())
		}
		t.Model.AddOrderRow(!e.Modifiers.Contain(key.ModShortcut)).Do()
	}
	if iv, err := strconv.Atoi(e.Name); err == nil {
		t.Model.Order().SetValue(oe.scrollTable.Table.Cursor(), iv)
		oe.scrollTable.EnsureCursorVisible()
	}
	if b := int(e.Name[0]) - 'A'; len(e.Name) == 1 && b >= 0 && b < 26 {
		t.Model.Order().SetValue(oe.scrollTable.Table.Cursor(), b+10)
		oe.scrollTable.EnsureCursorVisible()
	}
}

func patternIndexToString(index int) string {
	if index < 0 {
		return ""
	} else if index < len(patternIndexStrings) {
		return patternIndexStrings[index]
	}
	return "?"
}
