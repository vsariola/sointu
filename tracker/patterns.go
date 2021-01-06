package tracker

import (
	"fmt"
	"image"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"github.com/vsariola/sointu"
)

const patternCellHeight = 12
const patternCellWidth = 16

func (t *Tracker) layoutPatterns(tracks []sointu.Track, activeTrack, cursorPattern, cursorCol, playingPattern int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = patternCellWidth * len(tracks)
		gtx.Constraints.Max.X = patternCellWidth * len(tracks)
		gtx.Constraints.Max.Y = 50
		defer op.Push(gtx.Ops).Pop()
		clip.Rect{Max: gtx.Constraints.Max}.Add(gtx.Ops)
		paint.FillShape(gtx.Ops, panelColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, trackRowHeight)}.Op())
		for i, track := range tracks {
			pop := op.Push(gtx.Ops)
			clip.Rect{Max: gtx.Constraints.Max}.Add(gtx.Ops)
			if activeTrack == i {
				paint.FillShape(gtx.Ops, activeTrackColor, clip.Rect{
					Max: gtx.Constraints.Max,
				}.Op())
			} else {
				paint.FillShape(gtx.Ops, inactiveTrackColor, clip.Rect{
					Max: gtx.Constraints.Max,
				}.Op())
			}
			for j, p := range track.Sequence {
				if j == playingPattern {
					paint.FillShape(gtx.Ops, patternPlayColor, clip.Rect{Max: image.Pt(trackWidth, trackRowHeight)}.Op())
				}
				if j == cursorPattern {
					paint.ColorOp{Color: trackerActiveTextColor}.Add(gtx.Ops)
				} else {
					paint.ColorOp{Color: trackerTextColor}.Add(gtx.Ops)
				}
				widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, fmt.Sprintf("%d", p))
				op.Offset(f32.Pt(0, patternCellHeight)).Add(gtx.Ops)
			}
			pop.Pop()
			op.Offset(f32.Pt(patternCellWidth, 0)).Add(gtx.Ops)
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
