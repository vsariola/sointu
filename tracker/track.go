package tracker

import (
	"fmt"
	"math"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
)

// Track returns the Track view of the model, containing methods to manipulate
// the tracks.
func (m *Model) Track() *TrackModel { return (*TrackModel)(m) }

type TrackModel Model

// LinkInstrument returns a Bool controlling whether instruments and tracks are
// linked.
func (m *TrackModel) LinkInstrument() Bool { return MakeBoolFromPtr(&m.linkInstrTrack) }

// Title returns the title of the track for a given index.
func (m *TrackModel) Item(index int) TrackListItem {
	if index < 0 || index >= len(m.derived.tracks) {
		return TrackListItem{}
	}
	return TrackListItem{m.derived.tracks[index].title, m.d.Song.Score.Tracks[index].Effect}
}

type TrackListItem struct {
	Title  string
	Effect bool
}

// Add returns an Action to add a new track.
func (m *TrackModel) Add() Action { return MakeAction((*addTrack)(m)) }

type addTrack TrackModel

func (m *addTrack) Enabled() bool { return m.d.Song.Score.NumVoices() < vm.MAX_VOICES }
func (m *addTrack) Do() {
	defer (*Model)(m).change("AddTrack", SongChange, MajorChange)()
	voiceIndex := m.d.Song.Score.FirstVoiceForTrack(m.d.Cursor.Track)
	p := sointu.Patch{defaultInstrument.Copy()}
	t := []sointu.Track{{NumVoices: 1}}
	_, _, ok := (*Model)(m).addVoices(voiceIndex, p, t, (*Model)(m).linkInstrTrack, true)
	m.changeCancel = !ok
}

// Delete returns an Action to delete the selected track(s).
func (m *TrackModel) Delete() Action { return MakeAction((*deleteTrack)(m)) }

type deleteTrack TrackModel

func (m *deleteTrack) Enabled() bool { return len(m.d.Song.Score.Tracks) > 0 }
func (m *deleteTrack) Do()           { (*TrackModel)(m).List().DeleteElements(false) }

// Split returns an Action to split the selected track into two tracks,
// distributing the voices as evenly as possible.
func (m *TrackModel) Split() Action { return MakeAction((*splitTrack)(m)) }

type splitTrack TrackModel

func (m *splitTrack) Enabled() bool {
	return m.d.Cursor.Track >= 0 && m.d.Cursor.Track < len(m.d.Song.Score.Tracks) && m.d.Song.Score.Tracks[m.d.Cursor.Track].NumVoices > 1
}
func (m *splitTrack) Do() {
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
	newTrack := sointu.Track{NumVoices: end - middle}
	m.d.Song.Score.Tracks = append(left, newTrack)
	m.d.Song.Score.Tracks = append(m.d.Song.Score.Tracks, right...)
}

// Effect returns a Bool to toggle whether the currently selected track is an
// effect track and should be displayed as hexadecimals or not.
func (m *TrackModel) Effect() Bool { return MakeBool((*trackEffect)(m)) }

type trackEffect TrackModel

func (m *trackEffect) Value() bool {
	if m.d.Cursor.Track < 0 || m.d.Cursor.Track >= len(m.d.Song.Score.Tracks) {
		return false
	}
	return m.d.Song.Score.Tracks[m.d.Cursor.Track].Effect
}
func (m *trackEffect) SetValue(val bool) {
	if m.d.Cursor.Track < 0 || m.d.Cursor.Track >= len(m.d.Song.Score.Tracks) {
		return
	}
	m.d.Song.Score.Tracks[m.d.Cursor.Track].Effect = val
}

// Voices returns an Int to adjust the number of voices for the currently
// selected track.
func (m *TrackModel) Voices() Int { return MakeInt((*trackVoices)(m)) }

type trackVoices TrackModel

func (v *trackVoices) Value() int {
	t := v.d.Cursor.Track
	if t < 0 || t >= len(v.d.Song.Score.Tracks) {
		return 1
	}
	return max(v.d.Song.Score.Tracks[t].NumVoices, 1)
}
func (m *trackVoices) SetValue(value int) bool {
	defer (*Model)(m).change("TrackVoices", SongChange, MinorChange)()
	voiceIndex := m.d.Song.Score.FirstVoiceForTrack(m.d.Cursor.Track)
	voiceRange := Range{voiceIndex, voiceIndex + m.d.Song.Score.Tracks[m.d.Cursor.Track].NumVoices}
	ranges := MakeSetLength(voiceRange, value)
	ok := (*Model)(m).sliceInstrumentsTracks(m.linkInstrTrack, true, ranges...)
	if !ok {
		m.changeCancel = true
	}
	return ok
}
func (v *trackVoices) Range() RangeInclusive {
	t := v.d.Cursor.Track
	if t < 0 || t >= len(v.d.Song.Score.Tracks) {
		return RangeInclusive{1, 1}
	}
	return RangeInclusive{1, (*Model)(v).remainingVoices(v.linkInstrTrack, true) + v.d.Song.Score.Tracks[t].NumVoices}
}

// List returns a List of all the tracks, implementing MutableListData
func (m *TrackModel) List() List { return List{(*trackList)(m)} }

type trackList TrackModel

func (v *trackList) Selected() int          { return v.d.Cursor.Track }
func (v *trackList) Selected2() int         { return v.d.Cursor2.Track }
func (v *trackList) SetSelected(value int)  { v.d.Cursor.Track = value }
func (v *trackList) SetSelected2(value int) { v.d.Cursor2.Track = value }
func (v *trackList) Count() int             { return len((*Model)(v).d.Song.Score.Tracks) }

func (v *trackList) Move(r Range, delta int) (ok bool) {
	voiceDelta := 0
	if delta < 0 {
		voiceDelta = -VoiceRange(v.d.Song.Score.Tracks, Range{r.Start + delta, r.Start}).Len()
	} else if delta > 0 {
		voiceDelta = VoiceRange(v.d.Song.Score.Tracks, Range{r.End, r.End + delta}).Len()
	}
	if voiceDelta == 0 {
		return false
	}
	ranges := MakeMoveRanges(VoiceRange(v.d.Song.Score.Tracks, r), voiceDelta)
	return (*Model)(v).sliceInstrumentsTracks(v.linkInstrTrack, true, ranges[:]...)
}

func (v *trackList) Delete(r Range) (ok bool) {
	ranges := Complement(VoiceRange(v.d.Song.Score.Tracks, r))
	return (*Model)(v).sliceInstrumentsTracks(v.linkInstrTrack, true, ranges[:]...)
}

func (v *trackList) Change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("TrackList."+n, SongChange, severity)
}

func (v *trackList) Cancel() {
	v.changeCancel = true
}

func (v *trackList) Marshal(r Range) ([]byte, error) {
	return (*Model)(v).marshalVoices(VoiceRange(v.d.Song.Score.Tracks, r))
}

func (m *trackList) Unmarshal(data []byte) (r Range, err error) {
	voiceIndex := m.d.Song.Score.FirstVoiceForTrack(m.d.Cursor.Track)
	_, r, ok := (*Model)(m).unmarshalVoices(voiceIndex, data, m.linkInstrTrack, true)
	if !ok {
		return Range{}, fmt.Errorf("unmarshal: unmarshalVoices failed")
	}
	return r, nil
}
