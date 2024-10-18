package tracker

import (
	"math"

	"github.com/vsariola/sointu"
)

type (
	// Range is used to represent a range [Start,End) of integers
	Range struct {
		Start, End int
	}
)

func (r Range) Len() int { return r.End - r.Start }

func Exclusive(start, end int) Range {
	if start > end {
		start, end = end, start
	}
	return Range{start, end}
}

func Inclusive(start, end int) Range {
	if start > end {
		start, end = end, start
	}
	return Range{start, end + 1}
}

func MakeSwapRanges(a, b Range) [5]Range {
	if a.Start > b.Start {
		a, b = b, a
	}
	return [5]Range{
		{math.MinInt, a.Start},
		{b.Start, b.End},
		{a.End, b.Start},
		{a.Start, a.End},
		{b.End, math.MaxInt},
	}
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

func MakeComplementaryRanges(a Range) [2]Range {
	return [2]Range{
		{math.MinInt, a.Start},
		{a.End, math.MaxInt},
	}
}

// Slice is a more powerful version of the standard slice operator. It takes a
// slice and a list of ranges, and returns a new slice that is a concatenation
// of the slices defined by the ranges. The ranges are inclusive on the start
// and exclusive on the end. Ranges can go outside the bounds of the slice, in
// which case the function just ignores the out-of-bounds part. Ranges with End
// < Start are ignored. If the ranges overlap, the function returns false, so
// the slicing cannot accidentally make shallow copies of reference types. If
// the ranges are empty, the function returns an empty slice.
func Slice[T any, S ~[]T](slice S, ranges ...Range) (ret S, ok bool) {
	ret = make(S, 0, len(slice))
	used := make([]bool, len(slice))
	for _, r := range ranges {
		s := max(0, r.Start)
		e := min(len(slice), r.End)
		if s > e {
			continue
		}
		for i := s; i < e; i++ {
			if used[i] {
				return nil, false
			}
			used[i] = true
		}
		ret = append(ret, slice[s:e]...)
	}
	return ret, true
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

// VoiceSlice works similar to the Slice function, but takes a slice of
// NumVoicer:s and treats it as a "virtual slice", with element repeated by the
// number of voices it has. NumVoicer interface is implemented at least by
// sointu.Tracks and sointu.Instruments. For example, if parameter "slice" has
// three elements, returning GetNumVoices 2, 1, and 3, the VoiceSlice thinks of
// this as a virtual slice of 6 elements [0,0,1,2,2,2]. Then, the "ranges"
// parameter are slicing ranges to this virtual slice. Continuing with the
// example, if "ranges" was [2,5), the virtual slice would be [1,2,2], and the
// function would return a slice with two elements: first with NumVoices 1 and
// second with NumVoices 2. If multiple ranges are given, multiple virtual
// slices are concatenated. However, when doing so, splitting an element is not
// allowed. In the previous example, if the ranges were [1,3) and [0,1), the
// resulting concatenated virtual slice would be [0,1,0], and here the 0 element
// would be split. This is to avoid accidentally making shallow copies of
// reference types.
func VoiceSlice[T any, S ~[]T, P sointu.NumVoicerPointer[T]](slice S, ranges ...Range) (ret S, ok bool) {
	ret = make(S, 0, len(slice))
	last := -1
	used := make([]bool, len(slice))
outer:
	for _, r := range ranges {
		left := 0
		for i, elem := range slice {
			right := left + (P)(&slice[i]).GetNumVoices()
			if left >= r.End {
				continue outer
			}
			if right <= r.Start {
				left = right
				continue
			}
			overlap := min(right, r.End) - max(left, r.Start)
			if last == i {
				(P)(&ret[len(ret)-1]).SetNumVoices(
					(P)(&ret[len(ret)-1]).GetNumVoices() + overlap)
			} else {
				if last == math.MaxInt || used[i] {
					return nil, false
				}
				ret = append(ret, elem)
				(P)(&ret[len(ret)-1]).SetNumVoices(overlap)
				used[i] = true
			}
			last = i
			left = right
		}
		if left >= r.End {
			continue outer
		}
		last = math.MaxInt // the list is closed, adding more elements causes it to fail
	}
	return ret, true
}

// VoiceInsert tries adding the elements "added" to the slice "orig" at the
// voice index "index". Notice that index is the index into a virtual slice
// where each element is repeated by the number of voices it has. If the index
// is between elements, the new elements are added in between the old elements.
// If the addition would cause splitting of an element, we rather increase the
// number of voices the element has, but do not split it.
func VoiceInsert[T any, S ~[]T, P sointu.NumVoicerPointer[T]](orig S, index, length int, added ...T) (ret S, retRange Range, ok bool) {
	ret = make(S, 0, len(orig)+length)
	left := 0
	for i, elem := range orig {
		right := left + (P)(&orig[i]).GetNumVoices()
		if left == index { // we are between elements and it's safe to add there
			if sointu.TotalVoices[T, S, P](added) < length {
				return nil, Range{}, false // we are missing some elements
			}
			retRange = Range{len(ret), len(ret) + len(added)}
			ret = append(ret, added...)
		} else if left < index && index < right { // we are inside an element and would split it; just increase its voices instead of splitting
			(P)(&elem).SetNumVoices((P)(&orig[i]).GetNumVoices() + sointu.TotalVoices[T, S, P](added))
			retRange = Range{len(ret), len(ret)}
		}
		ret = append(ret, elem)
		left = right
	}
	if left == index { // we are at the end and it's safe to add there, even if we are missing some elements
		retRange = Range{len(ret), len(ret) + len(added)}
		ret = append(ret, added...)
	}
	return ret, retRange, true
}

func VoiceChange[T any, S ~[]T, P sointu.NumVoicerPointer[T]](orig S, index, delta int) (ret S, ok bool) {
	if delta == 0 {
		return orig, true
	}
	if delta < 0 {
		r := Range{index, index - delta}
		c := MakeComplementaryRanges(r)
		return VoiceSlice[T, S, P](orig, c[:]...)
	}
	ok = VoiceExpand[T, S, P](orig, index, delta)
	return orig, ok
}

func VoiceExpand[T any, S ~[]T, P sointu.NumVoicerPointer[T]](orig S, index, length int) (ok bool) {
	left := 0
	for i := range orig {
		right := left + (P)(&orig[i]).GetNumVoices()
		if left <= index && index < right { // we are inside an element and would split it; just increase its voices instead of splitting
			(P)(&orig[i]).SetNumVoices((P)(&orig[i]).GetNumVoices() + length)
			return true
		}
		left = right
	}
	return false
}

func VoiceRange[T any, S ~[]T, P sointu.NumVoicerPointer[T]](slice S, indexRange Range) (voiceRange Range) {
	indexRange.Start = max(0, indexRange.Start)
	indexRange.End = min(len(slice), indexRange.End)
	for _, e := range slice[:indexRange.Start] {
		voiceRange.Start += (P)(&e).GetNumVoices()
	}
	voiceRange.End = voiceRange.Start
	for i := indexRange.Start; i < indexRange.End; i++ {
		voiceRange.End += (P)(&slice[i]).GetNumVoices()
	}
	return
}
