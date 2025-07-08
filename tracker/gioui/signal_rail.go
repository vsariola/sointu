package gioui

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/vsariola/sointu/tracker"
)

const maxSignalsDrawn = 16

type (
	RailStyle struct {
		Color        color.NRGBA
		LineWidth    unit.Dp
		SignalWidth  unit.Dp
		PortDiameter unit.Dp
		PortColor    color.NRGBA
	}

	RailWidget struct {
		Style  *RailStyle
		Signal tracker.Rail
		Height unit.Dp
	}
)

func Rail(th *Theme, signal tracker.Rail) RailWidget {
	return RailWidget{
		Style:  &th.SignalRail,
		Signal: signal,
		Height: th.UnitEditor.Height,
	}
}

func (s RailWidget) Layout(gtx C) D {
	sw := gtx.Dp(s.Style.SignalWidth)
	h := gtx.Dp(s.Height)
	if s.Signal.PassThrough == 0 && len(s.Signal.StackUse.Inputs) == 0 && s.Signal.StackUse.NumOutputs == 0 {
		return D{Size: image.Pt(sw, h)}
	}
	lw := gtx.Dp(s.Style.LineWidth)
	pd := gtx.Dp(s.Style.PortDiameter)
	center := sw / 2
	var path clip.Path
	path.Begin(gtx.Ops)
	// Draw pass through signals
	for i := range min(maxSignalsDrawn, s.Signal.PassThrough) {
		x := float32(i*sw + center)
		path.MoveTo(f32.Pt(x, 0))
		path.LineTo(f32.Pt(x, float32(h)))
	}
	// Draw the routing of input signals
	for i := range min(len(s.Signal.StackUse.Inputs), maxSignalsDrawn-s.Signal.PassThrough) {
		input := s.Signal.StackUse.Inputs[i]
		x1 := float32((i+s.Signal.PassThrough)*sw + center)
		for _, link := range input {
			x2 := float32((link+s.Signal.PassThrough)*sw + center)
			path.MoveTo(f32.Pt(x1, 0))
			path.LineTo(f32.Pt(x2, float32(h/2)))
		}
	}
	if s.Signal.Send {
		for i := range min(len(s.Signal.StackUse.Inputs), maxSignalsDrawn-s.Signal.PassThrough) {
			d := gtx.Dp(8)
			from := f32.Pt(float32((i+s.Signal.PassThrough)*sw+center), float32(h/2))
			to := f32.Pt(float32(gtx.Constraints.Max.X), float32(h)-float32(d))
			ctrl := f32.Pt(from.X, to.Y)
			path.MoveTo(from)
			path.QuadTo(ctrl, to)
		}
	}
	// Draw the routing of output signals
	for i := range min(s.Signal.StackUse.NumOutputs, maxSignalsDrawn-s.Signal.PassThrough) {
		x := float32((i+s.Signal.PassThrough)*sw + center)
		path.MoveTo(f32.Pt(x, float32(h/2)))
		path.LineTo(f32.Pt(x, float32(h)))
	}
	// Signal paths finished
	paint.FillShape(gtx.Ops, s.Style.Color,
		clip.Stroke{
			Path:  path.End(),
			Width: float32(lw),
		}.Op())
	// Draw the circles on signals that get modified
	var circle clip.Path
	circle.Begin(gtx.Ops)
	for i := range min(len(s.Signal.StackUse.Modifies), maxSignalsDrawn-s.Signal.PassThrough) {
		if !s.Signal.StackUse.Modifies[i] {
			continue
		}
		f := f32.Pt(float32((i+s.Signal.PassThrough)*sw+center), float32(h/2))
		circle.MoveTo(f32.Pt(f.X-float32(pd/2), float32(h/2)))
		circle.ArcTo(f, f, float32(2*math.Pi))
	}
	p := clip.Outline{Path: circle.End()}.Op().Push(gtx.Ops)
	paint.ColorOp{Color: s.Style.PortColor}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	p.Pop()
	return D{Size: image.Pt(sw, h)}
}
