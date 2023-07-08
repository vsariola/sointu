package gioui

import (
	"image"

	"gioui.org/app"
	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

type C = layout.Context
type D = layout.Dimensions

func (t *Tracker) Layout(gtx layout.Context, w *app.Window) {
	// this is the top level input handler for the whole app
	// it handles all the global key events and clipboard events
	// we need to tell gio that we handle tabs too; otherwise
	// it will steal them for focus switching
	key.InputOp{Tag: t, Keys: "Tab|Shift-Tab"}.Add(gtx.Ops)
	for _, ev := range gtx.Events(t) {
		switch e := ev.(type) {
		case key.Event:
			t.KeyEvent(e, gtx.Ops)
		case clipboard.Event:
			t.UnmarshalContent([]byte(e.Text))
		}
	}

	paint.FillShape(gtx.Ops, backgroundColor, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())
	if t.InstrEnlarged() {
		t.layoutTop(gtx)
	} else {
		t.VerticalSplit.Layout(gtx,
			t.layoutTop,
			t.layoutBottom)
	}
	t.Alert.Layout(gtx)
	dstyle := ConfirmDialog(t.Theme, t.ConfirmSongDialog, "Do you want to save your changes to the song? Your changes will be lost if you don't save them.")
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
	dstyle = ConfirmDialog(t.Theme, t.WaveTypeDialog, "Export .wav in int16 or float32 sample format?")
	dstyle.ShowAlt = true
	dstyle.OkStyle.Text = "Int16"
	dstyle.AltStyle.Text = "Float32"
	dstyle.Layout(gtx)
	for t.WaveTypeDialog.BtnOk.Clicked() {
		t.ExportWav(true)
		t.WaveTypeDialog.Visible = false
	}
	for t.WaveTypeDialog.BtnAlt.Clicked() {
		t.ExportWav(false)
		t.WaveTypeDialog.Visible = false
	}
	for t.WaveTypeDialog.BtnCancel.Clicked() {
		t.WaveTypeDialog.Visible = false
	}
	if t.ModalDialog != nil {
		t.ModalDialog(gtx)
	}
}

func (t *Tracker) confirmedSongAction() {
	switch t.ConfirmSongActionType {
	case ConfirmLoad:
		t.OpenSongFile(true)
	case ConfirmNew:
		t.NewSong(true)
	case ConfirmQuit:
		t.Quit(true)
	}
}

func (t *Tracker) NewSong(forced bool) {
	if !forced && t.ChangedSinceSave() {
		t.ConfirmSongActionType = ConfirmNew
		t.ConfirmSongDialog.Visible = true
		return
	}
	t.ResetSong()
	t.SetFilePath("")
	t.ClearUndoHistory()
	t.SetChangedSinceSave(false)
}

func (t *Tracker) layoutBottom(gtx layout.Context) layout.Dimensions {
	return t.BottomHorizontalSplit.Layout(gtx,
		func(gtx C) D {
			return t.OrderEditor.Layout(gtx, t)
		},
		func(gtx C) D {
			return t.TrackEditor.Layout(gtx, t)
		},
	)
}

func (t *Tracker) layoutTop(gtx layout.Context) layout.Dimensions {
	return t.TopHorizontalSplit.Layout(gtx,
		t.layoutSongPanel,
		func(gtx C) D {
			return t.InstrumentEditor.Layout(gtx, t)
		},
	)
}
