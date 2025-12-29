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

func NewScopeModel(bpm int) *ScopeModel {
	s := &ScopeModel{
		bpm:           bpm,
		lengthInBeats: 4,
	}
	s.updateBufferLength()
	return s
}

func (s *ScopeModel) Waveform() RingBuffer[[2]float32] { return s.waveForm }

func (s *ScopeModel) Once() Bool          { return MakeEnabledBool((*SignalOnce)(s)) }
func (s *ScopeModel) Wrap() Bool          { return MakeEnabledBool((*SignalWrap)(s)) }
func (s *ScopeModel) LengthInBeats() Int  { return MakeInt((*SignalLengthInBeats)(s)) }
func (s *ScopeModel) TriggerChannel() Int { return MakeInt((*TriggerChannel)(s)) }

func (m *SignalOnce) Value() bool       { return m.once }
func (m *SignalOnce) SetValue(val bool) { m.once = val }

func (m *SignalWrap) Value() bool       { return m.wrap }
func (m *SignalWrap) SetValue(val bool) { m.wrap = val }

func (m *SignalLengthInBeats) Value() int { return m.lengthInBeats }
func (m *SignalLengthInBeats) SetValue(val int) bool {
	m.lengthInBeats = val
	(*ScopeModel)(m).updateBufferLength()
	return true
}
func (m *SignalLengthInBeats) Range() IntRange { return IntRange{1, 999} }

func (m *TriggerChannel) Value() int            { return m.triggerChannel }
func (m *TriggerChannel) SetValue(val int) bool { m.triggerChannel = val; return true }
func (m *TriggerChannel) Range() IntRange       { return IntRange{0, vm.MAX_VOICES} }

func (s *ScopeModel) ProcessAudioBuffer(bufPtr *sointu.AudioBuffer) {
	if s.wrap {
		s.waveForm.WriteWrap(*bufPtr)
	} else {
		s.waveForm.WriteOnce(*bufPtr)
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
