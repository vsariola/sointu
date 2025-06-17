package tracker

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
)

type (
	// Broker is the centralized message broker for the tracker. It is used to
	// communicate between the player, the model, and the loudness detector. At
	// the moment, the broker is just many-to-one communication, implemented
	// with one channel for each recipient. Additionally, the broker has a
	// sync.pool for *sointu.AudioBuffers, from which the player can get and
	// return buffers to pass buffers around without allocating new memory every
	// time. We can later consider making many-to-many types of communication
	// and more complex routing logic to the Broker if needed.
	//
	// For closing goroutines, the broker has two channels for each goroutine:
	// CloseXXX and FinishedXXX. The CloseXXX channel has a capacity of 1, so
	// you can always send a empty message (struct{}{}) to it without blocking.
	// If the channel is already full, that means someone else has already
	// requested its closure and the goroutine is already closing, so dropping
	// the message is fine. Then, FinishedXXX is used to signal that a goroutine
	// has succesfully closed and cleaned up. Nothing is ever sent to the
	// channel, it is only closed. You can wait until the goroutines is done
	// closing with "<- FinishedXXX", which for avoiding deadlocks can be
	// combined with a timeout:
	//    select {
	//      case <-FinishedXXX:
	//      case <-time.After(3 * time.Second):
	//    }

	Broker struct {
		ToModel    chan MsgToModel
		ToPlayer   chan any // TODO: consider using a sum type here, for a bit more type safety. See: https://www.jerf.org/iri/post/2917/
		ToDetector chan MsgToDetector
		ToGUI      chan any

		CloseDetector chan struct{}
		CloseGUI      chan struct{}

		FinishedGUI      chan struct{}
		FinishedDetector chan struct{}

		// mIDIEventsToGUI is true if all MIDI events should be sent to the GUI,
		// for inputting notes to tracks. If false, they should be sent to the
		// player instead.
		mIDIEventsToGUI atomic.Bool

		bufferPool sync.Pool
	}

	// MsgToModel is a message sent to the model. The most often sent data
	// (Panic, SongPosition, VoiceLevels and DetectorResult) are not boxed to
	// avoid allocations. All the infrequently passed messages can be boxed &
	// cast to any; casting pointer types to any is cheap (does not allocate).
	MsgToModel struct {
		HasPanicPosLevels bool
		Panic             bool
		SongPosition      sointu.SongPos
		VoiceLevels       [vm.MAX_VOICES]float32
		CPULoad           float64

		HasDetectorResult bool
		DetectorResult    DetectorResult

		TriggerChannel int  // note: 0 = no trigger, 1 = first channel, etc.
		Reset          bool // true: playing started, so should reset the detector and the scope cursor

		Data any // TODO: consider using a sum type here, for a bit more type safety. See: https://www.jerf.org/iri/post/2917/
	}

	// MsgToDetector is a message sent to the detector. It contains a reset flag
	// and a data field. The data field can contain many different messages,
	// including *sointu.AudioBuffer for the detector to analyze and func()
	// which gets executed in the detector goroutine.
	MsgToDetector struct {
		Reset bool
		Data  any // TODO: consider using a sum type here, for a bit more type safety. See: https://www.jerf.org/iri/post/2917/

		WeightingType    WeightingType
		HasWeightingType bool
		Oversampling     bool
		HasOversampling  bool
	}

	MsgToGUI struct {
		Kind  GUIMessageKind
		Param int
	}

	GUIMessageKind int
)

const (
	GUIMessageKindNone GUIMessageKind = iota
	GUIMessageCenterOnRow
	GUIMessageEnsureCursorVisible
)

func NewBroker() *Broker {
	return &Broker{
		ToPlayer:         make(chan interface{}, 1024),
		ToModel:          make(chan MsgToModel, 1024),
		ToDetector:       make(chan MsgToDetector, 1024),
		ToGUI:            make(chan any, 1024),
		CloseDetector:    make(chan struct{}, 1),
		CloseGUI:         make(chan struct{}, 1),
		FinishedGUI:      make(chan struct{}),
		FinishedDetector: make(chan struct{}),
		bufferPool:       sync.Pool{New: func() interface{} { return &sointu.AudioBuffer{} }},
	}
}

func (b *Broker) MIDIChannel() chan<- any {
	if b.mIDIEventsToGUI.Load() {
		return b.ToGUI
	}
	return b.ToPlayer
}

// GetAudioBuffer returns an audio buffer from the buffer pool. The buffer is
// guaranteed to be empty. After using the buffer, it should be returned to the
// pool with PutAudioBuffer.
func (b *Broker) GetAudioBuffer() *sointu.AudioBuffer {
	return b.bufferPool.Get().(*sointu.AudioBuffer)
}

// PutAudioBuffer returns an audio buffer to the buffer pool. If the buffer is
// not empty, its length is resetted (but capacity kept) before returning it to
// the pool.
func (b *Broker) PutAudioBuffer(buf *sointu.AudioBuffer) {
	if len(*buf) > 0 {
		*buf = (*buf)[:0]
	}
	b.bufferPool.Put(buf)
}

// TrySend is a helper function to send a value to a channel if it is not full.
// It is guaranteed to be non-blocking. Return true if the value was sent, false
// otherwise.
func TrySend[T any](c chan<- T, v T) bool {
	select {
	case c <- v:
	default:
		return false
	}
	return true
}

// TimeoutReceive is a helper function to block until a value is received from a
// channel, or timing out after t. ok will be false if the timeout occurred or
// if the channel is closed.
func TimeoutReceive[T any](c <-chan T, t time.Duration) (v T, ok bool) {
	select {
	case v, ok = <-c:
		return v, ok
	case <-time.After(t):
		return v, false
	}
}
