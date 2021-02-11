package tracker

import (
	"fmt"
	"strings"
	"sync"

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/vsariola/sointu"
)

type EditMode int

const (
	EditPatterns EditMode = iota
	EditTracks
	EditUnits
	EditParameters
)

type Tracker struct {
	QuitButton    *widget.Clickable
	songPlayMutex sync.RWMutex // protects song and playing
	song          sointu.Song
	Playing       bool
	// protects PlayPattern and PlayRow
	playRowPatMutex       sync.RWMutex // protects song and playing
	PlayPosition          SongRow
	EditMode              EditMode
	SelectionCorner       SongPoint
	Cursor                SongPoint
	CursorColumn          int
	CurrentInstrument     int
	CurrentUnit           int
	CurrentParam          int
	UnitGroupMenuVisible  bool
	UnitGroupMenuIndex    int
	UnitSubMenuIndex      int
	NoteTracking          bool
	Theme                 *material.Theme
	Octave                *NumberInput
	BPM                   *NumberInput
	RowsPerPattern        *NumberInput
	RowsPerBeat           *NumberInput
	InstrumentVoices      *NumberInput
	InstrumentNameEditor  *widget.Editor
	NewTrackBtn           *widget.Clickable
	NewInstrumentBtn      *widget.Clickable
	DeleteInstrumentBtn   *widget.Clickable
	LoadSongFileBtn       *widget.Clickable
	NewSongFileBtn        *widget.Clickable
	AddSemitoneBtn        *widget.Clickable
	SubtractSemitoneBtn   *widget.Clickable
	AddOctaveBtn          *widget.Clickable
	SubtractOctaveBtn     *widget.Clickable
	SongLength            *NumberInput
	SaveSongFileBtn       *widget.Clickable
	FileMenuBtn           *widget.Clickable
	PanicBtn              *widget.Clickable
	FileMenuVisible       bool
	ParameterSliders      []*widget.Float
	ParameterList         *layout.List
	UnitDragList          *DragList
	DeleteUnitBtn         *widget.Clickable
	ClearUnitBtn          *widget.Clickable
	ChooseUnitTypeList    *layout.List
	ChooseUnitTypeBtns    []*widget.Clickable
	AddUnitBtn            *widget.Clickable
	InstrumentDragList    *DragList
	TrackHexCheckBoxes    []*widget.Bool
	TrackShowHex          []bool
	TopHorizontalSplit    *Split
	BottomHorizontalSplit *Split
	VerticalSplit         *Split
	StackUse              []int
	KeyPlaying            map[string]func()

	sequencer    *Sequencer
	refresh      chan struct{}
	setPlaying   chan bool
	rowJump      chan int
	patternJump  chan int
	audioContext sointu.AudioContext
	playBuffer   []float32
	closer       chan struct{}
	undoStack    []sointu.Song
	redoStack    []sointu.Song
}

func (t *Tracker) LoadSong(song sointu.Song) error {
	if err := song.Validate(); err != nil {
		return fmt.Errorf("invalid song: %w", err)
	}
	t.songPlayMutex.Lock()
	defer t.songPlayMutex.Unlock()
	t.song = song
	t.ClampPositions()
	if t.sequencer != nil {
		t.sequencer.SetPatch(song.Patch)
		t.sequencer.SetRowLength(song.SamplesPerRow())
	}
	return nil
}

func clamp(a, min, max int) int {
	if a < min {
		return min
	}
	if a >= max {
		return max - 1
	}
	return a
}

func (t *Tracker) Close() {
	t.audioContext.Close()
	t.closer <- struct{}{}
}

func (t *Tracker) TogglePlay() {
	t.songPlayMutex.Lock()
	defer t.songPlayMutex.Unlock()
	t.Playing = !t.Playing
	if t.Playing {
		t.NoteTracking = true
		t.PlayPosition = t.Cursor.SongRow
		t.PlayPosition.Row-- // TODO: we advance soon to make up for this -1, but this is not very elegant way to do it
	}
}

func (t *Tracker) ChangeOctave(delta int) bool {
	newOctave := t.Octave.Value + delta
	if newOctave < 0 {
		newOctave = 0
	}
	if newOctave > 9 {
		newOctave = 9
	}
	if newOctave != t.Octave.Value {
		t.Octave.Value = newOctave
		return true
	}
	return false
}

