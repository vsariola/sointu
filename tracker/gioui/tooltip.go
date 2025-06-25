package gioui

import (
	"image"
	"image/color"
	"time"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/x/component"
)

// TipArea holds the state information for displaying a tooltip. The zero
// value will choose sensible defaults for all fields.
type TipArea struct {
	component.VisibilityAnimation
	Hover     component.InvalidateDeadline
	Press     component.InvalidateDeadline
	LongPress component.InvalidateDeadline
	Exit      component.InvalidateDeadline
	init      bool
	// HoverDelay is the delay between the cursor entering the tip area
	// and the tooltip appearing.
	HoverDelay time.Duration
	// LongPressDelay is the required duration of a press in the area for
	// it to count as a long press.
	LongPressDelay time.Duration
	// LongPressDuration is the amount of time the tooltip should be displayed
	// after being triggered by a long press.
	LongPressDuration time.Duration
	// FadeDuration is the amount of time it takes the tooltip to fade in
	// and out.
	FadeDuration time.Duration
	// ExitDuration is the amount of time the tooltip will remain visible at
	// maximum, to avoid tooltips staying visible indefinitely if the user
	// managed to leave the area without triggering a pointer.Leave event.
	ExitDuration time.Duration
}

const (
	tipAreaHoverDelay        = time.Millisecond * 500
	tipAreaLongPressDuration = time.Millisecond * 1500
	tipAreaFadeDuration      = time.Millisecond * 250
	longPressTheshold        = time.Millisecond * 500
	tipAreaExitDelay         = time.Millisecond * 5000
)

// Layout renders the provided widget with the provided tooltip. The tooltip
// will be summoned if the widget is hovered or long-pressed.
func (t *TipArea) Layout(gtx C, tip component.Tooltip, w layout.Widget) D {
	if !t.init {
		t.init = true
		t.VisibilityAnimation.State = component.Invisible
		if t.HoverDelay == time.Duration(0) {
			t.HoverDelay = tipAreaHoverDelay
		}
		if t.LongPressDelay == time.Duration(0) {
			t.LongPressDelay = longPressTheshold
		}
		if t.LongPressDuration == time.Duration(0) {
			t.LongPressDuration = tipAreaLongPressDuration
		}
		if t.FadeDuration == time.Duration(0) {
			t.FadeDuration = tipAreaFadeDuration
		}
		if t.ExitDuration == time.Duration(0) {
			t.ExitDuration = tipAreaExitDelay
		}
		t.VisibilityAnimation.Duration = t.FadeDuration
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: t,
			Kinds:  pointer.Press | pointer.Release | pointer.Enter | pointer.Leave,
		})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		// regardless of the event, we reset the exit timer to avoid tooltips
		// staying visible indefinitely
		t.Exit.SetTarget(gtx.Now.Add(t.ExitDuration))
		switch e.Kind {
		case pointer.Enter:
			t.Hover.SetTarget(gtx.Now.Add(t.HoverDelay))
			t.Exit.SetTarget(gtx.Now.Add(t.ExitDuration))
		case pointer.Leave:
			t.VisibilityAnimation.Disappear(gtx.Now)
			t.Hover.ClearTarget()
		case pointer.Press:
			t.Press.SetTarget(gtx.Now.Add(t.LongPressDelay))
		case pointer.Release:
			t.Press.ClearTarget()
		case pointer.Cancel:
			t.Hover.ClearTarget()
			t.Press.ClearTarget()
		}
	}
	if t.Hover.Process(gtx) {
		t.VisibilityAnimation.Appear(gtx.Now)
	}
	if t.Press.Process(gtx) {
		t.VisibilityAnimation.Appear(gtx.Now)
		t.LongPress.SetTarget(gtx.Now.Add(t.LongPressDuration))
	}
	if t.LongPress.Process(gtx) {
		t.VisibilityAnimation.Disappear(gtx.Now)
	}
	if t.Exit.Process(gtx) {
		t.VisibilityAnimation.Disappear(gtx.Now)
	}
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(w),
		layout.Expanded(func(gtx C) D {
			defer pointer.PassOp{}.Push(gtx.Ops).Pop()
			defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()
			event.Op(gtx.Ops, t)

			originalMin := gtx.Constraints.Min
			gtx.Constraints.Min = image.Point{}

			if t.Visible() {
				macro := op.Record(gtx.Ops)
				tip.Bg = component.Interpolate(color.NRGBA{}, tip.Bg, t.VisibilityAnimation.Revealed(gtx))
				dims := tip.Layout(gtx)
				call := macro.Stop()
				xOffset := (originalMin.X / 2) - (dims.Size.X / 2)
				yOffset := originalMin.Y
				macro = op.Record(gtx.Ops)
				op.Offset(image.Pt(xOffset, yOffset)).Add(gtx.Ops)
				call.Add(gtx.Ops)
				call = macro.Stop()
				op.Defer(gtx.Ops, call)
			}
			return D{}
		}),
	)
}

func Tooltip(th *Theme, tip string) component.Tooltip {
	tooltip := component.PlatformTooltip(&th.Material, tip)
	tooltip.Bg = th.Tooltip.Bg
	tooltip.Text.Color = th.Tooltip.Color
	return tooltip
}
