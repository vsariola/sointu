package gioui

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/vsariola/sointu/tracker"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

const trackRowHeight = unit.Dp(16)
const trackColWidth = unit.Dp(54)
const trackColTitleHeight = unit.Dp(16)
const trackPatMarkWidth = unit.Dp(25)
const trackRowMarkWidth = unit.Dp(25)

var noteStr [256]string
var hexStr [256]string

func init() {
	// initialize these strings once, so we don't have to do it every time we draw the note editor
	hexStr[0] = "--"
	hexStr[1] = ".."
	noteStr[0] = "---"
	noteStr[1] = "..."
	for i := 2; i < 256; i++ {
		hexStr[i] = fmt.Sprintf("%02x", i)
		oNote := mod(i-baseNote, 12)
		octave := (i - oNote - baseNote) / 12
		switch {
		case octave < 0:
			noteStr[i] = fmt.Sprintf("%s%s", notes[oNote], string(byte('Z'+1+octave)))
		case octave >= 10:
			noteStr[i] = fmt.Sprintf("%s%s", notes[oNote], string(byte('A'+octave-10)))
		default:
			noteStr[i] = fmt.Sprintf("%s%d", notes[oNote], octave)
		}
	}
}

type NoteEditor struct {
	TrackVoices         *NumberInput
	NewTrackBtn         *ActionClickable
	DeleteTrackBtn      *ActionClickable
	AddSemitoneBtn      *ActionClickable
	SubtractSemitoneBtn *ActionClickable
	AddOctaveBtn        *ActionClickable
	SubtractOctaveBtn   *ActionClickable
	NoteOffBtn          *ActionClickable
	EffectBtn           *BoolClickable

	scrollTable *ScrollTable
	tag         struct{}
}

func NewNoteEditor(model *tracker.Model) *NoteEditor {
	return &NoteEditor{
		TrackVoices:         NewNumberInput(model.TrackVoices().Int()),
		NewTrackBtn:         NewActionClickable(model.AddTrack()),
		DeleteTrackBtn:      NewActionClickable(model.DeleteTrack()),
		AddSemitoneBtn:      NewActionClickable(model.AddSemitone()),
		SubtractSemitoneBtn: NewActionClickable(model.SubtractSemitone()),
		AddOctaveBtn:        NewActionClickable(model.AddOctave()),
		SubtractOctaveBtn:   NewActionClickable(model.SubtractOctave()),
		NoteOffBtn:          NewActionClickable(model.EditNoteOff()),
		EffectBtn:           NewBoolClickable(model.Effect().Bool()),
		scrollTable: NewScrollTable(
			model.Notes().Table(),
			model.Tracks().List(),
			model.NoteRows().List(),
		),
	}
}

func (te *NoteEditor) Layout(gtx layout.Context, t *Tracker) layout.Dimensions {
	for _, e := range gtx.Events(&te.tag) {
		switch e := e.(type) {
		case key.Event:
			if e.State == key.Release {
				if noteID, ok := t.KeyPlaying[e.Name]; ok {
					noteID.NoteOff()
					delete(t.KeyPlaying, e.Name)
				}
				continue
			}
			te.command(gtx, t, e)
		}
	}

	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
	key.InputOp{Tag: &te.tag, Keys: "Ctrl-⌫|Ctrl-⌦|⏎|Ctrl-⏎|A|B|C|D|E|F|G|H|I|J|K|L|M|N|O|P|Q|R|S|T|U|V|W|X|Y|Z|0|1|2|3|4|5|6|7|8|9|,|."}.Add(gtx.Ops)

	return Surface{Gray: 24, Focus: te.scrollTable.Focused()}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return te.layoutButtons(gtx, t)
			}),
			layout.Flexed(1, func(gtx C) D {
				return te.layoutTracks(gtx, t)
			}),
		)
	})
}

