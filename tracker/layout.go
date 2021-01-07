package tracker

import (
	"fmt"
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

func (t *Tracker) Layout(gtx layout.Context) {
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(t.layoutControls),
		layout.Rigid(t.darkLine(true)),
		layout.Flexed(1, Raised(t.layoutTracker)),
	)
}

func (t *Tracker) layoutTracker(gtx layout.Context) layout.Dimensions {
	flexTracks := make([]layout.FlexChild, len(t.song.Tracks))
	t.playRowPatMutex.RLock()
	defer t.playRowPatMutex.RUnlock()

	playPat := t.PlayPattern
	if !t.Playing {
		playPat = -1
	}

	for i, trk := range t.song.Tracks {
		flexTracks[i] = layout.Rigid(Lowered(t.layoutTrack(
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

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(Raised(t.layoutPatterns(
			t.song.Tracks,
			t.ActiveTrack,
			t.DisplayPattern,
			t.CursorColumn,
			playPat,
		))),
		layout.Rigid(t.darkLine(false)),
		layout.Flexed(1, Raised(Label(fmt.Sprintf("Current octave: %v", t.CurrentOctave), white))),
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
