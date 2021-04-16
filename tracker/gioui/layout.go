package gioui

import (
	"image"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/vsariola/sointu/tracker"
)

type C = layout.Context
type D = layout.Dimensions

func (t *Tracker) Layout(gtx layout.Context) {
	paint.FillShape(gtx.Ops, backgroundColor, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())
	t.VerticalSplit.Layout(gtx,
		t.layoutTop,
		t.layoutBottom)
	t.Alert.Layout(gtx)
	dstyle := ConfirmDialog(t.Theme, t.ConfirmInstrDelete, "Are you sure you want to delete this instrument?")
	dstyle.Layout(gtx)
	dstyle = ConfirmDialog(t.Theme, t.ConfirmSongDialog, "Do you want to save your changes to the song? Your changes will be lost if you don't save them.")
	dstyle.ShowAlt = true
	dstyle.OkStyle.Text = "Save"
	dstyle.AltStyle.Text = "Don't save"
	dstyle.Layout(gtx)
	for t.ConfirmSongDialog.BtnOk.Clicked() {
		if t.SaveSongFile() {
			t.confirmedSongAction()
		}
		t.ConfirmSongDialog.Visible = false
	}
	for t.ConfirmSongDialog.BtnAlt.Clicked() {
		t.confirmedSongAction()
		t.ConfirmSongDialog.Visible = false
	}
	for t.ConfirmSongDialog.BtnCancel.Clicked() {
		t.ConfirmSongDialog.Visible = false
	}
}

func (t *Tracker) confirmedSongAction() {
	switch t.ConfirmSongActionType {
	case ConfirmLoad:
		t.loadSong()
	case ConfirmNew:
		t.ResetSong()
		t.SetFilePath("")
		t.window.Option(app.Title("Sointu Tracker"))
		t.ClearUndoHistory()
		t.SetChangedSinceSave(false)
	case ConfirmQuit:
		t.quitted = true
	}
}

func (t *Tracker) TryResetSong() {
	if t.ChangedSinceSave() {
		t.ConfirmSongActionType = ConfirmNew
		t.ConfirmSongDialog.Visible = true
		return
	}
	t.ResetSong()
	t.SetFilePath("")
	t.window.Option(app.Title("Sointu Tracker"))
	t.ClearUndoHistory()
	t.SetChangedSinceSave(false)
}

func (t *Tracker) TryQuit() bool {
	if t.ChangedSinceSave() {
		t.ConfirmSongActionType = ConfirmQuit
		t.ConfirmSongDialog.Visible = true
		return false
	}
	t.quitted = true
	return true
}

func (t *Tracker) layoutBottom(gtx layout.Context) layout.Dimensions {
	return t.BottomHorizontalSplit.Layout(gtx,
		func(gtx C) D {
			return Surface{Gray: 24, Focus: t.EditMode() == tracker.EditPatterns}.Layout(gtx, t.layoutPatterns)
		},
		func(gtx C) D {
			return Surface{Gray: 24, Focus: t.EditMode() == tracker.EditTracks}.Layout(gtx, t.layoutTracker)
		},
	)
}

func (t *Tracker) layoutTop(gtx layout.Context) layout.Dimensions {
	return t.TopHorizontalSplit.Layout(gtx,
		t.layoutSongPanel,
		t.layoutInstruments,
	)
}
