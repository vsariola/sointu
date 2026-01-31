package tracker

import (
	"math"
	"strconv"

	"github.com/vsariola/sointu"
)

// Scope returns the ScopeModel view of the Model, used for oscilloscope
// control.
func (m *Model) Scope() *ScopeModel { return (*ScopeModel)(m) }

type ScopeModel Model

type scopeData struct {
	waveForm       RingBuffer[[2]float32]
	once           bool
	triggered      bool
	wrap           bool
	triggerChannel int
	lengthInBeats  int
}

// Once returns a Bool for controlling whether the oscilloscope should only
// trigger once.
func (m *ScopeModel) Once() Bool { return MakeBoolFromPtr(&m.scopeData.once) }

// Wrap returns a Bool for controlling whether the oscilloscope should wrap the
// buffer when full.
func (m *ScopeModel) Wrap() Bool { return MakeBoolFromPtr(&m.scopeData.wrap) }

// LengthInBeats returns an Int for controlling the length of the oscilloscope
// buffer in beats.
func (m *ScopeModel) LengthInBeats() Int { return MakeInt((*scopeLengthInBeats)(m)) }

type scopeLengthInBeats Model

func (s *scopeLengthInBeats) Value() int { return s.scopeData.lengthInBeats }
func (s *scopeLengthInBeats) SetValue(val int) bool {
	s.scopeData.lengthInBeats = val
	(*ScopeModel)(s).updateBufferLength()
	return true
}
func (s *scopeLengthInBeats) Range() RangeInclusive { return RangeInclusive{1, 999} }

// TriggerChannel returns an Int for controlling the trigger channel of the
// oscilloscope. 0 = no trigger, 1 is the first channel etc.
func (m *ScopeModel) TriggerChannel() Int { return MakeInt((*scopeTriggerChannel)(m)) }

type scopeTriggerChannel Model

func (s *scopeTriggerChannel) Value() int { return s.scopeData.triggerChannel }
func (s *scopeTriggerChannel) SetValue(val int) bool {
	s.scopeData.triggerChannel = val
	return true
}
func (s *scopeTriggerChannel) Range() RangeInclusive { return RangeInclusive{0, len(s.d.Song.Patch)} }
func (s *scopeTriggerChannel) StringOf(value int) string {
	if value == 0 {
		return "Disabled"
	}
	if value >= 1 && value <= len(s.d.Song.Patch) && s.d.Song.Patch[value-1].Name != "" {
		return s.d.Song.Patch[value-1].Name
	}
	return strconv.Itoa(value)
}

// Waveform returns the oscilloscope waveform buffer.
func (s *ScopeModel) Waveform() RingBuffer[[2]float32] { return s.scopeData.waveForm }

func (s *ScopeModel) Envelope() (Envelope, bool) {
	i := s.d.InstrIndex
	u := s.d.UnitIndex
	if i < 0 || i >= len(s.d.Song.Patch) || u < 0 || u >= len(s.d.Song.Patch[i].Units) || s.d.Song.Patch[i].Units[u].Type != "envelope" {
		return Envelope{}, false
	}
	var ret Envelope = Envelope{{Position: math.MaxInt}, {Position: math.MaxInt}, {Position: math.MaxInt}, {Position: math.MaxInt}, {Position: math.MaxInt}}
	releasePos := len(s.scopeData.waveForm.Buffer) / 2
	p := s.d.Song.Patch[s.d.InstrIndex].Units[s.d.UnitIndex].Parameters
	attack := nonLinearMap((float32)(p["attack"]) / 128.0)
	decay := nonLinearMap((float32)(p["decay"]) / 128.0)
	sustain := (float32)(p["sustain"]) / 128.0
	release := nonLinearMap((float32)(p["release"]) / 128.0)
	gain := (float32)(p["gain"]) / 128.0
	curpos := 0
	for i := 0; i < 3; i++ {
		var nextpos int
		switch i {
		case 0:
			ret[i] = EnvelopePoint{Position: curpos, Level: 0, Slope: attack * gain}
			nextpos = curpos + int(math.Ceil(float64(1/attack)))
		case 1:
			ret[i] = EnvelopePoint{Position: curpos, Level: gain, Slope: -decay * gain}
			nextpos = curpos + int(math.Ceil(float64((1-sustain)/decay)))
		case 2:
			ret[i] = EnvelopePoint{Position: curpos, Level: sustain * gain, Slope: 0}
			nextpos = math.MaxInt
		}
		if nextpos >= releasePos {
			v := ret[i].Level + ret[i].Slope*float32(releasePos-curpos)
			ret[i+1] = EnvelopePoint{Position: releasePos, Level: v, Slope: -release * gain}
			ret[i+2] = EnvelopePoint{Position: releasePos + int(math.Ceil(float64(v/(release*gain)))), Level: 0, Slope: 0}
			break
		}
		curpos = nextpos
	}
	return ret, true
}

func nonLinearMap(value float32) float32 {
	return float32(math.Exp2(float64(-24 * value)))
}

type Envelope [5]EnvelopePoint

type EnvelopePoint struct {
	Position     int
	Level, Slope float32
}

func (e *Envelope) Value(position int) float32 {
	for i := len(e) - 1; i >= 0; i-- {
		if position >= e[i].Position {
			return e[i].Value(position + 1)
		}
	}
	return 0
}

func (e *EnvelopePoint) Value(position int) float32 {
	return e.Level + e.Slope*float32(position-e.Position)
}

// processAudioBuffer fills the oscilloscope buffer with audio data from the
// given buffer.
func (s *ScopeModel) processAudioBuffer(bufPtr *sointu.AudioBuffer) {
	if s.scopeData.wrap {
		s.scopeData.waveForm.WriteWrap(*bufPtr)
	} else {
		s.scopeData.waveForm.WriteOnce(*bufPtr)
	}
}

// trigger triggers the oscilloscope if the given channel matches the trigger
// channel.
func (s *ScopeModel) trigger(channel int) {
	if s.scopeData.triggerChannel > 0 && channel == s.scopeData.triggerChannel && !(s.scopeData.once && s.scopeData.triggered) {
		s.scopeData.waveForm.Cursor = 0
		s.scopeData.triggered = true
	}
}

// reset resets the oscilloscope buffer and cursor.
func (s *ScopeModel) reset() {
	s.scopeData.waveForm.Cursor = 0
	s.scopeData.triggered = false
	l := len(s.scopeData.waveForm.Buffer)
	s.scopeData.waveForm.Buffer = s.scopeData.waveForm.Buffer[:0]
	s.scopeData.waveForm.Buffer = append(s.scopeData.waveForm.Buffer, make([][2]float32, l)...)
}

func (s *ScopeModel) updateBufferLength() {
	if s.d.Song.BPM == 0 || s.scopeData.lengthInBeats == 0 {
		return
	}
	setSliceLength(&s.scopeData.waveForm.Buffer, s.d.Song.SamplesPerRow()*s.d.Song.RowsPerBeat*s.scopeData.lengthInBeats)
}

// RingBuffer is a generic ring buffer with buffer and a cursor. It is used by
// the oscilloscope.
type RingBuffer[T any] struct {
	Buffer []T
	Cursor int
}

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
