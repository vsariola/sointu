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
		for x := 0; x < gtx.Constraints.Max.X; x++ {
			if x < 0 || x >= len(s.Wave) {
				continue
			}
			y := int((s.Wave[x][chn] + 1) / 2 * float32(gtx.Constraints.Max.Y))
			stack := clip.Rect{Min: image.Pt(x, y), Max: image.Pt(x+1, y+1)}.Push(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			stack.Pop()
		}
	}
	return D{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}
}