func (t *Tracker) SetInstrumentVoices(value int) bool {
	if value < 1 {
		value = 1
	}
	maxRemain := 32 - t.song.Patch.TotalVoices() + t.song.Patch.Instruments[t.CurrentInstrument].NumVoices
	if maxRemain < 1 {
		maxRemain = 1
	}
	if value > maxRemain {
		value = maxRemain
	}
	if value != int(t.song.Patch.Instruments[t.CurrentInstrument].NumVoices) {
		t.SaveUndo()
		t.song.Patch.Instruments[t.CurrentInstrument].NumVoices = value
		t.sequencer.SetPatch(t.song.Patch)
		return true
	}
	return false
}

func (t *Tracker) SetInstrumentName(name string) {
	name = strings.TrimSpace(name)
	if name != t.song.Patch.Instruments[t.CurrentInstrument].Name {
		t.SaveUndo()
		t.song.Patch.Instruments[t.CurrentInstrument].Name = name
	}
}

func (t *Tracker) SetBPM(value int) bool {
	if value < 1 {
		value = 1
	}
	if value > 999 {
		value = 999
	}
	if value != int(t.song.BPM) {
		t.SaveUndo()
		t.song.BPM = value
		t.sequencer.SetRowLength(t.song.SamplesPerRow())
		return true
	}
	return false
}

func (t *Tracker) SetRowsPerBeat(value int) bool {
	if value < 1 {
		value = 1
	}
	if value > 32 {
		value = 32
	}
	if value != int(t.song.RowsPerBeat) {
		t.SaveUndo()
		t.song.RowsPerBeat = value
		t.sequencer.SetRowLength(t.song.SamplesPerRow())
		return true
	}
	return false
}

func (t *Tracker) AddTrack() {
	t.SaveUndo()
	if t.song.TotalTrackVoices() < t.song.Patch.TotalVoices() {
		seq := make([]byte, t.song.SequenceLength())
		patterns := [][]byte{make([]byte, t.song.RowsPerPattern)}
		t.song.Tracks = append(t.song.Tracks, sointu.Track{
			NumVoices: 1,
			Patterns:  patterns,
			Sequence:  seq,
		})
	}
}

func (t *Tracker) AddInstrument() {
	if t.song.Patch.TotalVoices() >= 32 {
		return
	}
	t.SaveUndo()
	instr := make([]sointu.Instrument, len(t.song.Patch.Instruments)+1)
	copy(instr, t.song.Patch.Instruments[:t.CurrentInstrument+1])
	instr[t.CurrentInstrument+1] = defaultInstrument.Copy()
	copy(instr[t.CurrentInstrument+2:], t.song.Patch.Instruments[t.CurrentInstrument+1:])
	t.song.Patch.Instruments = instr
	t.CurrentInstrument++
	t.ClampPositions()
	t.sequencer.SetPatch(t.song.Patch)
}

func (t *Tracker) SwapInstruments(i, j int) {
	if i < 0 || j < 0 || i >= len(t.song.Patch.Instruments) || j >= len(t.song.Patch.Instruments) {
		return
	}
	t.SaveUndo()
	instruments := t.song.Patch.Instruments
	instruments[i], instruments[j] = instruments[j], instruments[i]
	t.ClampPositions()
	t.sequencer.SetPatch(t.song.Patch)
}

func (t *Tracker) DeleteInstrument() {
	if len(t.song.Patch.Instruments) <= 1 {
		return
	}
	t.SaveUndo()
	t.song.Patch.Instruments = append(t.song.Patch.Instruments[:t.CurrentInstrument], t.song.Patch.Instruments[t.CurrentInstrument+1:]...)
	if t.CurrentInstrument >= len(t.song.Patch.Instruments) {
		t.CurrentInstrument = len(t.song.Patch.Instruments) - 1
	}
	t.ClampPositions()
	t.sequencer.SetPatch(t.song.Patch)
}

// SetCurrentNote sets the (note) value in current pattern under cursor to iv
func (t *Tracker) SetCurrentNote(iv byte) {
	t.SaveUndo()
	t.song.Tracks[t.Cursor.Track].Patterns[t.song.Tracks[t.Cursor.Track].Sequence[t.Cursor.Pattern]][t.Cursor.Row] = iv
}

