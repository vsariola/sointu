package gioui

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/x/stroke"
	"github.com/vsariola/sointu/tracker"
)

type (
	KnobState struct {
		click        gesture.Click
		drag         gesture.Drag
		dragStartPt  f32.Point // used to calculate the drag amount
		dragStartVal int
		tipArea      TipArea
	}

	KnobStyle struct {
		Diameter    unit.Dp
		StrokeWidth unit.Dp
		Bg          color.NRGBA
		Pos         struct {
			Color color.NRGBA
			Bg    color.NRGBA
		}
		Neg struct {
			Color color.NRGBA
			Bg    color.NRGBA
		}
		Indicator struct {
			Color     color.NRGBA
			Width     unit.Dp
			InnerDiam unit.Dp
			OuterDiam unit.Dp
		}
		Value LabelStyle
		Title LabelStyle
	}

	KnobWidget struct {
		Theme  *Theme
		Value  tracker.Parameter
		State  *KnobState
		Style  *KnobStyle
		Hint   string
		Scroll bool
	}
)

func Knob(v tracker.Parameter, th *Theme, state *KnobState, hint string, scroll bool) KnobWidget {
	return KnobWidget{
		Theme:  th,
		Value:  v,
		State:  state,
		Style:  &th.Knob,
		Hint:   hint,
		Scroll: scroll,
	}
}

func (k *KnobWidget) Layout(gtx C) D {
	k.update(gtx)
	knob := func(gtx C) D {
		m := k.Value.Range()
		amount := float32(k.Value.Value()-m.Min) / float32(m.Max-m.Min)
		sw := gtx.Dp(k.Style.StrokeWidth)
		d := gtx.Dp(k.Style.Diameter)
		defer clip.Rect(image.Rectangle{Max: image.Pt(d, d)}).Push(gtx.Ops).Pop()
		event.Op(gtx.Ops, k.State)
		k.State.drag.Add(gtx.Ops)
		k.State.click.Add(gtx.Ops)
		k.strokeKnobArc(gtx, k.Style.Pos.Bg, sw, d, amount, 1)
		k.strokeKnobArc(gtx, k.Style.Pos.Color, sw, d, 0, amount)
		k.strokeIndicator(gtx, amount)
		return D{Size: image.Pt(d, d)}
	}
	label := Label(k.Theme, &k.Style.Value, strconv.Itoa(k.Value.Value()))
	w := func(gtx C) D {
		return layout.Stack{Alignment: layout.Center}.Layout(gtx,
			layout.Stacked(knob),
			layout.Stacked(label.Layout))
	}
	if k.Hint != "" {
		c := gtx.Constraints
		gtx.Constraints.Max = image.Pt(1e6, 1e6)
		return k.State.tipArea.Layout(gtx, Tooltip(k.Theme, k.Hint), func(gtx C) D {
			gtx.Constraints = c
			return w(gtx)
		})
	}
	return w(gtx)
}

func (k *KnobWidget) update(gtx C) {
	for {
		p, ok := k.State.drag.Update(gtx.Metric, gtx.Source, gesture.Both)
		if !ok {
			break
		}
		switch p.Kind {
		case pointer.Press:
			k.State.dragStartPt = p.Position
			k.State.dragStartVal = k.Value.Value()
		case pointer.Drag:
			// update the value based on the drag amount
			m := k.Value.Range()
			d := p.Position.Sub(k.State.dragStartPt)
			amount := float32(d.X-d.Y) / float32(gtx.Dp(k.Style.Diameter)) / 4
			newValue := int(float32(k.State.dragStartVal) + amount*float32(m.Max-m.Min))
			k.Value.SetValue(newValue)
			k.State.tipArea.Appear(gtx.Now)
		}
	}
	for {
		g, ok := k.State.click.Update(gtx.Source)
		if !ok {
			break
		}
		if g.Kind == gesture.KindClick && g.NumClicks > 1 {
			k.Value.Reset()
		}
	}
	for k.Scroll {
		e, ok := gtx.Event(pointer.Filter{
			Target:  k.State,
			Kinds:   pointer.Scroll,
			ScrollY: pointer.ScrollRange{Min: -1e6, Max: 1e6},
		})
		if !ok {
			break
		}
		if ev, ok := e.(pointer.Event); ok && ev.Kind == pointer.Scroll {
			delta := math.Min(math.Max(float64(ev.Scroll.Y), -1), 1)
			k.Value.SetValue(k.Value.Value() - int(delta))
			k.State.tipArea.Appear(gtx.Now)
		}
	}
}

func (k *KnobWidget) strokeBg(gtx C) {
	diam := gtx.Dp(k.Style.Diameter)
	circle := clip.Ellipse{
		Min: image.Pt(0, 0),
		Max: image.Pt(diam, diam),
	}.Op(gtx.Ops)
	paint.FillShape(gtx.Ops, k.Style.Bg, circle)
}

func (k *KnobWidget) strokeKnobArc(gtx C, color color.NRGBA, strokeWidth, diameter int, start, end float32) {
	rad := float32(diameter) / 2
	end = min(max(end, 0), 1)
	if end <= 0 {
		return
	}
	startAngle := float64((start*8 + 1) / 10 * 2 * math.Pi)
	deltaAngle := (end - start) * 8 * math.Pi / 5
	center := f32.Point{X: rad, Y: rad}
	r2 := rad - float32(strokeWidth)/2
	startPt := f32.Point{X: rad - r2*float32(math.Sin(startAngle)), Y: rad + r2*float32(math.Cos(startAngle))}
	segments := [...]stroke.Segment{
		stroke.MoveTo(startPt),
		stroke.ArcTo(center, deltaAngle),
	}
	s := stroke.Stroke{
		Path:  stroke.Path{Segments: segments[:]},
		Width: float32(strokeWidth),
		Cap:   stroke.FlatCap,
	}
	paint.FillShape(gtx.Ops, color, s.Op(gtx.Ops))
}

func (k *KnobWidget) strokeIndicator(gtx C, amount float32) {
	innerRad := float32(gtx.Dp(k.Style.Indicator.InnerDiam)) / 2
	outerRad := float32(gtx.Dp(k.Style.Indicator.OuterDiam)) / 2
	center := float32(gtx.Dp(k.Style.Diameter)) / 2
	angle := (float64(amount)*8 + 1) / 10 * 2 * math.Pi
	start := f32.Point{
		X: center - innerRad*float32(math.Sin(angle)),
		Y: center + innerRad*float32(math.Cos(angle)),
	}
	end := f32.Point{
		X: center - outerRad*float32(math.Sin(angle)),
		Y: center + outerRad*float32(math.Cos(angle)),
	}
	segments := [...]stroke.Segment{
		stroke.MoveTo(start),
		stroke.LineTo(end),
	}
	s := stroke.Stroke{
		Path:  stroke.Path{Segments: segments[:]},
		Width: float32(k.Style.Indicator.Width),
		Cap:   stroke.FlatCap,
	}
	paint.FillShape(gtx.Ops, k.Style.Indicator.Color, s.Op(gtx.Ops))
}
