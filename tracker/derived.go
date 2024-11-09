package tracker

import (
	"github.com/vsariola/sointu"
	"iter"
	"slices"
)

/*
	from modelData we can derive useful information that can be cached for performance
	or easy access, because of nested iterations over the score or patch data.
	i.e. this needs to update when the model changes, and only then.
*/

type (
	derivedForUnit struct {
		unit       *sointu.Unit
		instrument *sointu.Instrument
		sends      []*sointu.Unit
	}

	derivedForTrack struct {
		instrumentRange          []int
		tracksWithSameInstrument []int
		title                    string
	}

	derivedModelData struct {
		// map unit by ID
		forUnit map[int]derivedForUnit
		// map track by index
		forTrack map[int]derivedForTrack
	}
)

// public access functions

func (m *Model) forUnitById(id int) *derivedForUnit {
	forUnit, ok := m.derived.forUnit[id]
	if !ok {
		return nil
	}
	return &forUnit
}

func (m *Model) InstrumentForUnit(id int) *sointu.Instrument {
	fu := m.forUnitById(id)
	if fu == nil {
		return nil
	}
	return fu.instrument
}

func (m *Model) UnitById(id int) *sointu.Unit {
	fu := m.forUnitById(id)
	if fu == nil {
		return nil
	}
	return fu.unit
}

func (m *Model) SendTargetsForUnit(id int) []*sointu.Unit {
	fu := m.forUnitById(id)
	if fu == nil {
		return nil
	}
	return fu.sends
}

func (m *Model) forTrackByIndex(index int) *derivedForTrack {
	forTrack, ok := m.derived.forTrack[index]
	if !ok {
		return nil
	}
	return &forTrack
}

func (m *Model) TrackTitle(index int) string {
	ft := m.forTrackByIndex(index)
	if ft == nil {
		return ""
	}
	return ft.title
}

// public getters with further model information

func (m *Model) TracksWithSameInstrumentAsCurrent() []int {
	currentTrack := m.d.Cursor.Track
	return m.derived.forTrack[currentTrack].tracksWithSameInstrument
}

func (m *Model) CountNextTracksForCurrentInstrument() int {
	currentTrack := m.d.Cursor.Track
	count := 0
	for t := range m.TracksWithSameInstrumentAsCurrent() {
		if t > currentTrack {
			count++
		}
	}
	return count
}

// init / update methods

func (m *Model) initDerivedData() {
	m.derived = derivedModelData{
		forUnit:  make(map[int]derivedForUnit),
		forTrack: make(map[int]derivedForTrack),
	}
	m.updateDerivedScoreData()
	m.updateDerivedPatchData()
}

func (m *Model) updateDerivedScoreData() {
	for index, _ := range m.d.Song.Score.Tracks {
		firstInstr, lastInstr, _ := m.instrumentRangeFor(index)
		m.derived.forTrack[index] = derivedForTrack{
			instrumentRange:          []int{firstInstr, lastInstr},
			tracksWithSameInstrument: slices.Collect(m.tracksWithSameInstrument(index)),
			title:                    m.buildTrackTitle(index),
		}
	}
}

func (m *Model) updateDerivedPatchData() {
	for _, instr := range m.d.Song.Patch {
		for _, unit := range instr.Units {
			m.derived.forUnit[unit.ID] = derivedForUnit{
				unit:       &unit,
				instrument: &instr,
				sends:      slices.Collect(m.collectSendsTo(unit)),
			}
		}
	}
}

// internals...

func (m *Model) collectSendsTo(unit sointu.Unit) iter.Seq[*sointu.Unit] {
	return func(yield func(*sointu.Unit) bool) {
		for _, instr := range m.d.Song.Patch {
			for _, u := range instr.Units {
				if u.Type != "send" {
					continue
				}
				targetId, ok := u.Parameters["target"]
				if !ok || targetId != unit.ID {
					continue
				}
				if !yield(&u) {
					return
				}
			}
		}
	}
}

func (m *Model) instrumentRangeFor(trackIndex int) (int, int, error) {
	track := m.d.Song.Score.Tracks[trackIndex]
	firstVoice := m.d.Song.Score.FirstVoiceForTrack(trackIndex)
	lastVoice := firstVoice + track.NumVoices - 1
	firstIndex, err1 := m.d.Song.Patch.InstrumentForVoice(firstVoice)
	if err1 != nil {
		return trackIndex, trackIndex, err1
	}
	lastIndex, err2 := m.d.Song.Patch.InstrumentForVoice(lastVoice)
	if err2 != nil {
		return trackIndex, trackIndex, err2
	}
	return firstIndex, lastIndex, nil
}

func (m *Model) buildTrackTitle(x int) (title string) {
	title = "?"
	if x < 0 || x >= len(m.d.Song.Score.Tracks) {
		return
	}
	firstIndex, lastIndex, err := m.instrumentRangeFor(x)
	if err != nil {
		return
	}
	switch diff := lastIndex - firstIndex; diff {
	case 0:
		title = m.d.Song.Patch[firstIndex].Name
	default:
		n1 := m.d.Song.Patch[firstIndex].Name
		n2 := m.d.Song.Patch[firstIndex+1].Name
		if len(n1) > 0 {
			n1 = string(n1[0])
		} else {
			n1 = "?"
		}
		if len(n2) > 0 {
			n2 = string(n2[0])
		} else {
			n2 = "?"
		}
		if diff > 1 {
			title = n1 + "/" + n2 + "..."
		} else {
			title = n1 + "/" + n2
		}
	}
	return
}

func (m *Model) instrumentForTrack(trackIndex int) (int, bool) {
	voiceIndex := m.d.Song.Score.FirstVoiceForTrack(trackIndex)
	instrument, err := m.d.Song.Patch.InstrumentForVoice(voiceIndex)
	return instrument, err == nil
}

func (m *Model) tracksWithSameInstrument(trackIndex int) iter.Seq[int] {
	return func(yield func(int) bool) {

		currentInstrument, currentExists := m.instrumentForTrack(trackIndex)
		if !currentExists {
			return
		}

		for i := 0; i < len(m.d.Song.Score.Tracks); i++ {
			instrument, exists := m.instrumentForTrack(i)
			if !exists {
				return
			}
			if instrument != currentInstrument {
				continue
			}
			if !yield(i) {
				return
			}
		}
	}
}
