package gioui

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
)

type LabelStyle struct {
	Color       color.NRGBA
	ShadowColor color.NRGBA
	Alignment   text.Alignment
	Font        font.Font
	TextSize    unit.Sp
}

type LabelWidget struct {
	Text   string
	Shaper *text.Shaper
	LabelStyle
}

func (l LabelWidget) Layout(gtx C) D {
	textColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: l.Color}.Add(gtx.Ops)
	textColor := textColorMacro.Stop()
	t := widget.Label{
		Alignment: l.Alignment,
		MaxLines:  1,
	}
	if l.ShadowColor.A > 0 {
		shadowColorMacro := op.Record(gtx.Ops)
		paint.ColorOp{Color: l.ShadowColor}.Add(gtx.Ops)
		shadowColor := shadowColorMacro.Stop()
		offs := op.Offset(image.Pt(2, 2)).Push(gtx.Ops)
		t.Layout(gtx, l.Shaper, l.Font, l.TextSize, l.Text, shadowColor)
		offs.Pop()
	}
	return t.Layout(gtx, l.Shaper, l.Font, l.TextSize, l.Text, textColor)
}

func Label(th *Theme, style *LabelStyle, txt string) LabelWidget {
	return LabelWidget{Text: txt, Shaper: th.Material.Shaper, LabelStyle: *style}
}
