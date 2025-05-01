package gioui

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/vsariola/sointu/tracker"
)

type (
	OscilloscopeState struct {
		onceBtn              *BoolClickable
		wrapBtn              *BoolClickable
		lengthInBeatsNumber  *NumberInput
		triggerChannelNumber *NumberInput
		xScale               int
		xOffset              float32
		yScale               float64
		dragging             bool
		dragId               pointer.ID
		dragStartPoint       f32.Point
	}

	OscilloscopeStyle struct {
		CurveColors [2]color.NRGBA `yaml:",flow"`
		LimitColor  color.NRGBA    `yaml:",flow"`
		CursorColor color.NRGBA    `yaml:",flow"`
	}

	Oscilloscope struct {
		State *OscilloscopeState
		Wave  tracker.RingBuffer[[2]float32]
		Theme *Theme
		OscilloscopeStyle
	}
)

func NewOscilloscope(model *tracker.Model) *OscilloscopeState {
	return &OscilloscopeState{
		onceBtn:              NewBoolClickable(model.SignalAnalyzer().Once().Bool()),
		wrapBtn:              NewBoolClickable(model.SignalAnalyzer().Wrap().Bool()),
		lengthInBeatsNumber:  NewNumberInput(model.SignalAnalyzer().LengthInBeats().Int()),
		triggerChannelNumber: NewNumberInput(model.SignalAnalyzer().TriggerChannel().Int()),
	}
}

func Scope(s *OscilloscopeState, wave tracker.RingBuffer[[2]float32], th *Theme) *Oscilloscope {
	return &Oscilloscope{State: s, Wave: wave, Theme: th}
}

func (s *Oscilloscope) Layout(gtx C) D {
	wrapBtnStyle := ToggleButton(gtx, s.Theme, s.State.wrapBtn, "Wrap")
	onceBtnStyle := ToggleButton(gtx, s.Theme, s.State.onceBtn, "Once")
	triggerChannelStyle := NumUpDown(s.Theme, s.State.triggerChannelNumber, "Trigger channel")
	lengthNumberStyle := NumUpDown(s.Theme, s.State.lengthInBeatsNumber, "Buffer length in beats")

	leftSpacer := layout.Spacer{Width: unit.Dp(6), Height: unit.Dp(24)}.Layout
	rightSpacer := layout.Spacer{Width: unit.Dp(6)}.Layout

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D { return s.layoutWave(gtx) }),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(leftSpacer),
				layout.Rigid(Label(s.Theme, &s.Theme.SongPanel.RowHeader, "Trigger").Layout),
				layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
				layout.Rigid(onceBtnStyle.Layout),
				layout.Rigid(triggerChannelStyle.Layout),
				layout.Rigid(rightSpacer),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(leftSpacer),
				layout.Rigid(Label(s.Theme, &s.Theme.SongPanel.RowHeader, "Buffer").Layout),
				layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
				layout.Rigid(wrapBtnStyle.Layout),
				layout.Rigid(lengthNumberStyle.Layout),
				layout.Rigid(rightSpacer),
			)
		}),
	)
}

