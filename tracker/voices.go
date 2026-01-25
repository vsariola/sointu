package tracker

import (
	"fmt"
	"math"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
	"gopkg.in/yaml.v3"
)

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

// helpers

func (m *Model) sliceInstrumentsTracks(instruments, tracks bool, ranges ...Range) (ok bool) {
	defer m.change("sliceInstrumentsTracks", PatchChange, MajorChange)()
	if instruments {
		m.d.Song.Patch, ok = VoiceSlice(m.d.Song.Patch, ranges...)
		if !ok {
			goto fail
		}
	}
	if tracks {
		m.d.Song.Score.Tracks, ok = VoiceSlice(m.d.Song.Score.Tracks, ranges...)
		if !ok {
			goto fail
		}
	}
	return true
fail:
	(*Model)(m).Alerts().AddNamed("slicesInstrumentsTracks", "Modify prevented by Instrument-Track linking", Warning)
	m.changeCancel = true
	return false
}

func (m *Model) marshalVoices(r Range) (data []byte, err error) {
	patch, ok := VoiceSlice(m.d.Song.Patch, r)
	if !ok {
		return nil, fmt.Errorf("marshalVoiceRange: slicing patch failed")
	}
	tracks, ok := VoiceSlice(m.d.Song.Score.Tracks, r)
	if !ok {
		return nil, fmt.Errorf("marshalVoiceRange: slicing tracks failed")
	}
	return yaml.Marshal(struct {
		Patch  sointu.Patch
		Tracks []sointu.Track
	}{patch, tracks})
}

func (m *Model) unmarshalVoices(voiceIndex int, data []byte, instruments, tracks bool) (instrRange, trackRange Range, ok bool) {
	var d struct {
		Patch  sointu.Patch
		Tracks []sointu.Track
	}
	if err := yaml.Unmarshal(data, &d); err != nil {
		return Range{}, Range{}, false
	}
	return m.addVoices(voiceIndex, d.Patch, d.Tracks, instruments, tracks)
}

func (m *Model) addVoices(voiceIndex int, p sointu.Patch, t []sointu.Track, instruments, tracks bool) (instrRange Range, trackRange Range, ok bool) {
	defer m.change("addVoices", PatchChange, MajorChange)()
	addedLength := max(p.NumVoices(), sointu.TotalVoices(t))
	if instruments {
		m.assignUnitIDsForPatch(p)
		m.d.Song.Patch, instrRange, ok = VoiceInsert(m.d.Song.Patch, voiceIndex, addedLength, p...)
		if !ok {
			goto fail
		}
	}
	if tracks {
		m.d.Song.Score.Tracks, trackRange, ok = VoiceInsert(m.d.Song.Score.Tracks, voiceIndex, addedLength, t...)
		if !ok {
			goto fail
		}
	}
	return instrRange, trackRange, true
fail:
	(*Model)(m).Alerts().AddNamed("addVoices", "Adding voices prevented by Instrument-Track linking", Warning)
	m.changeCancel = true
	return Range{}, Range{}, false
}

func (m *Model) remainingVoices(instruments, tracks bool) (ret int) {
	ret = math.MaxInt
	if instruments {
		ret = min(ret, vm.MAX_VOICES-m.d.Song.Patch.NumVoices())
	}
	if tracks {
		ret = min(ret, vm.MAX_VOICES-m.d.Song.Score.NumVoices())
	}
	return
}
