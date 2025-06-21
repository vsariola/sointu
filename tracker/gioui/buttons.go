package gioui

import (
	"image"
	"image/color"
	"math"
	"time"

	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/x/component"
	"github.com/vsariola/sointu/tracker"
)

type (
	Clickable struct {
		click   gesture.Click
		history []widget.Press

		requestClicks int
	}

	ClickableTip struct {
		Clickable Clickable
		TipArea   component.TipArea
	}

	ButtonStyle struct {
		// Color is the text color.
		Color        color.NRGBA
		Font         font.Font
		TextSize     unit.Sp
		Background   color.NRGBA
		CornerRadius unit.Dp
		Height       unit.Dp
		Inset        layout.Inset
	}

	IconButtonStyle struct {
		Background color.NRGBA
		// Color is the icon color.
		Color color.NRGBA
		// Size is the icon size.
		Size  unit.Dp
		Inset layout.Inset
	}

	TipIconButton struct {
		th      *Theme
		st      *IconButtonStyle
		t       *ClickableTip
		icon    []byte
		tip     string
		enabled bool
	}
)

func TipIconBtn(th *Theme, st *IconButtonStyle, t *ClickableTip, icon []byte, tip string, enabled bool) TipIconButton {
	return TipIconButton{
		th:      th,
		st:      st,
		t:       t,
		icon:    icon,
		tip:     tip,
		enabled: enabled,
	}
}

func (t TipIconButton) Layout(gtx C) D {
	iconBtn := IconBtn(t.th, t.st, &t.t.Clickable, t.th.Icon(t.icon), t.enabled)
	if t.tip != "" {
		return t.t.TipArea.Layout(gtx, Tooltip(t.th, t.tip), iconBtn)
	} else {
		return iconBtn(gtx)
	}
}

func ActionIconBtn(act tracker.Action, th *Theme, t *ClickableTip, icon []byte, tip string) layout.Widget {
	return func(gtx C) D {
		for t.Clickable.Clicked(gtx) {
			act.Do()
		}
		iconBtn := IconBtn(th, &th.IconButton, &t.Clickable, th.Icon(icon), act.Enabled())
		if tip != "" {
			return t.TipArea.Layout(gtx, Tooltip(th, tip), iconBtn)
		} else {
			return iconBtn(gtx)
		}
	}
}

func ToggleIconBtn(b tracker.Bool, th *Theme, t *ClickableTip, offIcon, onIcon []byte, offTip, onTip string) layout.Widget {
	return func(gtx C) D {
		icon := offIcon
		tip := offTip
		if b.Value() {
			icon = onIcon
			tip = onTip
		}
		for t.Clickable.Clicked(gtx) {
			b.Toggle()
		}
		iconBtn := IconBtn(th, &th.IconButton, &t.Clickable, th.Icon(icon), b.Enabled())
		if tip != "" {
			return t.TipArea.Layout(gtx, Tooltip(th, tip), iconBtn)
		} else {
			return iconBtn(gtx)
		}
	}
}

func ActionBtn(a tracker.Action, th *Theme, c *ClickableTip, text string, tip string) layout.Widget {
	return func(gtx C) D {
		for c.Clickable.Clicked(gtx) {
			a.Do()
		}
		var btn layout.Widget
		if !a.Enabled() {
			btn = Btn(th, &th.Button.Disabled, &c.Clickable, text)
		} else {
			btn = Btn(th, &th.Button.Text, &c.Clickable, text)
		}
		if tip != "" {
			return c.TipArea.Layout(gtx, Tooltip(th, tip), btn)
		} else {
			return btn(gtx)
		}
	}
}

func ToggleBtn(b tracker.Bool, th *Theme, c *ClickableTip, text string, tip string) layout.Widget {
	return func(gtx C) D {
		for c.Clickable.Clicked(gtx) {
			b.Toggle()
		}
		var btn layout.Widget
		if !b.Enabled() {
			btn = Btn(th, &th.Button.Disabled, &c.Clickable, text)
		} else if b.Value() {
			btn = Btn(th, &th.Button.Filled, &c.Clickable, text)
		} else {
			btn = Btn(th, &th.Button.Text, &c.Clickable, text)
		}
		if tip != "" {
			return c.TipArea.Layout(gtx, Tooltip(th, tip), btn)
		} else {
			return btn(gtx)
		}
	}
}

