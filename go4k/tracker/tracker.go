package tracker

import "gioui.org/widget"

type Tracker struct {
	QuitButton *widget.Clickable
}

func New() *Tracker {
	return &Tracker{
		QuitButton: new(widget.Clickable),
	}
}
