package tracker

import (
	"gioui.org/io/key"
	"os"
)

// KeyEvent handles incoming key events and returns true if repaint is needed.
func (t *Tracker) KeyEvent(e key.Event) bool {
	if e.State == key.Press {
		switch e.Name {
		case key.NameEscape:
			os.Exit(0)
		}
	}
	return false
}
