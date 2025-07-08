package gioui

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/x/stroke"
	"github.com/vsariola/sointu/tracker"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type (
	ParamState struct {
		drag         gesture.Drag
		dragStartPt  f32.Point // used to calculate the drag amount
		dragStartVal int
		tipArea      TipArea
		click        gesture.Click
		clickable    Clickable
	}

	ParamWidget struct {
		Parameter tracker.Parameter
		State     *ParamState
		Theme     *Theme
		Focus     bool
		Disabled  bool
	}

	PortStyle struct {
		Diameter    unit.Dp
		StrokeWidth unit.Dp
		Color       color.NRGBA
	}

	PortWidget struct {
		Theme *Theme
		Style *PortStyle
		State *ParamState
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
		State  *ParamState
		Style  *KnobStyle
		Hint   string
		Scroll bool
	}

	SwitchStyle struct {
		Neutral struct {
			Fg color.NRGBA
			Bg color.NRGBA
		}
		Pos struct {
			Fg color.NRGBA
			Bg color.NRGBA
		}
		Neg struct {
			Fg color.NRGBA
			Bg color.NRGBA
		}
		Width   unit.Dp
		Height  unit.Dp
		Outline unit.Dp
		Handle  unit.Dp
		Icon    unit.Dp
	}

	SwitchWidget struct {
		Theme    *Theme
		Value    tracker.Parameter
		State    *ParamState
		Style    *SwitchStyle
		Hint     string
		Scroll   bool
		Disabled bool
	}
)

// ParamState

func Param(Parameter tracker.Parameter, th *Theme, paramWidget *ParamState, focus, disabled bool) ParamWidget {
	return ParamWidget{
		Theme:     th,
		State:     paramWidget,
		Parameter: Parameter,
		Focus:     focus,
		Disabled:  disabled,
	}
}

func (p ParamWidget) Layout(gtx C) D {
	title := Label(p.Theme, &p.Theme.UnitEditor.Name, p.Parameter.Name())
	t := TrackerFromContext(gtx)
	widget := func(gtx C) D {
		if port, ok := p.Parameter.Port(); t.IsChoosingSendTarget() && ok {
			for p.State.clickable.Clicked(gtx) {
				t.ChooseSendTarget(p.Parameter.UnitID(), port).Do()
			}
			k := Port(p.Theme, p.State)
			return k.Layout(gtx)
		}
		switch p.Parameter.Type() {
		case tracker.IntegerParameter:
			k := Knob(p.Parameter, p.Theme, p.State, p.Parameter.Hint().Label, p.Focus, p.Disabled)
			return k.Layout(gtx)
		case tracker.BoolParameter:
			s := Switch(p.Parameter, p.Theme, p.State, p.Parameter.Hint().Label, p.Focus, p.Disabled)
			return s.Layout(gtx)
		case tracker.IDParameter:
			for p.State.clickable.Clicked(gtx) {
				t.ChooseSendSource(p.Parameter.UnitID()).Do()
			}
			btn := Btn(t.Theme, &t.Theme.Button.Text, &p.State.clickable, "Set", p.Parameter.Hint().Label)
			if p.Disabled {
				btn.Style = &t.Theme.Button.Disabled
			}
			return btn.Layout(gtx)
		}
		if _, ok := p.Parameter.Port(); ok {
			k := Port(p.Theme, p.State)
			return k.Layout(gtx)
		}
		return D{}
	}
	title.Layout(gtx)
	layout.Center.Layout(gtx, widget)
	return D{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}
}

