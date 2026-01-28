package tracker

import (
	"iter"
	"math"
	"math/bits"
	"strconv"
)

// Enabler is an interface that defines a single Enabled() method, which is used
// by the UI to check if UI Action/Bool/Int etc. is enabled or not.
type Enabler interface {
	Enabled() bool
}

// Action

type (
	// Action describes a user action that can be performed on the model, which
	// can be initiated by calling the Do() method. It is usually initiated by a
	// button press or a menu item. Action advertises whether it is enabled, so
	// UI can e.g. gray out buttons when the underlying action is not allowed.
	// The underlying Doer can optionally implement the Enabler interface to
	// decide if the action is enabled or not; if it does not implement the
	// Enabler interface, the action is always allowed.
	Action struct {
		doer Doer
	}

	// Doer is an interface that defines a single Do() method, which is called
	// when an action is performed.
	Doer interface {
		Do()
	}
)

func MakeAction(doer Doer) Action { return Action{doer: doer} }

func (a Action) Do() {
	e, ok := a.doer.(Enabler)
	if ok && !e.Enabled() {
		return
	}
	if a.doer != nil {
		a.doer.Do()
	}
}

func (a Action) Enabled() bool {
	if a.doer == nil {
		return false // no doer, not allowed
	}
	e, ok := a.doer.(Enabler)
	if !ok {
		return true // not enabler, always allowed
	}
	return e.Enabled()
}

// Bool

type (
	Bool struct {
		value BoolValue
	}

	BoolValue interface {
		Value() bool
		SetValue(bool)
	}

	simpleBool bool
)

func MakeBool(value BoolValue) Bool    { return Bool{value: value} }
func MakeBoolFromPtr(value *bool) Bool { return Bool{value: (*simpleBool)(value)} }
func (v Bool) Toggle()                 { v.SetValue(!v.Value()) }

func (v Bool) SetValue(value bool) (changed bool) {
	if !v.Enabled() || v.Value() == value {
		return false
	}
	v.value.SetValue(value)
	return true
}

func (v Bool) Value() bool {
	if v.value == nil {
		return false
	}
	return v.value.Value()
}

func (v Bool) Enabled() bool {
	if v.value == nil {
		return false
	}
	e, ok := v.value.(Enabler)
	if !ok {
		return true
	}
	return e.Enabled()
}

func (v *simpleBool) Value() bool         { return bool(*v) }
func (v *simpleBool) SetValue(value bool) { *v = simpleBool(value) }

// Int

type (
	// Int represents an integer value in the tracker model e.g. BPM, song
	// length, etc. It is a wrapper around an IntValue interface that provides
	// methods to manipulate the value, but Int guard that all changes are
	// within the range of the underlying IntValue implementation and that
	// SetValue is not called when the value is unchanged. The IntValue can
	// optionally implement the StringOfer interface to provide custom string
	// representations of the integer values.
	Int struct {
		value IntValue
	}

	IntValue interface {
		Value() int
		SetValue(int) (changed bool)
		Range() RangeInclusive
	}

	StringOfer interface {
		StringOf(value int) string
	}
)

func MakeInt(value IntValue) Int { return Int{value} }

func (v Int) Add(delta int) (changed bool) {
	return v.SetValue(v.Value() + delta)
}

func (v Int) SetValue(value int) (changed bool) {
	r := v.Range()
	value = r.Clamp(value)
	if value == v.Value() || value < r.Min || value > r.Max {
		return false
	}
	return v.value.SetValue(value)
}

func (v Int) Range() RangeInclusive {
	if v.value == nil {
		return RangeInclusive{0, 0}
	}
	return v.value.Range()
}

func (v Int) Value() int {
	if v.value == nil {
		return 0
	}
	return v.value.Value()
}

func (v Int) String() string {
	return v.StringOf(v.Value())
}

func (v Int) StringOf(value int) string {
	if s, ok := v.value.(StringOfer); ok {
		return s.StringOf(value)
	}
	return strconv.Itoa(value)
}

// String

type (
	String struct {
		value StringValue
	}

	StringValue interface {
		Value() string
		SetValue(string) (changed bool)
	}
)

func MakeString(value StringValue) String { return String{value: value} }

func (v String) SetValue(value string) (changed bool) {
	if v.value == nil || v.value.Value() == value {
		return false
	}
	return v.value.SetValue(value)
}

func (v String) Value() string {
	if v.value == nil {
		return ""
	}
	return v.value.Value()
}

// List

type (
	List struct {
		data ListData
	}

	ListData interface {
		Selected() int
		Selected2() int
		SetSelected(int)
		SetSelected2(int)
		Count() int
	}

	MutableListData interface {
		Change(kind string, severity ChangeSeverity) func()
		Cancel()
		Move(r Range, delta int) (ok bool)
		Delete(r Range) (ok bool)
		Marshal(r Range) ([]byte, error)
		Unmarshal([]byte) (r Range, err error)
	}
)

func MakeList(data ListData) List { return List{data} }