func (te *NoteEditor) layoutButtons(gtx C, t *Tracker) D {
	return Surface{Gray: 37, Focus: te.scrollTable.Focused() || te.scrollTable.ChildFocused(), FitSize: true}.Layout(gtx, func(gtx C) D {
		addSemitoneBtnStyle := ActionButton(t.Theme, te.AddSemitoneBtn, "+1")
		subtractSemitoneBtnStyle := ActionButton(t.Theme, te.SubtractSemitoneBtn, "-1")
		addOctaveBtnStyle := ActionButton(t.Theme, te.AddOctaveBtn, "+12")
		subtractOctaveBtnStyle := ActionButton(t.Theme, te.SubtractOctaveBtn, "-12")
		noteOffBtnStyle := ActionButton(t.Theme, te.NoteOffBtn, "Note Off")
		deleteTrackBtnStyle := ActionIcon(t.Theme, te.DeleteTrackBtn, icons.ActionDelete, "Delete track\n(Ctrl+Shift+T)")
		newTrackBtnStyle := ActionIcon(t.Theme, te.NewTrackBtn, icons.ContentAdd, "Add track\n(Ctrl+T)")
		in := layout.UniformInset(unit.Dp(1))
		voiceUpDown := func(gtx C) D {
			numStyle := NumericUpDown(t.Theme, te.TrackVoices, "Number of voices for this track")
			return in.Layout(gtx, numStyle.Layout)
		}
		effectBtnStyle := ToggleButton(t.Theme, te.EffectBtn, "Hex")
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D { return layout.Dimensions{Size: image.Pt(gtx.Dp(unit.Dp(12)), 0)} }),
			layout.Rigid(addSemitoneBtnStyle.Layout),
			layout.Rigid(subtractSemitoneBtnStyle.Layout),
			layout.Rigid(addOctaveBtnStyle.Layout),
			layout.Rigid(subtractOctaveBtnStyle.Layout),
			layout.Rigid(noteOffBtnStyle.Layout),
			layout.Rigid(effectBtnStyle.Layout),
			layout.Rigid(Label("  Voices:", white, t.Theme.Shaper)),
			layout.Rigid(voiceUpDown),
			layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
			layout.Rigid(deleteTrackBtnStyle.Layout),
			layout.Rigid(newTrackBtnStyle.Layout))
	})
}

const baseNote = 24

var notes = []string{
	"C-",
	"C#",
	"D-",
	"D#",
	"E-",
	"F-",
	"F#",
	"G-",
	"G#",
	"A-",
	"A#",
	"B-",
}

func (te *NoteEditor) layoutTracks(gtx C, t *Tracker) D {
	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()

	beatMarkerDensity := t.RowsPerBeat().Value()
	switch beatMarkerDensity {
	case 0, 1, 2:
		beatMarkerDensity = 4
	}

	playSongRow := t.PlaySongRow()
	pxWidth := gtx.Dp(trackColWidth)
	pxHeight := gtx.Dp(trackRowHeight)
	pxPatMarkWidth := gtx.Dp(trackPatMarkWidth)
	pxRowMarkWidth := gtx.Dp(trackRowMarkWidth)

	colTitle := func(gtx C, i int) D {
		h := gtx.Dp(unit.Dp(trackColTitleHeight))
		title := ((*tracker.Order)(t.Model)).Title(i)
		gtx.Constraints = layout.Exact(image.Pt(pxWidth, h))
		LabelStyle{Alignment: layout.N, Text: title, FontSize: unit.Sp(12), Color: mediumEmphasisTextColor, Shaper: t.Theme.Shaper}.Layout(gtx)
		return D{Size: image.Pt(pxWidth, h)}
	}

	rowTitleBg := func(gtx C, j int) D {
		if mod(j, beatMarkerDensity*2) == 0 {
			paint.FillShape(gtx.Ops, twoBeatHighlight, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, pxHeight)}.Op())
		} else if mod(j, beatMarkerDensity) == 0 {
			paint.FillShape(gtx.Ops, oneBeatHighlight, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, pxHeight)}.Op())
		}
		if t.SongPanel.PlayingBtn.Bool.Value() && j == playSongRow {
			paint.FillShape(gtx.Ops, trackerPlayColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, pxHeight)}.Op())
		}
		return D{}
	}

	rowTitle := func(gtx C, j int) D {
		rpp := intMax(t.RowsPerPattern().Value(), 1)
		pat := j / rpp
		row := j % rpp
		w := pxPatMarkWidth + pxRowMarkWidth
		defer op.Offset(image.Pt(0, -2)).Push(gtx.Ops).Pop()
		if row == 0 {
			color := rowMarkerPatternTextColor
			if l := t.Loop(); pat >= l.Start && pat < l.Start+l.Length {
				color = loopMarkerColor
			}
			paint.ColorOp{Color: color}.Add(gtx.Ops)
			widget.Label{}.Layout(gtx, t.Theme.Shaper, trackerFont, trackerFontSize, strings.ToUpper(fmt.Sprintf("%02x", pat)), op.CallOp{})
		}
		defer op.Offset(image.Pt(pxPatMarkWidth, 0)).Push(gtx.Ops).Pop()
		paint.ColorOp{Color: rowMarkerRowTextColor}.Add(gtx.Ops)
		widget.Label{}.Layout(gtx, t.Theme.Shaper, trackerFont, trackerFontSize, strings.ToUpper(fmt.Sprintf("%02x", row)), op.CallOp{})
		return D{Size: image.Pt(w, pxHeight)}
	}

	drawSelection := te.scrollTable.Table.Cursor() != te.scrollTable.Table.Cursor2()
	selection := te.scrollTable.Table.Range()

	cell := func(gtx C, x, y int) D {
		// draw the background, to indicate selection
		color := transparent
		point := tracker.Point{X: x, Y: y}
		if drawSelection && selection.Contains(point) {
			color = inactiveSelectionColor
			if te.scrollTable.Focused() {
				color = selectionColor
			}
		}
		paint.FillShape(gtx.Ops, color, clip.Rect{Min: image.Pt(0, 0), Max: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}.Op())
		// draw the cursor
		if point == te.scrollTable.Table.Cursor() {
			cw := gtx.Constraints.Min.X
			cx := 0
			if t.Model.Notes().Effect(x) {
				cw /= 2
				if t.Model.Notes().LowNibble() {
					cx += cw
				}
			}
			c := inactiveSelectionColor
			if te.scrollTable.Focused() {
				c = cursorColor
			}
			paint.FillShape(gtx.Ops, c, clip.Rect{Min: image.Pt(cx, 0), Max: image.Pt(cx+cw, gtx.Constraints.Min.Y)}.Op())
		}
		// draw the pattern marker
		rpp := intMax(t.RowsPerPattern().Value(), 1)
		pat := y / rpp
		row := y % rpp
		defer op.Offset(image.Pt(0, -2)).Push(gtx.Ops).Pop()
		s := t.Model.Order().Value(tracker.Point{X: x, Y: pat})
		if row == 0 { // draw the pattern marker
			paint.ColorOp{Color: trackerPatMarker}.Add(gtx.Ops)
			widget.Label{}.Layout(gtx, t.Theme.Shaper, trackerFont, trackerFontSize, patternIndexToString(s), op.CallOp{})
		}
		if row == 1 && t.Model.Notes().Unique(x, s) { // draw a * if the pattern is unique
			paint.ColorOp{Color: mediumEmphasisTextColor}.Add(gtx.Ops)
			widget.Label{}.Layout(gtx, t.Theme.Shaper, trackerFont, trackerFontSize, "*", op.CallOp{})
		}
		if te.scrollTable.Table.Cursor() == point && te.scrollTable.Focused() {
			paint.ColorOp{Color: trackerActiveTextColor}.Add(gtx.Ops)
		} else {
			paint.ColorOp{Color: trackerInactiveTextColor}.Add(gtx.Ops)
		}
		val := noteStr[byte(t.Model.Notes().Value(tracker.Point{X: x, Y: y}))]
		if t.Model.Notes().Effect(x) {
			val = hexStr[byte(t.Model.Notes().Value(tracker.Point{X: x, Y: y}))]
		}
		widget.Label{Alignment: text.Middle}.Layout(gtx, t.Theme.Shaper, trackerFont, trackerFontSize, val, op.CallOp{})
		return D{Size: image.Pt(pxWidth, pxHeight)}
	}
	table := FilledScrollTable(t.Theme, te.scrollTable, cell, colTitle, rowTitle, nil, rowTitleBg)
	table.RowTitleWidth = trackPatMarkWidth + trackRowMarkWidth
	table.ColumnTitleHeight = trackColTitleHeight
	table.CellWidth = trackColWidth
	table.CellHeight = trackRowHeight
	return table.Layout(gtx)
}