func (s *ParamState) update(gtx C, param tracker.Parameter, scroll bool) {
	for {
		p, ok := s.drag.Update(gtx.Metric, gtx.Source, gesture.Both)
		if !ok {
			break
		}
		switch p.Kind {
		case pointer.Press:
			s.dragStartPt = p.Position
			s.dragStartVal = param.Value()
		case pointer.Drag:
			// update the value based on the drag amount
			m := param.Range()
			d := p.Position.Sub(s.dragStartPt)
			amount := float32(d.X-d.Y) / float32(gtx.Dp(128))
			newValue := int(float32(s.dragStartVal) + amount*float32(m.Max-m.Min))
			param.SetValue(newValue)
			s.tipArea.Appear(gtx.Now)
		}
	}
	for {
		g, ok := s.click.Update(gtx.Source)
		if !ok {
			break
		}
		if g.Kind == gesture.KindClick && g.NumClicks > 1 {
			param.Reset()
		}
	}
	for scroll {
		e, ok := gtx.Event(pointer.Filter{
			Target:  s,
			Kinds:   pointer.Scroll,
			ScrollY: pointer.ScrollRange{Min: -1e6, Max: 1e6},
		})
		if !ok {
			break
		}
		if ev, ok := e.(pointer.Event); ok && ev.Kind == pointer.Scroll {
			delta := -int(math.Min(math.Max(float64(ev.Scroll.Y), -1), 1))
			param.Add(delta, ev.Modifiers.Contain(key.ModShortcut))
			s.tipArea.Appear(gtx.Now)
		}
	}
}

// KnobWidget

func Knob(v tracker.Parameter, th *Theme, state *ParamState, hint string, scroll, disabled bool) KnobWidget {
	ret := KnobWidget{
		Theme:  th,
		Value:  v,
		State:  state,
		Style:  &th.Knob,
		Hint:   hint,
		Scroll: scroll,
	}
	if disabled {
		ret.Style = &th.DisabledKnob
	}
	return ret
}

