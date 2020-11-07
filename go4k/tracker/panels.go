package tracker

import (
	"fmt"
	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"image"
	"image/color"
)

func Raised(w layout.Widget) layout.Widget {
	return Beveled(w, panelColor, panelLightColor, panelShadeColor)
}

func Lowered(w layout.Widget) layout.Widget {
	return Beveled(w, panelColor, panelShadeColor, panelLightColor)
}

func Beveled(w layout.Widget, base, light, shade color.RGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		fmt.Println("BR", gtx.Constraints)
		paint.FillShape(gtx.Ops, light, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, 1)).Op())
		paint.FillShape(gtx.Ops, light, clip.Rect(image.Rect(0, 0, 1, gtx.Constraints.Max.Y)).Op())
		paint.FillShape(gtx.Ops, base, clip.Rect(image.Rect(1, 1, gtx.Constraints.Max.X-1, gtx.Constraints.Max.Y-1)).Op())
		paint.FillShape(gtx.Ops, shade, clip.Rect(image.Rect(0, gtx.Constraints.Max.Y-1, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())
		paint.FillShape(gtx.Ops, shade, clip.Rect(image.Rect(gtx.Constraints.Max.X-1, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())
		fmt.Println("drawing sub..", gtx.Constraints)
		stack := op.Push(gtx.Ops)
		mcs := gtx.Constraints
		mcs.Max.X -= 2
		if mcs.Max.X < 0 {
			mcs.Max.X = 0
		}
		if mcs.Min.X > mcs.Max.X {
			mcs.Min.X = mcs.Max.X
		}
		mcs.Max.Y -= 2
		if mcs.Max.Y < 0 {
			mcs.Max.Y = 0
		}
		if mcs.Min.Y > mcs.Max.Y {
			mcs.Min.Y = mcs.Max.Y
		}
		op.Offset(f32.Pt(1, 1)).Add(gtx.Ops)
		gtx.Constraints = mcs
		dims := w(gtx)
		stack.Pop()
		return layout.Dimensions{
			Size:     dims.Size.Add(image.Point{X: 2, Y: 2}),
			Baseline: dims.Baseline + 1,
		}
	}
}
