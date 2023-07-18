package gioui

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
)

type TipClickable struct {
	Clickable widget.Clickable
	TipArea   component.TipArea
}

type TipIconButtonStyle struct {
	IconButtonStyle material.IconButtonStyle
	Tooltip         component.Tooltip
	tipArea         *component.TipArea
}

func Tooltip(th *material.Theme, tip string) component.Tooltip {
	tooltip := component.PlatformTooltip(th, tip)
	tooltip.Bg = black
	return tooltip
}

func IconButton(th *material.Theme, w *TipClickable, icon []byte, enabled bool, tip string) TipIconButtonStyle {
	ret := material.IconButton(th, &w.Clickable, widgetForIcon(icon), "")
	ret.Background = transparent
	ret.Inset = layout.UniformInset(unit.Dp(6))
	if enabled {
		ret.Color = primaryColor
	} else {
		ret.Color = disabledTextColor
	}
	return TipIconButtonStyle{
		IconButtonStyle: ret,
		Tooltip:         Tooltip(th, tip),
		tipArea:         &w.TipArea,
	}
}

func (t *TipIconButtonStyle) Layout(gtx C) D {
	return t.tipArea.Layout(gtx, t.Tooltip, t.IconButtonStyle.Layout)
}

func LowEmphasisButton(th *material.Theme, w *widget.Clickable, text string) material.ButtonStyle {
	ret := material.Button(th, w, text)
	ret.Color = th.Palette.Fg
	ret.Background = transparent
	ret.Inset = layout.UniformInset(unit.Dp(6))
	return ret
}

func HighEmphasisButton(th *material.Theme, w *widget.Clickable, text string) material.ButtonStyle {
	ret := material.Button(th, w, text)
	ret.Color = th.Palette.ContrastFg
	ret.Background = th.Palette.Fg
	ret.Inset = layout.UniformInset(unit.Dp(6))
	return ret
}
