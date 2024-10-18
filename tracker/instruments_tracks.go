package tracker

import (
	"fmt"
	"math"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
	"gopkg.in/yaml.v3"
)

type (
	Instruments      Model // Instruments is a list of instruments, implementing ListData & MutableListData interfaces
	Tracks           Model // Tracks is a list of all the tracks, implementing ListData & MutableListData interfaces
	InstrumentVoices Model
	TrackVoices      Model
)

// Model methods

func (m *Model) InstrumentVoices() *InstrumentVoices { return (*InstrumentVoices)(m) }
func (m *Model) TrackVoices() *TrackVoices           { return (*TrackVoices)(m) }
func (m *Model) Instruments() *Instruments           { return (*Instruments)(m) }
func (m *Model) Tracks() *Tracks                     { return (*Tracks)(m) }

func (m *Model) AddTrack() Action {
	return Action{
		allowed: func() bool { return m.d.Song.Score.NumVoices() < vm.MAX_VOICES },
		do: func() {
			defer (*Model)(m).change("AddTrack", SongChange, MajorChange)()
			voiceIndex := m.d.Song.Score.FirstVoiceForTrack(m.d.Cursor.Track)
			p := sointu.Patch{defaultInstrument.Copy()}
			t := []sointu.Track{defaultTrack}
			_, _, ok := m.addVoices(voiceIndex, p, t, (*Model)(m).linkInstrTrack, true)
			m.changeCancel = !ok
		},
	}
}

func (m *Model) DeleteTrack() Action {
	return Action{
		allowed: func() bool { return len(m.d.Song.Score.Tracks) > 0 },
		do:      func() { m.Tracks().List().DeleteElements(false) },
	}
}

func (m *Model) AddInstrument() Action {
	return Action{
		allowed: func() bool { return (*Model)(m).d.Song.Patch.NumVoices() < vm.MAX_VOICES },
		do: func() {
			defer (*Model)(m).change("AddInstrument", SongChange, MajorChange)()
			voiceIndex := m.d.Song.Patch.FirstVoiceForInstrument(m.d.InstrIndex)
			p := sointu.Patch{defaultInstrument.Copy()}
			t := []sointu.Track{defaultTrack}
			_, _, ok := m.addVoices(voiceIndex, p, t, true, (*Model)(m).linkInstrTrack)
			m.changeCancel = !ok
		},
	}
}

func (m *Model) DeleteInstrument() Action {
	return Action{
		allowed: func() bool { return len((*Model)(m).d.Song.Patch) > 0 },
		do:      func() { m.Instruments().List().DeleteElements(false) },
	}
}

func (m *Model) SplitTrack() Action {
	return Action{
		allowed: func() bool {
			return m.d.Cursor.Track >= 0 && m.d.Cursor.Track < len(m.d.Song.Score.Tracks) && m.d.Song.Score.Tracks[m.d.Cursor.Track].NumVoices > 1
		},
		do: func() {
			defer (*Model)(m).change("SplitTrack", SongChange, MajorChange)()
			voiceIndex := m.d.Song.Score.FirstVoiceForTrack(m.d.Cursor.Track)
			middle := voiceIndex + (m.d.Song.Score.Tracks[m.d.Cursor.Track].NumVoices+1)/2
			end := voiceIndex + m.d.Song.Score.Tracks[m.d.Cursor.Track].NumVoices
			left, ok := VoiceSlice(m.d.Song.Score.Tracks, Range{math.MinInt, middle})
			if !ok {
				m.changeCancel = true
				return
			}
			right, ok := VoiceSlice(m.d.Song.Score.Tracks, Range{end, math.MaxInt})
			if !ok {
				m.changeCancel = true
				return
			}
			newTrack := defaultTrack.Copy()
			newTrack.NumVoices = end - middle
			m.d.Song.Score.Tracks = append(left, newTrack)
			m.d.Song.Score.Tracks = append(m.d.Song.Score.Tracks, right...)
		},
	}
}

// Instruments methods

func (v *Instruments) List() List {
	return List{v}
}

func (v *Instruments) Item(i int) (name string, maxLevel float32, mute bool, ok bool) {
	if i < 0 || i >= len(v.d.Song.Patch) {
		return "", 0, false, false
	}
	name = v.d.Song.Patch[i].Name
	mute = v.d.Song.Patch[i].Mute
	start := v.d.Song.Patch.FirstVoiceForInstrument(i)
	end := start + v.d.Song.Patch[i].NumVoices
	if end >= vm.MAX_VOICES {
		end = vm.MAX_VOICES
	}
	if start < end {
		for _, level := range v.voiceLevels[start:end] {
			if maxLevel < level {
				maxLevel = level
			}
		}
	}
	ok = true
	return
}
func (v *Instruments) FirstID(i int) (id int, ok bool) {
	if i < 0 || i >= len(v.d.Song.Patch) {
		return 0, false
	}
	if len(v.d.Song.Patch[i].Units) == 0 {
		return 0, false
	}
	return v.d.Song.Patch[i].Units[0].ID, true
}

