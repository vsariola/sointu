package gioui

import (
	"image"
	"image/color"

	"gioui.org/font"
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
	Alignment  layout.Direction
	Font       font.Font
	FontSize   unit.Sp
	Shaper     *text.Shaper
}

func (l LabelStyle) Layout(gtx layout.Context) layout.Dimensions {
	return l.Alignment.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min = image.Point{}
		paint.ColorOp{Color: l.ShadeColor}.Add(gtx.Ops)
		offs := op.Offset(image.Pt(2, 2)).Push(gtx.Ops)
		widget.Label{
			Alignment: text.Start,
			MaxLines:  1,
		}.Layout(gtx, l.Shaper, l.Font, l.FontSize, l.Text, op.CallOp{})
		offs.Pop()
		paint.ColorOp{Color: l.Color}.Add(gtx.Ops)
		dims := widget.Label{
			Alignment: text.Start,
			MaxLines:  1,
		}.Layout(gtx, l.Shaper, l.Font, l.FontSize, l.Text, op.CallOp{})
		return layout.Dimensions{
			Size:     dims.Size,
			Baseline: dims.Baseline,
		}
	})
}

func Label(str string, color color.NRGBA, shaper *text.Shaper) layout.Widget {
	return LabelStyle{Text: str, Color: color, ShadeColor: black, Font: labelDefaultFont, FontSize: labelDefaultFontSize, Alignment: layout.W, Shaper: shaper}.Layout
}
