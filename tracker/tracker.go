package tracker

import (
	"fmt"
	"sync"

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/bridge"
)

type Tracker struct {
	QuitButton    *widget.Clickable
	songPlayMutex sync.RWMutex // protects song and playing
	song          sointu.Song
	Playing       bool
	// protects PlayPattern and PlayRow
	playRowPatMutex       sync.RWMutex // protects song and playing
	PlayPosition          SongRow
	SelectionCorner       SongPoint
	Cursor                SongPoint
	CursorColumn          int
	CurrentInstrument     int
	CurrentUnit           int
	NoteTracking          bool
	Theme                 *material.Theme
	Octave                *NumberInput
	BPM                   *NumberInput
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
	ParameterSliders      []*widget.Float
	UnitBtns              []*widget.Clickable
	InstrumentBtns        []*widget.Clickable
	InstrumentList        *layout.List
	TrackHexCheckBoxes    []*widget.Bool
	TrackShowHex          []bool
	TopHorizontalSplit    *Split
	BottomHorizontalSplit *Split
	VerticalSplit         *Split

	sequencer    *Sequencer
	ticked       chan struct{}
	setPlaying   chan bool
	rowJump      chan int
	patternJump  chan int
	audioContext sointu.AudioContext
	synth        sointu.Synth
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
	t.PlayPosition.Clamp(song)
	t.Cursor.Clamp(song)
	t.SelectionCorner.Clamp(song)
	if t.sequencer != nil {
		t.sequencer.SetPatch(song.Patch)
	}
	return nil
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

func (t *Tracker) sequencerLoop(closer <-chan struct{}) {
	output := t.audioContext.Output()
	defer output.Close()
	synth, err := bridge.Synth(t.song.Patch)
	if err != nil {
		panic("cannot create a synth with the default patch")
	}
	curVoices := make([]int, 32)
	t.sequencer = NewSequencer(synth, 44100*60/(4*t.song.BPM), func() ([]Note, bool) {
		t.playRowPatMutex.Lock()
		if !t.Playing {
			t.playRowPatMutex.Unlock()
			return nil, false
		}
		t.PlayPosition.Row++
		t.PlayPosition.Wrap(t.song)
		if t.NoteTracking {
			t.Cursor.SongRow = t.PlayPosition
			t.SelectionCorner.SongRow = t.PlayPosition
		}
		notes := make([]Note, 0, 32)
		for track := range t.song.Tracks {
			patternIndex := t.song.Tracks[track].Sequence[t.PlayPosition.Pattern]
			note := t.song.Tracks[track].Patterns[patternIndex][t.PlayPosition.Row]
			if note == 1 { // anything but hold causes an action.
				continue
			}
			first := t.song.FirstTrackVoice(track)
			notes = append(notes, Note{first + curVoices[track], 0})
			if note > 1 {
				curVoices[track]++
				if curVoices[track] >= t.song.Tracks[track].NumVoices {
					curVoices[track] = 0
				}
				notes = append(notes, Note{first + curVoices[track], note})
			}
		}
		t.playRowPatMutex.Unlock()
		t.ticked <- struct{}{}
		return notes, true
	})
	buffer := make([]float32, 8192)
	for {
		select {
		case <-closer:
			return
		default:
			t.sequencer.ReadAudio(buffer)
			output.WriteAudio(buffer)
		}
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
		t.sequencer.SetRowLength(44100 * 60 / (4 * t.song.BPM))
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
	t.SaveUndo()
	if t.song.Patch.TotalVoices() < 32 {
		units := make([]sointu.Unit, len(defaultInstrument.Units))
		for i, defUnit := range defaultInstrument.Units {
			units[i].Type = defUnit.Type
			units[i].Parameters = make(map[string]int)
			for k, v := range defUnit.Parameters {
				units[i].Parameters[k] = v
			}
		}
		t.song.Patch.Instruments = append(t.song.Patch.Instruments, sointu.Instrument{
			NumVoices: defaultInstrument.NumVoices,
			Units:     units,
		})
	}
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
	}
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

func New(audioContext sointu.AudioContext) *Tracker {
	t := &Tracker{
		Theme:                 material.NewTheme(gofont.Collection()),
		QuitButton:            new(widget.Clickable),
		audioContext:          audioContext,
		BPM:                   new(NumberInput),
		Octave:                new(NumberInput),
		SongLength:            new(NumberInput),
		NewTrackBtn:           new(widget.Clickable),
		NewInstrumentBtn:      new(widget.Clickable),
		DeleteInstrumentBtn:   new(widget.Clickable),
		NewSongFileBtn:        new(widget.Clickable),
		LoadSongFileBtn:       new(widget.Clickable),
		SaveSongFileBtn:       new(widget.Clickable),
		AddSemitoneBtn:        new(widget.Clickable),
		SubtractSemitoneBtn:   new(widget.Clickable),
		AddOctaveBtn:          new(widget.Clickable),
		SubtractOctaveBtn:     new(widget.Clickable),
		setPlaying:            make(chan bool),
		rowJump:               make(chan int),
		patternJump:           make(chan int),
		ticked:                make(chan struct{}),
		closer:                make(chan struct{}),
		undoStack:             []sointu.Song{},
		redoStack:             []sointu.Song{},
		InstrumentList:        &layout.List{Axis: layout.Horizontal},
		TopHorizontalSplit:    new(Split),
		BottomHorizontalSplit: new(Split),
		VerticalSplit:         new(Split),
	}
	t.Octave.Value = 4
	t.VerticalSplit.Axis = layout.Vertical
	t.BottomHorizontalSplit.Ratio = -.5
	t.Theme.Color.Primary = primaryColor
	t.Theme.Color.InvText = black
	go t.sequencerLoop(t.closer)
	if err := t.LoadSong(defaultSong); err != nil {
		panic(fmt.Errorf("cannot load default song: %w", err))
	}
	return t
}
