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
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/x/component"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/text"
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
	TextColor    color.NRGBA `yaml:",flow"`
	IconColor    color.NRGBA `yaml:",flow"`
	BgColor      color.NRGBA `yaml:",flow"`
	CornerRadius unit.Dp
	ButtonWidth  unit.Dp
	Width        unit.Dp
	Height       unit.Dp
	TextSize     unit.Sp
	DpPerStep    unit.Dp
}

type NumericUpDown struct {
	NumberInput *NumberInput
	Tooltip     component.Tooltip
	Theme       *Theme
	Font        font.Font
	NumericUpDownStyle
}

func NewNumberInput(v tracker.Int) *NumberInput {
	return &NumberInput{Int: v}
}

func NumUpDown(th *Theme, number *NumberInput, tooltip string) NumericUpDown {
	return NumericUpDown{
		NumberInput:        number,
		Theme:              th,
		Tooltip:            Tooltip(th, tooltip),
		NumericUpDownStyle: th.NumericUpDown,
	}
}

func (s *NumericUpDown) Update(gtx layout.Context) {
	// handle dragging
	pxPerStep := float32(gtx.Dp(s.DpPerStep))
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
				s.NumberInput.Int.SetValue(s.NumberInput.dragStartValue + int(deltaCoord/pxPerStep+0.5))
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

func (s NumericUpDown) Layout(gtx C) D {
	if s.Tooltip.Text.Text != "" {
		return s.NumberInput.tipArea.Layout(gtx, s.Tooltip, s.actualLayout)
	}
	return s.actualLayout(gtx)
}

func (s *NumericUpDown) actualLayout(gtx C) D {
	s.Update(gtx)
	gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(s.Width), gtx.Dp(s.Height)))
	width := gtx.Dp(s.ButtonWidth)
	height := gtx.Dp(s.Height)
	return layout.Background{}.Layout(gtx,
		func(gtx C) D {
			defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(s.CornerRadius)).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, s.BgColor)
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
						func(gtx C) D { return s.Theme.Icon(icons.ContentRemove).Layout(gtx, s.IconColor) },
					)
				}),
				layout.Flexed(1, func(gtx C) D {
					paint.ColorOp{Color: s.TextColor}.Add(gtx.Ops)
					return widget.Label{Alignment: text.Middle}.Layout(gtx, s.Theme.Material.Shaper, s.Font, s.TextSize, strconv.Itoa(s.NumberInput.Int.Value()), op.CallOp{})
				}),
				layout.Rigid(func(gtx C) D {
					gtx.Constraints = layout.Exact(image.Pt(width, height))
					return layout.Background{}.Layout(gtx,
						func(gtx C) D {
							defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()
							s.NumberInput.clickIncrease.Add(gtx.Ops)
							return D{Size: gtx.Constraints.Min}
						},
						func(gtx C) D { return s.Theme.Icon(icons.ContentAdd).Layout(gtx, s.IconColor) },
					)
				}),
			)
		},
	)
}
