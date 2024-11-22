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
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"github.com/vsariola/sointu/tracker"
)

type (
	TipClickable struct {
		Clickable Clickable
		TipArea   component.TipArea
	}

	ActionClickable struct {
		Action tracker.Action
		TipClickable
	}

	TipIconButtonStyle struct {
		TipArea         *component.TipArea
		IconButtonStyle IconButtonStyle
		Tooltip         component.Tooltip
	}

	BoolClickable struct {
		Clickable Clickable
		TipArea   component.TipArea
		Bool      tracker.Bool
	}

	MenuClickable struct {
		Clickable Clickable
		menu      Menu
		Selected  tracker.OptionalInt
		TipArea   component.TipArea
		Tooltip   component.Tooltip
	}
)

func NewActionClickable(a tracker.Action) *ActionClickable {
	return &ActionClickable{
		Action: a,
	}
}

func NewBoolClickable(b tracker.Bool) *BoolClickable {
	return &BoolClickable{
		Bool: b,
	}
}

func Tooltip(th *material.Theme, tip string) component.Tooltip {
	tooltip := component.PlatformTooltip(th, tip)
	tooltip.Bg = black
	return tooltip
}

func ActionIcon(gtx C, th *material.Theme, w *ActionClickable, icon []byte, tip string) TipIconButtonStyle {
	ret := TipIcon(th, &w.TipClickable, icon, tip)
	for w.Clickable.Clicked(gtx) {
		w.Action.Do()
	}
	if !w.Action.Allowed() {
		ret.IconButtonStyle.Color = disabledTextColor
	}
	return ret
}

func TipIcon(th *material.Theme, w *TipClickable, icon []byte, tip string) TipIconButtonStyle {
	iconButtonStyle := IconButton(th, &w.Clickable, widgetForIcon(icon), "")
	iconButtonStyle.Color = primaryColor
	iconButtonStyle.Background = transparent
	iconButtonStyle.Inset = layout.UniformInset(unit.Dp(6))
	return TipIconButtonStyle{
		TipArea:         &w.TipArea,
		IconButtonStyle: iconButtonStyle,
		Tooltip:         Tooltip(th, tip),
	}
}

func ToggleIcon(gtx C, th *material.Theme, w *BoolClickable, offIcon, onIcon []byte, offTip, onTip string) TipIconButtonStyle {
	icon := offIcon
	tip := offTip
	if w.Bool.Value() {
		icon = onIcon
		tip = onTip
	}
	for w.Clickable.Clicked(gtx) {
		w.Bool.Toggle()
	}
	ibStyle := IconButton(th, &w.Clickable, widgetForIcon(icon), "")
	ibStyle.Background = transparent
	ibStyle.Inset = layout.UniformInset(unit.Dp(6))
	ibStyle.Color = primaryColor
	if !w.Bool.Enabled() {
		ibStyle.Color = disabledTextColor
	}
	return TipIconButtonStyle{
		TipArea:         &w.TipArea,
		IconButtonStyle: ibStyle,
		Tooltip:         Tooltip(th, tip),
	}
}

func (t *TipIconButtonStyle) Layout(gtx C) D {
	return t.TipArea.Layout(gtx, t.Tooltip, t.IconButtonStyle.Layout)
}

func ActionButton(gtx C, th *material.Theme, w *ActionClickable, text string) ButtonStyle {
	for w.Clickable.Clicked(gtx) {
		w.Action.Do()
	}
	ret := Button(th, &w.Clickable, text)
	ret.Color = th.Palette.Fg
	if !w.Action.Allowed() {
		ret.Color = disabledTextColor
	}
	ret.Background = transparent
	ret.Inset = layout.UniformInset(unit.Dp(6))
	return ret
}

func ToggleButton(gtx C, th *material.Theme, b *BoolClickable, text string) ButtonStyle {
	for b.Clickable.Clicked(gtx) {
		b.Bool.Toggle()
	}
	ret := Button(th, &b.Clickable, text)
	ret.Background = transparent
	ret.Inset = layout.UniformInset(unit.Dp(6))
	if !b.Bool.Enabled() {
		ret.Color = disabledTextColor
		ret.Background = transparent
	} else if b.Bool.Value() {
		ret.Color = th.Palette.ContrastFg
		ret.Background = th.Palette.Fg
	} else {
		ret.Color = th.Palette.Fg
		ret.Background = transparent
	}
	return ret
}

func LowEmphasisButton(th *material.Theme, w *Clickable, text string) ButtonStyle {
	ret := Button(th, w, text)
	ret.Color = th.Palette.Fg
	ret.Background = transparent
	ret.Inset = layout.UniformInset(unit.Dp(6))
	return ret
}

