package tracker

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"golang.org/x/exp/shiny/materialdesign/icons"
)

var upIcon *widget.Icon
var downIcon *widget.Icon
var addIcon *widget.Icon
var loadIcon *widget.Icon
var saveIcon *widget.Icon

func init() {
	var err error
	upIcon, err = widget.NewIcon(icons.NavigationArrowUpward)
	if err != nil {
		log.Fatal(err)
	}
	downIcon, err = widget.NewIcon(icons.NavigationArrowDownward)
	if err != nil {
		log.Fatal(err)
	}
	addIcon, err = widget.NewIcon(icons.ContentAdd)
	if err != nil {
		log.Fatal(err)
	}
	loadIcon, err = widget.NewIcon(icons.FileFolder)
	if err != nil {
		log.Fatal(err)
	}
	saveIcon, err = widget.NewIcon(icons.ContentSave)
	if err != nil {
		log.Fatal(err)
	}
}

func smallButton(icStyle material.IconButtonStyle) material.IconButtonStyle {
	icStyle.Size = unit.Dp(14)
	icStyle.Inset = layout.UniformInset(unit.Dp(1))
	return icStyle
}

func enableButton(icStyle material.IconButtonStyle, enabled bool) material.IconButtonStyle {
	if !enabled {
		icStyle.Background = disabledContainerColor
		icStyle.Color = disabledTextColor
	}
	return icStyle
}

func trackButton(t *material.Theme, w *widget.Clickable, text string, enabled bool) material.ButtonStyle {
	ret := material.Button(t, w, text)
	if !enabled {
		ret.Background = disabledContainerColor
		ret.Color = disabledTextColor
	}
	return ret
}

func (t *Tracker) Layout(gtx layout.Context) {
	paint.FillShape(gtx.Ops, backgroundColor, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())
	layout.UniformInset(unit.Dp(2)).Layout(gtx, func(gtx2 layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx2,
			layout.Rigid(t.layoutControls),
			layout.Flexed(1, t.layoutTracksAndPatterns))
	})
	t.updateInstrumentScroll()
}

func (t *Tracker) layoutTracksAndPatterns(gtx layout.Context) layout.Dimensions {
	playPat := t.PlayPattern
	if !t.Playing {
		playPat = -1
	}
	return t.BottomSplit.Layout(gtx,
		t.layoutPatterns(
			t.song.Tracks,
			t.ActiveTrack,
			t.DisplayPattern,
			t.CursorColumn,
			playPat,
		),
		t.layoutTracks,
	)
}

