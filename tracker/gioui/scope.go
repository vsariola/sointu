package gioui

import (
	"image"
	"image/color"

	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/vsariola/sointu"
)

type ScopeStyle struct {
	Wave         sointu.AudioBuffer
	Colors       [2]color.NRGBA
	ClippedColor color.NRGBA
}

func FilledScope(wave sointu.AudioBuffer) *ScopeStyle {
	return &ScopeStyle{Wave: wave, Colors: [2]color.NRGBA{primaryColor, secondaryColor}}
}

func (s *ScopeStyle) Layout(gtx C) D {
	for chn := 0; chn < 2; chn++ {
		paint.ColorOp{Color: s.Colors[chn]}.Add(gtx.Ops)
		yprev := int((s.Wave[0][chn] + 1) / 2 * float32(gtx.Constraints.Max.Y))
		for px := 0; px < gtx.Constraints.Max.X; px++ {
			x := int(float32(px) / float32(gtx.Constraints.Max.X) * float32(len(s.Wave)))
			if x < 0 || x >= len(s.Wave) {
				continue
			}
			y := int((s.Wave[x][chn] + 1) / 2 * float32(gtx.Constraints.Max.Y))
			y1, y2 := yprev, y
			if y < yprev {
				y1, y2 = y, yprev-1
			} else if y > yprev {
				y1++
			}
			stack := clip.Rect{Min: image.Pt(px, y1), Max: image.Pt(px+1, y2+1)}.Push(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			stack.Pop()
			yprev = y
		}
	}
	return D{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}
}
