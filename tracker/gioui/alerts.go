package gioui

import (
	"image"
	"image/color"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/vsariola/sointu/tracker"
)

type (
	AlertsState struct {
		prevUpdate time.Time
	}

	AlertStyle struct {
		Bg   color.NRGBA
		Text LabelStyle
	}

	AlertStyles struct {
		Info    AlertStyle
		Warning AlertStyle
		Error   AlertStyle
		Margin  layout.Inset
		Inset   layout.Inset
	}

	AlertsWidget struct {
		Theme *Theme
		Model *tracker.Alerts
		State *AlertsState
	}
)

func NewAlertsState() *AlertsState {
	return &AlertsState{prevUpdate: time.Now()}
}

func Alerts(m *tracker.Alerts, th *Theme, st *AlertsState) AlertsWidget {
	return AlertsWidget{
		Theme: th,
		Model: m,
		State: st,
	}
}

func (a *AlertsWidget) Layout(gtx C) D {
	now := time.Now()
	if a.Model.Update(now.Sub(a.State.prevUpdate)) {
		gtx.Execute(op.InvalidateCmd{At: now.Add(50 * time.Millisecond)})
	}
	a.State.prevUpdate = now

	var totalY float64 = float64(gtx.Dp(38))
	for _, alert := range a.Model.Iterate {
		var alertStyle *AlertStyle
		switch alert.Priority {
		case tracker.Warning:
			alertStyle = &a.Theme.Alert.Warning
		case tracker.Error:
			alertStyle = &a.Theme.Alert.Error
		default:
			alertStyle = &a.Theme.Alert.Info
		}
		bgWidget := func(gtx C) D {
			paint.FillShape(gtx.Ops, alertStyle.Bg, clip.Rect{
				Max: gtx.Constraints.Min,
			}.Op())
			return D{Size: gtx.Constraints.Min}
		}
		labelStyle := Label(a.Theme, &alertStyle.Text, alert.Message)
		a.Theme.Alert.Margin.Layout(gtx, func(gtx C) D {
			return layout.S.Layout(gtx, func(gtx C) D {
				defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				recording := op.Record(gtx.Ops)
				dims := layout.Stack{Alignment: layout.Center}.Layout(gtx,
					layout.Expanded(bgWidget),
					layout.Stacked(func(gtx C) D {
						return a.Theme.Alert.Inset.Layout(gtx, labelStyle.Layout)
					}),
				)
				macro := recording.Stop()
				delta := float64(dims.Size.Y + gtx.Dp(a.Theme.Alert.Margin.Bottom))
				op.Offset(image.Point{0, int(-totalY*alert.FadeLevel + delta*(1-alert.FadeLevel))}).Add((gtx.Ops))
				totalY += delta
				macro.Add(gtx.Ops)
				return dims
			})
		})
	}
	return D{}
}
