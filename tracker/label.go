package tracker

import (
	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget"
	"image"
	"image/color"
)

type LabelStyle struct {
	Text       string
	Color      color.RGBA
	ShadeColor color.RGBA
}

func (l LabelStyle) Layout(gtx layout.Context) layout.Dimensions {
	return layout.Stack{Alignment: layout.Center}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			defer op.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: l.ShadeColor}.Add(gtx.Ops)
			op.Offset(f32.Pt(2, 2)).Add(gtx.Ops)
			dims := widget.Label{
				Alignment: text.Start,
				MaxLines:  1,
			}.Layout(gtx, textShaper, labelFont, labelFontSize, l.Text)
			return layout.Dimensions{
				Size:     dims.Size.Add(image.Pt(2, 2)),
				Baseline: dims.Baseline,
			}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			paint.ColorOp{Color: l.Color}.Add(gtx.Ops)
			return widget.Label{
				Alignment: text.Start,
				MaxLines:  1,
			}.Layout(gtx, textShaper, labelFont, labelFontSize, l.Text)
		}),
	)
}

func Label(text string, color color.RGBA) layout.Widget {
	return LabelStyle{Text: text, Color: color, ShadeColor: black}.Layout
}
