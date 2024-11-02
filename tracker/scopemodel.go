package tracker

import (
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
)

type (
	ScopeModel struct {
		waveForm       RingBuffer[[2]float32]
		once           bool
		triggered      bool
		wrap           bool
		triggerChannel int
		lengthInBeats  int
		bpm            int

		broker *Broker
	}

	RingBuffer[T any] struct {
		Buffer []T
		Cursor int
	}

	SignalOnce          ScopeModel
	SignalWrap          ScopeModel
	SignalLengthInBeats ScopeModel
	TriggerChannel      ScopeModel
)

func (r *RingBuffer[T]) WriteWrap(values []T) {
	r.Cursor = (r.Cursor + len(values)) % len(r.Buffer)
	a := min(len(values), r.Cursor)                 // how many values to copy before the cursor
	b := min(len(values)-a, len(r.Buffer)-r.Cursor) // how many values to copy to the end of the buffer
	copy(r.Buffer[r.Cursor-a:r.Cursor], values[len(values)-a:])
	copy(r.Buffer[len(r.Buffer)-b:], values[len(values)-a-b:])
}

func (r *RingBuffer[T]) WriteWrapSingle(value T) {
	r.Cursor = (r.Cursor + 1) % len(r.Buffer)
	r.Buffer[r.Cursor] = value
}

func (r *RingBuffer[T]) WriteOnce(values []T) {
	if r.Cursor < len(r.Buffer) {
		r.Cursor += copy(r.Buffer[r.Cursor:], values)
	}
}

func (r *RingBuffer[T]) WriteOnceSingle(value T) {
	if r.Cursor < len(r.Buffer) {
		r.Buffer[r.Cursor] = value
		r.Cursor++
	}
}

func NewScopeModel(broker *Broker, bpm int) *ScopeModel {
	s := &ScopeModel{
		broker:        broker,
		bpm:           bpm,
		lengthInBeats: 4,
	}
	s.updateBufferLength()
	return s
}

func (s *ScopeModel) Waveform() RingBuffer[[2]float32] { return s.waveForm }

func (s *ScopeModel) Once() *SignalOnce                   { return (*SignalOnce)(s) }
func (s *ScopeModel) Wrap() *SignalWrap                   { return (*SignalWrap)(s) }
func (s *ScopeModel) LengthInBeats() *SignalLengthInBeats { return (*SignalLengthInBeats)(s) }
func (s *ScopeModel) TriggerChannel() *TriggerChannel     { return (*TriggerChannel)(s) }

func (m *SignalOnce) Bool() Bool        { return Bool{m} }
func (m *SignalOnce) Value() bool       { return m.once }
func (m *SignalOnce) setValue(val bool) { m.once = val }
func (m *SignalOnce) Enabled() bool     { return true }

func (m *SignalWrap) Bool() Bool        { return Bool{m} }
func (m *SignalWrap) Value() bool       { return m.wrap }
func (m *SignalWrap) setValue(val bool) { m.wrap = val }
func (m *SignalWrap) Enabled() bool     { return true }

func (m *SignalLengthInBeats) Int() Int   { return Int{m} }
func (m *SignalLengthInBeats) Value() int { return m.lengthInBeats }
func (m *SignalLengthInBeats) setValue(val int) {
	m.lengthInBeats = val
	(*ScopeModel)(m).updateBufferLength()
}
func (m *SignalLengthInBeats) Enabled() bool        { return true }
func (m *SignalLengthInBeats) Range() intRange      { return intRange{1, 999} }
func (m *SignalLengthInBeats) change(string) func() { return func() {} }

func (m *TriggerChannel) Int() Int             { return Int{m} }
func (m *TriggerChannel) Value() int           { return m.triggerChannel }
func (m *TriggerChannel) setValue(val int)     { m.triggerChannel = val }
func (m *TriggerChannel) Enabled() bool        { return true }
func (m *TriggerChannel) Range() intRange      { return intRange{0, vm.MAX_VOICES} }
func (m *TriggerChannel) change(string) func() { return func() {} }

func (s *ScopeModel) ProcessAudioBuffer(bufPtr *sointu.AudioBuffer) {
	if s.wrap {
		s.waveForm.WriteWrap(*bufPtr)
	} else {
		s.waveForm.WriteOnce(*bufPtr)
	}
	// chain the messages: when we have a new audio buffer, try passing it on to the detector
	if !trySend(s.broker.ToDetector, MsgToDetector{Data: bufPtr}) {
		s.broker.PutAudioBuffer(bufPtr)
	}
}

// Note: channel 1 is the first channel
func (s *ScopeModel) Trigger(channel int) {
	if s.triggerChannel > 0 && channel == s.triggerChannel && !(s.once && s.triggered) {
		s.waveForm.Cursor = 0
		s.triggered = true
	}
}

func (s *ScopeModel) Reset() {
	s.waveForm.Cursor = 0
	s.triggered = false
	l := len(s.waveForm.Buffer)
	s.waveForm.Buffer = s.waveForm.Buffer[:0]
	s.waveForm.Buffer = append(s.waveForm.Buffer, make([][2]float32, l)...)
	trySend(s.broker.ToDetector, MsgToDetector{Reset: true}) // chain the messages: when the signal analyzer is reset, also reset the detector
}

func (s *ScopeModel) SetBpm(bpm int) {
	s.bpm = bpm
	s.updateBufferLength()
}

func (s *ScopeModel) updateBufferLength() {
	if s.bpm == 0 || s.lengthInBeats == 0 {
		return
	}
	setSliceLength(&s.waveForm.Buffer, 44100*60*s.lengthInBeats/s.bpm)
}
