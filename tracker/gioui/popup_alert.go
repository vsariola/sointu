package gioui

import (
	"image"
	"image/color"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"github.com/vsariola/sointu/tracker"
)

type PopupAlert struct {
	alerts     *tracker.Alerts
	prevUpdate time.Time
	shaper     *text.Shaper
}

var alertSpeed = 150 * time.Millisecond
var alertMargin = layout.UniformInset(unit.Dp(6))
var alertInset = layout.UniformInset(unit.Dp(6))

func NewPopupAlert(alerts *tracker.Alerts, shaper *text.Shaper) *PopupAlert {
	return &PopupAlert{alerts: alerts, shaper: shaper, prevUpdate: time.Now()}
}

func (a *PopupAlert) Layout(gtx C) D {
	now := time.Now()
	if a.alerts.Update(now.Sub(a.prevUpdate)) {
		op.InvalidateOp{At: now.Add(50 * time.Millisecond)}.Add(gtx.Ops)
	}
	a.prevUpdate = now

	var totalY float64
	a.alerts.Iterate(func(alert tracker.Alert) {
		var color, textColor, shadeColor color.NRGBA
		switch alert.Priority {
		case tracker.Warning:
			color = warningColor
			textColor = black
		case tracker.Error:
			color = errorColor
			textColor = black
		default:
			color = popupSurfaceColor
			textColor = white
			shadeColor = black
		}
		bgWidget := func(gtx C) D {
			paint.FillShape(gtx.Ops, color, clip.Rect{
				Max: gtx.Constraints.Min,
			}.Op())
			return D{Size: gtx.Constraints.Min}
		}
		labelStyle := LabelStyle{Text: alert.Message, Color: textColor, ShadeColor: shadeColor, Font: labelDefaultFont, Alignment: layout.Center, FontSize: unit.Sp(16), Shaper: a.shaper}
		alertMargin.Layout(gtx, func(gtx C) D {
			return layout.S.Layout(gtx, func(gtx C) D {
				defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				recording := op.Record(gtx.Ops)
				dims := layout.Stack{Alignment: layout.Center}.Layout(gtx,
					layout.Expanded(bgWidget),
					layout.Stacked(func(gtx C) D {
						return alertInset.Layout(gtx, labelStyle.Layout)
					}),
				)
				macro := recording.Stop()
				delta := float64(dims.Size.Y + gtx.Dp(alertMargin.Bottom))
				op.Offset(image.Point{0, int(-totalY*alert.FadeLevel + delta*(1-alert.FadeLevel))}).Add((gtx.Ops))
				totalY += delta
				macro.Add(gtx.Ops)
				return dims
			})
		})
	})
	return D{}
}
