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
}

func smallButton(icStyle material.IconButtonStyle) material.IconButtonStyle {
	icStyle.Size = unit.Dp(14)
	icStyle.Inset = layout.UniformInset(unit.Dp(1))
	return icStyle
}

func (t *Tracker) Layout(gtx layout.Context) {
	paint.FillShape(gtx.Ops, backgroundColor, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())
	layout.UniformInset(unit.Dp(2)).Layout(gtx, func(gtx2 layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx2,
			layout.Rigid(t.layoutControls),
			layout.Rigid(t.line(true, separatorLineColor)),
			layout.Flexed(1, t.layoutTracker))
	})
}

func (t *Tracker) layoutTracker(gtx layout.Context) layout.Dimensions {
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
		flexTracks[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return leftInset.Layout(gtx, t.layoutTrack(
				trk2.Patterns,
				trk2.Sequence,
				t.ActiveTrack == i2,
				t.CursorRow,
				t.DisplayPattern,
				t.CursorColumn,
				t.PlayRow,
				playPat,
			))
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
			iconBtn := material.IconButton(t.Theme, t.NewTrackBtn, addIcon)
			if t.song.TotalTrackVoices() >= t.song.Patch.TotalVoices() {
				iconBtn.Background = disabledContainerColor
				iconBtn.Color = disabledTextColor
			}
			return in2.Layout(gtx, iconBtn.Layout)
		})
		octLabel := layout.Rigid(Label("OCT:", white))
		in := layout.UniformInset(unit.Dp(1))
		octRow := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return in.Layout(gtx, smallButton(material.IconButton(t.Theme, t.OctaveUpBtn, upIcon)).Layout)
				}),
				layout.Rigid(Label(fmt.Sprintf("%v", t.CurrentOctave), white)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return in.Layout(gtx, smallButton(material.IconButton(t.Theme, t.OctaveDownBtn, downIcon)).Layout)
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
	gtx.Constraints.Min.Y = 200
	gtx.Constraints.Max.Y = 200

	playPat := t.PlayPattern
	if !t.Playing {
		playPat = -1
	}
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

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(t.layoutPatterns(
			t.song.Tracks,
			t.ActiveTrack,
			t.DisplayPattern,
			t.CursorColumn,
			playPat,
		)),
		layout.Rigid(Label(fmt.Sprintf("BPM: %3v", t.song.BPM), white)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return in.Layout(gtx, smallButton(material.IconButton(t.Theme, t.BPMUpBtn, upIcon)).Layout)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return in.Layout(gtx, smallButton(material.IconButton(t.Theme, t.BPMDownBtn, downIcon)).Layout)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			iconBtn := material.IconButton(t.Theme, t.NewInstrumentBtn, addIcon)
			if t.song.Patch.TotalVoices() >= 32 {
				iconBtn.Background = disabledContainerColor
				iconBtn.Color = disabledTextColor
			}
			return in.Layout(gtx, iconBtn.Layout)
		}),
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
