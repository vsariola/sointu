package tracker

import (
	"fmt"
	"iter"
	"slices"

	"github.com/vsariola/sointu"
)

/*
	from modelData we can derive useful information that can be cached for performance
	or easy access, because of nested iterations over the score or patch data.
	i.e. this needs to update when the model changes, and only then.
*/

type (
	derivedModelData struct {
		// map Unit by ID, other entities by their respective index
		forUnit    map[int]derivedForUnit
		forTrack   []derivedForTrack
		forPattern []derivedForPattern
	}

	derivedForUnit struct {
		unit            sointu.Unit
		instrument      sointu.Instrument
		instrumentIndex int
		unitIndex       int

		// map param by Name
		forParameter map[string]derivedForParameter
	}

	derivedForParameter struct {
		sendTooltip string
		sendSources []sendSourceData
	}

	sendSourceData struct {
		unitId          int
		paramName       string
		amount          int
		instrumentIndex int
		instrumentName  string
	}

	derivedForTrack struct {
		instrumentRange          []int
		tracksWithSameInstrument []int
		title                    string
	}

	derivedForPattern struct {
		useCount []int
	}
)

// public access functions

func (m *Model) InstrumentForUnit(id int) (sointu.Instrument, int, bool) {
	forUnit, ok := m.derived.forUnit[id]
	if !ok {
		return sointu.Instrument{}, -1, false
	}
	return forUnit.instrument, forUnit.instrumentIndex, true
}

func (m *Model) UnitInfo(id int) (instrName string, units []sointu.Unit, unitIndex int, ok bool) {
	fu, ok := m.derived.forUnit[id]
	return fu.instrument.Name, fu.instrument.Units, fu.unitIndex, ok
}

func (m *Model) UnitHintInfo(id int) (instrIndex int, unitType string, ok bool) {
	fu, ok := m.derived.forUnit[id]
	return fu.instrumentIndex, fu.unit.Type, ok
}

func (m *Model) ParameterInfo(unitId int, paramName string) (tooltip string, ok bool) {
	du, ok1 := m.derived.forUnit[unitId]
	if !ok1 {
		return "", false
	}
	dp, ok2 := du.forParameter[paramName]
	if !ok2 {
		return "", false
	}
	return dp.sendTooltip, len(dp.sendSources) > 0
}

func (m *Model) TrackTitle(index int) string {
	if index < 0 || index >= len(m.derived.forTrack) {
		return ""
	}
	return m.derived.forTrack[index].title
}

func (m *Model) PatternUseCount(index int) []int {
	if index < 0 || index >= len(m.derived.forPattern) {
		return nil
	}
	return m.derived.forPattern[index].useCount
}

func (m *Model) PatternUnique(t, p int) bool {
	if t < 0 || t >= len(m.derived.forPattern) {
		return false
	}
	forPattern := m.derived.forPattern[t]
	if p < 0 || p >= len(forPattern.useCount) {
		return false
	}
	return forPattern.useCount[p] == 1
}

// public getters with further model information

func (m *Model) TracksWithSameInstrumentAsCurrent() []int {
	currentTrack := m.d.Cursor.Track
	if currentTrack >= len(m.derived.forTrack) {
		return nil
	}
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
		forUnit:    make(map[int]derivedForUnit),
		forTrack:   make([]derivedForTrack, 0),
		forPattern: make([]derivedForPattern, 0),
	}
	m.updateDerivedScoreData()
	m.updateDerivedPatchData()
}

func (m *Model) updateDerivedScoreData() {
	m.derived.forTrack = m.derived.forTrack[:0]
	m.derived.forPattern = m.derived.forPattern[:0]
	for index, track := range m.d.Song.Score.Tracks {
		firstInstr, lastInstr, _ := m.instrumentRangeFor(index)
		m.derived.forTrack = append(
			m.derived.forTrack,
			derivedForTrack{
				instrumentRange:          []int{firstInstr, lastInstr},
				tracksWithSameInstrument: slices.Collect(m.tracksWithSameInstrument(index)),
				title:                    m.buildTrackTitle(index),
			},
		)
		m.derived.forPattern = append(
			m.derived.forPattern,
			derivedForPattern{
				useCount: m.calcPatternUseCounts(track),
			},
		)
	}
}

func (m *Model) updateDerivedPatchData() {
	clear(m.derived.forUnit)
	for i, instr := range m.d.Song.Patch {
		for u, unit := range instr.Units {
			m.derived.forUnit[unit.ID] = derivedForUnit{
				unit:            unit,
				unitIndex:       u,
				instrument:      instr,
				instrumentIndex: i,
				forParameter:    make(map[string]derivedForParameter),
			}
			m.updateDerivedParameterData(unit)
		}
	}
}

func (m *Model) updateDerivedParameterData(unit sointu.Unit) {
	fu, _ := m.derived.forUnit[unit.ID]
	for name := range fu.unit.Parameters {
		sendSources := slices.Collect(m.collectSendSources(unit, name))
		fu.forParameter[name] = derivedForParameter{
			sendSources: sendSources,
			sendTooltip: m.buildSendTargetTooltip(fu.instrumentIndex, sendSources),
		}
	}
}

// internals...

func (m *Model) collectSendSources(unit sointu.Unit, paramName string) iter.Seq[sendSourceData] {
	return func(yield func(sendSourceData) bool) {
		for i, instr := range m.d.Song.Patch {
			for _, u := range instr.Units {
				if u.Type != "send" {
					continue
				}
				targetId, ok := u.Parameters["target"]
				if !ok || targetId != unit.ID {
					continue
				}
				port := u.Parameters["port"]
				unitParam, ok := sointu.FindParamForModulationPort(unit.Type, port)
				if !ok || unitParam.Name != paramName {
					continue
				}
				sourceData := sendSourceData{
					unitId:          u.ID,
					paramName:       paramName,
					instrumentIndex: i,
					instrumentName:  instr.Name,
					amount:          u.Parameters["amount"],
				}
				if !yield(sourceData) {
					return
				}
			}
		}
	}
}

func (m *Model) buildSendTargetTooltip(ownInstrIndex int, sendSources []sendSourceData) string {
	if len(sendSources) == 0 {
		return ""
	}
	amounts := ""
	for _, sendSource := range sendSources {
		sourceInfo := ""
		if sendSource.instrumentIndex != ownInstrIndex {
			sourceInfo = fmt.Sprintf(" from \"%s\"", sendSource.instrumentName)
		}
		if amounts == "" {
			amounts = fmt.Sprintf("x %d%s", sendSource.amount, sourceInfo)
		} else {
			amounts = fmt.Sprintf("%s, x %d%s", amounts, sendSource.amount, sourceInfo)
		}
	}
	count := "1 send"
	if len(sendSources) > 1 {
		count = fmt.Sprintf("%d sends", len(sendSources))
	}
	return fmt.Sprintf("%s [%s]", count, amounts)
}

func (m *Model) instrumentRangeFor(trackIndex int) (int, int, error) {
	track := m.d.Song.Score.Tracks[trackIndex]
	if track.NumVoices <= 0 {
		return 0, 0, fmt.Errorf("track %d has no voices", trackIndex)
	}
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

func (m *Model) calcPatternUseCounts(track sointu.Track) []int {
	result := make([]int, len(m.d.Song.Score.Tracks))
	for j, _ := range result {
		result[j] = 0
	}
	for j := 0; j < m.d.Song.Score.Length; j++ {
		if j >= len(track.Order) {
			break
		}
		p := track.Order[j]
		for len(result) <= p {
			result = append(result, 0)
		}
		if p < 0 {
			continue
		}
		result[p]++
	}
	return result
}
