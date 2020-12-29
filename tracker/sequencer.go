package tracker

import (
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/vsariola/sointu"
)

// how many times the sequencer tries to fill the buffer. If the buffer is not
// filled after this many tries, there's probably an issue with rowlength (e.g.
// infinite BPM, rowlength = 0) or something else, so we error instead of
// letting ReadAudio hang.
const SEQUENCER_MAX_READ_TRIES = 1000

// Sequencer is a AudioSource that uses the given synth to render audio. In
// periods of rowLength, it pulls new notes to trigger/release from the given
// iterator. Note that the iterator should be thread safe, as the ReadAudio
// might be called from another go routine.
type Sequencer struct {
	// we use mutex to ensure that voices are not triggered during readaudio or
	// that the synth is not changed when audio is being read
	mutex sync.Mutex
	synth sointu.Synth
	// this iterator is a bit unconventional in the sense that it might return
	// hasNext false, but might still return hasNext true in future attempts if
	// new rows become available.
	iterator  func() ([]Note, bool)
	rowTime   int
	rowLength int
}

type Note struct {
	Voice int
	Note  byte
}

func NewSequencer(synth sointu.Synth, rowLength int, iterator func() ([]Note, bool)) *Sequencer {
	return &Sequencer{
		synth:     synth,
		iterator:  iterator,
		rowLength: rowLength,
		rowTime:   math.MaxInt32,
	}
}

func (s *Sequencer) ReadAudio(buffer []float32) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.synth == nil {
		return 0, errors.New("cannot Sequencer.ReadAudio; synth is nil")
	}
	totalRendered := 0
	for i := 0; i < SEQUENCER_MAX_READ_TRIES; i++ {
		gotRow := true
		if s.rowTime >= s.rowLength {
			var row []Note
			row, gotRow = s.iterator()
			if gotRow {
				for _, n := range row {
					s.doNote(n.Voice, n.Note)
				}
				s.rowTime = 0
			} else {
				for i := 0; i < 32; i++ {
					s.doNote(i, 0)
				}
			}
		}
		rowTimeRemaining := s.rowLength - s.rowTime
		if !gotRow {
			rowTimeRemaining = math.MaxInt32
		}
		rendered, timeAdvanced, err := s.synth.Render(buffer[totalRendered*2:], rowTimeRemaining)
		totalRendered += rendered
		s.rowTime += timeAdvanced
		if err != nil {
			return totalRendered * 2, fmt.Errorf("synth.Render failed: %v", err)
		}
		if totalRendered*2 >= len(buffer) {
			return totalRendered * 2, nil
		}
	}
	return totalRendered * 2, fmt.Errorf("despite %v attempts, Sequencer.ReadAudio could not fill the buffer (rowLength was %v, should be >> 0)", SEQUENCER_MAX_READ_TRIES, s.rowLength)
}

// Sets the synth used by the sequencer. This takes ownership of the synth: the
// synth should not be called by anyone else than the sequencer afterwards
func (s *Sequencer) SetSynth(synth sointu.Synth) {
	s.mutex.Lock()
	s.synth = synth
	s.mutex.Unlock()
}

func (s *Sequencer) SetRowLength(rowLength int) {
	s.mutex.Lock()
	s.rowLength = rowLength
	s.mutex.Unlock()
}

// Trigger is used to manually play a note on the sequencer when jamming. It is
// thread-safe.
func (s *Sequencer) Trigger(voice int, note byte) {
	s.mutex.Lock()
	s.doNote(voice, note)
	s.mutex.Unlock()
}

// Release is used to manually release a note on the sequencer when jamming. It
// is thread-safe.
func (s *Sequencer) Release(voice int) {
	s.Trigger(voice, 0)
}

// doNote is the internal trigger/release function that is not thread safe
func (s *Sequencer) doNote(voice int, note byte) {
	if note == 0 {
		s.synth.Release(voice)
	} else {
		s.synth.Trigger(voice, note)
	}
}
