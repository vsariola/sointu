package gioui

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
	"github.com/vsariola/sointu/tracker"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

const trackRowHeight = 16
const trackColWidth = 54
const patmarkWidth = 16

var trackPointerTag bool
var trackJumpPointerTag bool

func (t *Tracker) layoutTracker(gtx layout.Context) layout.Dimensions {
	rowMarkers := layout.Rigid(t.layoutRowMarkers)

	for t.NewTrackBtn.Clicked() {
		t.AddTrack(true)
	}

	for t.DeleteTrackBtn.Clicked() {
		t.DeleteTrack(false)
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
		deleteTrackBtnStyle := material.IconButton(t.Theme, t.DeleteTrackBtn, widgetForIcon(icons.ActionDelete))
		deleteTrackBtnStyle.Background = transparent
		deleteTrackBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
		if t.CanDeleteTrack() {
			deleteTrackBtnStyle.Color = primaryColor
		} else {
			deleteTrackBtnStyle.Color = disabledTextColor
		}
		newTrackBtnStyle := material.IconButton(t.Theme, t.NewTrackBtn, widgetForIcon(icons.ContentAdd))
		newTrackBtnStyle.Background = transparent
		newTrackBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
		if t.CanAddTrack() {
			newTrackBtnStyle.Color = primaryColor
		} else {
			newTrackBtnStyle.Color = disabledTextColor
		}
		in := layout.UniformInset(unit.Dp(1))
		octave := func(gtx C) D {
			t.OctaveNumberInput.Value = t.Octave()
			numStyle := NumericUpDown(t.Theme, t.OctaveNumberInput, 0, 9)
			gtx.Constraints.Min.Y = gtx.Px(unit.Dp(20))
			gtx.Constraints.Min.X = gtx.Px(unit.Dp(70))
			dims := in.Layout(gtx, numStyle.Layout)
			t.SetOctave(t.OctaveNumberInput.Value)
			return dims
		}
		n := t.Song().Score.Tracks[t.Cursor().Track].NumVoices
		t.TrackVoices.Value = n
		voiceUpDown := func(gtx C) D {
			numStyle := NumericUpDown(t.Theme, t.TrackVoices, 1, t.MaxTrackVoices())
			gtx.Constraints.Min.Y = gtx.Px(unit.Dp(20))
			gtx.Constraints.Min.X = gtx.Px(unit.Dp(70))
			return in.Layout(gtx, numStyle.Layout)
		}
		t.TrackHexCheckBox.Value = t.Song().Score.Tracks[t.Cursor().Track].Effect
		hexCheckBoxStyle := material.CheckBox(t.Theme, t.TrackHexCheckBox, "Hex")
		dims := layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(Label("OCT:", white)),
			layout.Rigid(octave),
			layout.Rigid(Label(" PITCH:", white)),
			layout.Rigid(addSemitoneBtnStyle.Layout),
			layout.Rigid(subtractSemitoneBtnStyle.Layout),
			layout.Rigid(addOctaveBtnStyle.Layout),
			layout.Rigid(subtractOctaveBtnStyle.Layout),
			layout.Rigid(hexCheckBoxStyle.Layout),
			layout.Rigid(Label("  Voices:", white)),
			layout.Rigid(voiceUpDown),
			layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
			layout.Rigid(deleteTrackBtnStyle.Layout),
			layout.Rigid(newTrackBtnStyle.Layout))
		t.Song().Score.Tracks[t.Cursor().Track].Effect = t.TrackHexCheckBox.Value // TODO: we should not modify the model, but how should this be done
		t.SetTrackVoices(t.TrackVoices.Value)
		return dims
	}

	for _, ev := range gtx.Events(&trackPointerTag) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		if e.Type == pointer.Press {
			t.SetEditMode(tracker.EditTracks)
		}
	}
	rect := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
	pointer.Rect(rect).Add(gtx.Ops)
	pointer.InputOp{Tag: &trackPointerTag,
		Types: pointer.Press,
	}.Add(gtx.Ops)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return Surface{Gray: 37, Focus: t.EditMode() == tracker.EditTracks, FitSize: true}.Layout(gtx, menu)
		}),
		layout.Flexed(1, func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				rowMarkers,
				layout.Flexed(1, t.layoutTracks))
		}),
	)
}

