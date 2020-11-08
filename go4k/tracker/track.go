package tracker

import (
	"fmt"
	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"image"
	"strings"
)

const trackRowHeight = 16
const trackWidth = 100

func (t *Tracker) layoutTrack(notes []byte, active bool, cursorRow, cursorCol int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = trackWidth
		gtx.Constraints.Max.X = trackWidth
		if active {
			paint.FillShape(gtx.Ops, activeTrackColor, clip.Rect{
				Max: gtx.Constraints.Max,
			}.Op())
		} else {
			paint.FillShape(gtx.Ops, inactiveTrackColor, clip.Rect{
				Max: gtx.Constraints.Max,
			}.Op())
		}
		defer op.Push(gtx.Ops).Pop()
		// clip.Rect{Max:gtx.Constraints.Max}.Add(gtx.Ops)
		op.Offset(f32.Pt(0, float32(gtx.Constraints.Max.Y/2)-trackRowHeight)).Add(gtx.Ops)
		paint.FillShape(gtx.Ops, panelColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, trackRowHeight)}.Op())
		if active {
			switch cursorCol {
			case 0:
				paint.FillShape(gtx.Ops, panelShadeColor, clip.Rect{Max: image.Pt(36, trackRowHeight)}.Op())
			case 1, 2:
				s := op.Push(gtx.Ops)
				op.Offset(f32.Pt(trackWidth/2+float32(cursorCol-1)*10, 0)).Add(gtx.Ops)
				paint.FillShape(gtx.Ops, panelShadeColor, clip.Rect{Max: image.Pt(10, trackRowHeight)}.Op())
				s.Pop()
			}
		}
		op.Offset(f32.Pt(0, (-1*trackRowHeight)*float32(cursorRow))).Add(gtx.Ops)
		for i, c := range notes {
			if i == cursorRow {
				paint.ColorOp{Color: trackerActiveTextColor}.Add(gtx.Ops)
			} else {
				paint.ColorOp{Color: trackerTextColor}.Add(gtx.Ops)
			}
			widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, valueAsNote(c))
			op.Offset(f32.Pt(trackWidth/2, 0)).Add(gtx.Ops)
			widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, strings.ToUpper(fmt.Sprintf("%02x", c)))
			op.Offset(f32.Pt(-trackWidth/2, trackRowHeight)).Add(gtx.Ops)
		}
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}