func (l List) Selected() int          { return max(min(l.data.Selected(), l.data.Count()-1), 0) }
func (l List) Selected2() int         { return max(min(l.data.Selected2(), l.data.Count()-1), 0) }
func (l List) SetSelected(value int)  { l.data.SetSelected(max(min(value, l.data.Count()-1), 0)) }
func (l List) SetSelected2(value int) { l.data.SetSelected2(max(min(value, l.data.Count()-1), 0)) }
func (l List) Count() int             { return l.data.Count() }

// MoveElements moves the selected elements in a list by delta. The list must
// implement the MutableListData interface.
func (v List) MoveElements(delta int) bool {
	s, ok := v.data.(MutableListData)
	if !ok {
		return false
	}
	r := v.listRange()
	if delta == 0 || r.Start+delta < 0 || r.End+delta > v.Count() {
		return false
	}
	defer s.Change("MoveElements", MajorChange)()
	if !s.Move(r, delta) {
		s.Cancel()
		return false
	}
	v.SetSelected(v.Selected() + delta)
	v.SetSelected2(v.Selected2() + delta)
	return true
}

// DeleteElements deletes the selected elements in a list. The list must
// implement the MutableListData interface.
func (v List) DeleteElements(backwards bool) bool {
	d, ok := v.data.(MutableListData)
	if !ok {
		return false
	}
	r := v.listRange()
	if r.Len() == 0 {
		return false
	}
	defer d.Change("DeleteElements", MajorChange)()
	if !d.Delete(r) {
		d.Cancel()
		return false
	}
	if backwards && r.Start > 0 {
		r.Start--
	}
	v.SetSelected(r.Start)
	v.SetSelected2(r.Start)
	return true
}

// CopyElements copies the selected elements in a list. The list must implement
// the MutableListData interface. Returns the copied data, marshaled into byte
// slice, and true if successful.
func (v List) CopyElements() ([]byte, bool) {
	m, ok := v.data.(MutableListData)
	if !ok {
		return nil, false
	}
	r := v.listRange()
	if r.Len() == 0 {
		return nil, false
	}
	ret, err := m.Marshal(r)
	if err != nil {
		return nil, false
	}
	return ret, true
}

// PasteElements pastes the data into the list. The data is unmarshaled from the
// byte slice. The list must implement the MutableListData interface. Returns
// true if successful.
func (v List) PasteElements(data []byte) (ok bool) {
	m, ok := v.data.(MutableListData)
	if !ok {
		return false
	}
	defer m.Change("PasteElements", MajorChange)()
	r, err := m.Unmarshal(data)
	if err != nil {
		m.Cancel()
		return false
	}
	v.SetSelected(r.Start)
	v.SetSelected2(r.End - 1)
	return true
}

func (v List) Mutable() bool {
	_, ok := v.data.(MutableListData)
	return ok
}

func (v *List) listRange() (r Range) {
	r.Start = max(min(v.Selected(), v.Selected2()), 0)
	r.End = min(max(v.Selected(), v.Selected2())+1, v.Count())
	return
}

// RangeInclusive

// RangeInclusive represents a range of integers [Min, Max], inclusive.
type RangeInclusive struct{ Min, Max int }

func (r RangeInclusive) Clamp(value int) int { return max(min(value, r.Max), r.Min) }

// Range is used to represent a range [Start,End) of integers, excluding End
type Range struct{ Start, End int }

func (r Range) Len() int { return r.End - r.Start }

func (r Range) Swaps(delta int) iter.Seq2[int, int] {
	if delta > 0 {
		return func(yield func(int, int) bool) {
			for i := r.End - 1; i >= r.Start; i-- {
				if !yield(i, i+delta) {
					return
				}
			}
		}
	}
	return func(yield func(int, int) bool) {
		for i := r.Start; i < r.End; i++ {
			if !yield(i, i+delta) {
				return
			}
		}
	}
}

func (r Range) Intersect(s Range) (ret Range) {
	ret.Start = max(r.Start, s.Start)
	ret.End = max(min(r.End, s.End), ret.Start)
	if ret.Len() == 0 {
		return Range{}
	}
	return
}

func MakeMoveRanges(a Range, delta int) [4]Range {
	if delta < 0 {
		return [4]Range{
			{math.MinInt, a.Start + delta},
			{a.Start, a.End},
			{a.Start + delta, a.Start},
			{a.End, math.MaxInt},
		}
	}
	return [4]Range{
		{math.MinInt, a.Start},
		{a.End, a.End + delta},
		{a.Start, a.End},
		{a.End + delta, math.MaxInt},
	}
}

// MakeSetLength takes a range and a length, and returns a slice of ranges that
// can be used with VoiceSlice to expand or shrink the range to the given
// length, by either duplicating or removing elements. The function tries to
// duplicate elements so all elements are equally spaced, and tries to remove
// elements from the middle of the range.
func MakeSetLength(a Range, length int) []Range {
	if length <= 0 || a.Len() <= 0 {
		return []Range{{a.Start, a.Start}}
	}
	ret := make([]Range, a.Len(), max(a.Len(), length)+2)
	for i := 0; i < a.Len(); i++ {
		ret[i] = Range{a.Start + i, a.Start + i + 1}
	}
	for x := len(ret); x < length; x++ {
		e := (x << 1) ^ (1 << bits.Len((uint)(x)))
		ret = append(ret[0:e+1], ret[e:]...)
	}
	for x := len(ret); x > length; x-- {
		e := (((x << 1) ^ (1 << bits.Len((uint)(x)))) + x - 1) % x
		ret = append(ret[0:e], ret[e+1:]...)
	}
	ret = append([]Range{{math.MinInt, a.Start}}, ret...)
	ret = append(ret, Range{a.End, math.MaxInt})
	return ret
}

