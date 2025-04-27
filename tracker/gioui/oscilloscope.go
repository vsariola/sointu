package gioui

import (
	"image"
	"image/color"
	"math"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/vsariola/sointu/tracker"
)

type (
	Oscilloscope struct {
		onceBtn              *BoolClickable
		wrapBtn              *BoolClickable
		lengthInBeatsNumber  *NumberInput
		triggerChannelNumber *NumberInput
		xScale               int
		xOffset              float32
		dragging             bool
		dragId               pointer.ID
		dragStartPx          float32
	}

	OscilloscopeStyle struct {
		Oscilloscope *Oscilloscope
		Wave         tracker.RingBuffer[[2]float32]
		Colors       [2]color.NRGBA
		ClippedColor color.NRGBA
		Theme        *material.Theme
	}
)

func NewOscilloscope(model *tracker.Model) *Oscilloscope {
	return &Oscilloscope{
		onceBtn:              NewBoolClickable(model.SignalAnalyzer().Once().Bool()),
		wrapBtn:              NewBoolClickable(model.SignalAnalyzer().Wrap().Bool()),
		lengthInBeatsNumber:  NewNumberInput(model.SignalAnalyzer().LengthInBeats().Int()),
		triggerChannelNumber: NewNumberInput(model.SignalAnalyzer().TriggerChannel().Int()),
	}
}

func LineOscilloscope(s *Oscilloscope, wave tracker.RingBuffer[[2]float32], th *material.Theme) *OscilloscopeStyle {
	return &OscilloscopeStyle{Oscilloscope: s, Wave: wave, Colors: [2]color.NRGBA{primaryColor, secondaryColor}, Theme: th, ClippedColor: errorColor}
}

func (s *OscilloscopeStyle) Layout(gtx C) D {
	wrapBtnStyle := ToggleButton(gtx, s.Theme, s.Oscilloscope.wrapBtn, "Wrap")
	onceBtnStyle := ToggleButton(gtx, s.Theme, s.Oscilloscope.onceBtn, "Once")
	triggerChannelStyle := NumericUpDown(s.Theme, s.Oscilloscope.triggerChannelNumber, "Trigger channel")
	lengthNumberStyle := NumericUpDown(s.Theme, s.Oscilloscope.lengthInBeatsNumber, "Buffer length in beats")

	leftSpacer := layout.Spacer{Width: unit.Dp(6), Height: unit.Dp(24)}.Layout
	rightSpacer := layout.Spacer{Width: unit.Dp(6)}.Layout

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D { return s.layoutWave(gtx) }),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(leftSpacer),
				layout.Rigid(LabelStyle{Text: "Trigger", Color: disabledTextColor, Alignment: layout.W, FontSize: s.Theme.TextSize * 14.0 / 16.0, Shaper: s.Theme.Shaper}.Layout),
				layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
				layout.Rigid(onceBtnStyle.Layout),
				layout.Rigid(triggerChannelStyle.Layout),
				layout.Rigid(rightSpacer),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(leftSpacer),
				layout.Rigid(LabelStyle{Text: "Buffer", Color: disabledTextColor, Alignment: layout.W, FontSize: s.Theme.TextSize * 14.0 / 16.0, Shaper: s.Theme.Shaper}.Layout),
				layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
				layout.Rigid(wrapBtnStyle.Layout),
				layout.Rigid(lengthNumberStyle.Layout),
				layout.Rigid(rightSpacer),
			)
		}),
	)
}

