package gioui

import (
	"fmt"
	"image"
	"strings"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
)

const rowMarkerWidth = 50

func (t *Tracker) layoutRowMarkers(gtx C) D {
	gtx.Constraints.Min.X = rowMarkerWidth
	paint.FillShape(gtx.Ops, rowMarkerSurfaceColor, clip.Rect{
		Max: gtx.Constraints.Max,
	}.Op())
	//defer op.Save(gtx.Ops).Load()
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	op.Offset(image.Pt(0, (gtx.Constraints.Max.Y-trackRowHeight)/2)).Add(gtx.Ops)
	cursorSongRow := t.Cursor().Pattern*t.Song().Score.RowsPerPattern + t.Cursor().Row
	playPos := t.PlayPosition()
	playSongRow := playPos.Pattern*t.Song().Score.RowsPerPattern + playPos.Row
	op.Offset(image.Pt(0, (-1*trackRowHeight)*(cursorSongRow))).Add(gtx.Ops)
	beatMarkerDensity := t.Song().RowsPerBeat
	for beatMarkerDensity <= 2 {
		beatMarkerDensity *= 2
	}
	for i := 0; i < t.Song().Score.Length; i++ {
		for j := 0; j < t.Song().Score.RowsPerPattern; j++ {
			songRow := i*t.Song().Score.RowsPerPattern + j
			if mod(songRow, beatMarkerDensity*2) == 0 {
				paint.FillShape(gtx.Ops, twoBeatHighlight, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, trackRowHeight)}.Op())
			} else if mod(songRow, beatMarkerDensity) == 0 {
				paint.FillShape(gtx.Ops, oneBeatHighlight, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, trackRowHeight)}.Op())
			}
			if t.Playing() && songRow == playSongRow {
				paint.FillShape(gtx.Ops, trackerPlayColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, trackRowHeight)}.Op())
			}
			if j == 0 {
				paint.ColorOp{Color: rowMarkerPatternTextColor}.Add(gtx.Ops)
				widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, strings.ToUpper(fmt.Sprintf("%02x", i)), op.CallOp{})
			}
			if t.TrackEditor.Focused() && songRow == cursorSongRow {
				paint.ColorOp{Color: trackerActiveTextColor}.Add(gtx.Ops)
			} else {
				paint.ColorOp{Color: rowMarkerRowTextColor}.Add(gtx.Ops)
			}
			op.Offset(image.Pt(rowMarkerWidth/2, 0)).Add(gtx.Ops)
			widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, strings.ToUpper(fmt.Sprintf("%02x", j)), op.CallOp{})
			op.Offset(image.Pt(-rowMarkerWidth/2, trackRowHeight)).Add(gtx.Ops)
		}
	}
	return layout.Dimensions{Size: image.Pt(rowMarkerWidth, gtx.Constraints.Max.Y)}
}

func mod(a, b int) int {
	m := a % b
	if a < 0 && b < 0 {
		m -= b
	}
	if a < 0 && b > 0 {
		m += b
	}
	return m
}
