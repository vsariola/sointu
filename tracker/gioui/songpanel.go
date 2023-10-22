package gioui

import (
	"image"
	"math"
	"time"

	"gioui.org/io/clipboard"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
	"gopkg.in/yaml.v3"
)

const shortcutKey = "Ctrl+"

var fileMenuItems []MenuItem = []MenuItem{
	{IconBytes: icons.ContentClear, Text: "New Song", ShortcutText: shortcutKey + "N"},
	{IconBytes: icons.FileFolder, Text: "Open Song", ShortcutText: shortcutKey + "O"},
	{IconBytes: icons.ContentSave, Text: "Save Song", ShortcutText: shortcutKey + "S"},
	{IconBytes: icons.ContentSave, Text: "Save Song As..."},
	{IconBytes: icons.ImageAudiotrack, Text: "Export Wav..."},
}

func init() {
	if CAN_QUIT {
		fileMenuItems = append(fileMenuItems, MenuItem{IconBytes: icons.ActionExitToApp, Text: "Quit"})
	}
}

func (t *Tracker) layoutSongPanel(gtx C) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(t.layoutMenuBar),
		layout.Rigid(t.layoutSongOptions),
	)
}

func (t *Tracker) layoutMenu(title string, clickable *widget.Clickable, menu *Menu, width unit.Dp, items ...MenuItem) layout.Widget {
	for clickable.Clicked() {
		menu.Visible = true
	}
	m := t.PopupMenu(menu)
	return func(gtx C) D {
		defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
		titleBtn := material.Button(t.Theme, clickable, title)
		titleBtn.Color = white
		titleBtn.Background = transparent
		titleBtn.CornerRadius = unit.Dp(0)
		dims := titleBtn.Layout(gtx)
		op.Offset(image.Pt(0, dims.Size.Y)).Add(gtx.Ops)
		gtx.Constraints.Max.X = gtx.Dp(width)
		gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(1000))
		m.Layout(gtx, items...)
		return dims
	}
}

func (t *Tracker) layoutMenuBar(gtx C) D {
	gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(36))
	gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(36))

	for clickedItem, hasClicked := t.Menus[0].Clicked(); hasClicked; {
		switch clickedItem {
		case 0:
			t.NewSong(false)
		case 1:
			t.OpenSongFile(false)
		case 2:
			t.SaveSongFile()
		case 3:
			t.SaveSongAsFile()
		case 4:
			t.WaveTypeDialog.Visible = true
		case 5:
			t.Quit(false)
		}
		clickedItem, hasClicked = t.Menus[0].Clicked()
	}

	for clickedItem, hasClicked := t.Menus[1].Clicked(); hasClicked; {
		switch clickedItem {
		case 0:
			t.Undo()
		case 1:
			t.Redo()
		case 2:
			if contents, err := yaml.Marshal(t.Song()); err == nil {
				clipboard.WriteOp{Text: string(contents)}.Add(gtx.Ops)
				t.Alert.Update("Song copied to clipboard", Notify, time.Second*3)
			}
		case 3:
			clipboard.ReadOp{Tag: t}.Add(gtx.Ops)
		case 4:
			t.RemoveUnusedData()
		}
		clickedItem, hasClicked = t.Menus[1].Clicked()
	}

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(t.layoutMenu("File", &t.MenuBar[0], &t.Menus[0], unit.Dp(200),
			fileMenuItems...,
		)),
		layout.Rigid(t.layoutMenu("Edit", &t.MenuBar[1], &t.Menus[1], unit.Dp(200),
			MenuItem{IconBytes: icons.ContentUndo, Text: "Undo", ShortcutText: shortcutKey + "Z", Disabled: !t.CanUndo()},
			MenuItem{IconBytes: icons.ContentRedo, Text: "Redo", ShortcutText: shortcutKey + "Y", Disabled: !t.CanRedo()},
			MenuItem{IconBytes: icons.ContentContentCopy, Text: "Copy", ShortcutText: shortcutKey + "C"},
			MenuItem{IconBytes: icons.ContentContentPaste, Text: "Paste", ShortcutText: shortcutKey + "V"},
			MenuItem{IconBytes: icons.ImageCrop, Text: "Remove unused data"},
		)),
	)
}

func (t *Tracker) layoutSongOptions(gtx C) D {
	paint.FillShape(gtx.Ops, songSurfaceColor, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())

	in := layout.UniformInset(unit.Dp(1))

	var panicBtnStyle material.ButtonStyle
	if !t.Panic() {
		panicBtnStyle = LowEmphasisButton(t.Theme, t.PanicBtn, "Panic")
	} else {
		panicBtnStyle = HighEmphasisButton(t.Theme, t.PanicBtn, "Panic")
	}

	for t.PanicBtn.Clicked() {
		t.SetPanic(!t.Panic())
	}

	var recordBtnStyle material.ButtonStyle
	if !t.Recording() {
		recordBtnStyle = LowEmphasisButton(t.Theme, t.RecordBtn, "Record")
	} else {
		recordBtnStyle = HighEmphasisButton(t.Theme, t.RecordBtn, "Record")
	}

	for t.RecordBtn.Clicked() {
		t.SetRecording(!t.Recording())
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("LEN:", white, t.TextShaper)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					t.SongLength.Value = t.Song().Score.Length
					numStyle := NumericUpDown(t.Theme, t.SongLength, 1, math.MaxInt32, "Song length")
					gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
					dims := in.Layout(gtx, numStyle.Layout)
					t.SetSongLength(t.SongLength.Value)
					return dims
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("BPM:", white, t.TextShaper)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					t.BPM.Value = t.Song().BPM
					numStyle := NumericUpDown(t.Theme, t.BPM, 1, 999, "Beats per minute")
					gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
					dims := in.Layout(gtx, numStyle.Layout)
					t.SetBPM(t.BPM.Value)
					return dims
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("RPP:", white, t.TextShaper)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					t.RowsPerPattern.Value = t.Song().Score.RowsPerPattern
					numStyle := NumericUpDown(t.Theme, t.RowsPerPattern, 1, 255, "Rows per pattern")
					gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
					dims := in.Layout(gtx, numStyle.Layout)
					t.SetRowsPerPattern(t.RowsPerPattern.Value)
					return dims
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("RPB:", white, t.TextShaper)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					t.RowsPerBeat.Value = t.Song().RowsPerBeat
					numStyle := NumericUpDown(t.Theme, t.RowsPerBeat, 1, 32, "Rows per beat")
					gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
					dims := in.Layout(gtx, numStyle.Layout)
					t.SetRowsPerBeat(t.RowsPerBeat.Value)
					return dims
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("STP:", white, t.TextShaper)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					numStyle := NumericUpDown(t.Theme, t.Step, 0, 8, "Cursor step")
					numStyle.UnitsPerStep = unit.Dp(20)
					dims := in.Layout(gtx, numStyle.Layout)
					return dims
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min = image.Pt(0, 0)
			return panicBtnStyle.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min = image.Pt(0, 0)
			return recordBtnStyle.Layout(gtx)
		}),
		layout.Rigid(VuMeter{AverageVolume: t.lastAvgVolume, PeakVolume: t.lastPeakVolume, Range: 100}.Layout),
	)
}