func (t *Tracker) layoutTracks(gtx C) D {
	defer op.Save(gtx.Ops).Load()
	clip.Rect{Max: gtx.Constraints.Max}.Add(gtx.Ops)
	cursorSongRow := t.Cursor().Pattern*t.Song().Score.RowsPerPattern + t.Cursor().Row
	for _, ev := range gtx.Events(&trackJumpPointerTag) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		if e.Type == pointer.Press {
			t.SetEditMode(tracker.EditTracks)
			track := int(e.Position.X) / trackColWidth
			row := int((e.Position.Y-float32(gtx.Constraints.Max.Y-trackRowHeight)/2)/trackRowHeight + float32(cursorSongRow))
			cursor := tracker.SongPoint{Track: track, SongRow: tracker.SongRow{Row: row}}.Clamp(t.Song().Score)
			t.SetCursor(cursor)
			t.SetSelectionCorner(cursor)
			cursorSongRow = cursor.Pattern*t.Song().Score.RowsPerPattern + cursor.Row
		}
	}
	rect := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
	pointer.Rect(rect).Add(gtx.Ops)
	pointer.InputOp{Tag: &trackJumpPointerTag,
		Types: pointer.Press,
	}.Add(gtx.Ops)
	op.Offset(f32.Pt(0, float32(gtx.Constraints.Max.Y-trackRowHeight)/2)).Add(gtx.Ops)
	op.Offset(f32.Pt(0, (-1*trackRowHeight)*float32(cursorSongRow))).Add(gtx.Ops)
	if t.EditMode() == tracker.EditPatterns || t.EditMode() == tracker.EditTracks {
		x1, y1 := t.Cursor().Track, t.Cursor().Pattern
		x2, y2 := t.SelectionCorner().Track, t.SelectionCorner().Pattern
		if x1 > x2 {
			x1, x2 = x2, x1
		}
		if y1 > y2 {
			y1, y2 = y2, y1
		}
		x2++
		y2++
		x1 *= trackColWidth
		y1 *= trackRowHeight * t.Song().Score.RowsPerPattern
		x2 *= trackColWidth
		y2 *= trackRowHeight * t.Song().Score.RowsPerPattern
		paint.FillShape(gtx.Ops, inactiveSelectionColor, clip.Rect{Min: image.Pt(x1, y1), Max: image.Pt(x2, y2)}.Op())
	}
	if t.EditMode() == tracker.EditTracks {
		x1, y1 := t.Cursor().Track, t.Cursor().Pattern*t.Song().Score.RowsPerPattern+t.Cursor().Row
		x2, y2 := t.SelectionCorner().Track, t.SelectionCorner().Pattern*t.Song().Score.RowsPerPattern+t.SelectionCorner().Row
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
		cx := t.Cursor().Track * trackColWidth
		cy := (t.Cursor().Pattern*t.Song().Score.RowsPerPattern + t.Cursor().Row) * trackRowHeight
		cw := trackColWidth
		if t.Song().Score.Tracks[t.Cursor().Track].Effect {
			cw /= 2
			if t.LowNibble() {
				cx += cw
			}
		}
		paint.FillShape(gtx.Ops, cursorColor, clip.Rect{Min: image.Pt(cx, cy), Max: image.Pt(cx+cw, cy+trackRowHeight)}.Op())
	}
	delta := (gtx.Constraints.Max.Y/2 + trackRowHeight - 1) / trackRowHeight
	firstRow := cursorSongRow - delta
	lastRow := cursorSongRow + delta
	if firstRow < 0 {
		firstRow = 0
	}
	if l := t.Song().Score.LengthInRows(); lastRow >= l {
		lastRow = l - 1
	}
	op.Offset(f32.Pt(0, float32(trackRowHeight*firstRow))).Add(gtx.Ops)
	for _, trk := range t.Song().Score.Tracks {
		stack := op.Save(gtx.Ops)
		for row := firstRow; row <= lastRow; row++ {
			pat := row / t.Song().Score.RowsPerPattern
			patRow := row % t.Song().Score.RowsPerPattern
			s := -1
			if pat >= 0 && pat < len(trk.Order) {
				s = trk.Order[pat]
			}
			if s < 0 {
				op.Offset(f32.Pt(0, trackRowHeight)).Add(gtx.Ops)
				continue
			}
			if s >= 0 && patRow == 0 {
				paint.ColorOp{Color: trackerPatMarker}.Add(gtx.Ops)
				widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, patternIndexToString(s))
			}
			op.Offset(f32.Pt(patmarkWidth, 0)).Add(gtx.Ops)
			if t.EditMode() == tracker.EditTracks && t.Cursor().Row == patRow && t.Cursor().Pattern == pat {
				paint.ColorOp{Color: trackerActiveTextColor}.Add(gtx.Ops)
			} else {
				paint.ColorOp{Color: trackerInactiveTextColor}.Add(gtx.Ops)
			}
			var c byte = 1
			if s >= 0 && s < len(trk.Patterns) {
				pattern := trk.Patterns[s]
				if patRow >= 0 && patRow < len(pattern) {
					c = pattern[patRow]
				}
			}
			if trk.Effect {
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
				widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, tracker.NoteStr(c))
			}
			op.Offset(f32.Pt(-patmarkWidth, trackRowHeight)).Add(gtx.Ops)
		}
		stack.Load()
		op.Offset(f32.Pt(trackColWidth, 0)).Add(gtx.Ops)
	}
	return layout.Dimensions{Size: gtx.Constraints.Max}
}
