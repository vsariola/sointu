package tracker

import (
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

func (t *Tracker) layoutSongPanel(gtx C) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(t.layoutSongButtons),
		layout.Flexed(1, t.layoutSongOptions),
	)
}

func (t *Tracker) layoutSongButtons(gtx C) D {
	gtx.Constraints.Max.Y = gtx.Px(unit.Dp(36))
	gtx.Constraints.Min.Y = gtx.Px(unit.Dp(36))

	//paint.FillShape(gtx.Ops, primaryColorDark, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())

	for t.NewSongFileBtn.Clicked() {
		t.LoadSong(defaultSong.Copy())
		t.FileMenuVisible = false
	}

	for t.LoadSongFileBtn.Clicked() {
		t.LoadSongFile()
		t.FileMenuVisible = false
	}

	for t.SaveSongFileBtn.Clicked() {
		t.SaveSongFile()
	}

	newBtnStyle := material.IconButton(t.Theme, t.NewSongFileBtn, widgetForIcon(icons.ContentClear))
	newBtnStyle.Background = transparent
	newBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
	newBtnStyle.Color = primaryColor

	loadBtnStyle := material.IconButton(t.Theme, t.LoadSongFileBtn, widgetForIcon(icons.FileFolder))
	loadBtnStyle.Background = transparent
	loadBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
	loadBtnStyle.Color = primaryColor

	menuContents := func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(newBtnStyle.Layout),
			layout.Rigid(loadBtnStyle.Layout),
		)
	}

	fileMenu := Popup(&t.FileMenuVisible)
	fileMenu.NE = unit.Dp(0)
	fileMenu.ShadowN = unit.Dp(0)
	fileMenu.NW = unit.Dp(0)

	saveBtnStyle := material.IconButton(t.Theme, t.SaveSongFileBtn, widgetForIcon(icons.ContentSave))
	saveBtnStyle.Background = transparent
	saveBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
	saveBtnStyle.Color = primaryColor

	fileMenuBtnStyle := material.IconButton(t.Theme, t.FileMenuBtn, widgetForIcon(icons.NavigationMoreVert))
	fileMenuBtnStyle.Background = transparent
	fileMenuBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
	fileMenuBtnStyle.Color = primaryColor

	for t.FileMenuBtn.Clicked() {
		t.FileMenuVisible = true
	}

	popupWidget := func(gtx C) D {
		defer op.Save(gtx.Ops).Load()
		dims := fileMenuBtnStyle.Layout(gtx)
		op.Offset(f32.Pt(0, float32(dims.Size.Y))).Add(gtx.Ops)
		gtx.Constraints.Max.X = 160
		gtx.Constraints.Max.Y = 300
		fileMenu.Layout(gtx, menuContents)
		return dims
	}

	layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(saveBtnStyle.Layout),
		layout.Rigid(popupWidget),
	)

	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func (t *Tracker) layoutSongOptions(gtx C) D {
	paint.FillShape(gtx.Ops, songSurfaceColor, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())

	in := layout.UniformInset(unit.Dp(1))

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("LEN:", white)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					t.SongLength.Value = t.song.SequenceLength()
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
					t.BPM.Value = t.song.BPM
					numStyle := NumericUpDown(t.Theme, t.BPM, 1, 999)
					gtx.Constraints.Min.Y = gtx.Px(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Px(unit.Dp(70))
					dims := in.Layout(gtx, numStyle.Layout)
					t.SetBPM(t.BPM.Value)
					return dims
					//return in.Layout(gtx, enableButton(smallButton(material.IconButton(t.Theme, t.BPMUpBtn, upIcon)), t.song.BPM < 999).Layout)
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("RPP:", white)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					t.RowsPerPattern.Value = t.song.RowsPerPattern
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
					t.RowsPerBeat.Value = t.song.RowsPerBeat
					numStyle := NumericUpDown(t.Theme, t.RowsPerBeat, 1, 32)
					gtx.Constraints.Min.Y = gtx.Px(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Px(unit.Dp(70))
					dims := in.Layout(gtx, numStyle.Layout)
					t.SetRowsPerBeat(t.RowsPerBeat.Value)
					return dims
				}),
			)
		}),
	)
}