func (t *Tracker) SetCurrentPattern(pat byte) {
	t.SaveUndo()
	length := len(t.song.Tracks[t.Cursor.Track].Patterns)
	if int(pat) >= length {
		tail := make([][]byte, int(pat)-length+1)
		for i := range tail {
			tail[i] = make([]byte, t.song.RowsPerPattern)
		}
		t.song.Tracks[t.Cursor.Track].Patterns = append(t.song.Tracks[t.Cursor.Track].Patterns, tail...)
	}
	t.song.Tracks[t.Cursor.Track].Sequence[t.Cursor.Pattern] = pat
}

func (t *Tracker) SetSongLength(value int) {
	if value < 1 {
		value = 1
	}
	if value != t.song.SequenceLength() {
		t.SaveUndo()
		for i := range t.song.Tracks {
			seq := t.song.Tracks[i].Sequence
			if len(t.song.Tracks[i].Sequence) > value {
				t.song.Tracks[i].Sequence = t.song.Tracks[i].Sequence[:value]
			} else if len(t.song.Tracks[i].Sequence) < value {
				for k := len(t.song.Tracks[i].Sequence); k < value; k++ {
					t.song.Tracks[i].Sequence = append(seq, seq[len(seq)-1])
				}
			}

		}
		t.ClampPositions()
	}
}

func (t *Tracker) SetRowsPerPattern(value int) {
	if value < 1 {
		value = 1
	}
	if value > 255 {
		value = 255
	}
	if value != t.song.RowsPerPattern {
		t.SaveUndo()
		for i := range t.song.Tracks {
			for j := range t.song.Tracks[i].Patterns {
				pat := t.song.Tracks[i].Patterns[j]
				if l := len(pat); l < value {
					tail := make([]byte, value-l)
					for k := range tail {
						tail[k] = 1
					}
					t.song.Tracks[i].Patterns[j] = append(pat, tail...)
				}
			}
		}
		t.song.RowsPerPattern = value
		t.ClampPositions()
	}
}

func (t *Tracker) SetUnit(typ string) {
	unit, ok := defaultUnits[typ]
	if !ok {
		return
	}
	if unit.Type == t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit].Type {
		return
	}
	t.SaveUndo()
	t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit] = unit.Copy()
	t.sequencer.SetPatch(t.song.Patch)
}

func (t *Tracker) AddUnit() {
	t.SaveUndo()
	units := make([]sointu.Unit, len(t.song.Patch.Instruments[t.CurrentInstrument].Units)+1)
	copy(units, t.song.Patch.Instruments[t.CurrentInstrument].Units[:t.CurrentUnit+1])
	copy(units[t.CurrentUnit+2:], t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit+1:])
	t.song.Patch.Instruments[t.CurrentInstrument].Units = units
	t.CurrentUnit++
	t.ClampPositions()
	t.sequencer.SetPatch(t.song.Patch)
}

func (t *Tracker) ClearUnit() {
	t.SaveUndo()
	t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit].Type = ""
	t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit].Parameters = make(map[string]int)
	t.ClampPositions()
	t.sequencer.SetPatch(t.song.Patch)
}

func (t *Tracker) DeleteUnit() {
	if len(t.song.Patch.Instruments[t.CurrentInstrument].Units) <= 1 {
		return
	}
	t.SaveUndo()
	units := make([]sointu.Unit, len(t.song.Patch.Instruments[t.CurrentInstrument].Units)-1)
	copy(units, t.song.Patch.Instruments[t.CurrentInstrument].Units[:t.CurrentUnit])
	copy(units[t.CurrentUnit:], t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit+1:])
	t.song.Patch.Instruments[t.CurrentInstrument].Units = units
	if t.CurrentUnit > 0 {
		t.CurrentUnit--
	}
	t.ClampPositions()
	t.sequencer.SetPatch(t.song.Patch)
}

func (t *Tracker) GetUnitParam() int {
	unit := t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit]
	paramtype := sointu.UnitTypes[unit.Type][t.CurrentParam]
	return unit.Parameters[paramtype.Name]
}

