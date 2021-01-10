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
	"github.com/vsariola/sointu"
)

const patternCellHeight = 16
const patternCellWidth = 16
const patternVisibleTracks = 8
const patternRowMarkerWidth = 30

func (t *Tracker) layoutPatterns(tracks []sointu.Track, activeTrack, cursorPattern, cursorCol, playingPattern int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = patternCellWidth*patternVisibleTracks + patternRowMarkerWidth
		gtx.Constraints.Max.X = patternCellWidth*patternVisibleTracks + patternRowMarkerWidth
		defer op.Push(gtx.Ops).Pop()
		clip.Rect{Max: gtx.Constraints.Max}.Add(gtx.Ops)
		paint.FillShape(gtx.Ops, patternSurfaceColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}.Op())
		for j := range tracks[0].Sequence {
			if j == playingPattern {
				paint.FillShape(gtx.Ops, patternPlayColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, patternCellHeight)}.Op())
			}
			paint.ColorOp{Color: rowMarkerPatternTextColor}.Add(gtx.Ops)
			widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, strings.ToUpper(fmt.Sprintf("%02x", j)))
			stack := op.Push(gtx.Ops)
			op.Offset(f32.Pt(patternRowMarkerWidth, 0)).Add(gtx.Ops)
			for i, track := range tracks {
				paint.ColorOp{Color: trackerTextColor}.Add(gtx.Ops)
				widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, fmt.Sprintf("%d", track.Sequence[j]))
				if activeTrack == i && j == cursorPattern {
					paint.FillShape(gtx.Ops, patternCursorColor, clip.Rect{Max: image.Pt(patternCellWidth, patternCellHeight)}.Op())
				}
				op.Offset(f32.Pt(patternCellWidth, 0)).Add(gtx.Ops)
			}
			stack.Pop()
			op.Offset(f32.Pt(0, patternCellHeight)).Add(gtx.Ops)
		}
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}

func patternIndexToString(index byte) string {
	if index < 10 {
		return string([]byte{'0' + index})
	}
	return string([]byte{'A' + index - 10})
}
