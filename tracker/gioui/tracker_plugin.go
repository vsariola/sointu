//go:build plugin

package gioui

const CAN_QUIT = false

func (t *Tracker) Quit(forced bool) bool {
	t.quitted = forced
	return forced
}
