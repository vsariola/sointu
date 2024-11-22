package tracker

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
