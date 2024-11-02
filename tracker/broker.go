package tracker

import (
	"sync"

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
	Broker struct {
		ToModel    chan MsgToModel
		ToPlayer   chan any // TODO: consider using a sum type here, for a bit more type safety. See: https://www.jerf.org/iri/post/2917/
		ToDetector chan MsgToDetector

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
		Quit  bool
		Data  any // TODO: consider using a sum type here, for a bit more type safety. See: https://www.jerf.org/iri/post/2917/
	}
)

func NewBroker() *Broker {
	return &Broker{
		ToPlayer:   make(chan interface{}, 1024),
		ToModel:    make(chan MsgToModel, 1024),
		ToDetector: make(chan MsgToDetector, 1024),
		bufferPool: sync.Pool{New: func() interface{} { return &sointu.AudioBuffer{} }},
	}
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

// trySend is a helper function to send a value to a channel if it is not full.
// It is guaranteed to be non-blocking. Return true if the value was sent, false
// otherwise.
func trySend[T any](c chan<- T, v T) bool {
	select {
	case c <- v:
	default:
		return false
	}
	return true
}
