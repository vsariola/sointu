//go:build !plugin

package gioui

const CAN_QUIT = true

func (t *Tracker) Quit(forced bool) bool {
	if !forced && t.ChangedSinceSave() {
		t.ConfirmSongActionType = ConfirmQuit
		t.ConfirmSongDialog.Visible = true
		return false
	}
	t.quitted = true
	return true
}
