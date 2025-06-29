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

const numSignalsDrawn = 8

type (
	SignalRailStyle struct {
		Color        color.NRGBA
		LineWidth    unit.Dp
		PortDiameter unit.Dp
		PortColor    color.NRGBA
	}

	SignalRailWidget struct {
		Style  *SignalRailStyle
		Signal tracker.Signal
		Width  unit.Dp
		Height unit.Dp
	}
)

func SignalRail(th *Theme, signal tracker.Signal) SignalRailWidget {
	return SignalRailWidget{
		Style:  &th.SignalRail,
		Signal: signal,
		Width:  th.UnitEditor.Width,
		Height: th.UnitEditor.Height,
	}
}

func (s SignalRailWidget) Layout(gtx C) D {
	w := gtx.Dp(s.Width)
	h := gtx.Dp(s.Height)
	l := gtx.Dp(s.Style.LineWidth)
	d := gtx.Dp(s.Style.PortDiameter)
	c := max(l, d) / 2
	stride := (w - c*2) / numSignalsDrawn
	var path clip.Path
	path.Begin(gtx.Ops)
	// Draw pass through signals
	for i := range min(numSignalsDrawn, s.Signal.PassThrough) {
		x := float32(i*stride + c)
		path.MoveTo(f32.Pt(x, 0))
		path.LineTo(f32.Pt(x, float32(h)))
	}
	// Draw the routing of input signals
	for i := range min(len(s.Signal.StackUse.Inputs), numSignalsDrawn-s.Signal.PassThrough) {
		input := s.Signal.StackUse.Inputs[i]
		x1 := float32((i+s.Signal.PassThrough)*stride + c)
		for _, link := range input {
			x2 := float32((link+s.Signal.PassThrough)*stride + c)
			path.MoveTo(f32.Pt(x1, 0))
			path.LineTo(f32.Pt(x2, float32(h/2)))
		}
	}
	// Draw the routing of output signals
	for i := range min(s.Signal.StackUse.NumOutputs, numSignalsDrawn-s.Signal.PassThrough) {
		x := float32((i+s.Signal.PassThrough)*stride + c)
		path.MoveTo(f32.Pt(x, float32(h/2)))
		path.LineTo(f32.Pt(x, float32(h)))
	}
	paint.FillShape(gtx.Ops, s.Style.Color,
		clip.Stroke{
			Path:  path.End(),
			Width: float32(l),
		}.Op())
	// Draw the circles on modified signals

	for i := range min(len(s.Signal.StackUse.Modifies), numSignalsDrawn-s.Signal.PassThrough) {
		if !s.Signal.StackUse.Modifies[i] {
			continue
		}
		var circle clip.Path
		x := float32((i + s.Signal.PassThrough) * stride)
		circle.Begin(gtx.Ops)
		circle.MoveTo(f32.Pt(x, float32(h/2)))
		f := f32.Pt(x+float32(c), float32(h/2))
		circle.ArcTo(f, f, float32(2*math.Pi))
		p := clip.Outline{Path: circle.End()}.Op().Push(gtx.Ops)
		paint.ColorOp{Color: s.Style.PortColor}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		p.Pop()
	}
	return D{Size: image.Pt(w, h)}
}
