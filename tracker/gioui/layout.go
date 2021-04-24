package gioui

import (
	"fmt"
	"image"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

type C = layout.Context
type D = layout.Dimensions

func (t *Tracker) Layout(gtx layout.Context) {
	paint.FillShape(gtx.Ops, backgroundColor, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())
	t.VerticalSplit.Layout(gtx,
		t.layoutTop,
		t.layoutBottom)
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
		t.exportWav(t.wavFilePath, true)
		t.WaveTypeDialog.Visible = false
	}
	for t.WaveTypeDialog.BtnAlt.Clicked() {
		t.exportWav(t.wavFilePath, false)
		t.WaveTypeDialog.Visible = false
	}
	for t.WaveTypeDialog.BtnCancel.Clicked() {
		t.WaveTypeDialog.Visible = false
	}
	fstyle := OpenFileDialog(t.Theme, t.OpenSongDialog)
	fstyle.Title = "Open Song File"
	fstyle.Layout(gtx)
	for ok, file := t.OpenSongDialog.FileSelected(); ok; ok, file = t.OpenSongDialog.FileSelected() {
		t.loadSong(file)
	}
	fstyle = SaveFileDialog(t.Theme, t.SaveSongDialog)
	fstyle.Title = "Save Song As"
	for ok, file := t.SaveSongDialog.FileSelected(); ok; ok, file = t.SaveSongDialog.FileSelected() {
		t.saveSong(file)
	}
	fstyle.Layout(gtx)
	exportWavDialogStyle := SaveFileDialog(t.Theme, t.ExportWavDialog)
	exportWavDialogStyle.Title = "Export Song As Wav"
	for ok, file := t.ExportWavDialog.FileSelected(); ok; ok, file = t.ExportWavDialog.FileSelected() {
		t.wavFilePath = file
		t.WaveTypeDialog.Visible = true
	}
	exportWavDialogStyle.ExtMain = ".wav"
	exportWavDialogStyle.ExtAlt = ""
	exportWavDialogStyle.Layout(gtx)
	fstyle = SaveFileDialog(t.Theme, t.SaveInstrumentDialog)
	fstyle.Title = "Save Instrument As"
	if t.SaveInstrumentDialog.Visible && t.Instrument().Name != "" {
		fstyle.Title = fmt.Sprintf("Save Instrument \"%v\" As", t.Instrument().Name)
	}
	for ok, file := t.SaveInstrumentDialog.FileSelected(); ok; ok, file = t.SaveInstrumentDialog.FileSelected() {
		t.saveInstrument(file)
		t.OpenInstrumentDialog.Directory.SetText(t.SaveInstrumentDialog.Directory.Text())
	}
	fstyle.Layout(gtx)
	fstyle = OpenFileDialog(t.Theme, t.OpenInstrumentDialog)
	fstyle.Title = "Open Instrument File"
	for ok, file := t.OpenInstrumentDialog.FileSelected(); ok; ok, file = t.OpenInstrumentDialog.FileSelected() {
		t.loadInstrument(file)
	}
	fstyle.Layout(gtx)
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
	t.window.Option(app.Title("Sointu Tracker"))
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
