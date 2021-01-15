package tracker

import (
	"image"
	"math"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
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
		t.LoadSong(defaultSong)
	}

	for t.LoadSongFileBtn.Clicked() {
		t.LoadSongFile()
	}

	for t.SaveSongFileBtn.Clicked() {
		t.SaveSongFile()
	}

	newBtnStyle := material.IconButton(t.Theme, t.NewSongFileBtn, clearIcon)
	newBtnStyle.Background = transparent
	newBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
	newBtnStyle.Color = primaryColor

	loadBtnStyle := material.IconButton(t.Theme, t.LoadSongFileBtn, loadIcon)
	loadBtnStyle.Background = transparent
	loadBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
	loadBtnStyle.Color = primaryColor

	saveBtnStyle := material.IconButton(t.Theme, t.SaveSongFileBtn, saveIcon)
	saveBtnStyle.Background = transparent
	saveBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
	saveBtnStyle.Color = primaryColor

	layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(newBtnStyle.Layout),
		layout.Rigid(loadBtnStyle.Layout),
		layout.Rigid(saveBtnStyle.Layout),
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
	)
}