func (s *OscilloscopeStyle) layoutWave(gtx C) D {
	s.update(gtx)
	if gtx.Constraints.Max.X == 0 || gtx.Constraints.Max.Y == 0 {
		return D{}
	}
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, s.Oscilloscope)
	paint.ColorOp{Color: disabledTextColor}.Add(gtx.Ops)
	cursorX := int(s.sampleToPx(gtx, float32(s.Wave.Cursor)))
	stack := clip.Rect{Min: image.Pt(cursorX, 0), Max: image.Pt(cursorX+1, gtx.Constraints.Max.Y)}.Push(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	stack.Pop()
	for chn := 0; chn < 2; chn++ {
		paint.ColorOp{Color: s.Colors[chn]}.Add(gtx.Ops)
		clippedColorSet := false
		yprev := int((s.Wave.Buffer[0][chn] + 1) / 2 * float32(gtx.Constraints.Max.Y))
		for px := 0; px < gtx.Constraints.Max.X; px++ {
			x := int(s.pxToSample(gtx, float32(px)))
			if x < 0 || x >= len(s.Wave.Buffer) {
				continue
			}
			y := int((s.Wave.Buffer[x][chn] + 1) / 2 * float32(gtx.Constraints.Max.Y))
			if y < 0 {
				y = 0
			} else if y >= gtx.Constraints.Max.Y {
				y = gtx.Constraints.Max.Y - 1
			}
			y1, y2 := yprev, y
			if y < yprev {
				y1, y2 = y, yprev-1
			} else if y > yprev {
				y1++
			}
			clipped := false
			if y1 == y2 && y1 == 0 {
				clipped = true
			}
			if y1 == y2 && y1 == gtx.Constraints.Max.Y-1 {
				clipped = true
			}
			if clippedColorSet != clipped {
				if clipped {
					paint.ColorOp{Color: s.ClippedColor}.Add(gtx.Ops)
				} else {
					paint.ColorOp{Color: s.Colors[chn]}.Add(gtx.Ops)
				}
				clippedColorSet = clipped
			}
			stack := clip.Rect{Min: image.Pt(px, y1), Max: image.Pt(px+1, y2+1)}.Push(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			stack.Pop()
			yprev = y
		}
	}
	return D{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}
}

func (o *OscilloscopeStyle) update(gtx C) {
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target:  o.Oscilloscope,
			Kinds:   pointer.Scroll | pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel,
			ScrollY: pointer.ScrollRange{Min: -1e6, Max: 1e6},
		})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok {
			switch e.Kind {
			case pointer.Scroll:
				s1 := o.pxToSample(gtx, e.Position.X)
				o.Oscilloscope.xScale += min(max(-1, int(e.Scroll.Y)), 1)
				s2 := o.pxToSample(gtx, e.Position.X)
				o.Oscilloscope.xOffset -= s1 - s2
			case pointer.Press:
				if e.Buttons&pointer.ButtonSecondary != 0 {
					o.Oscilloscope.xOffset = 0
					o.Oscilloscope.xScale = 0
				}
				if e.Buttons&pointer.ButtonPrimary != 0 {
					o.Oscilloscope.dragging = true
					o.Oscilloscope.dragId = e.PointerID
					o.Oscilloscope.dragStartPx = e.Position.X
				}
			case pointer.Drag:
				if e.Buttons&pointer.ButtonPrimary != 0 && o.Oscilloscope.dragging && e.PointerID == o.Oscilloscope.dragId {
					delta := o.pxToSample(gtx, e.Position.X) - o.pxToSample(gtx, o.Oscilloscope.dragStartPx)
					o.Oscilloscope.xOffset += delta
					o.Oscilloscope.dragStartPx = e.Position.X
				}
			case pointer.Release | pointer.Cancel:
				o.Oscilloscope.dragging = false
			}
		}
	}
}

func (o *OscilloscopeStyle) scaleFactor() float32 {
	return float32(math.Pow(1.1, float64(o.Oscilloscope.xScale)))
}

func (s *OscilloscopeStyle) pxToSample(gtx C, px float32) float32 {
	return px*s.scaleFactor()*float32(len(s.Wave.Buffer))/float32(gtx.Constraints.Max.X) - s.Oscilloscope.xOffset
}

func (s *OscilloscopeStyle) sampleToPx(gtx C, sample float32) float32 {
	return (sample + s.Oscilloscope.xOffset) * float32(gtx.Constraints.Max.X) / float32(len(s.Wave.Buffer)) / s.scaleFactor()
}
