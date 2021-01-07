package tracker

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

func Raised(w layout.Widget) layout.Widget {
	return Beveled(w, panelColor, panelLightColor, panelShadeColor)
}

func Lowered(w layout.Widget) layout.Widget {
	return Beveled(w, panelColor, panelShadeColor, panelLightColor)
}

func Beveled(w layout.Widget, base, light, shade color.RGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		stack := op.Push(gtx.Ops)
		gtx.Constraints.Max.X -= 2
		if gtx.Constraints.Max.X < 0 {
			gtx.Constraints.Max.X = 0
		}
		if gtx.Constraints.Min.X > gtx.Constraints.Max.X {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
		}
		gtx.Constraints.Max.Y -= 2
		if gtx.Constraints.Max.Y < 0 {
			gtx.Constraints.Max.Y = 0
		}
		if gtx.Constraints.Min.Y > gtx.Constraints.Max.Y {
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
		}
		macro := op.Record(gtx.Ops)
		op.Offset(f32.Pt(1, 1)).Add(gtx.Ops)
		dims := w(gtx)
		c := macro.Stop()
		stack.Pop()
		paint.FillShape(gtx.Ops, light, clip.Rect(image.Rect(0, 0, dims.Size.X+2, 1)).Op())
		paint.FillShape(gtx.Ops, light, clip.Rect(image.Rect(0, 0, 1, dims.Size.Y+2)).Op())
		paint.FillShape(gtx.Ops, base, clip.Rect(image.Rect(1, 1, dims.Size.X+1, dims.Size.Y+1)).Op())
		paint.FillShape(gtx.Ops, shade, clip.Rect(image.Rect(0, dims.Size.Y+1, dims.Size.X+2, dims.Size.Y+2)).Op())
		paint.FillShape(gtx.Ops, shade, clip.Rect(image.Rect(dims.Size.X+1, 0, dims.Size.X+2, dims.Size.Y+2)).Op())
		c.Add(gtx.Ops)
		return layout.Dimensions{
			Size:     dims.Size.Add(image.Point{X: 2, Y: 2}),
			Baseline: dims.Baseline + 1,
		}
	}
}
