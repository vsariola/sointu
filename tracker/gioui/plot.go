package gioui

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

type (
	Plot struct {
		origXlim, origYlim plotRange
		fixedYLevel        float32

		xScale, yScale float32
		xOffset        float32
		dragging       bool
		dragId         pointer.ID
		dragStartPoint f32.Point
	}

	PlotStyle struct {
		CurveColors [3]color.NRGBA `yaml:",flow"`
		LimitColor  color.NRGBA    `yaml:",flow"`
		CursorColor color.NRGBA    `yaml:",flow"`
		Ticks       LabelStyle
		DpPerTick   unit.Dp
	}

	PlotDataFunc func(chn int, xr plotRange) (yr plotRange, ok bool)
	PlotTickFunc func(r plotRange, num int, yield func(pos float32, label string))
	plotRange    struct{ a, b float32 }
	plotRel      float32
	plotPx       int
	plotLogScale float32
)

func NewPlot(xlim, ylim plotRange, fixedYLevel float32) *Plot {
	return &Plot{
		origXlim:    xlim,
		origYlim:    ylim,
		fixedYLevel: fixedYLevel,
	}
}

func (p *Plot) Layout(gtx C, data PlotDataFunc, xticks, yticks PlotTickFunc, cursornx float32, numchns int) D {
	p.update(gtx)
	t := TrackerFromContext(gtx)
	style := t.Theme.Plot
	s := gtx.Constraints.Max
	if s.X <= 1 || s.Y <= 1 {
		return D{}
	}
	defer clip.Rect(image.Rectangle{Max: s}).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, p)

	xlim := p.xlim()
	ylim := p.ylim()

	// draw tick marks
	numxticks := s.X / gtx.Dp(style.DpPerTick)
	xticks(xlim, numxticks, func(x float32, txt string) {
		paint.ColorOp{Color: style.LimitColor}.Add(gtx.Ops)
		sx := plotPx(s.X).toScreen(xlim.toRelative(x))
		fillRect(gtx, clip.Rect{Min: image.Pt(sx, 0), Max: image.Pt(sx+1, s.Y)})
		defer op.Offset(image.Pt(sx, gtx.Dp(2))).Push(gtx.Ops).Pop()
		Label(t.Theme, &t.Theme.Plot.Ticks, txt).Layout(gtx)
	})

	numyticks := s.Y / gtx.Dp(style.DpPerTick)
	yticks(ylim, numyticks, func(y float32, txt string) {
		paint.ColorOp{Color: style.LimitColor}.Add(gtx.Ops)
		sy := plotPx(s.Y).toScreen(ylim.toRelative(y))
		fillRect(gtx, clip.Rect{Min: image.Pt(0, sy), Max: image.Pt(s.X, sy+1)})
		defer op.Offset(image.Pt(gtx.Dp(2), sy)).Push(gtx.Ops).Pop()
		Label(t.Theme, &t.Theme.Plot.Ticks, txt).Layout(gtx)
	})

	// draw cursor
	if cursornx == cursornx { // check for NaN
		paint.ColorOp{Color: style.CursorColor}.Add(gtx.Ops)
		csx := plotPx(s.X).toScreen(xlim.toRelative(cursornx))
		fillRect(gtx, clip.Rect{Min: image.Pt(csx, 0), Max: image.Pt(csx+1, s.Y)})
	}

	// draw curves
	for chn := range numchns {
		paint.ColorOp{Color: style.CurveColors[chn]}.Add(gtx.Ops)
		right := xlim.fromRelative(plotPx(s.X).fromScreen(0))
		for sx := range s.X {
			// left and right is the sample range covered by the pixel
			left := right
			right = xlim.fromRelative(plotPx(s.X).fromScreen(sx + 1))
			yr, ok := data(chn, plotRange{left, right})
			if !ok {
				continue
			}
			y1 := plotPx(s.Y).toScreen(ylim.toRelative(yr.a))
			y2 := plotPx(s.Y).toScreen(ylim.toRelative(yr.b))
			fillRect(gtx, clip.Rect{Min: image.Pt(sx, min(y1, y2)), Max: image.Pt(sx+1, max(y1, y2)+1)})
		}
	}
	return D{Size: s}
}

func (r plotRange) toRelative(f float32) plotRel    { return plotRel((f - r.a) / (r.b - r.a)) }
func (r plotRange) fromRelative(pr plotRel) float32 { return float32(pr)*(r.b-r.a) + r.a }
func (r plotRange) offset(o float32) plotRange      { return plotRange{r.a + o, r.b + o} }
func (r plotRange) scale(logScale float32) plotRange {
	s := float32(math.Exp(float64(logScale)))
	return plotRange{r.a * s, r.b * s}
}

func (s plotPx) toScreen(pr plotRel) int          { return int(float32(pr)*float32(s-1) + 0.5) }
func (s plotPx) fromScreen(px int) plotRel        { return plotRel(float32(px) / float32(s-1)) }
func (s plotPx) fromScreenF32(px float32) plotRel { return plotRel(px / float32(s-1)) }

func (o *Plot) xlim() plotRange { return o.origXlim.scale(o.xScale).offset(o.xOffset) }
func (o *Plot) ylim() plotRange {
	return o.origYlim.offset(-o.fixedYLevel).scale(o.yScale).offset(o.fixedYLevel)
}

func fillRect(gtx C, rect clip.Rect) {
	stack := rect.Push(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	stack.Pop()
}

func (o *Plot) update(gtx C) {
	s := gtx.Constraints.Max
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
				x1 := o.xlim().fromRelative(plotPx(s.X).fromScreenF32(e.Position.X))
				o.xScale += float32(min(max(-1, int(e.Scroll.Y)), 1)) * 0.1
				x2 := o.xlim().fromRelative(plotPx(s.X).fromScreenF32(e.Position.X))
				o.xOffset += x1 - x2
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
					x1 := o.xlim().fromRelative(plotPx(s.X).fromScreenF32(o.dragStartPoint.X))
					x2 := o.xlim().fromRelative(plotPx(s.X).fromScreenF32(e.Position.X))
					o.xOffset += x1 - x2

					num := o.ylim().fromRelative(plotPx(s.Y).fromScreenF32(e.Position.Y))
					den := o.ylim().fromRelative(plotPx(s.Y).fromScreenF32(o.dragStartPoint.Y))
					num -= o.fixedYLevel
					den -= o.fixedYLevel
					if l := math.Abs(float64(num / den)); l > 1e-3 && l < 1e3 {
						o.yScale -= float32(math.Log(l))
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
