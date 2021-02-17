package tracker

import (
	"image/color"
	"time"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

type Alert struct {
	message       string
	alertType     AlertType
	duration      time.Duration
	showMessage   string
	showAlertType AlertType
	showDuration  time.Duration
	showTime      time.Time
	pos           float64
	lastUpdate    time.Time
}

type AlertType int

const (
	None AlertType = iota
	Notify
	Warning
	Error
)

var alertSpeed = 150 * time.Millisecond
var alertMargin = layout.UniformInset(unit.Dp(6))
var alertInset = layout.UniformInset(unit.Dp(6))

func (a *Alert) Update(message string, alertType AlertType, duration time.Duration) {
	if a.alertType < alertType {
		a.message = message
		a.alertType = alertType
		a.duration = duration
	}
}

func (a *Alert) Layout(gtx C) D {
	now := time.Now()
	if a.alertType != None {
		a.showMessage = a.message
		a.showAlertType = a.alertType
		a.showTime = now
		a.showDuration = a.duration
	}
	a.alertType = None
	var targetPos float64 = 0.0
	if now.Sub(a.showTime) <= a.showDuration {
		targetPos = 1.0
	}
	delta := float64(now.Sub(a.lastUpdate)) / float64(alertSpeed)
	if a.pos < targetPos {
		a.pos += delta
		if a.pos > targetPos {
			a.pos = targetPos
		} else {
			op.InvalidateOp{At: now.Add(50 * time.Millisecond)}.Add(gtx.Ops)
		}
	} else if a.pos > targetPos {
		a.pos -= delta
		if a.pos < targetPos {
			a.pos = targetPos
		} else {
			op.InvalidateOp{At: now.Add(50 * time.Millisecond)}.Add(gtx.Ops)
		}
	}
	a.lastUpdate = now
	var color, textColor, shadeColor color.NRGBA
	switch a.showAlertType {
	case Warning:
		color = warningColor
		textColor = black
	case Error:
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
	labelStyle := LabelStyle{Text: a.showMessage, Color: textColor, ShadeColor: shadeColor, Font: labelDefaultFont, Alignment: layout.Center, FontSize: unit.Dp(16)}
	return alertMargin.Layout(gtx, func(gtx C) D {
		return layout.S.Layout(gtx, func(gtx C) D {
			defer op.Save(gtx.Ops).Load()
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			recording := op.Record(gtx.Ops)
			dims := layout.Stack{Alignment: layout.Center}.Layout(gtx,
				layout.Expanded(bgWidget),
				layout.Stacked(func(gtx C) D {
					return alertInset.Layout(gtx, labelStyle.Layout)
				}),
			)
			macro := recording.Stop()
			totalY := dims.Size.Y + gtx.Px(alertMargin.Bottom)
			op.Offset(f32.Pt(0, float32((1-a.pos)*float64(totalY)))).Add((gtx.Ops))
			macro.Add(gtx.Ops)
			return dims
		})
	})
}
