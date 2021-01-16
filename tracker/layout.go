package tracker

import (
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
var clearIcon *widget.Icon

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
	clearIcon, err = widget.NewIcon(icons.ContentClear)
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
	t.VerticalSplit.Layout(gtx,
		t.layoutControls,
		t.layoutTracksAndPatterns)
	t.updateInstrumentScroll()
}

func (t *Tracker) layoutTracksAndPatterns(gtx layout.Context) layout.Dimensions {
	return t.BottomHorizontalSplit.Layout(gtx,
		t.layoutPatterns,
		t.layoutTracks,
	)
}

func (t *Tracker) layoutTracks(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, trackerSurfaceColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}.Op())

	flexTracks := make([]layout.FlexChild, len(t.song.Tracks))
	t.playRowPatMutex.RLock()
	defer t.playRowPatMutex.RUnlock()

	playPat := t.PlayPosition.Pattern
	if !t.Playing {
		playPat = -1
	}

	rowMarkers := layout.Rigid(t.layoutRowMarkers(
		len(t.song.Tracks[0].Patterns[0]),
		len(t.song.Tracks[0].Sequence),
		t.Cursor.Row,
		t.Cursor.Pattern,
		t.CursorColumn,
		t.PlayPosition.Row,
		playPat,
	))
	leftInset := layout.Inset{Left: unit.Dp(4)}
	for i := range t.song.Tracks {
		i2 := i // avoids i being updated in the closure
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
					return leftInset.Layout(gtx, t.layoutTrack(i2))
				}),
				layout.Stacked(cbStyle.Layout),
			)
			t.TrackShowHex[i2] = t.TrackHexCheckBoxes[i2].Value
			return ret
		})
	}
	menuBg := func(gtx C) D {
		paint.FillShape(gtx.Ops, trackMenuSurfaceColor, clip.Rect{
			Max: gtx.Constraints.Min,
		}.Op())
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}
	menu := func(gtx C) D {
		newTrackBtnStyle := material.IconButton(t.Theme, t.NewTrackBtn, addIcon)
		newTrackBtnStyle.Background = transparent
		newTrackBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
		if t.song.TotalTrackVoices() < t.song.Patch.TotalVoices() {
			newTrackBtnStyle.Color = primaryColor
		} else {
			newTrackBtnStyle.Color = disabledTextColor
		}
		in := layout.UniformInset(unit.Dp(1))
		octave := func(gtx C) D {
			numStyle := NumericUpDown(t.Theme, t.Octave, 0, 9)
			gtx.Constraints.Min.Y = gtx.Px(unit.Dp(20))
			gtx.Constraints.Min.X = gtx.Px(unit.Dp(70))
			return in.Layout(gtx, numStyle.Layout)
		}
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(Label("OCT:", white)),
			layout.Rigid(octave),
			layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
			layout.Rigid(newTrackBtnStyle.Layout))
	}
	go func() {
		for t.NewTrackBtn.Clicked() {
			t.AddTrack()
		}
	}()
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Stack{Alignment: layout.Center}.Layout(gtx,
				layout.Expanded(menuBg),
				layout.Stacked(menu),
			)
		}),
		layout.Flexed(1, func(gtx C) D {
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
				}))
		}),
	)
}

func (t *Tracker) layoutControls(gtx layout.Context) layout.Dimensions {
	go func() {
		/*for t.BPMUpBtn.Clicked() {
			t.ChangeBPM(1)
		}
		for t.BPMDownBtn.Clicked() {
			t.ChangeBPM(-1)
		}*/
		for t.NewInstrumentBtn.Clicked() {
			t.AddInstrument()
		}
	}()

	return t.TopHorizontalSplit.Layout(gtx,
		t.layoutSongPanel,
		t.layoutInstruments(),
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
