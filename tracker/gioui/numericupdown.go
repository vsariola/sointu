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

type NumericUpDown struct {
	DpPerStep unit.Dp

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
	Font         font.Font
}

func NewNumericUpDown() *NumericUpDown {
	return &NumericUpDown{DpPerStep: unit.Dp(8)}
}

func (s *NumericUpDown) Update(gtx layout.Context, v tracker.Int) {
	// handle dragging
	pxPerStep := float32(gtx.Dp(s.DpPerStep))
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: s,
			Kinds:  pointer.Press | pointer.Drag | pointer.Release,
		})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok {
			switch e.Kind {
			case pointer.Press:
				s.dragStartValue = v.Value()
				s.dragStartXY = e.Position.X - e.Position.Y
			case pointer.Drag:
				var deltaCoord float32
				deltaCoord = e.Position.X - e.Position.Y - s.dragStartXY
				v.SetValue(s.dragStartValue + int(deltaCoord/pxPerStep+0.5))
			}
		}
	}
	// handle decrease clicks
	for ev, ok := s.clickDecrease.Update(gtx.Source); ok; ev, ok = s.clickDecrease.Update(gtx.Source) {
		if ev.Kind == gesture.KindClick {
			v.Add(-1)
		}
	}
	// handle increase clicks
	for ev, ok := s.clickIncrease.Update(gtx.Source); ok; ev, ok = s.clickIncrease.Update(gtx.Source) {
		if ev.Kind == gesture.KindClick {
			v.Add(1)
		}
	}
}

func (s *NumericUpDown) Widget(v tracker.Int, th *Theme, st *NumericUpDownStyle, tooltip string) func(gtx C) D {
	return func(gtx C) D {
		return s.Layout(gtx, v, th, st, tooltip)
	}
}

func (s *NumericUpDown) Layout(gtx C, v tracker.Int, th *Theme, st *NumericUpDownStyle, tooltip string) D {
	s.Update(gtx, v)
	if tooltip != "" {
		return s.tipArea.Layout(gtx, Tooltip(th, tooltip), func(gtx C) D {
			return s.actualLayout(gtx, v, th, st)
		})
	}
	return s.actualLayout(gtx, v, th, st)
}

func (s *NumericUpDown) actualLayout(gtx C, v tracker.Int, th *Theme, st *NumericUpDownStyle) D {
	gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(st.Width), gtx.Dp(st.Height)))
	width := gtx.Dp(st.ButtonWidth)
	height := gtx.Dp(st.Height)
	return layout.Background{}.Layout(gtx,
		func(gtx C) D {
			defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(st.CornerRadius)).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, st.BgColor)
			event.Op(gtx.Ops, s) // register drag inputs, if not hitting the clicks
			return D{Size: gtx.Constraints.Min}
		},
		func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					gtx.Constraints = layout.Exact(image.Pt(width, height))
					return layout.Background{}.Layout(gtx,
						func(gtx C) D {
							defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()
							s.clickDecrease.Add(gtx.Ops)
							return D{Size: gtx.Constraints.Min}
						},
						func(gtx C) D { return th.Icon(icons.ContentRemove).Layout(gtx, st.IconColor) },
					)
				}),
				layout.Flexed(1, func(gtx C) D {
					paint.ColorOp{Color: st.TextColor}.Add(gtx.Ops)
					return widget.Label{Alignment: text.Middle}.Layout(gtx, th.Material.Shaper, st.Font, st.TextSize, strconv.Itoa(v.Value()), op.CallOp{})
				}),
				layout.Rigid(func(gtx C) D {
					gtx.Constraints = layout.Exact(image.Pt(width, height))
					return layout.Background{}.Layout(gtx,
						func(gtx C) D {
							defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()
							s.clickIncrease.Add(gtx.Ops)
							return D{Size: gtx.Constraints.Min}
						},
						func(gtx C) D { return th.Icon(icons.ContentAdd).Layout(gtx, st.IconColor) },
					)
				}),
			)
		},
	)
}
