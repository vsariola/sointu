package gioui

import (
	"fmt"
	"image"
	"strings"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"github.com/vsariola/sointu/tracker"
)

const patternCellHeight = 16
const patternCellWidth = 16
const patternRowMarkerWidth = 30

var patternPointerTag = false

func (t *Tracker) layoutPatterns(gtx C) D {
	defer op.Save(gtx.Ops).Load()
	clip.Rect{Max: gtx.Constraints.Max}.Add(gtx.Ops)
	for _, ev := range gtx.Events(&patternPointerTag) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		if e.Type == pointer.Press {
			t.SetEditMode(tracker.EditPatterns)
		}
	}
	rect := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
	pointer.Rect(rect).Add(gtx.Ops)
	pointer.InputOp{Tag: &patternPointerTag,
		Types: pointer.Press,
	}.Add(gtx.Ops)
	patternRect := tracker.SongRect{
		Corner1: tracker.SongPoint{SongRow: tracker.SongRow{Pattern: t.Cursor().Pattern}, Track: t.Cursor().Track},
		Corner2: tracker.SongPoint{SongRow: tracker.SongRow{Pattern: t.SelectionCorner().Pattern}, Track: t.SelectionCorner().Track},
	}
	for j := 0; j < t.Song().Score.Length; j++ {
		if playPos, ok := t.player.Position(); ok && j == playPos.Pattern {
			paint.FillShape(gtx.Ops, patternPlayColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, patternCellHeight)}.Op())
		}
		paint.ColorOp{Color: rowMarkerPatternTextColor}.Add(gtx.Ops)
		widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, strings.ToUpper(fmt.Sprintf("%02x", j)))
		stack := op.Save(gtx.Ops)
		op.Offset(f32.Pt(patternRowMarkerWidth, 0)).Add(gtx.Ops)
		for i, track := range t.Song().Score.Tracks {
			paint.ColorOp{Color: patternTextColor}.Add(gtx.Ops)
			if j < len(track.Order) && track.Order[j] >= 0 {
				widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, patternIndexToString(track.Order[j]))
			}
			point := tracker.SongPoint{Track: i, SongRow: tracker.SongRow{Pattern: j}}
			if t.EditMode() == tracker.EditPatterns || t.EditMode() == tracker.EditTracks {
				if patternRect.Contains(point) {
					color := inactiveSelectionColor
					if t.EditMode() == tracker.EditPatterns {
						color = selectionColor
						if point.Pattern == t.Cursor().Pattern && point.Track == t.Cursor().Track {
							color = cursorColor
						}
					}
					paint.FillShape(gtx.Ops, color, clip.Rect{Max: image.Pt(patternCellWidth, patternCellHeight)}.Op())
				}
			}
			op.Offset(f32.Pt(patternCellWidth, 0)).Add(gtx.Ops)
		}
		stack.Load()
		op.Offset(f32.Pt(0, patternCellHeight)).Add(gtx.Ops)
	}
	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func patternIndexToString(index int) string {
	if index < 0 {
		return ""
	} else if index < 10 {
		return string('0' + byte(index))
	}
	return string('A' + byte(index-10))
}