func (v *Instruments) Selected() int {
	return max(min(v.d.InstrIndex, v.Count()-1), 0)
}

func (v *Instruments) Selected2() int {
	return max(min(v.d.InstrIndex2, v.Count()-1), 0)
}

func (v *Instruments) SetSelected(value int) {
	v.d.InstrIndex = max(min(value, v.Count()-1), 0)
	v.d.UnitIndex = 0
	v.d.UnitIndex2 = 0
	v.d.UnitSearching = false
	v.d.UnitSearchString = ""
}

func (v *Instruments) SetSelected2(value int) {
	v.d.InstrIndex2 = max(min(value, v.Count()-1), 0)
}

func (v *Instruments) move(r Range, delta int) (ok bool) {
	voiceDelta := 0
	if delta < 0 {
		voiceDelta = -VoiceRange(v.d.Song.Patch, Exclusive(r.Start+delta, r.Start)).Len()
	} else if delta > 0 {
		voiceDelta = VoiceRange(v.d.Song.Patch, Exclusive(r.End, r.End+delta)).Len()
	}
	if voiceDelta == 0 {
		return false
	}
	voiceRange := VoiceRange(v.d.Song.Patch, r)
	return (*Model)(v).moveVoices(voiceRange, voiceDelta, true, v.linkInstrTrack)
}

func (v *Instruments) delete(r Range) (ok bool) {
	voiceRange := VoiceRange(v.d.Song.Patch, r)
	return (*Model)(v).deleteVoices(voiceRange, true, v.linkInstrTrack)
}

func (v *Instruments) change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("Instruments."+n, SongChange, severity)
}

func (v *Instruments) cancel() {
	v.changeCancel = true
}

func (v *Instruments) Count() int {
	return len(v.d.Song.Patch)
}

func (v *Instruments) marshal(r Range) ([]byte, error) {
	return (*Model)(v).marshalVoices(VoiceRange(v.d.Song.Patch, r))
}

func (m *Instruments) unmarshal(data []byte) (r Range, err error) {
	voiceIndex := m.d.Song.Patch.FirstVoiceForInstrument(m.d.InstrIndex)
	r, _, ok := (*Model)(m).unmarshalVoices(voiceIndex, data, true, m.linkInstrTrack)
	if !ok {
		return Range{}, fmt.Errorf("unmarshal: unmarshalVoices failed")
	}
	return r, nil
}

// Tracks methods

func (v *Tracks) List() List {
	return List{v}
}

func (v *Tracks) Selected() int {
	return max(min(v.d.Cursor.Track, v.Count()-1), 0)
}

func (v *Tracks) Selected2() int {
	return max(min(v.d.Cursor2.Track, v.Count()-1), 0)
}

func (v *Tracks) SetSelected(value int) {
	v.d.Cursor.Track = max(min(value, v.Count()-1), 0)
}

func (v *Tracks) SetSelected2(value int) {
	v.d.Cursor2.Track = max(min(value, v.Count()-1), 0)
}

func (v *Tracks) move(r Range, delta int) (ok bool) {
	voiceDelta := 0
	if delta < 0 {
		voiceDelta = -VoiceRange(v.d.Song.Score.Tracks, Exclusive(r.Start+delta, r.Start)).Len()
	} else if delta > 0 {
		voiceDelta = VoiceRange(v.d.Song.Score.Tracks, Exclusive(r.End, r.End+delta)).Len()
	}
	if voiceDelta == 0 {
		return false
	}
	voiceRange := VoiceRange(v.d.Song.Score.Tracks, r)
	return (*Model)(v).moveVoices(voiceRange, voiceDelta, v.linkInstrTrack, true)
}

func (v *Tracks) delete(r Range) (ok bool) {
	voiceRange := VoiceRange(v.d.Song.Score.Tracks, r)
	return (*Model)(v).deleteVoices(voiceRange, v.linkInstrTrack, true)
}

func (v *Tracks) change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("TrackList."+n, SongChange, severity)
}

func (v *Tracks) cancel() {
	v.changeCancel = true
}

func (v *Tracks) Count() int {
	return len((*Model)(v).d.Song.Score.Tracks)
}

func (v *Tracks) marshal(r Range) ([]byte, error) {
	return (*Model)(v).marshalVoices(VoiceRange(v.d.Song.Score.Tracks, r))
}

func (m *Tracks) unmarshal(data []byte) (r Range, err error) {
	voiceIndex := m.d.Song.Score.FirstVoiceForTrack(m.d.Cursor.Track)
	_, r, ok := (*Model)(m).unmarshalVoices(voiceIndex, data, m.linkInstrTrack, true)
	if !ok {
		return Range{}, fmt.Errorf("unmarshal: unmarshalVoices failed")
	}
	return r, nil
}

// InstrumentVoicesInt

func (v *InstrumentVoices) Int() Int {
	return Int{v}
}

func (v *InstrumentVoices) Value() int {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return 1
	}
	return max(v.d.Song.Patch[v.d.InstrIndex].NumVoices, 1)
}