func (t *Tracker) SetUnitParam(value int) {
	unit := t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit]
	unittype := sointu.UnitTypes[unit.Type][t.CurrentParam]
	if value < unittype.MinValue {
		value = unittype.MinValue
	} else if value > unittype.MaxValue {
		value = unittype.MaxValue
	}
	if unit.Parameters[unittype.Name] == value {
		return
	}
	t.SaveUndo()
	unit.Parameters[unittype.Name] = value
	t.ClampPositions()
	t.sequencer.SetPatch(t.song.Patch)
}

func (t *Tracker) SwapUnits(i, j int) {
	if i < 0 || j < 0 || i >= len(t.song.Patch.Instruments[t.CurrentInstrument].Units) || j >= len(t.song.Patch.Instruments[t.CurrentInstrument].Units) {
		return
	}
	t.SaveUndo()
	units := t.song.Patch.Instruments[t.CurrentInstrument].Units
	units[i], units[j] = units[j], units[i]
	t.ClampPositions()
	t.sequencer.SetPatch(t.song.Patch)
}

func (t *Tracker) ClampPositions() {
	t.PlayPosition.Clamp(t.song)
	t.Cursor.Clamp(t.song)
	t.SelectionCorner.Clamp(t.song)
	if t.Cursor.Track >= len(t.TrackShowHex) || !t.TrackShowHex[t.Cursor.Track] {
		t.CursorColumn = 0
	}
	t.CurrentInstrument = clamp(t.CurrentInstrument, 0, len(t.song.Patch.Instruments))
	t.CurrentUnit = clamp(t.CurrentUnit, 0, len(t.song.Patch.Instruments[t.CurrentInstrument].Units))
	numSettableParams := 0
	for _, t := range sointu.UnitTypes[t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit].Type] {
		if t.CanSet {
			numSettableParams++
		}
	}
	if t.CurrentParam < 0 && t.CurrentUnit > 0 {
		t.CurrentUnit--
		numSettableParams = 0
		for _, t := range sointu.UnitTypes[t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit].Type] {
			if t.CanSet {
				numSettableParams++
			}
		}
		t.CurrentParam = numSettableParams - 1
	}
	if t.CurrentParam >= numSettableParams && t.CurrentUnit < len(t.song.Patch.Instruments[t.CurrentInstrument].Units)-1 {
		t.CurrentUnit++
		numSettableParams = 0
		for _, t := range sointu.UnitTypes[t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit].Type] {
			if t.CanSet {
				numSettableParams++
			}
		}
		t.CurrentParam = 0
	}
	t.CurrentParam = clamp(t.CurrentParam, 0, numSettableParams)
}

func (t *Tracker) getSelectionRange() (int, int, int, int) {
	r1 := t.Cursor.Pattern*t.song.RowsPerPattern + t.Cursor.Row
	r2 := t.SelectionCorner.Pattern*t.song.RowsPerPattern + t.SelectionCorner.Row
	if r2 < r1 {
		r1, r2 = r2, r1
	}
	t1 := t.Cursor.Track
	t2 := t.SelectionCorner.Track
	if t2 < t1 {
		t1, t2 = t2, t1
	}
	return r1, r2, t1, t2
}

func (t *Tracker) AdjustSelectionPitch(delta int) {
	t.SaveUndo()
	r1, r2, t1, t2 := t.getSelectionRange()
	for c := t1; c <= t2; c++ {
		adjustedNotes := map[struct {
			Pat byte
			Row int
		}]bool{}
		for r := r1; r <= r2; r++ {
			s := SongRow{Row: r}
			s.Wrap(t.song)
			p := t.song.Tracks[c].Sequence[s.Pattern]
			noteIndex := struct {
				Pat byte
				Row int
			}{p, s.Row}
			if !adjustedNotes[noteIndex] {
				if val := t.song.Tracks[c].Patterns[p][s.Row]; val > 1 {
					newVal := int(val) + delta
					if newVal < 2 {
						newVal = 2
					} else if newVal > 255 {
						newVal = 255
					}
					t.song.Tracks[c].Patterns[p][s.Row] = byte(newVal)
				}
				adjustedNotes[noteIndex] = true
			}
		}
	}
}

func (t *Tracker) DeleteSelection() {
	t.SaveUndo()
	r1, r2, t1, t2 := t.getSelectionRange()
	for r := r1; r <= r2; r++ {
		s := SongRow{Row: r}
		s.Wrap(t.song)
		for c := t1; c <= t2; c++ {
			p := t.song.Tracks[c].Sequence[s.Pattern]
			t.song.Tracks[c].Patterns[p][s.Row] = 1
		}
	}
}

