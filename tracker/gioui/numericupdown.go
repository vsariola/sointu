package gioui

import (
	"image"
	"image/color"
	"strconv"

	"github.com/vsariola/sointu/tracker"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"gioui.org/font"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/x/component"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
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
	shaper          text.Shaper
}

func NewNumberInput(v tracker.Int) *NumberInput {
	return &NumberInput{Int: v}
}

func NumericUpDown(th *material.Theme, number *NumberInput, tooltip string) NumericUpDownStyle {
	return NumericUpDownStyle{
		NumberInput:     number,
		Color:           white,
		IconColor:       th.Palette.Fg,
		BackgroundColor: numberInputBgColor,
		CornerRadius:    unit.Dp(4),
		ButtonWidth:     unit.Dp(16),
		Border:          unit.Dp(1),
		UnitsPerStep:    unit.Dp(8),
		TextSize:        th.TextSize * 14 / 16,
		Tooltip:         Tooltip(th, tooltip),
		Width:           unit.Dp(70),
		Height:          unit.Dp(20),
		shaper:          *th.Shaper,
	}
}

func (s *NumericUpDownStyle) Update(gtx layout.Context) {
	// handle dragging
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
	// handle decrease clicks
	for ev, ok := s.NumberInput.clickDecrease.Update(gtx.Source); ok; ev, ok = s.NumberInput.clickDecrease.Update(gtx.Source) {
		if ev.Kind == gesture.KindClick {
			s.NumberInput.Int.Add(-1)
		}
	}
	// handle increase clicks
	for ev, ok := s.NumberInput.clickIncrease.Update(gtx.Source); ok; ev, ok = s.NumberInput.clickIncrease.Update(gtx.Source) {
		if ev.Kind == gesture.KindClick {
			s.NumberInput.Int.Add(1)
		}
	}
}

func (s NumericUpDownStyle) Layout(gtx C) D {
	if s.Tooltip.Text.Text != "" {
		return s.NumberInput.tipArea.Layout(gtx, s.Tooltip, s.actualLayout)
	}
	return s.actualLayout(gtx)
}

func (s *NumericUpDownStyle) actualLayout(gtx C) D {
	s.Update(gtx)
	gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(s.Width), gtx.Dp(s.Height)))
	width := gtx.Dp(s.ButtonWidth)
	height := gtx.Dp(s.Height)
	return layout.Background{}.Layout(gtx,
		func(gtx C) D {
			defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(s.CornerRadius)).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, s.BackgroundColor)
			event.Op(gtx.Ops, s.NumberInput) // register drag inputs, if not hitting the clicks
			return D{Size: gtx.Constraints.Min}
		},
		func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					gtx.Constraints = layout.Exact(image.Pt(width, height))
					return layout.Background{}.Layout(gtx,
						func(gtx C) D {
							defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()
							s.NumberInput.clickDecrease.Add(gtx.Ops)
							return D{Size: gtx.Constraints.Min}
						},
						func(gtx C) D { return widgetForIcon(icons.ContentRemove).Layout(gtx, s.IconColor) },
					)
				}),
				layout.Flexed(1, func(gtx C) D {
					paint.ColorOp{Color: s.Color}.Add(gtx.Ops)
					return widget.Label{Alignment: text.Middle}.Layout(gtx, &s.shaper, s.Font, s.TextSize, strconv.Itoa(s.NumberInput.Int.Value()), op.CallOp{})
				}),
				layout.Rigid(func(gtx C) D {
					gtx.Constraints = layout.Exact(image.Pt(width, height))
					return layout.Background{}.Layout(gtx,
						func(gtx C) D {
							defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()
							s.NumberInput.clickIncrease.Add(gtx.Ops)
							return D{Size: gtx.Constraints.Min}
						},
						func(gtx C) D { return widgetForIcon(icons.ContentAdd).Layout(gtx, s.IconColor) },
					)
				}),
			)
		},
	)
}
