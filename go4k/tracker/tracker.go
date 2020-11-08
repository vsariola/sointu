package tracker

import (
	"fmt"
	"gioui.org/widget"
	"github.com/vsariola/sointu/go4k"
	"github.com/vsariola/sointu/go4k/audio"
	"github.com/vsariola/sointu/go4k/bridge"
)

type Tracker struct {
	QuitButton     *widget.Clickable
	song           go4k.Song
	CursorRow      int
	CursorColumn   int
	DisplayPattern int
	PlayPattern    int32
	PlayRow        int32
	ActiveTrack    int
	CurrentOctave  byte
	Playing        bool
	ticked         chan struct{}
	setPlaying     chan bool
	rowJump        chan int
	patternJump    chan int
	player         audio.Player
	synth          go4k.Synth
	playBuffer     []float32
	closer         chan struct{}
}

func (t *Tracker) LoadSong(song go4k.Song) error {
	if err := song.Validate(); err != nil {
		return fmt.Errorf("invalid song: %w", err)
	}
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
	t.player.Close()
	t.closer <- struct{}{}
}

func New(player audio.Player) *Tracker {
	t := &Tracker{
		QuitButton:    new(widget.Clickable),
		CurrentOctave: 4,
		player:        player,
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
