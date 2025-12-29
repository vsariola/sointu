package tracker

import (
	"math"
)

type (
	// Int represents an integer value in the tracker model e.g. BPM, song
	// length, etc. It is a wrapper around an IntValue interface that provides
	// methods to manipulate the value, but Int guard that all changes are
	// within the range of the underlying IntValue implementation and that
	// SetValue is not called when the value is unchanged.
	Int struct {
		value IntValue
	}

	IntValue interface {
		Value() int
		SetValue(int) bool // returns true if the value was changed
		Range() IntRange
	}

	IntRange struct {
		Min, Max int
	}

	InstrumentVoices  Model
	TrackVoices       Model
	SongLength        Model
	BPM               Model
	RowsPerPattern    Model
	RowsPerBeat       Model
	Step              Model
	Octave            Model
	DetectorWeighting Model
	SyntherIndex      Model
	SpecAnSpeed       Model
	SpecAnResolution  Model
	SpecAnChannelsInt Model
)

func MakeInt(value IntValue) Int {
	return Int{value}
}

func (v Int) Add(delta int) (ok bool) {
	return v.SetValue(v.Value() + delta)
}

func (v Int) SetValue(value int) (ok bool) {
	r := v.Range()
	value = r.Clamp(value)
	if value == v.Value() || value < r.Min || value > r.Max {
		return false
	}
	return v.value.SetValue(value)
}

func (v Int) Range() IntRange {
	if v.value == nil {
		return IntRange{0, 0}
	}
	return v.value.Range()
}

func (v Int) Value() int {
	if v.value == nil {
		return 0
	}
	return v.value.Value()
}

func (r IntRange) Clamp(value int) int {
	return max(min(value, r.Max), r.Min)
}

// Model methods

func (m *Model) BPM() Int               { return MakeInt((*BPM)(m)) }
func (m *Model) InstrumentVoices() Int  { return MakeInt((*InstrumentVoices)(m)) }
func (m *Model) TrackVoices() Int       { return MakeInt((*TrackVoices)(m)) }
func (m *Model) SongLength() Int        { return MakeInt((*SongLength)(m)) }
func (m *Model) RowsPerPattern() Int    { return MakeInt((*RowsPerPattern)(m)) }
func (m *Model) RowsPerBeat() Int       { return MakeInt((*RowsPerBeat)(m)) }
func (m *Model) Step() Int              { return MakeInt((*Step)(m)) }
func (m *Model) Octave() Int            { return MakeInt((*Octave)(m)) }
func (m *Model) DetectorWeighting() Int { return MakeInt((*DetectorWeighting)(m)) }
func (m *Model) SyntherIndex() Int      { return MakeInt((*SyntherIndex)(m)) }
func (m *Model) SpecAnSpeed() Int       { return MakeInt((*SpecAnSpeed)(m)) }
func (m *Model) SpecAnResolution() Int  { return MakeInt((*SpecAnResolution)(m)) }
func (m *Model) SpecAnChannelsInt() Int { return MakeInt((*SpecAnChannelsInt)(m)) }

// BeatsPerMinuteInt

func (v *BPM) Value() int { return v.d.Song.BPM }
func (v *BPM) SetValue(value int) bool {
	defer (*Model)(v).change("BPMInt", SongChange, MinorChange)()
	v.d.Song.BPM = value
	return true
}
func (v *BPM) Range() IntRange { return IntRange{1, 999} }

// RowsPerPatternInt

func (v *RowsPerPattern) Value() int { return v.d.Song.Score.RowsPerPattern }
func (v *RowsPerPattern) SetValue(value int) bool {
	defer (*Model)(v).change("RowsPerPatternInt", SongChange, MinorChange)()
	v.d.Song.Score.RowsPerPattern = value
	return true
}
func (v *RowsPerPattern) Range() IntRange { return IntRange{1, 256} }

// SongLengthInt

func (v *SongLength) Value() int { return v.d.Song.Score.Length }
func (v *SongLength) SetValue(value int) bool {
	defer (*Model)(v).change("SongLengthInt", SongChange, MinorChange)()
	v.d.Song.Score.Length = value
	return true
}
func (v *SongLength) Range() IntRange { return IntRange{1, math.MaxInt32} }

// StepInt

func (v *Step) Value() int { return v.d.Step }
func (v *Step) SetValue(value int) bool {
	defer (*Model)(v).change("StepInt", NoChange, MinorChange)()
	v.d.Step = value
	return true
}
func (v *Step) Range() IntRange { return IntRange{0, 8} }

// OctaveInt

func (v *Octave) Value() int              { return v.d.Octave }
func (v *Octave) SetValue(value int) bool { v.d.Octave = value; return true }
func (v *Octave) Range() IntRange         { return IntRange{0, 9} }

