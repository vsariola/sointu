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

type (
	NumericUpDownState struct {
		DpPerStep unit.Dp

		dragStartValue int
		dragStartXY    float32
		clickDecrease  gesture.Click
		clickIncrease  gesture.Click
		tipArea        component.TipArea
	}

	NumericUpDownStyle struct {
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

	NumericUpDown struct {
		Int   tracker.Int
		Theme *Theme
		State *NumericUpDownState
		Style *NumericUpDownStyle
		Tip   string
	}
)

func NewNumericUpDownState() *NumericUpDownState {
	return &NumericUpDownState{DpPerStep: unit.Dp(8)}
}

func NumUpDown(v tracker.Int, th *Theme, n *NumericUpDownState, tip string) NumericUpDown {
	return NumericUpDown{
		Int:   v,
		Theme: th,
		State: n,
		Style: &th.NumericUpDown,
		Tip:   tip,
	}
}

func (s *NumericUpDownState) Update(gtx layout.Context, v tracker.Int) {
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

func (n *NumericUpDown) Layout(gtx C) D {
	n.State.Update(gtx, n.Int)
	if n.Tip != "" {
		return n.State.tipArea.Layout(gtx, Tooltip(n.Theme, n.Tip), n.actualLayout)
	}
	return n.actualLayout(gtx)
}

func (n *NumericUpDown) actualLayout(gtx C) D {
	gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(n.Style.Width), gtx.Dp(n.Style.Height)))
	width := gtx.Dp(n.Style.ButtonWidth)
	height := gtx.Dp(n.Style.Height)
	return layout.Background{}.Layout(gtx,
		func(gtx C) D {
			defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(n.Style.CornerRadius)).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, n.Style.BgColor)
			event.Op(gtx.Ops, n.State) // register drag inputs, if not hitting the clicks
			return D{Size: gtx.Constraints.Min}
		},
		func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					gtx.Constraints = layout.Exact(image.Pt(width, height))
					return layout.Background{}.Layout(gtx,
						func(gtx C) D {
							defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()
							n.State.clickDecrease.Add(gtx.Ops)
							return D{Size: gtx.Constraints.Min}
						},
						func(gtx C) D { return n.Theme.Icon(icons.ContentRemove).Layout(gtx, n.Style.IconColor) },
					)
				}),
				layout.Flexed(1, func(gtx C) D {
					paint.ColorOp{Color: n.Style.TextColor}.Add(gtx.Ops)
					return widget.Label{Alignment: text.Middle}.Layout(gtx, n.Theme.Material.Shaper, n.Style.Font, n.Style.TextSize, strconv.Itoa(n.Int.Value()), op.CallOp{})
				}),
				layout.Rigid(func(gtx C) D {
					gtx.Constraints = layout.Exact(image.Pt(width, height))
					return layout.Background{}.Layout(gtx,
						func(gtx C) D {
							defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()
							n.State.clickIncrease.Add(gtx.Ops)
							return D{Size: gtx.Constraints.Min}
						},
						func(gtx C) D { return n.Theme.Icon(icons.ContentAdd).Layout(gtx, n.Style.IconColor) },
					)
				}),
			)
		},
	)
}