func (s *Oscilloscope) layoutWave(gtx C) D {
	s.update(gtx)
	if gtx.Constraints.Max.X == 0 || gtx.Constraints.Max.Y == 0 {
		return D{}
	}
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, s.State)
	paint.ColorOp{Color: s.Theme.Oscilloscope.CursorColor}.Add(gtx.Ops)
	cursorX := int(s.sampleToPx(gtx, float32(s.Wave.Cursor)))
	fillRect(gtx, clip.Rect{Min: image.Pt(cursorX, 0), Max: image.Pt(cursorX+1, gtx.Constraints.Max.Y)})
	paint.ColorOp{Color: s.Theme.Oscilloscope.LimitColor}.Add(gtx.Ops)
	minusOneY := int(s.ampToY(gtx, -1))
	fillRect(gtx, clip.Rect{Min: image.Pt(0, minusOneY), Max: image.Pt(gtx.Constraints.Max.X, minusOneY+1)})
	plusOneY := int(s.ampToY(gtx, 1))
	fillRect(gtx, clip.Rect{Min: image.Pt(0, plusOneY), Max: image.Pt(gtx.Constraints.Max.X, plusOneY+1)})
	leftX := int(s.sampleToPx(gtx, 0))
	fillRect(gtx, clip.Rect{Min: image.Pt(leftX, 0), Max: image.Pt(leftX+1, gtx.Constraints.Max.Y)})
	rightX := int(s.sampleToPx(gtx, float32(len(s.Wave.Buffer)-1)))
	fillRect(gtx, clip.Rect{Min: image.Pt(rightX, 0), Max: image.Pt(rightX+1, gtx.Constraints.Max.Y)})
	for chn := range 2 {
		paint.ColorOp{Color: s.Theme.Oscilloscope.CurveColors[chn]}.Add(gtx.Ops)
		for px := range gtx.Constraints.Max.X {
			// left and right is the sample range covered by the pixel
			left := int(s.pxToSample(gtx, float32(px)-0.5))
			right := int(s.pxToSample(gtx, float32(px)+0.5))
			if right < 0 || left >= len(s.Wave.Buffer) {
				continue
			}
			right = min(right, len(s.Wave.Buffer)-1)
			left = max(left, 0)
			// smin and smax are the smallest and largest sample values in the pixel range
			smax := float32(math.Inf(-1))
			smin := float32(math.Inf(1))
			for x := left; x <= right; x++ {
				smax = max(smax, s.Wave.Buffer[x][chn])
				smin = min(smin, s.Wave.Buffer[x][chn])
			}
			// y1 and y2 are the pixel range covered by the sample value
			y1 := min(max(int(s.ampToY(gtx, smax)+0.5), 0), gtx.Constraints.Max.Y-1)
			y2 := min(max(int(s.ampToY(gtx, smin)+0.5), 0), gtx.Constraints.Max.Y-1)
			fillRect(gtx, clip.Rect{Min: image.Pt(px, y1), Max: image.Pt(px+1, y2+1)})
		}
	}
	return D{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}
}

func fillRect(gtx C, rect clip.Rect) {
	stack := rect.Push(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	stack.Pop()
}

func (o *Oscilloscope) update(gtx C) {
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target:  o.State,
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
				o.State.xScale += min(max(-1, int(e.Scroll.Y)), 1)
				s2 := o.pxToSample(gtx, e.Position.X)
				o.State.xOffset -= s1 - s2
			case pointer.Press:
				if e.Buttons&pointer.ButtonSecondary != 0 {
					o.State.xOffset = 0
					o.State.xScale = 0
					o.State.yScale = 0
				}
				if e.Buttons&pointer.ButtonPrimary != 0 {
					o.State.dragging = true
					o.State.dragId = e.PointerID
					o.State.dragStartPoint = e.Position
				}
			case pointer.Drag:
				if e.Buttons&pointer.ButtonPrimary != 0 && o.State.dragging && e.PointerID == o.State.dragId {
					deltaX := o.pxToSample(gtx, e.Position.X) - o.pxToSample(gtx, o.State.dragStartPoint.X)
					o.State.xOffset += deltaX
					num := o.yToAmp(gtx, e.Position.Y)
					den := o.yToAmp(gtx, o.State.dragStartPoint.Y)
					if l := math.Abs(float64(num / den)); l > 1e-3 && l < 1e3 {
						o.State.yScale += math.Log(l)
						o.State.yScale = min(max(o.State.yScale, -1e3), 1e3)
					}
					o.State.dragStartPoint = e.Position

				}
			case pointer.Release | pointer.Cancel:
				o.State.dragging = false
			}
		}
	}
}

func (o *Oscilloscope) scaleFactor() float32 {
	return float32(math.Pow(1.1, float64(o.State.xScale)))
}

func (s *Oscilloscope) pxToSample(gtx C, px float32) float32 {
	return px*s.scaleFactor()*float32(len(s.Wave.Buffer))/float32(gtx.Constraints.Max.X) - s.State.xOffset
}

func (s *Oscilloscope) sampleToPx(gtx C, sample float32) float32 {
	return (sample + s.State.xOffset) * float32(gtx.Constraints.Max.X) / float32(len(s.Wave.Buffer)) / s.scaleFactor()
}

func (s *Oscilloscope) ampToY(gtx C, amp float32) float32 {
	scale := float32(math.Exp(s.State.yScale))
	return (1 - amp*scale) / 2 * float32(gtx.Constraints.Max.Y-1)
}

func (s *Oscilloscope) yToAmp(gtx C, y float32) float32 {
	scale := float32(math.Exp(s.State.yScale))
	return (1 - y/float32(gtx.Constraints.Max.Y-1)*2) / scale
}