func Btn(th *Theme, st *ButtonStyle, b *Clickable, txt string) layout.Widget {
	return func(gtx C) D {
		min := gtx.Constraints.Min
		min.Y = gtx.Dp(st.Height)
		return b.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			semantic.Button.Add(gtx.Ops)
			return layout.Background{}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					rr := gtx.Dp(st.CornerRadius)
					defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, rr).Push(gtx.Ops).Pop()
					background := st.Background
					switch {
					case b.Hovered():
						background = hoveredColor(background)
					}
					paint.Fill(gtx.Ops, background)
					for _, c := range b.History() {
						drawInk(gtx, (widget.Press)(c))
					}
					return layout.Dimensions{Size: gtx.Constraints.Min}
				},
				func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min = min
					return layout.Center.Layout(gtx, func(gtx C) D {
						return st.Inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							colMacro := op.Record(gtx.Ops)
							paint.ColorOp{Color: st.Color}.Add(gtx.Ops)
							return widget.Label{Alignment: text.Middle}.Layout(gtx, th.Material.Shaper, st.Font, st.TextSize, txt, colMacro.Stop())
						})
					})
				},
			)
		})
	}
}

func IconBtn(th *Theme, st *IconButtonStyle, b *Clickable, icon *widget.Icon, enabled bool) layout.Widget {
	return func(gtx C) D {
		m := op.Record(gtx.Ops)
		dims := b.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			semantic.Button.Add(gtx.Ops)
			return layout.Background{}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					rr := (gtx.Constraints.Min.X + gtx.Constraints.Min.Y) / 4
					defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, rr).Push(gtx.Ops).Pop()
					background := st.Background
					switch {
					case b.Hovered():
						background = hoveredColor(background)
					}
					paint.Fill(gtx.Ops, background)
					for _, c := range b.History() {
						drawInk(gtx, (widget.Press)(c))
					}
					return layout.Dimensions{Size: gtx.Constraints.Min}
				},
				func(gtx layout.Context) layout.Dimensions {
					return st.Inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						size := gtx.Dp(st.Size)
						if icon != nil {
							gtx.Constraints.Min = image.Point{X: size}
							icon.Layout(gtx, st.Color)
						}
						return layout.Dimensions{
							Size: image.Point{X: size, Y: size},
						}
					})
				},
			)
		})
		c := m.Stop()
		bounds := image.Rectangle{Max: dims.Size}
		defer clip.Ellipse(bounds).Push(gtx.Ops).Pop()
		c.Add(gtx.Ops)
		return dims
	}
}

func Tooltip(th *Theme, tip string) component.Tooltip {
	tooltip := component.PlatformTooltip(&th.Material, tip)
	tooltip.Bg = th.Tooltip.Bg
	tooltip.Text.Color = th.Tooltip.Color
	return tooltip
}

// Click executes a simple programmatic click.
func (b *Clickable) Click() {
	b.requestClicks++
}

// Clicked calls Update and reports whether a click was registered.
func (b *Clickable) Clicked(gtx layout.Context) bool {
	return b.clicked(b, gtx)
}

func (b *Clickable) clicked(t event.Tag, gtx layout.Context) bool {
	_, clicked := b.update(t, gtx)
	return clicked
}

// Hovered reports whether a pointer is over the element.
func (b *Clickable) Hovered() bool {
	return b.click.Hovered()
}

// Pressed reports whether a pointer is pressing the element.
func (b *Clickable) Pressed() bool {
	return b.click.Pressed()
}

// History is the past pointer presses useful for drawing markers.
// History is retained for a short duration (about a second).
func (b *Clickable) History() []widget.Press {
	return b.history
}

// Layout and update the button state.
func (b *Clickable) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	return b.layout(b, gtx, w)
}

func (b *Clickable) layout(t event.Tag, gtx layout.Context, w layout.Widget) layout.Dimensions {
	for {
		_, ok := b.update(t, gtx)
		if !ok {
			break
		}
	}
	m := op.Record(gtx.Ops)
	dims := w(gtx)
	c := m.Stop()
	defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
	semantic.EnabledOp(gtx.Enabled()).Add(gtx.Ops)
	b.click.Add(gtx.Ops)
	event.Op(gtx.Ops, t)
	c.Add(gtx.Ops)
	return dims
}

// Update the button state by processing events, and return the next
// click, if any.
func (b *Clickable) Update(gtx layout.Context) (widget.Click, bool) {
	return b.update(b, gtx)
}

func (b *Clickable) update(_ event.Tag, gtx layout.Context) (widget.Click, bool) {
	for len(b.history) > 0 {
		c := b.history[0]
		if c.End.IsZero() || gtx.Now.Sub(c.End) < 1*time.Second {
			break
		}
		n := copy(b.history, b.history[1:])
		b.history = b.history[:n]
	}
	if c := b.requestClicks; c > 0 {
		b.requestClicks = 0
		return widget.Click{
			NumClicks: c,
		}, true
	}
	for {
		e, ok := b.click.Update(gtx.Source)
		if !ok {
			break
		}
		switch e.Kind {
		case gesture.KindClick:
			if l := len(b.history); l > 0 {
				b.history[l-1].End = gtx.Now
			}
			return widget.Click{
				Modifiers: e.Modifiers,
				NumClicks: e.NumClicks,
			}, true
		case gesture.KindCancel:
			for i := range b.history {
				b.history[i].Cancelled = true
				if b.history[i].End.IsZero() {
					b.history[i].End = gtx.Now
				}
			}
		case gesture.KindPress:
			b.history = append(b.history, widget.Press{
				Position: e.Position,
				Start:    gtx.Now,
			})
		}
	}
	return widget.Click{}, false
}

