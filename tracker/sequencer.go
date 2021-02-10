package tracker

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"

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
	mutex      sync.Mutex
	synth      sointu.Synth
	validSynth int32
	service    sointu.SynthService
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

func NewSequencer(service sointu.SynthService, iterator func() ([]Note, bool)) *Sequencer {
	return &Sequencer{
		service:   service,
		iterator:  iterator,
		rowLength: math.MaxInt32,
		rowTime:   math.MaxInt32,
	}
}

func (s *Sequencer) ReadAudio(buffer []float32) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	totalRendered := 0
	for i := 0; i < SEQUENCER_MAX_READ_TRIES; i++ {
		gotRow := true
		if s.rowTime >= s.rowLength {
			var row []Note
			s.mutex.Unlock()
			row, gotRow = s.iterator()
			s.mutex.Lock()
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
		if s.Enabled() {
			rendered, timeAdvanced, err := s.synth.Render(buffer[totalRendered*2:], rowTimeRemaining)
			if err != nil {
				s.Disable()
			}
			totalRendered += rendered
			s.rowTime += timeAdvanced
		} else {
			for totalRendered*2 < len(buffer) && rowTimeRemaining > 0 {
				buffer[totalRendered*2] = 0
				buffer[totalRendered*2+1] = 0
				totalRendered++
				s.rowTime++
				rowTimeRemaining--
			}
		}
		if totalRendered*2 >= len(buffer) {
			return totalRendered * 2, nil
		}
	}
	return totalRendered * 2, fmt.Errorf("despite %v attempts, Sequencer.ReadAudio could not fill the buffer (rowLength was %v, should be >> 0)", SEQUENCER_MAX_READ_TRIES, s.rowLength)
}

// Updates the patch of the synth
func (s *Sequencer) SetPatch(patch sointu.Patch) {
	s.mutex.Lock()
	var err error
	if s.Enabled() {
		err = s.synth.Update(patch)
	} else {
		s.synth, err = s.service.Compile(patch)
	}
	if err == nil {
		atomic.StoreInt32(&s.validSynth, 1)
	}
	s.mutex.Unlock()
}

func (s *Sequencer) Enabled() bool {
	return atomic.LoadInt32(&s.validSynth) == 1
}

func (s *Sequencer) Disable() {
	atomic.StoreInt32(&s.validSynth, 0)
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
	if s.synth != nil {
		if note == 0 {
			s.synth.Release(voice)
		} else {
			s.synth.Trigger(voice, note)
		}
	}
}
