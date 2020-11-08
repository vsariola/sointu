package tracker

import (
	"gioui.org/layout"
)

func (t *Tracker) Layout(gtx layout.Context) {
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(t.layoutControls),
		layout.Flexed(1, Lowered(t.layoutTracker)),
	)
}

func (t *Tracker) layoutTracker(gtx layout.Context) layout.Dimensions {
	flexTracks := make([]layout.FlexChild, len(t.song.Tracks))
	for i, trk := range t.song.Tracks {
		flexTracks[i] = layout.Rigid(Lowered(t.layoutTrack(
			t.song.Patterns[trk.Sequence[t.DisplayPattern]],
			t.ActiveTrack == i,
			t.CursorRow,
			t.CursorColumn,
		)))
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		flexTracks...,
	)
}

func (t *Tracker) layoutControls(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.Y = 400
	gtx.Constraints.Max.Y = 400
	return layout.Stack{Alignment: layout.NW}.Layout(gtx,
		layout.Expanded(t.QuitButton.Layout),
		layout.Stacked(Raised(Label("Hello", white))),
	)
}