func (m *InstrumentVoices) setValue(value int) {
	voiceIndex := m.d.Song.Patch.FirstVoiceForInstrument(m.d.InstrIndex)
	ok := (*Model)(m).changeNumVoice(voiceIndex, value-m.Value(), true, m.linkInstrTrack)
	if !ok {
		m.changeCancel = true
	}
}

func (v *InstrumentVoices) Range() intRange {
	return intRange{1, vm.MAX_VOICES - v.d.Song.Patch.NumVoices() + v.Value()}
}

func (v *InstrumentVoices) change(kind string) func() {
	if v.linkInstrTrack {
		return (*Model)(v).change("InstrumentVoicesInt."+kind, SongChange, MinorChange)
	}
	return (*Model)(v).change("InstrumentVoicesInt."+kind, PatchChange, MinorChange)
}

// TrackVoicesInt

func (v *TrackVoices) Int() Int {
	return Int{v}
}

func (v *TrackVoices) Value() int {
	t := v.d.Cursor.Track
	if t < 0 || t >= len(v.d.Song.Score.Tracks) {
		return 1
	}
	return max(v.d.Song.Score.Tracks[t].NumVoices, 1)
}

func (m *TrackVoices) setValue(value int) {
	voiceIndex := m.d.Song.Score.FirstVoiceForTrack(m.d.Cursor.Track)
	ok := (*Model)(m).changeNumVoice(voiceIndex, value-m.Value(), m.linkInstrTrack, true)
	if !ok {
		m.changeCancel = true
	}
}

func (v *TrackVoices) Range() intRange {
	t := v.d.Cursor.Track
	if t < 0 || t >= len(v.d.Song.Score.Tracks) {
		return intRange{1, 1}
	}
	return intRange{1, vm.MAX_VOICES - v.d.Song.Score.NumVoices() + v.d.Song.Score.Tracks[t].NumVoices}
}

func (v *TrackVoices) change(kind string) func() {
	return (*Model)(v).change("TrackVoicesInt."+kind, ScoreChange, MinorChange)
}

// helpers

func (m *Model) moveVoices(voiceRange Range, delta int, instruments, tracks bool) (ok bool) {
	defer m.change("moveVoices", PatchChange, MajorChange)()
	ranges := MakeMoveRanges(voiceRange, delta)
	if instruments {
		m.d.Song.Patch, ok = VoiceSlice(m.d.Song.Patch, ranges[:]...)
		if !ok {
			(*Model)(m).Alerts().AddNamed("moveVoices", "Move prevented by Instrument-Track linking", Warning)
			m.changeCancel = true
			return false
		}
	}
	if tracks {
		m.d.Song.Score.Tracks, ok = VoiceSlice(m.d.Song.Score.Tracks, ranges[:]...)
		if !ok {
			(*Model)(m).Alerts().AddNamed("moveVoices", "Move prevented by Instrument-Track linking", Warning)
			m.changeCancel = true
			return false
		}
	}
	return true
}

func (m *Model) deleteVoices(voiceRange Range, instruments, tracks bool) (ok bool) {
	defer m.change("deleteVoices", PatchChange, MajorChange)()
	r := MakeComplementaryRanges(voiceRange)
	if instruments {
		m.d.Song.Patch, ok = VoiceSlice(m.d.Song.Patch, r[:]...)
		if !ok {
			(*Model)(m).Alerts().AddNamed("deleteVoices", "Deleting prevented by Instrument-Track linking", Warning)
			m.changeCancel = true
			return false
		}
	}
	if tracks {
		m.d.Song.Score.Tracks, ok = VoiceSlice(m.d.Song.Score.Tracks, r[:]...)
		if !ok {
			(*Model)(m).Alerts().AddNamed("deleteVoices", "Deleting prevented by Instrument-Track linking", Warning)
			m.changeCancel = true
			return false
		}
	}
	return true
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
		p.AvoidUnitIDs(m.d.Song.Patch)
		m.d.Song.Patch, instrRange, ok = VoiceInsert(m.d.Song.Patch, voiceIndex, addedLength, p...)
		if !ok {
			m.changeCancel = true
			return Range{}, Range{}, false
		}
	}
	if tracks {
		m.d.Song.Score.Tracks, trackRange, ok = VoiceInsert(m.d.Song.Score.Tracks, voiceIndex, addedLength, t...)
		if !ok {
			m.changeCancel = true
			return Range{}, Range{}, false
		}
	}
	return instrRange, trackRange, true
}

func (m *Model) changeNumVoice(voiceIndex, delta int, instruments, tracks bool) (ok bool) {
	defer m.change("setVoices", PatchChange, MinorChange)()
	if instruments {
		m.d.Song.Patch, ok = VoiceChange(m.d.Song.Patch, voiceIndex, delta)
		if !ok {
			m.changeCancel = true
			return false
		}
	}
	if tracks {
		m.d.Song.Score.Tracks, ok = VoiceChange(m.d.Song.Score.Tracks, voiceIndex, delta)
		if !ok {
			m.changeCancel = true
			return false
		}
	}
	return true
}
