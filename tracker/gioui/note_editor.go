package gioui

import (
	"fmt"
	"image"
	"image/color"
	"strconv"

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

var noteName [256]string
var noteHex [256]string
var hexStr [256]string

func init() {
	// initialize these strings once, so we don't have to do it every time we draw the note editor
	for i := range 256 {
		hexStr[i] = fmt.Sprintf("%02X", i)
	}
	noteHex[0] = "--"
	noteHex[1] = ".."
	noteName[0] = "---"
	noteName[1] = "..."
	for i := 2; i < 256; i++ {
		noteHex[i] = fmt.Sprintf("%02x", i)
		oNote := mod(i-baseNote, 12)
		octave := (i - oNote - baseNote) / 12
		switch {
		case octave < 0:
			noteName[i] = fmt.Sprintf("%s%s", notes[oNote], string(byte('Z'+1+octave)))
		case octave >= 10:
			noteName[i] = fmt.Sprintf("%s%s", notes[oNote], string(byte('A'+octave-10)))
		default:
			noteName[i] = fmt.Sprintf("%s%d", notes[oNote], octave)
		}
	}
}

type NoteEditor struct {
	TrackVoices    *NumericUpDownState
	NewTrackBtn    *Clickable
	DeleteTrackBtn *Clickable
	SplitTrackBtn  *Clickable

	AddSemitoneBtn      *Clickable
	SubtractSemitoneBtn *Clickable
	AddOctaveBtn        *Clickable
	SubtractOctaveBtn   *Clickable
	NoteOffBtn          *Clickable
	EffectBtn           *Clickable
	UniqueBtn           *Clickable
	TrackMidiInBtn      *Clickable

	scrollTable  *ScrollTable
	eventFilters []event.Filter

	deleteTrackHint           string
	addTrackHint              string
	uniqueOffTip, uniqueOnTip string
	splitTrackHint            string
}

func NewNoteEditor(model *tracker.Model) *NoteEditor {
	ret := &NoteEditor{
		TrackVoices:         NewNumericUpDownState(),
		NewTrackBtn:         new(Clickable),
		DeleteTrackBtn:      new(Clickable),
		SplitTrackBtn:       new(Clickable),
		AddSemitoneBtn:      new(Clickable),
		SubtractSemitoneBtn: new(Clickable),
		AddOctaveBtn:        new(Clickable),
		SubtractOctaveBtn:   new(Clickable),
		NoteOffBtn:          new(Clickable),
		EffectBtn:           new(Clickable),
		UniqueBtn:           new(Clickable),
		TrackMidiInBtn:      new(Clickable),
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

func (te *NoteEditor) Layout(gtx layout.Context) layout.Dimensions {
	t := TrackerFromContext(gtx)
	for {
		e, ok := gtx.Event(te.eventFilters...)
		if !ok {
			break
		}
		switch e := e.(type) {
		case key.Event:
			if e.State == key.Release {
				t.KeyNoteMap.Release(e.Name)
				continue
			}
			te.command(t, e)
		}
	}

	for gtx.Focused(te.scrollTable) && len(t.noteEvents) > 0 {
		ev := t.noteEvents[0]
		ev.IsTrack = true
		ev.Channel = t.Model.Notes().Cursor().X
		ev.Source = te
		if ev.On {
			t.Model.Notes().Input(ev.Note)
		}
		copy(t.noteEvents, t.noteEvents[1:])
		t.noteEvents = t.noteEvents[:len(t.noteEvents)-1]
		tracker.TrySend(t.Broker().ToPlayer, any(ev))
	}

	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()

	return Surface{Gray: 24, Focus: te.scrollTable.TreeFocused(gtx)}.Layout(gtx, func(gtx C) D {
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
	return Surface{Gray: 37, Focus: te.scrollTable.TreeFocused(gtx)}.Layout(gtx, func(gtx C) D {
		addSemitoneBtn := ActionBtn(t.AddSemitone(), t.Theme, te.AddSemitoneBtn, "+1", "Add semitone")
		subtractSemitoneBtn := ActionBtn(t.SubtractSemitone(), t.Theme, te.SubtractSemitoneBtn, "-1", "Subtract semitone")
		addOctaveBtn := ActionBtn(t.AddOctave(), t.Theme, te.AddOctaveBtn, "+12", "Add octave")
		subtractOctaveBtn := ActionBtn(t.SubtractOctave(), t.Theme, te.SubtractOctaveBtn, "-12", "Subtract octave")
		noteOffBtn := ActionBtn(t.EditNoteOff(), t.Theme, te.NoteOffBtn, "Note Off", "")
		deleteTrackBtn := ActionIconBtn(t.DeleteTrack(), t.Theme, te.DeleteTrackBtn, icons.ActionDelete, te.deleteTrackHint)
		splitTrackBtn := ActionIconBtn(t.SplitTrack(), t.Theme, te.SplitTrackBtn, icons.CommunicationCallSplit, te.splitTrackHint)
		newTrackBtn := ActionIconBtn(t.AddTrack(), t.Theme, te.NewTrackBtn, icons.ContentAdd, te.addTrackHint)
		trackVoices := NumUpDown(t.Model.TrackVoices(), t.Theme, te.TrackVoices, "Track voices")
		in := layout.UniformInset(unit.Dp(1))
		trackVoicesInsetted := func(gtx C) D {
			return in.Layout(gtx, trackVoices.Layout)
		}
		effectBtn := ToggleBtn(t.Effect(), t.Theme, te.EffectBtn, "Hex", "Input notes as hex values")
		uniqueBtn := ToggleIconBtn(t.UniquePatterns(), t.Theme, te.UniqueBtn, icons.ToggleStarBorder, icons.ToggleStar, te.uniqueOffTip, te.uniqueOnTip)
		midiInBtn := ToggleBtn(t.TrackMidiIn(), t.Theme, te.TrackMidiInBtn, "MIDI", "Input notes from MIDI keyboard")
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D { return layout.Dimensions{Size: image.Pt(gtx.Dp(unit.Dp(12)), 0)} }),
			layout.Rigid(addSemitoneBtn.Layout),
			layout.Rigid(subtractSemitoneBtn.Layout),
			layout.Rigid(addOctaveBtn.Layout),
			layout.Rigid(subtractOctaveBtn.Layout),
			layout.Rigid(noteOffBtn.Layout),
			layout.Rigid(effectBtn.Layout),
			layout.Rigid(uniqueBtn.Layout),
			layout.Rigid(layout.Spacer{Width: 10}.Layout),
			layout.Rigid(Label(t.Theme, &t.Theme.NoteEditor.Header, "Voices").Layout),
			layout.Rigid(layout.Spacer{Width: 4}.Layout),
			layout.Rigid(trackVoicesInsetted),
			layout.Rigid(splitTrackBtn.Layout),
			layout.Rigid(midiInBtn.Layout),
			layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
			layout.Rigid(deleteTrackBtn.Layout),
			layout.Rigid(newTrackBtn.Layout))
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
		Label(t.Theme, &t.Theme.NoteEditor.TrackTitle, t.Model.TrackTitle(i)).Layout(gtx)
		return D{Size: image.Pt(pxWidth, h)}
	}

	rowTitleBg := func(gtx C, j int) D {
		if mod(j, beatMarkerDensity*2) == 0 {
			paint.FillShape(gtx.Ops, t.Theme.NoteEditor.TwoBeat, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, pxHeight)}.Op())
		} else if mod(j, beatMarkerDensity) == 0 {
			paint.FillShape(gtx.Ops, t.Theme.NoteEditor.OneBeat, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, pxHeight)}.Op())
		}
		if t.Model.Playing().Value() && j == playSongRow {
			paint.FillShape(gtx.Ops, t.Theme.NoteEditor.Play, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, pxHeight)}.Op())
		}
		return D{}
	}

	orderRowOp := colorOp(gtx, t.Theme.NoteEditor.OrderRow.Color)
	loopColorOp := colorOp(gtx, t.Theme.OrderEditor.Loop)
	patternRowOp := colorOp(gtx, t.Theme.NoteEditor.PatternRow.Color)

	rowTitle := func(gtx C, j int) D {
		rpp := max(t.RowsPerPattern().Value(), 1)
		pat := j / rpp
		row := j % rpp
		w := pxPatMarkWidth + pxRowMarkWidth
		defer op.Offset(image.Pt(0, -2)).Push(gtx.Ops).Pop()
		if row == 0 {
			op := orderRowOp
			if l := t.Loop(); pat >= l.Start && pat < l.Start+l.Length {
				op = loopColorOp
			}
			widget.Label{}.Layout(gtx, t.Theme.Material.Shaper, t.Theme.NoteEditor.OrderRow.Font, t.Theme.NoteEditor.OrderRow.TextSize, hexStr[pat&255], op)
		}
		defer op.Offset(image.Pt(pxPatMarkWidth, 0)).Push(gtx.Ops).Pop()
		widget.Label{}.Layout(gtx, t.Theme.Material.Shaper, t.Theme.NoteEditor.PatternRow.Font, t.Theme.NoteEditor.PatternRow.TextSize, hexStr[row&255], patternRowOp)
		return D{Size: image.Pt(w, pxHeight)}
	}

	cursor := te.scrollTable.Table.Cursor()
	drawSelection := cursor != te.scrollTable.Table.Cursor2()
	selection := te.scrollTable.Table.Range()
	hasTrackMidiIn := t.Model.TrackMidiIn().Value()

	patternNoOp := colorOp(gtx, t.Theme.NoteEditor.PatternNo.Color)
	uniqueOp := colorOp(gtx, t.Theme.NoteEditor.Unique.Color)
	noteOp := colorOp(gtx, t.Theme.NoteEditor.Note.Color)

	cell := func(gtx C, x, y int) D {
		// draw the background, to indicate selection
		point := tracker.Point{X: x, Y: y}
		if drawSelection && selection.Contains(point) {
			color := t.Theme.Selection.Inactive
			if gtx.Focused(te.scrollTable) {
				color = t.Theme.Selection.Active
			}
			paint.FillShape(gtx.Ops, color, clip.Rect{Min: image.Pt(0, 0), Max: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}.Op())
		}
		// draw the cursor
		if point == cursor {
			c := t.Theme.Cursor.Inactive
			if gtx.Focused(te.scrollTable) {
				c = t.Theme.Cursor.Active
			}
			if hasTrackMidiIn {
				c = t.Theme.Cursor.ActiveAlt
			}
			te.paintColumnCell(gtx, x, t, c)
		}

		// draw the pattern marker
		rpp := max(t.RowsPerPattern().Value(), 1)
		pat := y / rpp
		row := y % rpp
		defer op.Offset(image.Pt(0, -2)).Push(gtx.Ops).Pop()
		s := t.Model.Order().Value(tracker.Point{X: x, Y: pat})
		if row == 0 { // draw the pattern marker
			widget.Label{}.Layout(gtx, t.Theme.Material.Shaper, t.Theme.NoteEditor.PatternNo.Font, t.Theme.NoteEditor.PatternNo.TextSize, patternIndexToString(s), patternNoOp)
		}
		if row == 1 && t.Model.PatternUnique(x, s) { // draw a * if the pattern is unique
			widget.Label{}.Layout(gtx, t.Theme.Material.Shaper, t.Theme.NoteEditor.Unique.Font, t.Theme.NoteEditor.Unique.TextSize, "*", uniqueOp)
		}
		op := noteOp
		val := noteName[byte(t.Model.Notes().Value(tracker.Point{X: x, Y: y}))]
		if t.Model.Notes().Effect(x) {
			val = noteHex[byte(t.Model.Notes().Value(tracker.Point{X: x, Y: y}))]
		}
		widget.Label{Alignment: text.Middle}.Layout(gtx, t.Theme.Material.Shaper, t.Theme.NoteEditor.Note.Font, t.Theme.NoteEditor.Note.TextSize, val, op)
		return D{Size: image.Pt(pxWidth, pxHeight)}
	}
	table := FilledScrollTable(t.Theme, te.scrollTable)
	table.RowTitleWidth = trackPatMarkWidth + trackRowMarkWidth
	table.ColumnTitleHeight = trackColTitleHeight
	table.CellWidth = trackColWidth
	table.CellHeight = trackRowHeight
	return table.Layout(gtx, cell, colTitle, rowTitle, nil, rowTitleBg)
}

func (t *NoteEditor) Tags(level int, yield TagYieldFunc) bool {
	return yield(level+1, t.scrollTable.RowTitleList) &&
		yield(level+1, t.scrollTable.ColTitleList) &&
		yield(level, t.scrollTable)
}

func colorOp(gtx C, c color.NRGBA) op.CallOp {
	macro := op.Record(gtx.Ops)
	paint.ColorOp{Color: c}.Add(gtx.Ops)
	return macro.Stop()
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
			ev := t.Model.Notes().InputNibble(byte(nibbleValue))
			t.KeyNoteMap.Press(e.Name, ev)
		}
	} else {
		action, ok := keyBindingMap[e]
		if !ok {
			return
		}
		if action == "NoteOff" {
			ev := t.Model.Notes().Input(0)
			t.KeyNoteMap.Press(e.Name, ev)
			return
		}
		if action[:4] == "Note" {
			val, err := strconv.Atoi(string(action[4:]))
			if err != nil {
				return
			}
			n = noteAsValue(t.Octave().Value(), val-12)
			ev := t.Model.Notes().Input(n)
			t.KeyNoteMap.Press(e.Name, ev)
		}
	}
}