func (t *Tracker) layoutTracks(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, trackerSurfaceColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}.Op())

	flexTracks := make([]layout.FlexChild, len(t.song.Tracks))
	t.playRowPatMutex.RLock()
	defer t.playRowPatMutex.RUnlock()

	playPat := t.PlayPattern
	if !t.Playing {
		playPat = -1
	}

	rowMarkers := layout.Rigid(t.layoutRowMarkers(
		len(t.song.Tracks[0].Patterns[0]),
		len(t.song.Tracks[0].Sequence),
		t.CursorRow,
		t.DisplayPattern,
		t.CursorColumn,
		t.PlayRow,
		playPat,
	))
	leftInset := layout.Inset{Left: unit.Dp(4)}
	for i, trk := range t.song.Tracks {
		i2 := i     // avoids i being updated in the closure
		trk2 := trk // avoids trk being updated in the closure
		if len(t.TrackHexCheckBoxes) <= i {
			t.TrackHexCheckBoxes = append(t.TrackHexCheckBoxes, new(widget.Bool))
		}
		if len(t.TrackShowHex) <= i {
			t.TrackShowHex = append(t.TrackShowHex, false)
		}
		flexTracks[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			t.TrackHexCheckBoxes[i2].Value = t.TrackShowHex[i2]
			cbStyle := material.CheckBox(t.Theme, t.TrackHexCheckBoxes[i2], "hex")
			cbStyle.Color = white
			ret := layout.Stack{}.Layout(gtx,
				layout.Stacked(func(gtx layout.Context) D {
					return leftInset.Layout(gtx, t.layoutTrack(
						trk2.Patterns,
						trk2.Sequence,
						t.ActiveTrack == i2,
						t.TrackShowHex[i2],
						t.CursorRow,
						t.DisplayPattern,
						t.CursorColumn,
						t.PlayRow,
						playPat,
					))
				}),
				layout.Stacked(cbStyle.Layout),
			)
			t.TrackShowHex[i2] = t.TrackHexCheckBoxes[i2].Value
			return ret
		})
	}
	in2 := layout.UniformInset(unit.Dp(8))
	for t.OctaveUpBtn.Clicked() {
		t.ChangeOctave(1)
	}
	for t.OctaveDownBtn.Clicked() {
		t.ChangeOctave(-1)
	}
	menu := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		newTrack := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, trackMenuSurfaceColor, clip.Rect{
				Max: gtx.Constraints.Max,
			}.Op())
			return in2.Layout(gtx, enableButton(material.IconButton(t.Theme, t.NewTrackBtn, addIcon), t.song.TotalTrackVoices() < t.song.Patch.TotalVoices()).Layout)
		})
		octLabel := layout.Rigid(Label("OCT:", white))
		in := layout.UniformInset(unit.Dp(1))
		octRow := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return in.Layout(gtx, enableButton(smallButton(material.IconButton(t.Theme, t.OctaveUpBtn, upIcon)), t.CurrentOctave < 9).Layout)
				}),
				layout.Rigid(Label(fmt.Sprintf("%v", t.CurrentOctave), white)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return in.Layout(gtx, enableButton(smallButton(material.IconButton(t.Theme, t.OctaveDownBtn, downIcon)), t.CurrentOctave > 0).Layout)
				}),
			)
		})
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, newTrack, octLabel, octRow)
	})
	go func() {
		for t.NewTrackBtn.Clicked() {
			t.AddTrack()
		}
	}()
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		rowMarkers,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			defer op.Push(gtx.Ops).Pop()
			clip.Rect{Max: gtx.Constraints.Max}.Add(gtx.Ops)
			dims := layout.Flex{Axis: layout.Horizontal}.Layout(gtx, flexTracks...)
			if dims.Size.X > gtx.Constraints.Max.X {
				dims.Size.X = gtx.Constraints.Max.X
			}
			return dims
		}),
		menu,
	)
}

func (t *Tracker) layoutControls(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.Y = 250
	gtx.Constraints.Max.Y = 250

	in := layout.UniformInset(unit.Dp(1))

	go func() {
		for t.BPMUpBtn.Clicked() {
			t.ChangeBPM(1)
		}
		for t.BPMDownBtn.Clicked() {
			t.ChangeBPM(-1)
		}
		for t.NewInstrumentBtn.Clicked() {
			t.AddInstrument()
		}
	}()

	for t.SongLengthUpBtn.Clicked() {
		t.IncreaseSongLength()
	}

	for t.SongLengthDownBtn.Clicked() {
		t.DecreaseSongLength()
	}

	return t.TopSplit.Layout(gtx,
		t.layoutSongPanel,
		func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, t.layoutInstruments()),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					iconBtn := enableButton(material.IconButton(t.Theme, t.NewInstrumentBtn, addIcon), t.song.Patch.TotalVoices() < 32)
					return in.Layout(gtx, iconBtn.Layout)
				}),
			)
		},
	)

}

func (t *Tracker) line(horizontal bool, color color.RGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if horizontal {
			gtx.Constraints.Min.Y = 1
			gtx.Constraints.Max.Y = 1
		} else {
			gtx.Constraints.Min.X = 1
			gtx.Constraints.Max.X = 1
		}
		defer op.Push(gtx.Ops).Pop()
		clip.Rect{Max: gtx.Constraints.Max}.Add(gtx.Ops)
		paint.FillShape(gtx.Ops, color, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}.Op())
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}