func (k *KnobWidget) Layout(gtx C) D {
	k.State.update(gtx, k.Value, k.Scroll)
	knob := func(gtx C) D {
		m := k.Value.Range()
		amount := float32(k.Value.Value()-m.Min) / float32(m.Max-m.Min)
		sw := gtx.Dp(k.Style.StrokeWidth)
		d := gtx.Dp(k.Style.Diameter)
		defer clip.Rect(image.Rectangle{Max: image.Pt(d, d)}).Push(gtx.Ops).Pop()
		event.Op(gtx.Ops, k.State)
		k.State.drag.Add(gtx.Ops)
		k.State.click.Add(gtx.Ops)
		middle := float32(k.Value.Neutral()-m.Min) / float32(m.Max-m.Min)
		pos := max(amount, middle)
		neg := min(amount, middle)
		if middle > 0 {
			k.strokeKnobArc(gtx, k.Style.Neg.Bg, sw, d, 0, neg)
		}
		if middle < 1 {
			k.strokeKnobArc(gtx, k.Style.Pos.Bg, sw, d, pos, 1)
		}
		if pos > middle {
			k.strokeKnobArc(gtx, k.Style.Pos.Color, sw, d, middle, pos)
		}
		if neg < middle {
			k.strokeKnobArc(gtx, k.Style.Neg.Color, sw, d, neg, middle)
		}
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

// SwitchWidget

func Switch(v tracker.Parameter, th *Theme, state *ParamState, hint string, scroll, disabled bool) SwitchWidget {
	return SwitchWidget{
		Theme:    th,
		Value:    v,
		State:    state,
		Style:    &th.Switch,
		Hint:     hint,
		Scroll:   scroll,
		Disabled: disabled,
	}
}

func (s *SwitchWidget) Layout(gtx C) D {
	s.State.update(gtx, s.Value, s.Scroll)
	for s.Scroll {
		ev, ok := gtx.Event(pointer.Filter{Target: s.State, Kinds: pointer.Press})
		if !ok {
			break
		}
		if pe, ok := ev.(pointer.Event); ok && pe.Kind == pointer.Press {
			curVal := s.Value.Value()
			if pe.Buttons == pointer.ButtonPrimary {
				if curVal >= 1 {
					s.Value.SetValue(0)
				} else {
					s.Value.SetValue(curVal + 1)
				}
			}
			if pe.Buttons == pointer.ButtonSecondary {
				if curVal <= -1 {
					s.Value.SetValue(0)
				} else {
					s.Value.SetValue(curVal - 1)
				}
			}
			s.State.tipArea.Appear(gtx.Now)
		}
	}
	width := gtx.Dp(s.Style.Width)
	height := gtx.Dp(s.Style.Height)
	var fg, bg color.NRGBA
	o := 0
	switch {
	case s.Disabled || s.Value.Value() == 0:
		fg = s.Style.Neutral.Fg
		bg = s.Style.Neutral.Bg
		o = gtx.Dp(s.Style.Outline)
	case s.Value.Value() < 0:
		fg = s.Style.Neg.Fg
		bg = s.Style.Neg.Bg
	case s.Value.Value() > 0:
		fg = s.Style.Pos.Fg
		bg = s.Style.Pos.Bg
	}
	r := min(width, height) / 2
	fillRoundRect := func(ops *op.Ops, rect image.Rectangle, r int, c color.NRGBA) {
		defer clip.UniformRRect(rect, r).Push(ops).Pop()
		paint.ColorOp{Color: c}.Add(ops)
		paint.PaintOp{}.Add(ops)
	}
	if o > 0 {
		fillRoundRect(gtx.Ops, image.Rect(0, 0, width, height), r, fg)
	}
	fillRoundRect(gtx.Ops, image.Rect(o, o, width-o, height-o), r-o, bg)
	a := r
	b := width - r
	p := a + (b-a)*(s.Value.Value()-s.Value.Range().Min)/(s.Value.Range().Max-s.Value.Range().Min)
	circle := func(x, y, r int) clip.Op {
		b := image.Rectangle{
			Min: image.Pt(x-r, y-r),
			Max: image.Pt(x+r, y+r),
		}
		return clip.Ellipse(b).Op(gtx.Ops)
	}
	paint.FillShape(gtx.Ops, fg, circle(p, height/2, gtx.Dp(s.Style.Handle)/2))
	defer clip.Rect(image.Rectangle{Max: image.Pt(width, height)}).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, s.State)
	s.State.drag.Add(gtx.Ops)
	s.State.click.Add(gtx.Ops)
	icon := icons.NavigationClose
	if s.Value.Range().Min < 0 {
		if s.Value.Value() < 0 {
			icon = icons.ImageExposureNeg1
		} else if s.Value.Value() > 0 {
			icon = icons.ImageExposurePlus1
		}
	} else if s.Value.Value() > 0 {
		icon = icons.NavigationCheck
	}
	w := s.Theme.Icon(icon)
	i := gtx.Dp(s.Style.Icon)
	defer op.Offset(image.Pt(p-i/2, (height-i)/2)).Push(gtx.Ops).Pop()
	gtx.Constraints = layout.Exact(image.Pt(i, i))
	w.Layout(gtx, bg)
	return D{Size: image.Pt(width, height)}
}

//

func Port(t *Theme, p *ParamState) PortWidget {
	return PortWidget{Theme: t, Style: &t.Port, State: p}
}

func (p *PortWidget) Layout(gtx C) D {
	return p.State.clickable.layout(p.State, gtx, func(gtx C) D {
		d := gtx.Dp(p.Style.Diameter)
		defer clip.Rect(image.Rectangle{Max: image.Pt(d, d)}).Push(gtx.Ops).Pop()
		p.strokeCircle(gtx)
		return D{Size: image.Pt(d, d)}
	})
}

func (p *PortWidget) strokeCircle(gtx C) {
	sw := float32(gtx.Dp(p.Style.StrokeWidth))
	d := float32(gtx.Dp(p.Style.Diameter))
	rad := d / 2
	center := f32.Point{X: rad, Y: rad}
	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(f32.Pt(sw/2, rad))
	path.ArcTo(center, center, float32(math.Pi*2))
	paint.FillShape(gtx.Ops, p.Style.Color,
		clip.Stroke{
			Path:  path.End(),
			Width: sw,
		}.Op())
}
