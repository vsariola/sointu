package gioui

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

type (
	PortState struct {
		click gesture.Click
	}

	PortStyle struct {
		Diameter    unit.Dp
		StrokeWidth unit.Dp
		Color       color.NRGBA
	}

	PortWidget struct {
		Theme *Theme
		Style *PortStyle
		State *PortState
	}
)

func Port(t *Theme, p *PortState) PortWidget {
	return PortWidget{Theme: t, Style: &t.Port, State: p}
}

func (p *PortWidget) Layout(gtx C) D {
	d := gtx.Dp(p.Style.Diameter)
	defer clip.Rect(image.Rectangle{Max: image.Pt(d, d)}).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, p.State)
	p.State.click.Add(gtx.Ops)
	p.strokeCircle(gtx)
	return D{Size: image.Pt(d, d)}
}

func (p *PortState) Clicked(gtx C) bool {
	ev, ok := p.click.Update(gtx.Source)
	return ok && ev.Kind == gesture.KindClick
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
