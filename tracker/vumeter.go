package tracker

import (
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

type VuMeter struct {
	avg     [2]float32
	max     [2]float32
	speed   [2]float32
	FallOff float32
	Decay   float32
	RangeDb float32
}

func (v *VuMeter) Update(buffer []float32) {
	for j := 0; j < 2; j++ {
		for i := 0; i < len(buffer); i += 2 {
			sample2 := buffer[i+j] * buffer[i+j]
			db := float32(10*math.Log10(float64(sample2))) + v.RangeDb
			v.speed[j] += v.FallOff
			v.max[j] -= v.speed[j]
			if v.max[j] < 0 {
				v.max[j] = 0
			}
			if v.max[j] < db {
				v.max[j] = db
				v.speed[j] = 0
			}
			v.avg[j] += (sample2 - v.avg[j]) * v.Decay
			if math.IsNaN(float64(v.avg[j])) {
				v.avg[j] = 0
			}
		}
	}
}

func (v *VuMeter) Reset() {
	v.avg = [2]float32{}
	v.max = [2]float32{}
}

func (v *VuMeter) Layout(gtx C) D {
	defer op.Save(gtx.Ops).Load()
	gtx.Constraints.Max.Y = gtx.Px(unit.Dp(12))
	height := gtx.Px(unit.Dp(6))
	for j := 0; j < 2; j++ {
		value := float32(10*math.Log10(float64(v.avg[j]))) + v.RangeDb
		if value > 0 {
			x := int(value/v.RangeDb*float32(gtx.Constraints.Max.X) + 0.5)
			if x > gtx.Constraints.Max.X {
				x = gtx.Constraints.Max.X
			}
			paint.FillShape(gtx.Ops, mediumEmphasisTextColor, clip.Rect(image.Rect(0, 0, x, height)).Op())
		}
		valueMax := v.max[j]
		if valueMax > 0 {
			color := white
			if valueMax >= v.RangeDb {
				color = errorColor
			}
			x := int(valueMax/v.RangeDb*float32(gtx.Constraints.Max.X) + 0.5)
			if x > gtx.Constraints.Max.X {
				x = gtx.Constraints.Max.X
			}
			paint.FillShape(gtx.Ops, color, clip.Rect(image.Rect(x-1, 0, x, height)).Op())
		}
		op.Offset(f32.Pt(0, float32(height))).Add(gtx.Ops)
	}
	return D{Size: gtx.Constraints.Max}
}
