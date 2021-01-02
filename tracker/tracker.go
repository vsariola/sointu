package tracker

import (
	"fmt"
	"sync"

	"gioui.org/widget"
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/bridge"
)

type Tracker struct {
	QuitButton    *widget.Clickable
	songPlayMutex sync.RWMutex // protects song and playing
	song          sointu.Song
	Playing       bool
	// protects PlayPattern and PlayRow
	playRowPatMutex sync.RWMutex // protects song and playing
	PlayPattern     int
	PlayRow         int
	CursorRow       int
	CursorColumn    int
	DisplayPattern  int
	ActiveTrack     int
	CurrentOctave   byte
	NoteTracking    bool

	ticked       chan struct{}
	setPlaying   chan bool
	rowJump      chan int
	patternJump  chan int
	audioContext sointu.AudioContext
	synth        sointu.Synth
	playBuffer   []float32
	closer       chan struct{}
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
	sequencer := NewSequencer(synth, 44100*60/(4*t.song.BPM), func() ([]Note, bool) {
		t.songPlayMutex.RLock()
		defer t.songPlayMutex.RUnlock()
		if !t.Playing {
			return nil, false
		}
		t.playRowPatMutex.Lock()
		defer t.playRowPatMutex.Unlock()
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
			note := t.song.Patterns[patternIndex][t.PlayRow]
			if note == 1 { // anything but hold causes an action.
				continue
			}
			notes = append(notes, Note{curVoices[track], 0})
			if note > 1 {
				curVoices[track]++
				first := t.song.FirstTrackVoice(track)
				if curVoices[track] >= first+t.song.Tracks[track].NumVoices {
					curVoices[track] = first
				}
				notes = append(notes, Note{curVoices[track], note})
			}
		}
		t.ticked <- struct{}{}
		return notes, true
	})
	buffer := make([]float32, 8192)
	for {
		select {
		case <-closer:
			return
		default:
			sequencer.ReadAudio(buffer)
			output.WriteAudio(buffer)
		}
	}
}

func New(audioContext sointu.AudioContext) *Tracker {
	t := &Tracker{
		QuitButton:    new(widget.Clickable),
		CurrentOctave: 4,
		audioContext:  audioContext,
		setPlaying:    make(chan bool),
		rowJump:       make(chan int),
		patternJump:   make(chan int),
		ticked:        make(chan struct{}),
		closer:        make(chan struct{}),
	}
	go t.sequencerLoop(t.closer)
	if err := t.LoadSong(defaultSong); err != nil {
		panic(fmt.Errorf("cannot load default song: %w", err))
	}
	return t
}