func drawInk(gtx layout.Context, c widget.Press) {
	// duration is the number of seconds for the
	// completed animation: expand while fading in, then
	// out.
	const (
		expandDuration = float32(0.5)
		fadeDuration   = float32(0.9)
	)

	now := gtx.Now

	t := float32(now.Sub(c.Start).Seconds())

	end := c.End
	if end.IsZero() {
		// If the press hasn't ended, don't fade-out.
		end = now
	}

	endt := float32(end.Sub(c.Start).Seconds())

	// Compute the fade-in/out position in [0;1].
	var alphat float32
	{
		var haste float32
		if c.Cancelled {
			// If the press was cancelled before the inkwell
			// was fully faded in, fast forward the animation
			// to match the fade-out.
			if h := 0.5 - endt/fadeDuration; h > 0 {
				haste = h
			}
		}
		// Fade in.
		half1 := t/fadeDuration + haste
		if half1 > 0.5 {
			half1 = 0.5
		}

		// Fade out.
		half2 := float32(now.Sub(end).Seconds())
		half2 /= fadeDuration
		half2 += haste
		if half2 > 0.5 {
			// Too old.
			return
		}

		alphat = half1 + half2
	}

	// Compute the expand position in [0;1].
	sizet := t
	if c.Cancelled {
		// Freeze expansion of cancelled presses.
		sizet = endt
	}
	sizet /= expandDuration

	// Animate only ended presses, and presses that are fading in.
	if !c.End.IsZero() || sizet <= 1.0 {
		gtx.Execute(op.InvalidateCmd{})
	}

	if sizet > 1.0 {
		sizet = 1.0
	}

	if alphat > .5 {
		// Start fadeout after half the animation.
		alphat = 1.0 - alphat
	}
	// Twice the speed to attain fully faded in at 0.5.
	t2 := alphat * 2
	// Beziér ease-in curve.
	alphaBezier := t2 * t2 * (3.0 - 2.0*t2)
	sizeBezier := sizet * sizet * (3.0 - 2.0*sizet)
	size := gtx.Constraints.Min.X
	if h := gtx.Constraints.Min.Y; h > size {
		size = h
	}
	// Cover the entire constraints min rectangle and
	// apply curve values to size and color.
	size = int(float32(size) * 2 * float32(math.Sqrt(2)) * sizeBezier)
	alpha := 0.7 * alphaBezier
	const col = 0.8
	ba, bc := byte(alpha*0xff), byte(col*0xff)
	rgba := color.NRGBA{A: 0xff, R: bc, G: bc, B: bc}
	rgba.A = uint8(uint32(rgba.A) * uint32(ba) / 0xFF)
	ink := paint.ColorOp{Color: rgba}
	ink.Add(gtx.Ops)
	rr := size / 2
	defer op.Offset(c.Position.Add(image.Point{
		X: -rr,
		Y: -rr,
	})).Push(gtx.Ops).Pop()
	defer clip.UniformRRect(image.Rectangle{Max: image.Pt(size, size)}, rr).Push(gtx.Ops).Pop()
	paint.PaintOp{}.Add(gtx.Ops)
}

func hoveredColor(c color.NRGBA) (h color.NRGBA) {
	if c.A == 0 {
		// Provide a reasonable default for transparent widgets.
		return color.NRGBA{A: 0x44, R: 0x88, G: 0x88, B: 0x88}
	}
	const ratio = 0x20
	m := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: c.A}
	if int(c.R)+int(c.G)+int(c.B) > 384 {
		m = color.NRGBA{A: c.A}
	}
	return mix(m, c, ratio)
}

// mix mixes c1 and c2 weighted by (1 - a/256) and a/256 respectively.
func mix(c1, c2 color.NRGBA, a uint8) color.NRGBA {
	ai := int(a)
	return color.NRGBA{
		R: byte((int(c1.R)*ai + int(c2.R)*(256-ai)) / 256),
		G: byte((int(c1.G)*ai + int(c2.G)*(256-ai)) / 256),
		B: byte((int(c1.B)*ai + int(c2.B)*(256-ai)) / 256),
		A: byte((int(c1.A)*ai + int(c2.A)*(256-ai)) / 256),
	}
}
