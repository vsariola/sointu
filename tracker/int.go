package tracker

import (
	"math"

	"github.com/vsariola/sointu/vm"
)

type (
	Int struct {
		IntData
	}

	IntData interface {
		Value() int
		Range() intRange

		setValue(int)
		change(kind string) func()
	}

	intRange struct {
		Min, Max int
	}

	InstrumentVoices Model
	TrackVoices      Model
	SongLength       Model
	BPM              Model
	RowsPerPattern   Model
	RowsPerBeat      Model
	Step             Model
	Octave           Model
)

func (v Int) Add(delta int) (ok bool) {
	r := v.Range()
	value := r.Clamp(v.Value() + delta)
	if value == v.Value() || value < r.Min || value > r.Max {
		return false
	}
	defer v.change("Add")()
	v.setValue(value)
	return true
}

func (v Int) Set(value int) (ok bool) {
	r := v.Range()
	value = v.Range().Clamp(value)
	if value == v.Value() || value < r.Min || value > r.Max {
		return false
	}
	defer v.change("Set")()
	v.setValue(value)
	return true
}

func (r intRange) Clamp(value int) int {
	return max(min(value, r.Max), r.Min)
}

// Model methods

func (m *Model) InstrumentVoices() *InstrumentVoices { return (*InstrumentVoices)(m) }
func (m *Model) TrackVoices() *TrackVoices           { return (*TrackVoices)(m) }
func (m *Model) SongLength() *SongLength             { return (*SongLength)(m) }
func (m *Model) BPM() *BPM                           { return (*BPM)(m) }
func (m *Model) RowsPerPattern() *RowsPerPattern     { return (*RowsPerPattern)(m) }
func (m *Model) RowsPerBeat() *RowsPerBeat           { return (*RowsPerBeat)(m) }
func (m *Model) Step() *Step                         { return (*Step)(m) }
func (m *Model) Octave() *Octave                     { return (*Octave)(m) }

// BeatsPerMinuteInt

func (v *BPM) Int() Int           { return Int{v} }
func (v *BPM) Value() int         { return v.d.Song.BPM }
func (v *BPM) setValue(value int) { v.d.Song.BPM = value }
func (v *BPM) Range() intRange    { return intRange{1, 999} }
func (v *BPM) change(kind string) func() {
	return (*Model)(v).change("BPMInt."+kind, SongChange, MinorChange)
}

// RowsPerPatternInt

func (v *RowsPerPattern) Int() Int           { return Int{v} }
func (v *RowsPerPattern) Value() int         { return v.d.Song.Score.RowsPerPattern }
func (v *RowsPerPattern) setValue(value int) { v.d.Song.Score.RowsPerPattern = value }
func (v *RowsPerPattern) Range() intRange    { return intRange{1, 256} }
func (v *RowsPerPattern) change(kind string) func() {
	return (*Model)(v).change("RowsPerPatternInt."+kind, SongChange, MinorChange)
}

// SongLengthInt

func (v *SongLength) Int() Int           { return Int{v} }
func (v *SongLength) Value() int         { return v.d.Song.Score.Length }
func (v *SongLength) setValue(value int) { v.d.Song.Score.Length = value }
func (v *SongLength) Range() intRange    { return intRange{1, math.MaxInt32} }
func (v *SongLength) change(kind string) func() {
	return (*Model)(v).change("SongLengthInt."+kind, SongChange, MinorChange)
}

// StepInt

func (v *Step) Int() Int           { return Int{v} }
func (v *Step) Value() int         { return v.d.Step }
func (v *Step) setValue(value int) { v.d.Step = value }
func (v *Step) Range() intRange    { return intRange{0, 8} }
func (v *Step) change(kind string) func() {
	return (*Model)(v).change("StepInt"+kind, NoChange, MinorChange)
}

// OctaveInt

func (v *Octave) Int() Int                  { return Int{v} }
func (v *Octave) Value() int                { return v.d.Octave }
func (v *Octave) setValue(value int)        { v.d.Octave = value }
func (v *Octave) Range() intRange           { return intRange{0, 9} }
func (v *Octave) change(kind string) func() { return func() {} }

// RowsPerBeatInt

func (v *RowsPerBeat) Int() Int           { return Int{v} }
func (v *RowsPerBeat) Value() int         { return v.d.Song.RowsPerBeat }
func (v *RowsPerBeat) setValue(value int) { v.d.Song.RowsPerBeat = value }
func (v *RowsPerBeat) Range() intRange    { return intRange{1, 32} }
func (v *RowsPerBeat) change(kind string) func() {
	return (*Model)(v).change("RowsPerBeatInt."+kind, SongChange, MinorChange)
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

func (v *InstrumentVoices) setValue(value int) {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return
	}
	v.d.Song.Patch[v.d.InstrIndex].NumVoices = value
}

func (v *InstrumentVoices) Range() intRange {
	return intRange{1, vm.MAX_VOICES - v.d.Song.Patch.NumVoices() + v.Value()}
}

func (v *InstrumentVoices) change(kind string) func() {
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

func (v *TrackVoices) setValue(value int) {
	t := v.d.Cursor.Track
	if t < 0 || t >= len(v.d.Song.Score.Tracks) {
		return
	}
	v.d.Song.Score.Tracks[t].NumVoices = value
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
