package gioui

import (
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"

	"gioui.org/io/event"
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
	TrackVoices    *NumberInput
	NewTrackBtn    *ActionClickable
	DeleteTrackBtn *ActionClickable
	SplitTrackBtn  *ActionClickable

	AddSemitoneBtn      *ActionClickable
	SubtractSemitoneBtn *ActionClickable
	AddOctaveBtn        *ActionClickable
	SubtractOctaveBtn   *ActionClickable
	NoteOffBtn          *ActionClickable
	EffectBtn           *BoolClickable
	UniqueBtn           *BoolClickable
	TrackMidiInBtn      *BoolClickable

	scrollTable  *ScrollTable
	eventFilters []event.Filter

	deleteTrackHint           string
	addTrackHint              string
	uniqueOffTip, uniqueOnTip string
	splitTrackHint            string
}

func NewNoteEditor(model *tracker.Model) *NoteEditor {
	ret := &NoteEditor{
		TrackVoices:         NewNumberInput(model.TrackVoices().Int()),
		NewTrackBtn:         NewActionClickable(model.AddTrack()),
		DeleteTrackBtn:      NewActionClickable(model.DeleteTrack()),
		SplitTrackBtn:       NewActionClickable(model.SplitTrack()),
		AddSemitoneBtn:      NewActionClickable(model.AddSemitone()),
		SubtractSemitoneBtn: NewActionClickable(model.SubtractSemitone()),
		AddOctaveBtn:        NewActionClickable(model.AddOctave()),
		SubtractOctaveBtn:   NewActionClickable(model.SubtractOctave()),
		NoteOffBtn:          NewActionClickable(model.EditNoteOff()),
		EffectBtn:           NewBoolClickable(model.Effect().Bool()),
		UniqueBtn:           NewBoolClickable(model.UniquePatterns().Bool()),
		TrackMidiInBtn:      NewBoolClickable(model.TrackMidiIn().Bool()),
		scrollTable: NewScrollTable(
			model.Notes().Table(),
			model.Tracks().List(),
			model.NoteRows().List(),
		),
	}
	for k, a := range keyBindingMap {
		if len(a) < 4 || a[:4] != "Note" {
			continue
		}
		ret.eventFilters = append(ret.eventFilters, key.Filter{Focus: ret.scrollTable, Required: k.Modifiers, Name: k.Name})
	}
	for c := 'A'; c <= 'F'; c++ {
		ret.eventFilters = append(ret.eventFilters, key.Filter{Focus: ret.scrollTable, Name: key.Name(c)})
	}
	for c := '0'; c <= '9'; c++ {
		ret.eventFilters = append(ret.eventFilters, key.Filter{Focus: ret.scrollTable, Name: key.Name(c)})
	}
	ret.deleteTrackHint = makeHint("Delete\ntrack", "\n(%s)", "DeleteTrack")
	ret.addTrackHint = makeHint("Add\ntrack", "\n(%s)", "AddTrack")
	ret.uniqueOnTip = makeHint("Duplicate non-unique patterns", " (%s)", "UniquePatternsToggle")
	ret.uniqueOffTip = makeHint("Allow editing non-unique patterns", " (%s)", "UniquePatternsToggle")
	ret.splitTrackHint = makeHint("Split track", " (%s)", "SplitTrack")
	return ret
}

