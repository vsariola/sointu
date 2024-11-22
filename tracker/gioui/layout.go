package gioui

import "gioui.org/layout"

// general helpers for layout that do not belong to any specific widget

func EmptyWidget() layout.Spacer {
	return layout.Spacer{}
}

func OnlyIf(condition bool, widget layout.Widget) layout.Widget {
	if condition {
		return widget
	}
	return EmptyWidget().Layout
}
