package tracker

import (
	"log"

	"gioui.org/widget"
)

var iconCache = map[*byte]*widget.Icon{}

// widgetForIcon returns a widget for IconVG data, but caching the results
func widgetForIcon(icon []byte) *widget.Icon {
	if widget, ok := iconCache[&icon[0]]; ok {
		return widget
	}
	widget, err := widget.NewIcon(icon)
	if err != nil {
		log.Fatal(err)
	}
	iconCache[&icon[0]] = widget
	return widget
}
