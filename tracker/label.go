package tracker

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
)

type LabelStyle struct {
	Text       string
	Color      color.NRGBA
	ShadeColor color.NRGBA
	Font       text.Font
	FontSize   unit.Value
}

func (l LabelStyle) Layout(gtx layout.Context) layout.Dimensions {
	return layout.Stack{Alignment: layout.Center}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			defer op.Save(gtx.Ops).Load()
			paint.ColorOp{Color: l.ShadeColor}.Add(gtx.Ops)
			op.Offset(f32.Pt(2, 2)).Add(gtx.Ops)
			dims := widget.Label{
				Alignment: text.Start,
				MaxLines:  1,
			}.Layout(gtx, textShaper, l.Font, l.FontSize, l.Text)
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
			}.Layout(gtx, textShaper, l.Font, l.FontSize, l.Text)
		}),
	)
}

func Label(text string, color color.NRGBA) layout.Widget {
	return LabelStyle{Text: text, Color: color, ShadeColor: black, Font: labelDefaultFont, FontSize: labelDefaultFontSize}.Layout
}