func mod(x, d int) int {
	x = x % d
	if x >= 0 {
		return x
	}
	if d < 0 {
		return x - d
	}
	return x + d
}

func noteAsValue(octave, note int) byte {
	return byte(baseNote + (octave * 12) + note)
}

func (te *NoteEditor) command(gtx C, t *Tracker, e key.Event) {
	if e.Name == "A" || e.Name == "1" {
		t.Model.Notes().Table().Fill(0)
		te.scrollTable.EnsureCursorVisible()
		return
	}
	var n byte
	if t.Model.Notes().Effect(te.scrollTable.Table.Cursor().X) {
		if nibbleValue, err := strconv.ParseInt(e.Name, 16, 8); err == nil {
			n = t.Model.Notes().Value(te.scrollTable.Table.Cursor())
			t.Model.Notes().FillNibble(byte(nibbleValue), t.Model.Notes().LowNibble())
			goto validNote
		}
	} else {
		if val, ok := noteMap[e.Name]; ok {
			n = noteAsValue(t.OctaveNumberInput.Int.Value(), val)
			t.Model.Notes().Table().Fill(int(n))
			goto validNote
		}
	}
	return
validNote:
	te.scrollTable.EnsureCursorVisible()
	if _, ok := t.KeyPlaying[e.Name]; !ok {
		trk := te.scrollTable.Table.Cursor().X
		t.KeyPlaying[e.Name] = t.TrackNoteOn(trk, n)
	}
}

/*

case "+":
	if e.Modifiers.Contain(key.ModShortcut) {
		te.AddOctaveBtn.Action.Do()
	} else {
		te.AddSemitoneBtn.Action.Do()
	}
case "-":
	if e.Modifiers.Contain(key.ModShortcut) {
		te.SubtractSemitoneBtn.Action.Do()
	} else {
		te.SubtractOctaveBtn.Action.Do()
	}
}*/
