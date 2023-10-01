//go:build plugin

package gioui

const CAN_QUIT = false

func (t *Tracker) Quit(forced bool) bool {
	t.sendQuit()
	return forced
}
