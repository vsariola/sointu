package gioui

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"gioui.org/io/key"
	"gioui.org/io/pointer"
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

type OrderEditor struct {
	list         *layout.List
	titleList    *DragList
	scrollBar    *ScrollBar
	tag          bool
	focused      bool
	requestFocus bool
}

func NewOrderEditor() *OrderEditor {
	return &OrderEditor{
		list:      &layout.List{Axis: layout.Vertical},
		titleList: &DragList{List: &layout.List{Axis: layout.Horizontal}},
		scrollBar: &ScrollBar{Axis: layout.Vertical},
	}
}

func (oe *OrderEditor) Focus() {
	oe.requestFocus = true
}

func (oe *OrderEditor) Focused() bool {
	return oe.focused
}

func (oe *OrderEditor) Layout(gtx C, t *Tracker) D {
	return Surface{Gray: 24, Focus: oe.focused}.Layout(gtx, func(gtx C) D {
		return oe.doLayout(gtx, t)
	})
}

func (oe *OrderEditor) doLayout(gtx C, t *Tracker) D {
	for _, e := range gtx.Events(&oe.tag) {
		switch e := e.(type) {
		case key.FocusEvent:
			oe.focused = e.Focus
		case pointer.Event:
			if e.Type == pointer.Press {
				key.FocusOp{Tag: &oe.tag}.Add(gtx.Ops)
			}
		case key.Event:
			if e.State != key.Press {
				continue
			}
			switch e.Name {
			case key.NameDeleteForward, key.NameDeleteBackward:
				if e.Modifiers.Contain(key.ModShortcut) {
					t.DeleteOrderRow(e.Name == key.NameDeleteForward)
				} else {
					t.DeletePatternSelection()
					if !(t.NoteTracking() && t.Playing()) && t.Step.Value > 0 {
						t.SetCursor(t.Cursor().AddPatterns(1))
						t.SetSelectionCorner(t.Cursor())
					}
				}
			case "Space":
				if !t.Playing() {
					t.SetNoteTracking(!e.Modifiers.Contain(key.ModShortcut))
					startRow := t.Cursor().ScoreRow
					startRow.Row = 0
					t.PlayFromPosition(startRow)
				} else {
					t.SetPlaying(false)
				}
			case key.NameReturn:
				t.AddOrderRow(!e.Modifiers.Contain(key.ModShortcut))
			case key.NameUpArrow:
				cursor := t.Cursor()
				if e.Modifiers.Contain(key.ModShortcut) {
					cursor.ScoreRow = tracker.ScoreRow{}
				} else {
					cursor.Row -= t.Song().Score.RowsPerPattern
				}
				t.SetNoteTracking(false)
				t.SetCursor(cursor)
			case key.NameDownArrow:
				cursor := t.Cursor()
				if e.Modifiers.Contain(key.ModShortcut) {
					cursor.Row = t.Song().Score.LengthInRows() - 1
				} else {
					cursor.Row += t.Song().Score.RowsPerPattern
				}
				t.SetNoteTracking(false)
				t.SetCursor(cursor)
			case key.NameLeftArrow:
				cursor := t.Cursor()
				if e.Modifiers.Contain(key.ModShortcut) {
					cursor.Track = 0
				} else {
					cursor.Track--
				}
				t.SetCursor(cursor)
			case key.NameRightArrow:
				cursor := t.Cursor()
				if e.Modifiers.Contain(key.ModShortcut) {
					cursor.Track = len(t.Song().Score.Tracks) - 1
				} else {
					cursor.Track++
				}
				t.SetCursor(cursor)
			case "+":
				t.AdjustPatternNumber(1, e.Modifiers.Contain(key.ModShortcut))
				continue
			case "-":
				t.AdjustPatternNumber(-1, e.Modifiers.Contain(key.ModShortcut))
				continue
			case key.NameHome:
				cursor := t.Cursor()
				cursor.Track = 0
				t.SetCursor(cursor)
			case key.NameEnd:
				cursor := t.Cursor()
				cursor.Track = len(t.Song().Score.Tracks) - 1
				t.SetCursor(cursor)
			}
			if (e.Name != key.NameLeftArrow &&
				e.Name != key.NameRightArrow &&
				e.Name != key.NameUpArrow &&
				e.Name != key.NameDownArrow) ||
				!e.Modifiers.Contain(key.ModShift) {
				t.SetSelectionCorner(t.Cursor())
			}
			if e.Modifiers.Contain(key.ModShortcut) {
				continue
			}
			if iv, err := strconv.Atoi(e.Name); err == nil {
				t.SetCurrentPattern(iv)
				if !(t.NoteTracking() && t.Playing()) && t.Step.Value > 0 {
					t.SetCursor(t.Cursor().AddPatterns(1))
					t.SetSelectionCorner(t.Cursor())
				}
			}
			if b := int(e.Name[0]) - 'A'; len(e.Name) == 1 && b >= 0 && b < 26 {
				t.SetCurrentPattern(b + 10)
				if !(t.NoteTracking() && t.Playing()) && t.Step.Value > 0 {
					t.SetCursor(t.Cursor().AddPatterns(1))
					t.SetSelectionCorner(t.Cursor())
				}
			}
		}
	}
	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	if oe.requestFocus {
		oe.requestFocus = false
		key.FocusOp{Tag: &oe.tag}.Add(gtx.Ops)
	}
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
	pointer.InputOp{Tag: &oe.tag,
		Types: pointer.Press,
	}.Add(gtx.Ops)

	key.InputOp{Tag: &oe.tag, Keys: "←|→|↑|↓|Shift-←|Shift-→|Shift-↑|Shift-↓|⏎|⇱|⇲|⌫|⌦|Ctrl-⌫|Ctrl-⌦|+|-|Space|0|1|2|3|4|5|6|7|8|9|A|B|C|D|E|F|G|H|I|J|K|L|M|N|O|P|Q|R|S|T|U|V|W|X|Y|Z"}.Add(gtx.Ops)

	patternRect := tracker.ScoreRect{
		Corner1: tracker.ScorePoint{ScoreRow: tracker.ScoreRow{Pattern: t.Cursor().Pattern}, Track: t.Cursor().Track},
		Corner2: tracker.ScorePoint{ScoreRow: tracker.ScoreRow{Pattern: t.SelectionCorner().Pattern}, Track: t.SelectionCorner().Track},
	}

	// draw the single letter titles for tracks
	{
		gtx := gtx
		stack := op.Offset(image.Pt(patternRowMarkerWidth, 0)).Push(gtx.Ops)
		gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X-patternRowMarkerWidth, patternCellHeight))
		elem := func(gtx C, i int) D {
			gtx.Constraints = layout.Exact(image.Pt(patternCellWidth, patternCellHeight))
			instr, err := t.Song().Patch.InstrumentForVoice(t.Song().Score.FirstVoiceForTrack(i))
			var title string
			if err == nil && len(t.Song().Patch[instr].Name) > 0 {
				title = string(t.Song().Patch[instr].Name[0])
			} else {
				title = "?"
			}
			LabelStyle{Alignment: layout.N, Text: title, FontSize: unit.Sp(12), Color: mediumEmphasisTextColor, Shaper: t.TextShaper}.Layout(gtx)
			return D{Size: gtx.Constraints.Min}
		}
		style := FilledDragList(t.Theme, oe.titleList, len(t.Song().Score.Tracks), elem, t.SwapTracks)
		style.HoverColor = transparent
		style.SelectedColor = transparent
		style.Layout(gtx)
		stack.Pop()
	}
	op.Offset(image.Pt(0, patternCellHeight)).Add(gtx.Ops)
	gtx.Constraints.Max.Y -= patternCellHeight
	gtx.Constraints.Min.Y -= patternCellHeight
	element := func(gtx C, j int) D {
		if playPos := t.PlayPosition(); t.Playing() && j == playPos.Pattern {
			paint.FillShape(gtx.Ops, patternPlayColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, patternCellHeight)}.Op())
		}
		paint.ColorOp{Color: rowMarkerPatternTextColor}.Add(gtx.Ops)
		widget.Label{}.Layout(gtx, t.TextShaper, trackerFont, trackerFontSize, strings.ToUpper(fmt.Sprintf("%02x", j)), op.CallOp{})
		stack := op.Offset(image.Pt(patternRowMarkerWidth, 0)).Push(gtx.Ops)
		for i, track := range t.Song().Score.Tracks {
			paint.FillShape(gtx.Ops, patternCellColor, clip.Rect{Min: image.Pt(1, 1), Max: image.Pt(patternCellWidth-1, patternCellHeight-1)}.Op())
			paint.ColorOp{Color: patternTextColor}.Add(gtx.Ops)
			if j >= 0 && j < len(track.Order) && track.Order[j] >= 0 {
				gtx := gtx
				gtx.Constraints.Max.X = patternCellWidth
				op.Offset(image.Pt(0, -2)).Add(gtx.Ops)
				widget.Label{Alignment: text.Middle}.Layout(gtx, t.TextShaper, trackerFont, trackerFontSize, patternIndexToString(track.Order[j]), op.CallOp{})
				op.Offset(image.Pt(0, 2)).Add(gtx.Ops)
			}
			point := tracker.ScorePoint{Track: i, ScoreRow: tracker.ScoreRow{Pattern: j}}
			if oe.focused || t.TrackEditor.Focused() {
				if patternRect.Contains(point) {
					color := inactiveSelectionColor
					if oe.focused {
						color = selectionColor
						if point.Pattern == t.Cursor().Pattern && point.Track == t.Cursor().Track {
							color = cursorColor
						}
					}
					paint.FillShape(gtx.Ops, color, clip.Rect{Max: image.Pt(patternCellWidth, patternCellHeight)}.Op())
				}
			}
			op.Offset(image.Pt(patternCellWidth, 0)).Add(gtx.Ops)
		}
		stack.Pop()
		return D{Size: image.Pt(gtx.Constraints.Max.X, patternCellHeight)}
	}

	return layout.Stack{Alignment: layout.NE}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			return oe.list.Layout(gtx, t.Song().Score.Length, element)
		}),
		layout.Expanded(func(gtx C) D {
			return oe.scrollBar.Layout(gtx, unit.Dp(10), t.Song().Score.Length, &oe.list.Position)
		}),
	)
}

func patternIndexToString(index int) string {
	if index < 0 {
		return ""
	} else if index < 10 {
		return string('0' + byte(index))
	}
	return string('A' + byte(index-10))
}
