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
		onceBtn              *Clickable
		wrapBtn              *Clickable
		lengthInBeatsNumber  *NumericUpDown
		triggerChannelNumber *NumericUpDown
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
)

func NewOscilloscope(model *tracker.Model) *OscilloscopeState {
	return &OscilloscopeState{
		onceBtn:              new(Clickable),
		wrapBtn:              new(Clickable),
		lengthInBeatsNumber:  NewNumericUpDown(),
		triggerChannelNumber: NewNumericUpDown(),
	}
}

func (s *OscilloscopeState) Layout(gtx C, vtrig, vlen tracker.Int, once, wrap tracker.Bool, wave tracker.RingBuffer[[2]float32], th *Theme, st *OscilloscopeStyle) D {
	leftSpacer := layout.Spacer{Width: unit.Dp(6), Height: unit.Dp(24)}.Layout
	rightSpacer := layout.Spacer{Width: unit.Dp(6)}.Layout

	onceBtn := ToggleBtn(once, th, s.onceBtn, "Once", "Trigger once on next event")
	wrapBtn := ToggleBtn(wrap, th, s.wrapBtn, "Wrap", "Wrap buffer when full")

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D { return s.layoutWave(gtx, wave, th) }),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(leftSpacer),
				layout.Rigid(Label(th, &th.SongPanel.RowHeader, "Trigger").Layout),
				layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
				layout.Rigid(onceBtn.Layout),
				layout.Rigid(func(gtx C) D {
					return s.triggerChannelNumber.Layout(gtx, vtrig, th, &th.NumericUpDown, "Trigger channel")
				}),
				layout.Rigid(rightSpacer),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(leftSpacer),
				layout.Rigid(Label(th, &th.SongPanel.RowHeader, "Buffer").Layout),
				layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
				layout.Rigid(wrapBtn.Layout),
				layout.Rigid(func(gtx C) D {
					return s.lengthInBeatsNumber.Layout(gtx, vlen, th, &th.NumericUpDown, "Buffer length in beats")
				}),
				layout.Rigid(rightSpacer),
			)
		}),
	)
}

func (s *OscilloscopeState) layoutWave(gtx C, wave tracker.RingBuffer[[2]float32], th *Theme) D {
	s.update(gtx, wave)
	if gtx.Constraints.Max.X == 0 || gtx.Constraints.Max.Y == 0 {
		return D{}
	}
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, s)
	paint.ColorOp{Color: th.Oscilloscope.CursorColor}.Add(gtx.Ops)
	cursorX := int(s.sampleToPx(gtx, float32(wave.Cursor), wave))
	fillRect(gtx, clip.Rect{Min: image.Pt(cursorX, 0), Max: image.Pt(cursorX+1, gtx.Constraints.Max.Y)})
	paint.ColorOp{Color: th.Oscilloscope.LimitColor}.Add(gtx.Ops)
	minusOneY := int(s.ampToY(gtx, -1))
	fillRect(gtx, clip.Rect{Min: image.Pt(0, minusOneY), Max: image.Pt(gtx.Constraints.Max.X, minusOneY+1)})
	plusOneY := int(s.ampToY(gtx, 1))
	fillRect(gtx, clip.Rect{Min: image.Pt(0, plusOneY), Max: image.Pt(gtx.Constraints.Max.X, plusOneY+1)})
	leftX := int(s.sampleToPx(gtx, 0, wave))
	fillRect(gtx, clip.Rect{Min: image.Pt(leftX, 0), Max: image.Pt(leftX+1, gtx.Constraints.Max.Y)})
	rightX := int(s.sampleToPx(gtx, float32(len(wave.Buffer)-1), wave))
	fillRect(gtx, clip.Rect{Min: image.Pt(rightX, 0), Max: image.Pt(rightX+1, gtx.Constraints.Max.Y)})
	for chn := range 2 {
		paint.ColorOp{Color: th.Oscilloscope.CurveColors[chn]}.Add(gtx.Ops)
		for px := range gtx.Constraints.Max.X {
			// left and right is the sample range covered by the pixel
			left := int(s.pxToSample(gtx, float32(px)-0.5, wave))
			right := int(s.pxToSample(gtx, float32(px)+0.5, wave))
			if right < 0 || left >= len(wave.Buffer) {
				continue
			}
			right = min(right, len(wave.Buffer)-1)
			left = max(left, 0)
			// smin and smax are the smallest and largest sample values in the pixel range
			smax := float32(math.Inf(-1))
			smin := float32(math.Inf(1))
			for x := left; x <= right; x++ {
				smax = max(smax, wave.Buffer[x][chn])
				smin = min(smin, wave.Buffer[x][chn])
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

func (o *OscilloscopeState) update(gtx C, wave tracker.RingBuffer[[2]float32]) {
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target:  o,
			Kinds:   pointer.Scroll | pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel,
			ScrollY: pointer.ScrollRange{Min: -1e6, Max: 1e6},
		})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok {
			switch e.Kind {
			case pointer.Scroll:
				s1 := o.pxToSample(gtx, e.Position.X, wave)
				o.xScale += min(max(-1, int(e.Scroll.Y)), 1)
				s2 := o.pxToSample(gtx, e.Position.X, wave)
				o.xOffset -= s1 - s2
			case pointer.Press:
				if e.Buttons&pointer.ButtonSecondary != 0 {
					o.xOffset = 0
					o.xScale = 0
					o.yScale = 0
				}
				if e.Buttons&pointer.ButtonPrimary != 0 {
					o.dragging = true
					o.dragId = e.PointerID
					o.dragStartPoint = e.Position
				}
			case pointer.Drag:
				if e.Buttons&pointer.ButtonPrimary != 0 && o.dragging && e.PointerID == o.dragId {
					deltaX := o.pxToSample(gtx, e.Position.X, wave) - o.pxToSample(gtx, o.dragStartPoint.X, wave)
					o.xOffset += deltaX
					num := o.yToAmp(gtx, e.Position.Y)
					den := o.yToAmp(gtx, o.dragStartPoint.Y)
					if l := math.Abs(float64(num / den)); l > 1e-3 && l < 1e3 {
						o.yScale += math.Log(l)
						o.yScale = min(max(o.yScale, -1e3), 1e3)
					}
					o.dragStartPoint = e.Position

				}
			case pointer.Release | pointer.Cancel:
				o.dragging = false
			}
		}
	}
}

func (o *OscilloscopeState) scaleFactor() float32 {
	return float32(math.Pow(1.1, float64(o.xScale)))
}

func (s *OscilloscopeState) pxToSample(gtx C, px float32, wave tracker.RingBuffer[[2]float32]) float32 {
	return px*s.scaleFactor()*float32(len(wave.Buffer))/float32(gtx.Constraints.Max.X) - s.xOffset
}

func (s *OscilloscopeState) sampleToPx(gtx C, sample float32, wave tracker.RingBuffer[[2]float32]) float32 {
	return (sample + s.xOffset) * float32(gtx.Constraints.Max.X) / float32(len(wave.Buffer)) / s.scaleFactor()
}

func (s *OscilloscopeState) ampToY(gtx C, amp float32) float32 {
	scale := float32(math.Exp(s.yScale))
	return (1 - amp*scale) / 2 * float32(gtx.Constraints.Max.Y-1)
}

func (s *OscilloscopeState) yToAmp(gtx C, y float32) float32 {
	scale := float32(math.Exp(s.yScale))
	return (1 - y/float32(gtx.Constraints.Max.Y-1)*2) / scale
}
