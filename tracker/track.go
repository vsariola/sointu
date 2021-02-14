package tracker

import (
	"fmt"
	"image"
	"strings"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

const trackRowHeight = 16
const trackColWidth = 54
const patmarkWidth = 16

var trackPointerTag bool
var trackJumpPointerTag bool

func (t *Tracker) layoutTracker(gtx layout.Context) layout.Dimensions {
	t.playRowPatMutex.RLock()
	defer t.playRowPatMutex.RUnlock()

	playPat := t.PlayPosition.Pattern
	if !t.Playing {
		playPat = -1
	}

	rowMarkers := layout.Rigid(t.layoutRowMarkers(
		t.song.RowsPerPattern,
		len(t.song.Tracks[0].Sequence),
		t.Cursor.Row,
		t.Cursor.Pattern,
		t.CursorColumn,
		t.PlayPosition.Row,
		playPat,
	))

	for t.NewTrackBtn.Clicked() {
		t.AddTrack()
	}

	for len(t.TrackHexCheckBoxes) < len(t.song.Tracks) {
		t.TrackHexCheckBoxes = append(t.TrackHexCheckBoxes, new(widget.Bool))
	}

	for len(t.TrackShowHex) < len(t.song.Tracks) {
		t.TrackShowHex = append(t.TrackShowHex, false)
	}

	//t.TrackHexCheckBoxes[i2].Value = t.TrackShowHex[i2]
	//cbStyle := material.CheckBox(t.Theme, t.TrackHexCheckBoxes[i2], "hex")
	//cbStyle.Color = white
	//cbStyle.IconColor = t.Theme.Fg

	for t.AddSemitoneBtn.Clicked() {
		t.AdjustSelectionPitch(1)
	}

	for t.SubtractSemitoneBtn.Clicked() {
		t.AdjustSelectionPitch(-1)
	}

	for t.AddOctaveBtn.Clicked() {
		t.AdjustSelectionPitch(12)
	}

	for t.SubtractOctaveBtn.Clicked() {
		t.AdjustSelectionPitch(-12)
	}

	menu := func(gtx C) D {
		addSemitoneBtnStyle := material.Button(t.Theme, t.AddSemitoneBtn, "+1")
		addSemitoneBtnStyle.Color = primaryColor
		addSemitoneBtnStyle.Background = transparent
		addSemitoneBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
		subtractSemitoneBtnStyle := material.Button(t.Theme, t.SubtractSemitoneBtn, "-1")
		subtractSemitoneBtnStyle.Color = primaryColor
		subtractSemitoneBtnStyle.Background = transparent
		subtractSemitoneBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
		addOctaveBtnStyle := material.Button(t.Theme, t.AddOctaveBtn, "+12")
		addOctaveBtnStyle.Color = primaryColor
		addOctaveBtnStyle.Background = transparent
		addOctaveBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
		subtractOctaveBtnStyle := material.Button(t.Theme, t.SubtractOctaveBtn, "-12")
		subtractOctaveBtnStyle.Color = primaryColor
		subtractOctaveBtnStyle.Background = transparent
		subtractOctaveBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
		newTrackBtnStyle := material.IconButton(t.Theme, t.NewTrackBtn, widgetForIcon(icons.ContentAdd))
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
		n := t.song.Tracks[t.Cursor.Track].NumVoices
		maxRemain := t.song.Patch.TotalVoices() - t.song.TotalTrackVoices() + n
		if maxRemain < 1 {
			maxRemain = 1
		}
		t.TrackVoices.Value = n
		voiceUpDown := func(gtx C) D {
			numStyle := NumericUpDown(t.Theme, t.TrackVoices, 1, maxRemain)
			gtx.Constraints.Min.Y = gtx.Px(unit.Dp(20))
			gtx.Constraints.Min.X = gtx.Px(unit.Dp(70))
			return in.Layout(gtx, numStyle.Layout)
		}
		dims := layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(Label("OCT:", white)),
			layout.Rigid(octave),
			layout.Rigid(Label(" PITCH:", white)),
			layout.Rigid(addSemitoneBtnStyle.Layout),
			layout.Rigid(subtractSemitoneBtnStyle.Layout),
			layout.Rigid(addOctaveBtnStyle.Layout),
			layout.Rigid(subtractOctaveBtnStyle.Layout),
			layout.Rigid(Label("Voices:", white)),
			layout.Rigid(voiceUpDown),
			layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
			layout.Rigid(newTrackBtnStyle.Layout))
		t.SetTrackVoices(t.TrackVoices.Value)
		return dims
	}

	for _, ev := range gtx.Events(&trackPointerTag) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		if e.Type == pointer.Press {
			t.EditMode = EditTracks
		}
	}
	rect := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
	pointer.Rect(rect).Add(gtx.Ops)
	pointer.InputOp{Tag: &trackPointerTag,
		Types: pointer.Press,
	}.Add(gtx.Ops)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return Surface{Gray: 37, Focus: t.EditMode == 1, FitSize: true}.Layout(gtx, menu)
		}),
		layout.Flexed(1, func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				rowMarkers,
				layout.Flexed(1, func(gtx C) D {
					return layout.Stack{Alignment: layout.NW}.Layout(gtx,
						layout.Stacked(t.layoutTracks),
						layout.Stacked(t.layoutTrackTitles),
					)
				}))

		}),
	)
}