func (te *NoteEditor) Layout(gtx layout.Context, t *Tracker) layout.Dimensions {
	for {
		e, ok := gtx.Event(te.eventFilters...)
		if !ok {
			break
		}
		switch e := e.(type) {
		case key.Event:
			if e.State == key.Release {
				if noteID, ok := t.KeyPlaying[e.Name]; ok {
					noteID.NoteOff()
					delete(t.KeyPlaying, e.Name)
				}
				continue
			}
			te.command(t, e)
		}
	}

	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()

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
		addSemitoneBtnStyle := ActionButton(gtx, t.Theme, te.AddSemitoneBtn, "+1")
		subtractSemitoneBtnStyle := ActionButton(gtx, t.Theme, te.SubtractSemitoneBtn, "-1")
		addOctaveBtnStyle := ActionButton(gtx, t.Theme, te.AddOctaveBtn, "+12")
		subtractOctaveBtnStyle := ActionButton(gtx, t.Theme, te.SubtractOctaveBtn, "-12")
		noteOffBtnStyle := ActionButton(gtx, t.Theme, te.NoteOffBtn, "Note Off")
		deleteTrackBtnStyle := ActionIcon(gtx, t.Theme, te.DeleteTrackBtn, icons.ActionDelete, te.deleteTrackHint)
		splitTrackBtnStyle := ActionIcon(gtx, t.Theme, te.SplitTrackBtn, icons.CommunicationCallSplit, te.splitTrackHint)
		newTrackBtnStyle := ActionIcon(gtx, t.Theme, te.NewTrackBtn, icons.ContentAdd, te.addTrackHint)
		in := layout.UniformInset(unit.Dp(1))
		voiceUpDown := func(gtx C) D {
			numStyle := NumericUpDown(t.Theme, te.TrackVoices, "Track voices")
			return in.Layout(gtx, numStyle.Layout)
		}
		effectBtnStyle := ToggleButton(gtx, t.Theme, te.EffectBtn, "Hex")
		uniqueBtnStyle := ToggleIcon(gtx, t.Theme, te.UniqueBtn, icons.ToggleStarBorder, icons.ToggleStar, te.uniqueOffTip, te.uniqueOnTip)
		midiInBtnStyle := ToggleButton(gtx, t.Theme, te.TrackMidiInBtn, "MIDI")
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D { return layout.Dimensions{Size: image.Pt(gtx.Dp(unit.Dp(12)), 0)} }),
			layout.Rigid(addSemitoneBtnStyle.Layout),
			layout.Rigid(subtractSemitoneBtnStyle.Layout),
			layout.Rigid(addOctaveBtnStyle.Layout),
			layout.Rigid(subtractOctaveBtnStyle.Layout),
			layout.Rigid(noteOffBtnStyle.Layout),
			layout.Rigid(effectBtnStyle.Layout),
			layout.Rigid(uniqueBtnStyle.Layout),
			layout.Rigid(layout.Spacer{Width: 10}.Layout),
			layout.Rigid(LabelStyle{Text: "Voices", Color: disabledTextColor, Alignment: layout.W, FontSize: t.Theme.TextSize * 14.0 / 16.0, Shaper: t.Theme.Shaper}.Layout),
			layout.Rigid(layout.Spacer{Width: 4}.Layout),
			layout.Rigid(voiceUpDown),
			layout.Rigid(splitTrackBtnStyle.Layout),
			layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
			layout.Rigid(midiInBtnStyle.Layout),
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
		h := gtx.Dp(trackColTitleHeight)
		gtx.Constraints = layout.Exact(image.Pt(pxWidth, h))
		LabelStyle{
			Alignment: layout.N,
			Text:      t.Model.TrackTitle(i),
			FontSize:  unit.Sp(12),
			Color:     mediumEmphasisTextColor,
			Shaper:    t.Theme.Shaper,
		}.Layout(gtx)
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
		rpp := max(t.RowsPerPattern().Value(), 1)
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

	cursor := te.scrollTable.Table.Cursor()
	drawSelection := cursor != te.scrollTable.Table.Cursor2()
	selection := te.scrollTable.Table.Range()
	hasTrackMidiIn := te.TrackMidiInBtn.Bool.Value()

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
		if point == cursor {
			c := inactiveSelectionColor
			if te.scrollTable.Focused() {
				c = cursorColor
			}
			if hasTrackMidiIn {
				c = cursorForTrackMidiInColor
			}
			te.paintColumnCell(gtx, x, t, c)
		}
		// draw the corresponding "fake cursors" for instrument-track-groups (for polyphony)
		if hasTrackMidiIn {
			for _, trackIndex := range t.Model.TracksWithSameInstrumentAsCurrent() {
				if x == trackIndex && y == cursor.Y {
					te.paintColumnCell(gtx, x, t, cursorNeighborForTrackMidiInColor)
				}
			}
		}

		// draw the pattern marker
		rpp := max(t.RowsPerPattern().Value(), 1)
		pat := y / rpp
		row := y % rpp
		defer op.Offset(image.Pt(0, -2)).Push(gtx.Ops).Pop()
		s := t.Model.Order().Value(tracker.Point{X: x, Y: pat})
		if row == 0 { // draw the pattern marker
			paint.ColorOp{Color: trackerPatMarker}.Add(gtx.Ops)
			widget.Label{}.Layout(gtx, t.Theme.Shaper, trackerFont, trackerFontSize, patternIndexToString(s), op.CallOp{})
		}
		if row == 1 && t.Model.PatternUnique(x, s) { // draw a * if the pattern is unique
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

func (te *NoteEditor) paintColumnCell(gtx C, x int, t *Tracker, c color.NRGBA) {
	cw := gtx.Constraints.Min.X
	cx := 0
	if t.Model.Notes().Effect(x) {
		cw /= 2
		if t.Model.Notes().LowNibble() {
			cx += cw
		}
	}
	paint.FillShape(gtx.Ops, c, clip.Rect{Min: image.Pt(cx, 0), Max: image.Pt(cx+cw, gtx.Constraints.Min.Y)}.Op())
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

func (te *NoteEditor) command(t *Tracker, e key.Event) {
	var n byte
	if t.Model.Notes().Effect(te.scrollTable.Table.Cursor().X) {
		if nibbleValue, err := strconv.ParseInt(string(e.Name), 16, 8); err == nil {
			t.Model.Notes().FillNibble(byte(nibbleValue), t.Model.Notes().LowNibble())
			n = t.Model.Notes().Value(te.scrollTable.Table.Cursor())
			te.finishNoteInsert(t, n, e.Name)
		}
	} else {
		action, ok := keyBindingMap[e]
		if !ok {
			return
		}
		if action == "NoteOff" {
			t.Model.Notes().Table().Fill(0)
			te.finishNoteInsert(t, 0, "")
			return
		}
		if action[:4] == "Note" {
			val, err := strconv.Atoi(string(action[4:]))
			if err != nil {
				return
			}
			n = noteAsValue(t.OctaveNumberInput.Int.Value(), val-12)
			t.Model.Notes().Table().Fill(int(n))
			te.finishNoteInsert(t, n, e.Name)
		}
	}
}

func (te *NoteEditor) finishNoteInsert(t *Tracker, note byte, keyName key.Name) {
	if step := t.Model.Step().Value(); step > 0 {
		te.scrollTable.Table.MoveCursor(0, step)
		te.scrollTable.Table.SetCursor2(te.scrollTable.Table.Cursor())
	}
	te.scrollTable.EnsureCursorVisible()

	if keyName == "" {
		return
	}
	if _, ok := t.KeyPlaying[keyName]; !ok {
		trk := te.scrollTable.Table.Cursor().X
		t.KeyPlaying[keyName] = t.TrackNoteOn(trk, note)
	}
}

func (te *NoteEditor) HandleMidiInput(t *Tracker) {
	inputDeactivated := !t.Model.TrackMidiIn().Value()
	if inputDeactivated {
		return
	}
	te.scrollTable.Table.SetCursor2(te.scrollTable.Table.Cursor())
	remaining := t.Model.CountNextTracksForCurrentInstrument()
	for i, note := range t.MidiNotePlaying {
		t.Model.Notes().Table().Set(note)
		te.scrollTable.Table.MoveCursor(1, 0)
		te.scrollTable.EnsureCursorVisible()
		if i >= remaining {
			break
		}
	}
	te.scrollTable.Table.SetCursor(te.scrollTable.Table.Cursor2())
}
