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
