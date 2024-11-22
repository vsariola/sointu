package types

type (
	// OptionalInteger is the simple struct, not to be confused with tracker.OptionalInt.
	// It implements the tracker.optionalIntData interface, without needing to know so.
	OptionalInteger struct {
		value  int
		exists bool
	}
)

func NewOptionalInteger(value int, exists bool) OptionalInteger {
	return OptionalInteger{value, exists}
}

func NewOptionalIntegerOf(value int) OptionalInteger {
	return OptionalInteger{
		value:  value,
		exists: true,
	}
}

func NewEmptyOptionalInteger() OptionalInteger {
	// could also just use OptionalInteger{}
	return OptionalInteger{
		exists: false,
	}
}

func (i OptionalInteger) Unpack() (int, bool) {
	return i.value, i.exists
}

func (i OptionalInteger) Value() int {
	if !i.exists {
		panic("Access value of empty OptionalInteger")
	}
	return i.value
}

func (i OptionalInteger) Empty() bool {
	return !i.exists
}

func (i OptionalInteger) Equals(value int) bool {
	return i.exists && i.value == value
}