func Complement(a Range) [2]Range {
	return [2]Range{
		{math.MinInt, a.Start},
		{a.End, math.MaxInt},
	}
}

// Insert inserts elements into a slice at the given index. If the index is out
// of bounds, the function returns false.
func Insert[T any, S ~[]T](slice S, index int, inserted ...T) (ret S, ok bool) {
	if index < 0 || index > len(slice) {
		return nil, false
	}
	ret = make(S, 0, len(slice)+len(inserted))
	ret = append(ret, slice[:index]...)
	ret = append(ret, inserted...)
	ret = append(ret, slice[index:]...)
	return ret, true
}

// Table

type (
	Table struct {
		TableData
	}

	TableData interface {
		Cursor() Point
		Cursor2() Point
		SetCursor(Point)
		SetCursor2(Point)
		Width() int
		Height() int
		MoveCursor(dx, dy int) (ok bool)

		clear(p Point)
		set(p Point, value int)
		add(rect Rect, delta int, largestep bool) (ok bool)
		marshal(rect Rect) (data []byte, ok bool)
		unmarshalAtCursor(data []byte) (ok bool)
		unmarshalRange(rect Rect, data []byte) (ok bool)
		change(kind string, severity ChangeSeverity) func()
		cancel()
	}

	Point struct {
		X, Y int
	}

	Rect struct {
		TopLeft, BottomRight Point
	}
)

// Rect methods

func (r *Rect) Contains(p Point) bool {
	return r.TopLeft.X <= p.X && p.X <= r.BottomRight.X &&
		r.TopLeft.Y <= p.Y && p.Y <= r.BottomRight.Y
}

func (r *Rect) Width() int {
	return r.BottomRight.X - r.TopLeft.X + 1
}

func (r *Rect) Height() int {
	return r.BottomRight.Y - r.TopLeft.Y + 1
}

func (r *Rect) Limit(width, height int) {
	if r.TopLeft.X < 0 {
		r.TopLeft.X = 0
	}
	if r.TopLeft.Y < 0 {
		r.TopLeft.Y = 0
	}
	if r.BottomRight.X >= width {
		r.BottomRight.X = width - 1
	}
	if r.BottomRight.Y >= height {
		r.BottomRight.Y = height - 1
	}
}

func (v Table) Range() (rect Rect) {
	rect.TopLeft.X = min(v.Cursor().X, v.Cursor2().X)
	rect.TopLeft.Y = min(v.Cursor().Y, v.Cursor2().Y)
	rect.BottomRight.X = max(v.Cursor().X, v.Cursor2().X)
	rect.BottomRight.Y = max(v.Cursor().Y, v.Cursor2().Y)
	return
}

func (v Table) Copy() ([]byte, bool) {
	ret, ok := v.marshal(v.Range())
	if !ok {
		return nil, false
	}
	return ret, true
}

func (v Table) Paste(data []byte) bool {
	defer v.change("Paste", MajorChange)()
	if v.Cursor() == v.Cursor2() {
		return v.unmarshalAtCursor(data)
	} else {
		return v.unmarshalRange(v.Range(), data)
	}
}

func (v Table) Clear() {
	defer v.change("Clear", MajorChange)()
	rect := v.Range()
	rect.Limit(v.Width(), v.Height())
	for y := rect.TopLeft.Y; y <= rect.BottomRight.Y; y++ {
		for x := rect.TopLeft.X; x <= rect.BottomRight.X; x++ {
			v.clear(Point{x, y})
		}
	}
}

func (v Table) Set(value byte) {
	defer v.change("Set", MajorChange)()
	cursor := v.Cursor()
	// TODO: might check for visibility
	v.set(cursor, int(value))
}

func (v Table) Fill(value int) {
	defer v.change("Fill", MajorChange)()
	rect := v.Range()
	rect.Limit(v.Width(), v.Height())
	for y := rect.TopLeft.Y; y <= rect.BottomRight.Y; y++ {
		for x := rect.TopLeft.X; x <= rect.BottomRight.X; x++ {
			v.set(Point{x, y}, value)
		}
	}
}

func (v Table) Add(delta int, largeStep bool) {
	defer v.change("Add", MinorChange)()
	if !v.add(v.Range(), delta, largeStep) {
		v.cancel()
	}
}

func (v Table) SetCursorX(x int) {
	p := v.Cursor()
	p.X = x
	v.SetCursor(p)
}

func (v Table) SetCursorY(y int) {
	p := v.Cursor()
	p.Y = y
	v.SetCursor(p)
}