func (t *Tracker) layoutTrackTitles(gtx C) D {
	defer op.Save(gtx.Ops).Load()
	hexFlexChildren := make([]layout.FlexChild, len(t.song.Tracks))
	for trkIndex := range t.song.Tracks {
		trkIndex2 := trkIndex
		hexFlexChildren[trkIndex] = layout.Rigid(func(gtx C) D {
			t.TrackHexCheckBoxes[trkIndex2].Value = t.TrackShowHex[trkIndex2]
			cbStyle := material.CheckBox(t.Theme, t.TrackHexCheckBoxes[trkIndex2], "hex")
			dims := cbStyle.Layout(gtx)
			t.TrackShowHex[trkIndex2] = t.TrackHexCheckBoxes[trkIndex2].Value
			return layout.Dimensions{Size: image.Pt(trackColWidth, dims.Size.Y)}
		})
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, hexFlexChildren...)
}

func (t *Tracker) layoutTracks(gtx C) D {
	defer op.Save(gtx.Ops).Load()
	clip.Rect{Max: gtx.Constraints.Max}.Add(gtx.Ops)
	cursorSongRow := t.Cursor.Pattern*t.song.RowsPerPattern + t.Cursor.Row
	for _, ev := range gtx.Events(&trackJumpPointerTag) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		if e.Type == pointer.Press {
			t.EditMode = EditTracks
			t.Cursor.Track = int(e.Position.X) / trackColWidth
			t.Cursor.Pattern = 0
			t.Cursor.Row = int(e.Position.Y) / trackRowHeight
			t.Cursor.Clamp(t.song)
			t.SelectionCorner = t.Cursor
			cursorSongRow = t.Cursor.Pattern*t.song.RowsPerPattern + t.Cursor.Row
		}
	}
	op.Offset(f32.Pt(0, float32(gtx.Constraints.Max.Y-trackRowHeight)/2)).Add(gtx.Ops)
	op.Offset(f32.Pt(0, (-1*trackRowHeight)*float32(cursorSongRow))).Add(gtx.Ops)
	rect := image.Rect(0, 0, trackColWidth*len(t.song.Tracks), trackRowHeight*t.song.TotalRows())
	pointer.Rect(rect).Add(gtx.Ops)
	pointer.InputOp{Tag: &trackJumpPointerTag,
		Types: pointer.Press,
	}.Add(gtx.Ops)
	if t.EditMode == EditPatterns || t.EditMode == EditTracks {
		x1, y1 := t.Cursor.Track, t.Cursor.Pattern
		x2, y2 := t.SelectionCorner.Track, t.SelectionCorner.Pattern
		if x1 > x2 {
			x1, x2 = x2, x1
		}
		if y1 > y2 {
			y1, y2 = y2, y1
		}
		x2++
		y2++
		x1 *= trackColWidth
		y1 *= trackRowHeight * t.song.RowsPerPattern
		x2 *= trackColWidth
		y2 *= trackRowHeight * t.song.RowsPerPattern
		paint.FillShape(gtx.Ops, inactiveSelectionColor, clip.Rect{Min: image.Pt(x1, y1), Max: image.Pt(x2, y2)}.Op())
	}
	if t.Playing {
		py := trackRowHeight * (t.PlayPosition.Pattern*t.song.RowsPerPattern + t.PlayPosition.Row)
		paint.FillShape(gtx.Ops, trackerPlayColor, clip.Rect{Min: image.Pt(0, py), Max: image.Pt(gtx.Constraints.Max.X, py+trackRowHeight)}.Op())
	}
	if t.EditMode == EditTracks {
		x1, y1 := t.Cursor.Track, t.Cursor.Pattern*t.song.RowsPerPattern+t.Cursor.Row
		x2, y2 := t.SelectionCorner.Track, t.SelectionCorner.Pattern*t.song.RowsPerPattern+t.SelectionCorner.Row
		if x1 > x2 {
			x1, x2 = x2, x1
		}
		if y1 > y2 {
			y1, y2 = y2, y1
		}
		x2++
		y2++
		x1 *= trackColWidth
		y1 *= trackRowHeight
		x2 *= trackColWidth
		y2 *= trackRowHeight
		paint.FillShape(gtx.Ops, selectionColor, clip.Rect{Min: image.Pt(x1, y1), Max: image.Pt(x2, y2)}.Op())
		cx := t.Cursor.Track * trackColWidth
		cy := (t.Cursor.Pattern*t.song.RowsPerPattern + t.Cursor.Row) * trackRowHeight
		paint.FillShape(gtx.Ops, cursorColor, clip.Rect{Min: image.Pt(cx, cy), Max: image.Pt(cx+trackColWidth, cy+trackRowHeight)}.Op())
	}
	delta := (gtx.Constraints.Max.Y/2 + trackRowHeight - 1) / trackRowHeight
	firstRow := cursorSongRow - delta
	lastRow := cursorSongRow + delta
	if firstRow < 0 {
		firstRow = 0
	}
	if l := t.song.TotalRows(); lastRow >= l {
		lastRow = l - 1
	}
	op.Offset(f32.Pt(0, float32(trackRowHeight*firstRow))).Add(gtx.Ops)
	for trkIndex, trk := range t.song.Tracks {
		stack := op.Save(gtx.Ops)
		for row := firstRow; row <= lastRow; row++ {
			pat := row / t.song.RowsPerPattern
			patRow := row % t.song.RowsPerPattern
			s := trk.Sequence[pat]
			if patRow == 0 {
				paint.ColorOp{Color: trackerPatMarker}.Add(gtx.Ops)
				widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, patternIndexToString(s))
			}
			op.Offset(f32.Pt(patmarkWidth, 0)).Add(gtx.Ops)
			if t.EditMode == EditTracks && t.Cursor.SongRow.Row == patRow && t.Cursor.SongRow.Pattern == pat {
				paint.ColorOp{Color: trackerActiveTextColor}.Add(gtx.Ops)
			} else {
				paint.ColorOp{Color: trackerInactiveTextColor}.Add(gtx.Ops)
			}
			c := trk.Patterns[s][patRow]
			if t.TrackShowHex[trkIndex] {
				var text string
				switch c {
				case 0:
					text = "--"
				case 1:
					text = ".."
				default:
					text = fmt.Sprintf("%02x", c)
				}
				widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, strings.ToUpper(text))
			} else {
				widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, valueAsNote(c))
			}
			op.Offset(f32.Pt(-patmarkWidth, trackRowHeight)).Add(gtx.Ops)
		}
		stack.Load()
		op.Offset(f32.Pt(trackColWidth, 0)).Add(gtx.Ops)
	}
	return layout.Dimensions{Size: gtx.Constraints.Max}
}
