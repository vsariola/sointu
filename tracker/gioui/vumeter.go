package gioui

import (
	"image"

	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/vsariola/sointu/tracker"
)

type VuMeter struct {
	Loudness tracker.Decibel
	Peak     [2]tracker.Decibel
	Range    float32
}

func (v VuMeter) Layout(gtx C) D {
	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(12))
	height := gtx.Dp(unit.Dp(6))
	for j := 0; j < 2; j++ {
		value := float32(v.Loudness) + v.Range
		if value > 0 {
			x := int(value/v.Range*float32(gtx.Constraints.Max.X) + 0.5)
			if x > gtx.Constraints.Max.X {
				x = gtx.Constraints.Max.X
			}
			paint.FillShape(gtx.Ops, mediumEmphasisTextColor, clip.Rect(image.Rect(0, 0, x, height)).Op())
		}
		valueMax := float32(v.Peak[j]) + v.Range
		if valueMax > 0 {
			color := white
			if valueMax >= v.Range {
				color = errorColor
			}
			x := int(valueMax/v.Range*float32(gtx.Constraints.Max.X) + 0.5)
			if x > gtx.Constraints.Max.X {
				x = gtx.Constraints.Max.X
			}
			paint.FillShape(gtx.Ops, color, clip.Rect(image.Rect(x-1, 0, x, height)).Op())
		}
		op.Offset(image.Point{0, height}).Add(gtx.Ops)
	}
	return D{Size: gtx.Constraints.Max}
}