func (t *Tracker) Unselect() {
	t.SelectionCorner = t.Cursor
}

func New(audioContext sointu.AudioContext, synthService sointu.SynthService) *Tracker {
	t := &Tracker{
		Theme:                 material.NewTheme(gofont.Collection()),
		QuitButton:            new(widget.Clickable),
		audioContext:          audioContext,
		BPM:                   new(NumberInput),
		Octave:                new(NumberInput),
		SongLength:            new(NumberInput),
		RowsPerPattern:        new(NumberInput),
		RowsPerBeat:           new(NumberInput),
		InstrumentVoices:      new(NumberInput),
		InstrumentNameEditor:  &widget.Editor{SingleLine: true, Submit: true, Alignment: text.Middle},
		NewTrackBtn:           new(widget.Clickable),
		NewInstrumentBtn:      new(widget.Clickable),
		DeleteInstrumentBtn:   new(widget.Clickable),
		NewSongFileBtn:        new(widget.Clickable),
		FileMenuBtn:           new(widget.Clickable),
		LoadSongFileBtn:       new(widget.Clickable),
		SaveSongFileBtn:       new(widget.Clickable),
		AddSemitoneBtn:        new(widget.Clickable),
		SubtractSemitoneBtn:   new(widget.Clickable),
		AddOctaveBtn:          new(widget.Clickable),
		SubtractOctaveBtn:     new(widget.Clickable),
		AddUnitBtn:            new(widget.Clickable),
		DeleteUnitBtn:         new(widget.Clickable),
		ClearUnitBtn:          new(widget.Clickable),
		PanicBtn:              new(widget.Clickable),
		UnitDragList:          &DragList{List: &layout.List{Axis: layout.Vertical}},
		setPlaying:            make(chan bool),
		rowJump:               make(chan int),
		patternJump:           make(chan int),
		refresh:               make(chan struct{}, 1), // use non-blocking sends; no need to queue extra ticks if one is queued already
		closer:                make(chan struct{}),
		undoStack:             []sointu.Song{},
		redoStack:             []sointu.Song{},
		InstrumentDragList:    &DragList{List: &layout.List{Axis: layout.Horizontal}},
		ParameterList:         &layout.List{Axis: layout.Vertical},
		TopHorizontalSplit:    new(Split),
		BottomHorizontalSplit: new(Split),
		VerticalSplit:         new(Split),
		ChooseUnitTypeList:    &layout.List{Axis: layout.Vertical},
		KeyPlaying:            make(map[string]func()),
	}
	t.UnitDragList.HoverItem = -1
	t.InstrumentDragList.HoverItem = -1
	t.Octave.Value = 4
	t.VerticalSplit.Axis = layout.Vertical
	t.BottomHorizontalSplit.Ratio = -.5
	t.Theme.Palette.Fg = primaryColor
	t.Theme.Palette.ContrastFg = black
	for range allUnits {
		t.ChooseUnitTypeBtns = append(t.ChooseUnitTypeBtns, new(widget.Clickable))
	}
	t.sequencer = NewSequencer(2048, synthService, audioContext, func(row []RowNote) []RowNote {
		t.playRowPatMutex.Lock()
		if !t.Playing {
			t.playRowPatMutex.Unlock()
			return nil
		}
		t.PlayPosition.Row++
		t.PlayPosition.Wrap(t.song)
		if t.NoteTracking {
			t.Cursor.SongRow = t.PlayPosition
			t.SelectionCorner.SongRow = t.PlayPosition
		}
		for _, track := range t.song.Tracks {
			patternIndex := track.Sequence[t.PlayPosition.Pattern]
			note := track.Patterns[patternIndex][t.PlayPosition.Row]
			row = append(row, RowNote{Note: note, NumVoices: track.NumVoices})
		}
		t.playRowPatMutex.Unlock()
		select {
		case t.refresh <- struct{}{}:
		default:
			// message dropped, there's already a tick queued, so no need to queue extra
		}

		return row
	})
	if err := t.LoadSong(defaultSong.Copy()); err != nil {
		panic(fmt.Errorf("cannot load default song: %w", err))
	}
	return t
}
