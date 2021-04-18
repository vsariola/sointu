package gioui

import (
	"image"
	"math"
	"runtime"
	"time"

	"gioui.org/f32"
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

func (t *Tracker) layoutSongPanel(gtx C) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(t.layoutMenuBar),
		layout.Rigid(t.layoutSongOptions),
	)
}

func (t *Tracker) layoutMenu(title string, clickable *widget.Clickable, menu *Menu, width unit.Value, items ...MenuItem) layout.Widget {
	for clickable.Clicked() {
		menu.Visible = true
	}
	m := PopupMenu(t.Theme, menu)
	return func(gtx C) D {
		defer op.Save(gtx.Ops).Load()
		titleBtn := material.Button(t.Theme, clickable, title)
		titleBtn.Color = white
		titleBtn.Background = transparent
		titleBtn.CornerRadius = unit.Dp(0)
		dims := titleBtn.Layout(gtx)
		op.Offset(f32.Pt(0, float32(dims.Size.Y))).Add(gtx.Ops)
		gtx.Constraints.Max.X = gtx.Px(width)
		gtx.Constraints.Max.Y = gtx.Px(unit.Dp(1000))
		m.Layout(gtx, items...)
		return dims
	}
}

func (t *Tracker) layoutMenuBar(gtx C) D {
	gtx.Constraints.Max.Y = gtx.Px(unit.Dp(36))
	gtx.Constraints.Min.Y = gtx.Px(unit.Dp(36))

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
			t.ExportWav()
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
			clipboard.ReadOp{Tag: &t.Menus[1]}.Add(gtx.Ops)
		}
		clickedItem, hasClicked = t.Menus[1].Clicked()
	}

	shortcutKey := "Ctrl+"
	if runtime.GOOS == "darwin" {
		shortcutKey = "Cmd+"
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(t.layoutMenu("File", &t.MenuBar[0], &t.Menus[0], unit.Dp(200),
			MenuItem{IconBytes: icons.ContentClear, Text: "New Song", ShortcutText: shortcutKey + "N"},
			MenuItem{IconBytes: icons.FileFolder, Text: "Open Song", ShortcutText: shortcutKey + "O"},
			MenuItem{IconBytes: icons.ContentSave, Text: "Save Song", ShortcutText: shortcutKey + "S"},
			MenuItem{IconBytes: icons.ContentSave, Text: "Save Song As..."},
			MenuItem{IconBytes: icons.ImageAudiotrack, Text: "Export Wav..."},
			MenuItem{IconBytes: icons.ActionExitToApp, Text: "Quit"},
		)),
		layout.Rigid(t.layoutMenu("Edit", &t.MenuBar[1], &t.Menus[1], unit.Dp(160),
			MenuItem{IconBytes: icons.ContentUndo, Text: "Undo", ShortcutText: shortcutKey + "Z", Disabled: !t.CanUndo()},
			MenuItem{IconBytes: icons.ContentRedo, Text: "Redo", ShortcutText: shortcutKey + "Y", Disabled: !t.CanRedo()},
			MenuItem{IconBytes: icons.ContentContentCopy, Text: "Copy", ShortcutText: shortcutKey + "C"},
			MenuItem{IconBytes: icons.ContentContentPaste, Text: "Paste", ShortcutText: shortcutKey + "V"},
		)),
	)
}

func (t *Tracker) layoutSongOptions(gtx C) D {
	paint.FillShape(gtx.Ops, songSurfaceColor, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())

	in := layout.UniformInset(unit.Dp(1))

	panicBtnStyle := material.Button(t.Theme, t.PanicBtn, "Panic")
	if t.player.Enabled() {
		panicBtnStyle.Background = transparent
		panicBtnStyle.Color = t.Theme.Palette.Fg
	} else {
		panicBtnStyle.Background = t.Theme.Palette.Fg
		panicBtnStyle.Color = t.Theme.Palette.ContrastFg
	}

	for t.PanicBtn.Clicked() {
		t.player.Disable()
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("LEN:", white)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					t.SongLength.Value = t.Song().Score.Length
					numStyle := NumericUpDown(t.Theme, t.SongLength, 1, math.MaxInt32)
					gtx.Constraints.Min.Y = gtx.Px(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Px(unit.Dp(70))
					dims := in.Layout(gtx, numStyle.Layout)
					t.SetSongLength(t.SongLength.Value)
					return dims
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("BPM:", white)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					t.BPM.Value = t.Song().BPM
					numStyle := NumericUpDown(t.Theme, t.BPM, 1, 999)
					gtx.Constraints.Min.Y = gtx.Px(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Px(unit.Dp(70))
					dims := in.Layout(gtx, numStyle.Layout)
					t.SetBPM(t.BPM.Value)
					return dims
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("RPP:", white)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					t.RowsPerPattern.Value = t.Song().Score.RowsPerPattern
					numStyle := NumericUpDown(t.Theme, t.RowsPerPattern, 1, 255)
					gtx.Constraints.Min.Y = gtx.Px(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Px(unit.Dp(70))
					dims := in.Layout(gtx, numStyle.Layout)
					t.SetRowsPerPattern(t.RowsPerPattern.Value)
					return dims
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("RPB:", white)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					t.RowsPerBeat.Value = t.Song().RowsPerBeat
					numStyle := NumericUpDown(t.Theme, t.RowsPerBeat, 1, 32)
					gtx.Constraints.Min.Y = gtx.Px(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Px(unit.Dp(70))
					dims := in.Layout(gtx, numStyle.Layout)
					t.SetRowsPerBeat(t.RowsPerBeat.Value)
					return dims
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("STP:", white)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					numStyle := NumericUpDown(t.Theme, t.Step, 0, 8)
					numStyle.UnitsPerStep = unit.Dp(20)
					gtx.Constraints.Min.Y = gtx.Px(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Px(unit.Dp(70))
					dims := in.Layout(gtx, numStyle.Layout)
					return dims
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min = image.Pt(0, 0)
			return panicBtnStyle.Layout(gtx)
		}),
		layout.Rigid(VuMeter{Volume: t.lastVolume, Range: 100}.Layout),
	)
}
