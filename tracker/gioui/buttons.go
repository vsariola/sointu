package gioui

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func IconButton(th *material.Theme, w *widget.Clickable, icon []byte, enabled bool) material.IconButtonStyle {
	ret := material.IconButton(th, w, widgetForIcon(icon))
	ret.Background = transparent
	ret.Inset = layout.UniformInset(unit.Dp(6))
	if enabled {
		ret.Color = primaryColor
	} else {
		ret.Color = disabledTextColor
	}
	return ret
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
