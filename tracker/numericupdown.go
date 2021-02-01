package tracker

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"gioui.org/f32"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

var defaultNumericLeftIcon *widget.Icon
var defaultNumericRightIcon *widget.Icon
var defaultNumericUpIcon *widget.Icon
var defaultNumericDownIcon *widget.Icon

type NumberInput struct {
	Value          int
	dragStartValue int
	dragStartXY    float32
	clickDecrease  gesture.Click
	clickIncrease  gesture.Click
}

type NumericUpDownStyle struct {
	NumberInput     *NumberInput
	Min             int
	Max             int
	Color           color.RGBA
	Font            text.Font
	TextSize        unit.Value
	BorderColor     color.RGBA
	IconColor       color.RGBA
	BackgroundColor color.RGBA
	CornerRadius    unit.Value
	Border          unit.Value
	ButtonWidth     unit.Value
	UnitsPerStep    unit.Value
	shaper          text.Shaper
}

func NumericUpDown(th *material.Theme, number *NumberInput, min, max int) NumericUpDownStyle {
	bgColor := th.Color.Primary
	bgColor.R /= 4
	bgColor.G /= 4
	bgColor.B /= 4
	return NumericUpDownStyle{
		NumberInput:     number,
		Min:             min,
		Max:             max,
		Color:           white,
		BorderColor:     th.Color.Primary,
		IconColor:       th.Color.InvText,
		BackgroundColor: bgColor,
		CornerRadius:    unit.Dp(4),
		ButtonWidth:     unit.Dp(16),
		Border:          unit.Dp(1),
		UnitsPerStep:    unit.Dp(8),
		TextSize:        th.TextSize.Scale(14.0 / 16.0),
		shaper:          th.Shaper,
	}
}

func (s NumericUpDownStyle) Layout(gtx C) D {
	size := gtx.Constraints.Min
	defer op.Push(gtx.Ops).Pop()
	rr := float32(gtx.Px(s.CornerRadius))
	border := float32(gtx.Px(s.Border))
	clip.UniformRRect(f32.Rectangle{Max: f32.Point{
		X: float32(gtx.Constraints.Min.X),
		Y: float32(gtx.Constraints.Min.Y),
	}}, rr).Add(gtx.Ops)
	paint.Fill(gtx.Ops, s.BorderColor)
	op.Offset(f32.Pt(border, border)).Add(gtx.Ops)
	clip.UniformRRect(f32.Rectangle{Max: f32.Point{
		X: float32(gtx.Constraints.Min.X) - border*2,
		Y: float32(gtx.Constraints.Min.Y) - border*2,
	}}, rr-border).Add(gtx.Ops)
	gtx.Constraints.Min.X -= int(border * 2)
	gtx.Constraints.Min.Y -= int(border * 2)
	gtx.Constraints.Max = gtx.Constraints.Min
	layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(s.button(gtx.Constraints.Max.Y, defaultNumericLeftIcon, -1, &s.NumberInput.clickDecrease)),
		layout.Flexed(1, s.layoutText),
		layout.Rigid(s.button(gtx.Constraints.Max.Y, defaultNumericRightIcon, 1, &s.NumberInput.clickIncrease)),
	)
	if s.NumberInput.Value < s.Min {
		s.NumberInput.Value = s.Min
	}
	if s.NumberInput.Value > s.Max {
		s.NumberInput.Value = s.Max
	}
	return layout.Dimensions{Size: size}
}

func (s NumericUpDownStyle) button(height int, icon *widget.Icon, delta int, click *gesture.Click) layout.Widget {
	return func(gtx C) D {
		btnWidth := gtx.Px(s.ButtonWidth)
		return layout.Stack{Alignment: layout.Center}.Layout(gtx,
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				//paint.FillShape(gtx.Ops, black, clip.Rect(image.Rect(0, 0, btnWidth, height)).Op())
				return layout.Dimensions{Size: image.Point{X: btnWidth, Y: height}}
			}),
			layout.Expanded(func(gtx C) D {
				size := btnWidth
				if height < size {
					size = height
				}
				if icon != nil {
					icon.Color = s.IconColor
					return icon.Layout(gtx, unit.Px(float32(size)))
				}
				return layout.Dimensions{}
			}),
			layout.Expanded(func(gtx C) D {
				return s.layoutClick(gtx, delta, click)
			}),
		)
	}
}

func (s NumericUpDownStyle) layoutText(gtx C) D {
	return layout.Stack{Alignment: layout.Center}.Layout(gtx,
		layout.Stacked(func(gtx C) D {
			paint.FillShape(gtx.Ops, s.BackgroundColor, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paint.ColorOp{Color: s.Color}.Add(gtx.Ops)
			return widget.Label{Alignment: text.Middle}.Layout(gtx, s.shaper, s.Font, s.TextSize, fmt.Sprintf("%v", s.NumberInput.Value))
		}),
		layout.Expanded(s.layoutDrag),
	)
}

func (s NumericUpDownStyle) layoutDrag(gtx layout.Context) layout.Dimensions {
	{ // handle dragging
		pxPerStep := float32(gtx.Px(s.UnitsPerStep))
		for _, ev := range gtx.Events(s.NumberInput) {
			if e, ok := ev.(pointer.Event); ok {
				switch e.Type {
				case pointer.Press:
					s.NumberInput.dragStartValue = s.NumberInput.Value
					s.NumberInput.dragStartXY = e.Position.X - e.Position.Y

				case pointer.Drag:
					var deltaCoord float32
					deltaCoord = e.Position.X - e.Position.Y - s.NumberInput.dragStartXY
					s.NumberInput.Value = s.NumberInput.dragStartValue + int(deltaCoord/pxPerStep+0.5)
				}
			}
		}

		// Avoid affecting the input tree with pointer events.
		stack := op.Push(gtx.Ops)
		// register for input
		dragRect := image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
		pointer.Rect(dragRect).Add(gtx.Ops)
		pointer.InputOp{
			Tag:   s.NumberInput,
			Types: pointer.Press | pointer.Drag | pointer.Release,
		}.Add(gtx.Ops)
		stack.Pop()
	}
	return layout.Dimensions{Size: gtx.Constraints.Min}
}

func (s NumericUpDownStyle) layoutClick(gtx layout.Context, delta int, click *gesture.Click) layout.Dimensions {
	// handle clicking
	for _, e := range click.Events(gtx) {
		switch e.Type {
		case gesture.TypeClick:
			s.NumberInput.Value += delta
		}
	}
	// Avoid affecting the input tree with pointer events.
	stack := op.Push(gtx.Ops)
	// register for input
	clickRect := image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
	pointer.Rect(clickRect).Add(gtx.Ops)
	click.Add(gtx.Ops)
	stack.Pop()
	return layout.Dimensions{Size: gtx.Constraints.Min}
}

func init() {
	var err error
	defaultNumericLeftIcon, err = widget.NewIcon(icons.NavigationArrowBack)
	if err != nil {
		log.Fatal(err)
	}
	defaultNumericRightIcon, err = widget.NewIcon(icons.NavigationArrowForward)
	if err != nil {
		log.Fatal(err)
	}
	defaultNumericUpIcon, err = widget.NewIcon(icons.NavigationArrowDropUp)
	if err != nil {
		log.Fatal(err)
	}
	defaultNumericDownIcon, err = widget.NewIcon(icons.NavigationArrowDropDown)
	if err != nil {
		log.Fatal(err)
	}
}
