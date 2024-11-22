package gioui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/vsariola/sointu/tracker"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"gioui.org/font"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/x/component"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

type NumberInput struct {
	Int            tracker.Int
	dragStartValue int
	dragStartXY    float32
	clickDecrease  gesture.Click
	clickIncrease  gesture.Click
	tipArea        component.TipArea
}

type NumericUpDownStyle struct {
	NumberInput     *NumberInput
	Color           color.NRGBA
	Font            font.Font
	TextSize        unit.Sp
	BorderColor     color.NRGBA
	IconColor       color.NRGBA
	BackgroundColor color.NRGBA
	CornerRadius    unit.Dp
	Border          unit.Dp
	ButtonWidth     unit.Dp
	UnitsPerStep    unit.Dp
	Tooltip         component.Tooltip
	Width           unit.Dp
	Height          unit.Dp
	Padding         unit.Dp
	shaper          text.Shaper
}

func NewNumberInput(v tracker.Int) *NumberInput {
	return &NumberInput{Int: v}
}

func NumericUpDown(th *material.Theme, number *NumberInput, tooltip string) NumericUpDownStyle {
	bgColor := th.Palette.Fg
	bgColor.R /= 4
	bgColor.G /= 4
	bgColor.B /= 4
	return NumericUpDownStyle{
		NumberInput:     number,
		Color:           white,
		BorderColor:     th.Palette.Fg,
		IconColor:       th.Palette.ContrastFg,
		BackgroundColor: bgColor,
		CornerRadius:    unit.Dp(4),
		ButtonWidth:     unit.Dp(16),
		Border:          unit.Dp(1),
		UnitsPerStep:    unit.Dp(8),
		TextSize:        th.TextSize * 14 / 16,
		Tooltip:         Tooltip(th, tooltip),
		Width:           unit.Dp(70),
		Height:          unit.Dp(20),
		Padding:         unit.Dp(0),
		shaper:          *th.Shaper,
	}
}

func (s *NumericUpDownStyle) Layout(gtx C) D {
	if s.Padding <= 0 {
		return s.layoutWithTooltip(gtx)
	}
	return layout.UniformInset(s.Padding).Layout(gtx, s.layoutWithTooltip)
}

func (s *NumericUpDownStyle) layoutWithTooltip(gtx C) D {
	if s.Tooltip.Text.Text != "" {
		return s.NumberInput.tipArea.Layout(gtx, s.Tooltip, s.actualLayout)
	}
	return s.actualLayout(gtx)
}

func (s *NumericUpDownStyle) actualLayout(gtx C) D {
	size := image.Pt(gtx.Dp(s.Width), gtx.Dp(s.Height))
	gtx.Constraints.Min = size
	rr := gtx.Dp(s.CornerRadius)
	border := gtx.Dp(s.Border)
	c := clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, rr).Push(gtx.Ops)
	paint.Fill(gtx.Ops, s.BorderColor)
	c.Pop()
	off := op.Offset(image.Pt(border, border)).Push(gtx.Ops)
	c2 := clip.UniformRRect(image.Rectangle{Max: image.Pt(
		gtx.Constraints.Min.X-border*2,
		gtx.Constraints.Min.Y-border*2,
	)}, rr-border).Push(gtx.Ops)
	gtx.Constraints.Min.X -= int(border * 2)
	gtx.Constraints.Min.Y -= int(border * 2)
	gtx.Constraints.Max = gtx.Constraints.Min
	layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(s.button(gtx.Constraints.Max.Y, widgetForIcon(icons.NavigationArrowBack), -1, &s.NumberInput.clickDecrease)),
		layout.Flexed(1, s.layoutText),
		layout.Rigid(s.button(gtx.Constraints.Max.Y, widgetForIcon(icons.NavigationArrowForward), 1, &s.NumberInput.clickIncrease)),
	)
	off.Pop()
	c2.Pop()
	return layout.Dimensions{Size: size}
}

func (s *NumericUpDownStyle) button(height int, icon *widget.Icon, delta int, click *gesture.Click) layout.Widget {
	return func(gtx C) D {
		width := gtx.Dp(s.ButtonWidth)
		return layout.Background{}.Layout(gtx,
			func(gtx C) D {
				if icon != nil {
					return icon.Layout(gtx, s.IconColor)
				}
				return layout.Dimensions{Size: image.Point{X: width, Y: height}}
			},
			func(gtx C) D {
				gtx.Constraints = layout.Exact(image.Pt(width, height))
				return s.layoutClick(gtx, delta, click)
			})
	}
}

func (s *NumericUpDownStyle) layoutText(gtx C) D {
	return layout.Background{}.Layout(gtx,
		func(gtx C) D {
			paint.FillShape(gtx.Ops, s.BackgroundColor, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())
			paint.ColorOp{Color: s.Color}.Add(gtx.Ops)
			return widget.Label{Alignment: text.Middle}.Layout(gtx, &s.shaper, s.Font, s.TextSize, fmt.Sprintf("%v", s.NumberInput.Int.Value()), op.CallOp{})
		},
		func(gtx C) D {
			gtx.Constraints.Min = gtx.Constraints.Max
			return s.layoutDrag(gtx)
		})
}

func (s *NumericUpDownStyle) layoutDrag(gtx layout.Context) layout.Dimensions {
	{ // handle dragging
		pxPerStep := float32(gtx.Dp(s.UnitsPerStep))
		for {
			ev, ok := gtx.Event(pointer.Filter{
				Target: s.NumberInput,
				Kinds:  pointer.Press | pointer.Drag | pointer.Release,
			})
			if !ok {
				break
			}
			if e, ok := ev.(pointer.Event); ok {
				switch e.Kind {
				case pointer.Press:
					s.NumberInput.dragStartValue = s.NumberInput.Int.Value()
					s.NumberInput.dragStartXY = e.Position.X - e.Position.Y

				case pointer.Drag:
					var deltaCoord float32
					deltaCoord = e.Position.X - e.Position.Y - s.NumberInput.dragStartXY
					s.NumberInput.Int.Set(s.NumberInput.dragStartValue + int(deltaCoord/pxPerStep+0.5))
				}
			}
		}

		// Avoid affecting the input tree with pointer events.
		stack := op.Offset(image.Point{}).Push(gtx.Ops)
		// register for input
		dragRect := image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
		area := clip.Rect(dragRect).Push(gtx.Ops)
		event.Op(gtx.Ops, s.NumberInput)
		area.Pop()
		stack.Pop()
	}
	return layout.Dimensions{Size: gtx.Constraints.Min}
}

func (s *NumericUpDownStyle) layoutClick(gtx layout.Context, delta int, click *gesture.Click) layout.Dimensions {
	// handle clicking
	for {
		ev, ok := click.Update(gtx.Source)
		if !ok {
			break
		}
		switch ev.Kind {
		case gesture.KindClick:
			s.NumberInput.Int.Add(delta)
		}
	}
	// Avoid affecting the input tree with pointer events.
	stack := op.Offset(image.Point{}).Push(gtx.Ops)

	// register for input
	clickRect := image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
	area := clip.Rect(clickRect).Push(gtx.Ops)
	click.Add(gtx.Ops)
	area.Pop()
	stack.Pop()
	return layout.Dimensions{Size: gtx.Constraints.Min}
}
