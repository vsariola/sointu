package tracker

import "github.com/vsariola/sointu/tracker/types"

type (
	// OptionalInt tries to follow the same convention as e.g. Int{...} or Bool{...}
	// Do not confuse with types.OptionalInteger, which you might use as a model,
	// but don't necessarily have to.
	OptionalInt struct {
		optionalIntData
	}

	optionalIntData interface {
		Unpack() (int, bool)
		Value() int
		Range() intRange

		setValue(int)
		unsetValue()
		change(kind string) func()
	}

	TrackMidiVelIn Model
)

func (v OptionalInt) Set(value int, present bool) (ok bool) {
	if !present {
		v.unsetValue()
		return true
	}
	// TODO: can we deduplicate this by referencing Int{...}.Set(value) ?
	r := v.Range()
	if v.Equals(value, present) || value < r.Min || value > r.Max {
		return false
	}
	defer v.change("Set")()
	v.setValue(value)
	return true
}

func (v OptionalInt) Equals(value int, present bool) bool {
	oldValue, oldPresent := v.Unpack()
	return value == oldValue && present == oldPresent
}

// Model methods

func (m *Model) TrackForMidiVelIn() *TrackMidiVelIn { return (*TrackMidiVelIn)(m) }

// TrackForMidiVelIn - to record Velocity in the track with given number (-1 = off)

func (m *TrackMidiVelIn) OptionalInt() OptionalInt { return OptionalInt{m} }
func (m *TrackMidiVelIn) Range() intRange          { return intRange{0, len(m.d.Song.Score.Tracks) - 1} }
func (m *TrackMidiVelIn) change(string) func()     { return func() {} }

func (m *TrackMidiVelIn) setValue(val int) {
	m.trackForMidiVelIn = types.NewOptionalInteger(val, val >= 0)
}

func (m *TrackMidiVelIn) unsetValue() {
	m.trackForMidiVelIn = types.NewEmptyOptionalInteger()
}

func (m *TrackMidiVelIn) Unpack() (int, bool) {
	return m.trackForMidiVelIn.Unpack()
}

func (m *TrackMidiVelIn) Value() int {
	return m.trackForMidiVelIn.Value()
}

func (m *TrackMidiVelIn) IsValid() bool {
	if m.trackForMidiVelIn.Empty() {
		return true
	}
	return (*Model)(m).CanUseTrackForMidiVelInput(m.trackForMidiVelIn.Value())
}

func (m *TrackMidiVelIn) Equals(value int) bool {
	return m.trackForMidiVelIn.Equals(value)
}
