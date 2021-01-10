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
	playRowPatMutex   sync.RWMutex // protects song and playing
	PlayPattern       int
	PlayRow           int
	CursorRow         int
	CursorColumn      int
	DisplayPattern    int
	ActiveTrack       int
	CurrentInstrument int
	CurrentUnit       int
	CurrentOctave     byte
	NoteTracking      bool
	Theme             *material.Theme
	OctaveUpBtn       *widget.Clickable
	OctaveDownBtn     *widget.Clickable
	BPMUpBtn          *widget.Clickable
	BPMDownBtn        *widget.Clickable
	NewTrackBtn       *widget.Clickable
	NewInstrumentBtn  *widget.Clickable
	LoadSongFileBtn   *widget.Clickable
	SaveSongFileBtn   *widget.Clickable
	ParameterSliders  []*widget.Float
	UnitBtns          []*widget.Clickable
	InstrumentBtns    []*widget.Clickable
	InstrumentList    *layout.List

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
	if synth, err := bridge.Synth(song.Patch); err != nil {
		fmt.Printf("error loading synth: %v\n", err)
		t.synth = nil
	} else {
		t.synth = synth
	}
	if t.DisplayPattern >= song.SequenceLength() {
		t.DisplayPattern = song.SequenceLength() - 1
	}
	if t.CursorRow >= song.PatternRows() {
		t.CursorRow = song.PatternRows() - 1
	}
	if t.PlayPattern >= song.SequenceLength() {
		t.PlayPattern = song.SequenceLength() - 1
	}
	if t.PlayRow >= song.PatternRows() {
		t.PlayRow = song.PatternRows() - 1
	}
	if t.ActiveTrack >= len(song.Tracks) {
		t.ActiveTrack = len(song.Tracks) - 1
	}
	if t.sequencer != nil {
		t.sequencer.SetSynth(t.synth)
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
		t.PlayPattern = t.DisplayPattern
		t.PlayRow = t.CursorRow - 1
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
		t.PlayRow++
		if t.PlayRow >= t.song.PatternRows() {
			t.PlayRow = 0
			t.PlayPattern++
		}
		if t.PlayPattern >= t.song.SequenceLength() {
			t.PlayPattern = 0
		}
		if t.NoteTracking {
			t.DisplayPattern = t.PlayPattern
			t.CursorRow = t.PlayRow
		}
		notes := make([]Note, 0, 32)
		for track := range t.song.Tracks {
			patternIndex := t.song.Tracks[track].Sequence[t.PlayPattern]
			note := t.song.Tracks[track].Patterns[patternIndex][t.PlayRow]
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
	newOctave := int(t.CurrentOctave) + delta
	if newOctave < 0 {
		newOctave = 0
	}
	if newOctave > 9 {
		newOctave = 9
	}
	if newOctave != int(t.CurrentOctave) {
		t.CurrentOctave = byte(newOctave)
		return true
	}
	return false
}

func (t *Tracker) ChangeBPM(delta int) bool {
	t.SaveUndo()
	newBPM := t.song.BPM + delta
	if newBPM < 1 {
		newBPM = 1
	}
	if newBPM > 999 {
		newBPM = 999
	}
	if newBPM != int(t.song.BPM) {
		t.song.BPM = newBPM
		t.sequencer.SetRowLength(44100 * 60 / (4 * t.song.BPM))
		return true
	}
	return false
}

func (t *Tracker) AddTrack() {
	t.SaveUndo()
	if t.song.TotalTrackVoices() < t.song.Patch.TotalVoices() {
		seq := make([]byte, t.song.SequenceLength())
		patterns := [][]byte{make([]byte, t.song.PatternRows())}
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
	synth, err := bridge.Synth(t.song.Patch)
	if err == nil {
		t.sequencer.SetSynth(synth)
	} else {
		fmt.Printf("%v", err)
	}
}

// SetCurrentNote sets the (note) value in current pattern under cursor to iv
func (t *Tracker) SetCurrentNote(iv byte) {
	t.SaveUndo()
	t.song.Tracks[t.ActiveTrack].Patterns[t.song.Tracks[t.ActiveTrack].Sequence[t.DisplayPattern]][t.CursorRow] = iv
}

func (t *Tracker) SetCurrentPattern(pat byte) {
	t.SaveUndo()
	length := len(t.song.Tracks[t.ActiveTrack].Patterns)
	if int(pat) >= length {
		tail := make([][]byte, int(pat)-length+1)
		for i := range tail {
			tail[i] = make([]byte, t.song.PatternRows())
		}
		t.song.Tracks[t.ActiveTrack].Patterns = append(t.song.Tracks[t.ActiveTrack].Patterns, tail...)
	}
	t.song.Tracks[t.ActiveTrack].Sequence[t.DisplayPattern] = pat
}

func New(audioContext sointu.AudioContext) *Tracker {
	t := &Tracker{
		Theme:            material.NewTheme(gofont.Collection()),
		QuitButton:       new(widget.Clickable),
		CurrentOctave:    4,
		audioContext:     audioContext,
		OctaveUpBtn:      new(widget.Clickable),
		OctaveDownBtn:    new(widget.Clickable),
		BPMUpBtn:         new(widget.Clickable),
		BPMDownBtn:       new(widget.Clickable),
		NewTrackBtn:      new(widget.Clickable),
		NewInstrumentBtn: new(widget.Clickable),
		LoadSongFileBtn:  new(widget.Clickable),
		SaveSongFileBtn:  new(widget.Clickable),
		setPlaying:       make(chan bool),
		rowJump:          make(chan int),
		patternJump:      make(chan int),
		ticked:           make(chan struct{}),
		closer:           make(chan struct{}),
		undoStack:        []sointu.Song{},
		redoStack:        []sointu.Song{},
		InstrumentList:   &layout.List{Axis: layout.Horizontal},
	}
	t.Theme.Color.Primary = primaryColor
	t.Theme.Color.InvText = black
	go t.sequencerLoop(t.closer)
	if err := t.LoadSong(defaultSong); err != nil {
		panic(fmt.Errorf("cannot load default song: %w", err))
	}
	return t
}
