package tracker

import (
	"fmt"
	"image"
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
}

func (t *Tracker) Layout(gtx layout.Context) {
	paint.FillShape(gtx.Ops, black, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())
	layout.UniformInset(unit.Dp(2)).Layout(gtx, func(gtx2 layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx2,
			layout.Rigid(t.layoutControls),
			layout.Rigid(t.darkLine(true)),
			layout.Flexed(1, Raised(t.layoutTracker)))
	})
}

func (t *Tracker) layoutTracker(gtx layout.Context) layout.Dimensions {
	flexTracks := make([]layout.FlexChild, len(t.song.Tracks)+1)
	t.playRowPatMutex.RLock()
	defer t.playRowPatMutex.RUnlock()

	playPat := t.PlayPattern
	if !t.Playing {
		playPat = -1
	}

	flexTracks[0] = layout.Rigid(Lowered(t.layoutRowMarkers(
		len(t.song.Tracks[0].Patterns[0]),
		len(t.song.Tracks[0].Sequence),
		t.CursorRow,
		t.DisplayPattern,
		t.CursorColumn,
		t.PlayRow,
		playPat,
	)))
	for i, trk := range t.song.Tracks {
		flexTracks[i+1] = layout.Rigid(Lowered(t.layoutTrack(
			trk.Patterns,
			trk.Sequence,
			t.ActiveTrack == i,
			t.CursorRow,
			t.DisplayPattern,
			t.CursorColumn,
			t.PlayRow,
			playPat,
		)))
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		flexTracks...,
	)
}

func (t *Tracker) layoutControls(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.Y = 200
	gtx.Constraints.Max.Y = 200

	playPat := t.PlayPattern
	if !t.Playing {
		playPat = -1
	}
	in := layout.UniformInset(unit.Dp(8))

	for t.OctaveUpBtn.Clicked() {
		t.ChangeOctave(1)
	}
	for t.OctaveDownBtn.Clicked() {
		t.ChangeOctave(-1)
	}
	for t.BPMUpBtn.Clicked() {
		t.ChangeBPM(1)
	}
	for t.BPMDownBtn.Clicked() {
		t.ChangeBPM(-1)
	}

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(Raised(t.layoutPatterns(
			t.song.Tracks,
			t.ActiveTrack,
			t.DisplayPattern,
			t.CursorColumn,
			playPat,
		))),
		layout.Rigid(t.darkLine(false)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return in.Layout(gtx, material.IconButton(t.Theme, t.OctaveUpBtn, upIcon).Layout)
		}),
		layout.Rigid(t.darkLine(false)),
		layout.Rigid(Raised(Label(fmt.Sprintf("OCT: %v", t.CurrentOctave), white))),
		layout.Rigid(t.darkLine(false)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return in.Layout(gtx, material.IconButton(t.Theme, t.OctaveDownBtn, downIcon).Layout)
		}),
		layout.Rigid(t.darkLine(false)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return in.Layout(gtx, material.IconButton(t.Theme, t.BPMUpBtn, upIcon).Layout)
		}),
		layout.Rigid(t.darkLine(false)),
		layout.Rigid(Raised(Label(fmt.Sprintf("BPM: %3v", t.song.BPM), white))),
		layout.Rigid(t.darkLine(false)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return in.Layout(gtx, material.IconButton(t.Theme, t.BPMDownBtn, downIcon).Layout)
		}),
	)
}

func (t *Tracker) darkLine(horizontal bool) layout.Widget {
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
		paint.FillShape(gtx.Ops, black, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}.Op())
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}
