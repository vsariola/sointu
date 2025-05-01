package gioui

import (
	"image"
	"image/color"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/vsariola/sointu/tracker"
)

type PopupAlert struct {
	alerts     *tracker.Alerts
	prevUpdate time.Time
}

type PopupAlertStyle struct {
	Bg   color.NRGBA
	Text LabelStyle
}

var alertMargin = layout.UniformInset(unit.Dp(6))
var alertInset = layout.UniformInset(unit.Dp(6))

func NewPopupAlert(alerts *tracker.Alerts) *PopupAlert {
	return &PopupAlert{alerts: alerts, prevUpdate: time.Now()}
}

func (a *PopupAlert) Layout(gtx C, th *Theme) D {
	now := time.Now()
	if a.alerts.Update(now.Sub(a.prevUpdate)) {
		gtx.Execute(op.InvalidateCmd{At: now.Add(50 * time.Millisecond)})
	}
	a.prevUpdate = now

	var totalY float64 = float64(gtx.Dp(38))
	for _, alert := range a.alerts.Iterate {
		var alertStyle *PopupAlertStyle
		switch alert.Priority {
		case tracker.Warning:
			alertStyle = &th.Alert.Warning
		case tracker.Error:
			alertStyle = &th.Alert.Error
		default:
			alertStyle = &th.Alert.Info
		}
		bgWidget := func(gtx C) D {
			paint.FillShape(gtx.Ops, alertStyle.Bg, clip.Rect{
				Max: gtx.Constraints.Min,
			}.Op())
			return D{Size: gtx.Constraints.Min}
		}
		labelStyle := Label(th, &alertStyle.Text, alert.Message)
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
	}
	return D{}
}