func HighEmphasisButton(th *material.Theme, w *Clickable, text string) ButtonStyle {
	ret := Button(th, w, text)
	ret.Color = th.Palette.ContrastFg
	ret.Background = th.Palette.Fg
	ret.Inset = layout.UniformInset(unit.Dp(6))
	return ret
}

// Clickable represents a clickable area.
type Clickable struct {
	click   gesture.Click
	history []widget.Press

	requestClicks int
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

func (b *Clickable) update(t event.Tag, gtx layout.Context) (widget.Click, bool) {
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

type ButtonStyle struct {
	Text string
	// Color is the text color.
	Color        color.NRGBA
	Font         font.Font
	TextSize     unit.Sp
	Background   color.NRGBA
	CornerRadius unit.Dp
	Inset        layout.Inset
	Button       *Clickable
	shaper       *text.Shaper
	Hidden       bool
}

type ButtonLayoutStyle struct {
	Background   color.NRGBA
	CornerRadius unit.Dp
	Button       *Clickable
}

type IconButtonStyle struct {
	Background color.NRGBA
	// Color is the icon color.
	Color color.NRGBA
	Icon  *widget.Icon
	// Size is the icon size.
	Size        unit.Dp
	Inset       layout.Inset
	Button      *Clickable
	Description string
}

func Button(th *material.Theme, button *Clickable, txt string) ButtonStyle {
	b := ButtonStyle{
		Text:         txt,
		Color:        th.Palette.ContrastFg,
		CornerRadius: 4,
		Background:   th.Palette.ContrastBg,
		TextSize:     th.TextSize * 14.0 / 16.0,
		Inset: layout.Inset{
			Top: 10, Bottom: 10,
			Left: 12, Right: 12,
		},
		Button: button,
		shaper: th.Shaper,
	}
	b.Font.Typeface = th.Face
	return b
}

func ButtonLayout(th *material.Theme, button *Clickable) ButtonLayoutStyle {
	return ButtonLayoutStyle{
		Button:       button,
		Background:   th.Palette.ContrastBg,
		CornerRadius: 4,
	}
}

func IconButton(th *material.Theme, button *Clickable, icon *widget.Icon, description string) IconButtonStyle {
	return IconButtonStyle{
		Background:  th.Palette.ContrastBg,
		Color:       th.Palette.ContrastFg,
		Icon:        icon,
		Size:        24,
		Inset:       layout.UniformInset(12),
		Button:      button,
		Description: description,
	}
}

func (b ButtonStyle) Layout(gtx layout.Context) layout.Dimensions {
	return ButtonLayoutStyle{
		Background:   b.Background,
		CornerRadius: b.CornerRadius,
		Button:       b.Button,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if b.Hidden {
			return layout.Dimensions{}
		}
		return b.Inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			colMacro := op.Record(gtx.Ops)
			paint.ColorOp{Color: b.Color}.Add(gtx.Ops)
			return widget.Label{Alignment: text.Middle}.Layout(gtx, b.shaper, b.Font, b.TextSize, b.Text, colMacro.Stop())
		})
	})
}

func (b ButtonLayoutStyle) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	min := gtx.Constraints.Min
	return b.Button.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		semantic.Button.Add(gtx.Ops)
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				rr := gtx.Dp(b.CornerRadius)
				defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, rr).Push(gtx.Ops).Pop()
				background := b.Background
				switch {
				case b.Button.Hovered():
					background = hoveredColor(background)
				}
				paint.Fill(gtx.Ops, background)
				for _, c := range b.Button.History() {
					drawInk(gtx, (widget.Press)(c))
				}
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min = min
				return layout.Center.Layout(gtx, w)
			},
		)
	})
}

func (b IconButtonStyle) Layout(gtx layout.Context) layout.Dimensions {
	m := op.Record(gtx.Ops)
	dims := b.Button.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		semantic.Button.Add(gtx.Ops)
		if d := b.Description; d != "" {
			semantic.DescriptionOp(b.Description).Add(gtx.Ops)
		}
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				rr := (gtx.Constraints.Min.X + gtx.Constraints.Min.Y) / 4
				defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, rr).Push(gtx.Ops).Pop()
				background := b.Background
				switch {
				case b.Button.Hovered():
					background = hoveredColor(background)
				}
				paint.Fill(gtx.Ops, background)
				for _, c := range b.Button.History() {
					drawInk(gtx, (widget.Press)(c))
				}
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			func(gtx layout.Context) layout.Dimensions {
				return b.Inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					size := gtx.Dp(b.Size)
					if b.Icon != nil {
						gtx.Constraints.Min = image.Point{X: size}
						b.Icon.Layout(gtx, b.Color)
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
