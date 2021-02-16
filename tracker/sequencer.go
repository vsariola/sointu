package tracker

import (
	"math"
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
	validSynth    int32
	closer        chan struct{}
	setPatch      chan sointu.Patch
	setRowLength  chan int
	noteOn        chan noteOnEvent
	noteOff       chan uint32
	synth         sointu.Synth
	voiceNoteID   []uint32
	voiceReleased []bool
	idCounter     uint32
}

type RowNote struct {
	NumVoices int
	Note      byte
}

type noteOnEvent struct {
	voiceStart int
	voiceEnd   int
	note       byte
	id         uint32
}

type noteID struct {
	voice int
	id    uint32
}

func NewSequencer(bufferSize int, service sointu.SynthService, context sointu.AudioContext, callBack func([]float32), iterator func([]RowNote) []RowNote) *Sequencer {
	ret := &Sequencer{
		closer:        make(chan struct{}),
		setPatch:      make(chan sointu.Patch, 32),
		setRowLength:  make(chan int, 32),
		noteOn:        make(chan noteOnEvent, 32),
		noteOff:       make(chan uint32, 32),
		voiceNoteID:   make([]uint32, 32),
		voiceReleased: make([]bool, 32),
	}
	// the iterator is a bit unconventional in the sense that it might return
	// false to indicate that there is no row available, but might still return
	// true in future attempts if new rows become available.
	go ret.loop(bufferSize, service, context, callBack, iterator)
	return ret
}

func (s *Sequencer) loop(bufferSize int, service sointu.SynthService, context sointu.AudioContext, callBack func([]float32), iterator func([]RowNote) []RowNote) {
	buffer := make([]float32, bufferSize)
	renderTries := 0
	audioOut := context.Output()
	defer audioOut.Close()
	rowIn := make([]RowNote, 32)
	rowLength := math.MaxInt32
	rowTimeRemaining := 0
	trackNotes := make([]uint32, 32)
	for {
		for !s.Enabled() {
			select {
			case <-s.closer:
				return
			case <-s.noteOn:
			case <-s.noteOff:
			case rowLength = <-s.setRowLength:
			case patch := <-s.setPatch:
				var err error
				s.synth, err = service.Compile(patch)
				if err == nil {
					s.enable()
					for i := range s.voiceReleased {
						s.voiceReleased[i] = true
						s.synth.Release(i)
					}
					break
				}
			}
		}
		released := false
		for s.Enabled() {
			select {
			case <-s.closer:
				return
			case n := <-s.noteOn:
				s.trigger(n.voiceStart, n.voiceEnd, n.note, n.id)
			case n := <-s.noteOff:
				s.release(n)
			case rowLength = <-s.setRowLength:
			case patch := <-s.setPatch:
				err := s.synth.Update(patch)
				if err != nil {
					s.Disable()
					break
				}
			default:
				renderTime := rowTimeRemaining
				if rowTimeRemaining <= 0 {
					rowOut := iterator(rowIn[:0])
					if len(rowOut) > 0 {
						curVoice := 0
						for i, rn := range rowOut {
							end := curVoice + rn.NumVoices
							if rn.Note != 1 {
								s.release(trackNotes[i])
							}
							if rn.Note > 1 {
								id := s.getNewID()
								s.trigger(curVoice, end, rn.Note, id)
								trackNotes[i] = id
							}
							curVoice = end
						}
						rowTimeRemaining = rowLength
						renderTime = rowLength
						released = false
					} else {
						if !released {
							s.releaseVoiceRange(0, len(s.voiceNoteID))
							released = true
						}
						rowTimeRemaining = 0
						renderTime = math.MaxInt32
					}
				}
				rendered, timeAdvanced, err := s.synth.Render(buffer, renderTime)
				callBack(buffer)
				if err != nil {
					s.Disable()
					break
				}
				rowTimeRemaining -= timeAdvanced
				if timeAdvanced == 0 {
					renderTries++
				} else {
					renderTries = 0
				}
				if renderTries >= SEQUENCER_MAX_READ_TRIES {
					s.Disable()
					break
				}
				err = audioOut.WriteAudio(buffer[:2*rendered])
				if err != nil {
					s.Disable()
					break
				}
			}
		}
	}
}

func (s *Sequencer) Enabled() bool {
	return atomic.LoadInt32(&s.validSynth) == 1
}

func (s *Sequencer) Disable() {
	atomic.StoreInt32(&s.validSynth, 0)
}

func (s *Sequencer) SetRowLength(rowLength int) {
	s.setRowLength <- rowLength
}

// Close closes the sequencer and releases all its resources
func (s *Sequencer) Close() {
	s.closer <- struct{}{}
}

// SetPatch updates the synth to match given patch
func (s *Sequencer) SetPatch(patch sointu.Patch) {
	s.setPatch <- patch.Copy()
}

// Trigger is used to manually play a note on the sequencer when jamming. It is
// thread-safe. It starts to play one of the voice in the range voiceStart
// (inclusive) and voiceEnd (exclusive). It returns a release function that can
// be called to release the voice playing the note (in case the voice has not
// been captured by someone else already). Note that Trigger will never block,
// but calling the release function might block until the sequencer has been
// able to assign a voice to the note.
func (s *Sequencer) Trigger(voiceStart, voiceEnd int, note byte) func() {
	if note <= 1 {
		return func() {}
	}
	id := s.getNewID()
	e := noteOnEvent{
		voiceStart: voiceStart,
		voiceEnd:   voiceEnd,
		note:       note,
		id:         id,
	}
	s.noteOn <- e
	return func() {
		s.noteOff <- id // now, tell the sequencer to stop it
	}
}

func (s *Sequencer) getNewID() uint32 {
	return atomic.AddUint32(&s.idCounter, 1)
}

func (s *Sequencer) enable() {
	atomic.StoreInt32(&s.validSynth, 1)
}

func (s *Sequencer) trigger(voiceStart, voiceEnd int, note byte, newID uint32) {
	if !s.Enabled() {
		return
	}
	var oldestID uint32 = math.MaxUint32
	oldestReleased := false
	oldestVoice := 0
	for i := voiceStart; i < voiceEnd; i++ {
		// find a suitable voice to trigger. if the voice has been released,
		// then we prefer to trigger that over a voice that is still playing. in
		// case two voices are both playing or or both are released, we prefer
		// the older one
		id := s.voiceNoteID[i]
		isReleased := s.voiceReleased[i]
		if id < oldestID && (oldestReleased == isReleased) || (!oldestReleased && isReleased) {
			oldestVoice = i
			oldestID = id
			oldestReleased = isReleased
		}
	}
	s.voiceNoteID[oldestVoice] = newID
	s.voiceReleased[oldestVoice] = false
	s.synth.Trigger(oldestVoice, note)
}

func (s *Sequencer) release(id uint32) {
	if !s.Enabled() {
		return
	}
	for i := 0; i < len(s.voiceNoteID); i++ {
		if s.voiceNoteID[i] == id && !s.voiceReleased[i] {
			s.voiceReleased[i] = true
			s.synth.Release(i)
			return
		}
	}
}

func (s *Sequencer) releaseVoiceRange(voiceStart, voiceEnd int) {
	if !s.Enabled() {
		return
	}
	for i := voiceStart; i < voiceEnd; i++ {
		if !s.voiceReleased[i] {
			s.voiceReleased[i] = true
			s.synth.Release(i)
		}
	}
}
