package tracker

import (
	"fmt"

	"gioui.org/layout"
)

func (t *Tracker) Layout(gtx layout.Context) {
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(t.layoutControls),
		layout.Rigid(Lowered(t.layoutPatterns(
			t.song.Tracks,
			t.ActiveTrack,
			t.DisplayPattern,
			t.CursorColumn,
			t.PlayPattern,
		))),
		layout.Flexed(1, Lowered(t.layoutTracker)),
	)
}

func (t *Tracker) layoutTracker(gtx layout.Context) layout.Dimensions {
	flexTracks := make([]layout.FlexChild, len(t.song.Tracks))
	t.playRowPatMutex.RLock()
	defer t.playRowPatMutex.RUnlock()
	playRow := int(t.PlayRow)
	if t.DisplayPattern != t.PlayPattern {
		playRow = -1
	}
	for i, trk := range t.song.Tracks {
		flexTracks[i] = layout.Rigid(Lowered(t.layoutTrack(
			trk.Patterns[trk.Sequence[t.DisplayPattern]],
			t.ActiveTrack == i,
			t.CursorRow,
			t.CursorColumn,
			playRow,
		)))
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		flexTracks...,
	)
}

func (t *Tracker) layoutControls(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.Y = 200
	gtx.Constraints.Max.Y = 200
	return layout.Stack{Alignment: layout.NW}.Layout(gtx,
		layout.Expanded(t.QuitButton.Layout),
		layout.Stacked(Raised(Label(fmt.Sprintf("Current octave: %v", t.CurrentOctave), white))),
	)
}