// RowsPerBeatInt

func (v *RowsPerBeat) Value() int { return v.d.Song.RowsPerBeat }
func (v *RowsPerBeat) SetValue(value int) bool {
	defer (*Model)(v).change("RowsPerBeatInt", SongChange, MinorChange)()
	v.d.Song.RowsPerBeat = value
	return true
}
func (v *RowsPerBeat) Range() IntRange { return IntRange{1, 32} }

// ModelLoudnessType

func (v *DetectorWeighting) Value() int { return int(v.weightingType) }
func (v *DetectorWeighting) SetValue(value int) bool {
	v.weightingType = WeightingType(value)
	TrySend(v.broker.ToDetector, MsgToDetector{HasWeightingType: true, WeightingType: WeightingType(value)})
	return true
}
func (v *DetectorWeighting) Range() IntRange { return IntRange{0, int(NumLoudnessTypes) - 1} }

// SpecAn stuff

func (v *SpecAnSpeed) Value() int { return int(v.specAnSettings.Smooth) }
func (v *SpecAnSpeed) SetValue(value int) bool {
	v.specAnSettings.Smooth = value
	TrySend(v.broker.ToSpecAn, MsgToSpecAn{HasSettings: true, SpecSettings: v.specAnSettings})
	return true
}
func (v *SpecAnSpeed) Range() IntRange { return IntRange{SpecSpeedMin, SpecSpeedMax} }

func (v *SpecAnResolution) Value() int { return v.specAnSettings.Resolution }
func (v *SpecAnResolution) SetValue(value int) bool {
	v.specAnSettings.Resolution = value
	TrySend(v.broker.ToSpecAn, MsgToSpecAn{HasSettings: true, SpecSettings: v.specAnSettings})
	return true
}
func (v *SpecAnResolution) Range() IntRange { return IntRange{SpecResolutionMin, SpecResolutionMax} }

func (v *SpecAnChannelsInt) Value() int { return int(v.specAnSettings.ChnMode) }
func (v *SpecAnChannelsInt) SetValue(value int) bool {
	v.specAnSettings.ChnMode = SpecChnMode(value)
	TrySend(v.broker.ToSpecAn, MsgToSpecAn{HasSettings: true, SpecSettings: v.specAnSettings})
	return true
}
func (v *SpecAnChannelsInt) Range() IntRange { return IntRange{0, int(NumSpecChnModes) - 1} }

// InstrumentVoicesInt

func (v *InstrumentVoices) Value() int {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return 1
	}
	return max(v.d.Song.Patch[v.d.InstrIndex].NumVoices, 1)
}

func (m *InstrumentVoices) SetValue(value int) bool {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	defer (*Model)(m).change("InstrumentVoices", SongChange, MinorChange)()
	voiceIndex := m.d.Song.Patch.FirstVoiceForInstrument(m.d.InstrIndex)
	voiceRange := Range{voiceIndex, voiceIndex + m.d.Song.Patch[m.d.InstrIndex].NumVoices}
	ranges := MakeSetLength(voiceRange, value)
	ok := (*Model)(m).sliceInstrumentsTracks(true, m.linkInstrTrack, ranges...)
	if !ok {
		m.changeCancel = true
	}
	return ok
}

func (v *InstrumentVoices) Range() IntRange {
	return IntRange{1, (*Model)(v).remainingVoices(true, v.linkInstrTrack) + v.Value()}
}

// TrackVoicesInt

func (v *TrackVoices) Value() int {
	t := v.d.Cursor.Track
	if t < 0 || t >= len(v.d.Song.Score.Tracks) {
		return 1
	}
	return max(v.d.Song.Score.Tracks[t].NumVoices, 1)
}

func (m *TrackVoices) SetValue(value int) bool {
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

func (v *TrackVoices) Range() IntRange {
	t := v.d.Cursor.Track
	if t < 0 || t >= len(v.d.Song.Score.Tracks) {
		return IntRange{1, 1}
	}
	return IntRange{1, (*Model)(v).remainingVoices(v.linkInstrTrack, true) + v.d.Song.Score.Tracks[t].NumVoices}
}

// SyntherIndex

func (v *SyntherIndex) Value() int      { return v.syntherIndex }
func (v *SyntherIndex) Range() IntRange { return IntRange{0, len(v.synthers) - 1} }
func (v *Model) SyntherName() string    { return v.synthers[v.syntherIndex].Name() }
func (v *SyntherIndex) SetValue(value int) bool {
	if value < 0 || value >= len(v.synthers) {
		return false
	}
	v.syntherIndex = value
	TrySend(v.broker.ToPlayer, any(v.synthers[value]))
	return true
}
