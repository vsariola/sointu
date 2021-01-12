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

const trackRowHeight = 16
const trackWidth = 54
const patmarkWidth = 16

func (t *Tracker) layoutTrack(patterns [][]byte, sequence []byte, active bool, hex bool, cursorRow, cursorPattern, cursorCol, playRow, playPattern int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = trackWidth
		gtx.Constraints.Max.X = trackWidth
		paint.FillShape(gtx.Ops, trackSurfaceColor, clip.Rect{
			Max: gtx.Constraints.Max,
		}.Op())
		defer op.Push(gtx.Ops).Pop()
		clip.Rect{Max: gtx.Constraints.Max}.Add(gtx.Ops)
		op.Offset(f32.Pt(0, float32(gtx.Constraints.Max.Y/2)-trackRowHeight)).Add(gtx.Ops)
		// TODO: this is a time bomb; as soon as one of the patterns is not the same length as rest. Find a solution
		// to fix the pattern lengths to a constant value
		cursorSongRow := cursorPattern*len(patterns[0]) + cursorRow
		playSongRow := playPattern*len(patterns[0]) + playRow
		op.Offset(f32.Pt(0, (-1*trackRowHeight)*float32(cursorSongRow))).Add(gtx.Ops)
		for i, s := range sequence {
			if cursorPattern == i && active {
				paint.FillShape(gtx.Ops, activeTrackColor, clip.Rect{Max: image.Pt(trackWidth, trackRowHeight*len(patterns[0]))}.Op())
			}
			for j, c := range patterns[s] {
				songRow := i*len(patterns[0]) + j
				if songRow == playSongRow {
					paint.FillShape(gtx.Ops, trackerPlayColor, clip.Rect{Max: image.Pt(trackWidth, trackRowHeight)}.Op())
				}
				if j == 0 {
					paint.ColorOp{Color: trackerPatMarker}.Add(gtx.Ops)
					widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, patternIndexToString(s))
				}
				if songRow == cursorSongRow {
					paint.ColorOp{Color: trackerActiveTextColor}.Add(gtx.Ops)
				} else {
					paint.ColorOp{Color: trackerInactiveTextColor}.Add(gtx.Ops)
				}
				op.Offset(f32.Pt(patmarkWidth, 0)).Add(gtx.Ops)
				if hex {
					var text string
					switch c {
					case 0:
						text = "--"
					case 1:
						text = ".."
					default:
						text = fmt.Sprintf("%02x", c)
					}
					widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, strings.ToUpper(text))
					if active && songRow == cursorSongRow {
						paint.FillShape(gtx.Ops, trackerCursorColor, clip.Rect{Min: image.Pt(cursorCol*10, 0), Max: image.Pt(cursorCol*10+10, trackRowHeight)}.Op())
					}
				} else {
					widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, valueAsNote(c))
					if active && cursorCol == 0 && songRow == cursorSongRow {
						paint.FillShape(gtx.Ops, trackerCursorColor, clip.Rect{Max: image.Pt(30, trackRowHeight)}.Op())
					}
				}
				op.Offset(f32.Pt(-patmarkWidth, trackRowHeight)).Add(gtx.Ops)
			}
		}
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}
