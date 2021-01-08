package tracker

import (
	"fmt"
	"image"
	"strings"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
)

const rowMarkerWidth = 50

func (t *Tracker) layoutRowMarkers(patternRows, sequenceLength, cursorRow, cursorPattern, cursorCol, playRow, playPattern int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = rowMarkerWidth
		gtx.Constraints.Max.X = rowMarkerWidth
		paint.FillShape(gtx.Ops, inactiveTrackColor, clip.Rect{
			Max: gtx.Constraints.Max,
		}.Op())
		defer op.Push(gtx.Ops).Pop()
		clip.Rect{Max: gtx.Constraints.Max}.Add(gtx.Ops)
		op.Offset(f32.Pt(0, float32(gtx.Constraints.Max.Y/2)-trackRowHeight)).Add(gtx.Ops)
		cursorSongRow := cursorPattern*patternRows + cursorRow
		playSongRow := playPattern*patternRows + playRow
		op.Offset(f32.Pt(0, (-1*trackRowHeight)*float32(cursorSongRow))).Add(gtx.Ops)
		for i := 0; i < sequenceLength; i++ {
			for j := 0; j < patternRows; j++ {
				songRow := i*patternRows + j
				if songRow == playSongRow {
					paint.FillShape(gtx.Ops, trackerPlayColor, clip.Rect{Max: image.Pt(trackWidth, trackRowHeight)}.Op())
				}
				if j == 0 {
					paint.ColorOp{Color: trackerPatMarker}.Add(gtx.Ops)
					widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, strings.ToUpper(fmt.Sprintf("%02x", i)))
				}
				if songRow == cursorSongRow {
					paint.ColorOp{Color: trackerActiveTextColor}.Add(gtx.Ops)
				} else {
					paint.ColorOp{Color: trackerPatternRowTextColor}.Add(gtx.Ops)
				}
				op.Offset(f32.Pt(rowMarkerWidth/2, 0)).Add(gtx.Ops)
				widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, strings.ToUpper(fmt.Sprintf("%02x", j)))
				op.Offset(f32.Pt(-rowMarkerWidth/2, trackRowHeight)).Add(gtx.Ops)
			}
		}
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}
