package tracker

import (
	"fmt"
	"time"

	"github.com/vsariola/sointu"
)

type (
	Rail struct {
		PassThrough int
		Send        bool
		StackUse    sointu.StackUse
	}

	Wire struct {
		From      int
		FromSet   bool
		To        Point
		ToSet     bool
		Hint      string
		Highlight bool
	}

	RailError struct {
		InstrIndex, UnitIndex int
		Err                   error
	}

	// derivedModelData contains useful information derived from the modelData,
	// cached for performance and/or easy access. This needs to be updated when
	// corresponding part of the model changes.
	derivedModelData struct {
		// map Unit by ID, other entities by their respective index
		patch        []derivedInstrument
		tracks       []derivedTrack
		railError    RailError
		presetSearch derivedPresetSearch
	}

	derivedInstrument struct {
		wires       []Wire
		rails       []Rail
		railWidth   int
		params      [][]Parameter
		paramsWidth int
	}

	derivedTrack struct {
		title            string
		patternUseCounts []int
	}
)

// public methods to access the derived data

func (s *Model) RailError() RailError { return s.derived.railError }

func (s *Model) RailWidth() int {
	i := s.d.InstrIndex
	if i < 0 || i >= len(s.derived.patch) {
		return 0
	}
	return s.derived.patch[i].railWidth
}

func (m *Model) Wires(yield func(wire Wire) bool) {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.derived.patch) {
		return
	}
	for _, wire := range m.derived.patch[i].wires {
		wire.Highlight = (wire.FromSet && m.d.UnitIndex == wire.From) || (wire.ToSet && m.d.UnitIndex == wire.To.Y && m.d.ParamIndex == wire.To.X)
		if !yield(wire) {
			return
		}
	}
}

func (m *Model) TrackTitle(index int) string {
	if index < 0 || index >= len(m.derived.tracks) {
		return ""
	}
	return m.derived.tracks[index].title
}

func (m *Model) PatternUnique(track, pat int) bool {
	if track < 0 || track >= len(m.derived.tracks) {
		return false
	}
	if pat < 0 || pat >= len(m.derived.tracks[track].patternUseCounts) {
		return false
	}
	return m.derived.tracks[track].patternUseCounts[pat] <= 1
}

func (e *RailError) Error() string { return e.Err.Error() }

func (s *Rail) StackAfter() int { return s.PassThrough + s.StackUse.NumOutputs }

// init / update methods

func (m *Model) updateDeriveData(changeType ChangeType) {
	setSliceLength(&m.derived.tracks, len(m.d.Song.Score.Tracks))
	if changeType&ScoreChange != 0 {
		for index, track := range m.d.Song.Score.Tracks {
			m.derived.tracks[index].patternUseCounts = m.buildPatternUseCounts(track)
		}
	}
	if changeType&ScoreChange != 0 || changeType&PatchChange != 0 {
		for index := range m.d.Song.Score.Tracks {
			m.derived.tracks[index].title = m.buildTrackTitle(index)
		}
	}
	setSliceLength(&m.derived.patch, len(m.d.Song.Patch))
	if changeType&PatchChange != 0 {
		m.updateParams()
		m.updateRails()
		m.updateWires()
	}
}

func (m *Model) updateParams() {
	for i, instr := range m.d.Song.Patch {
		setSliceLength(&m.derived.patch[i].params, len(instr.Units))
		paramsWidth := 0
		for u := range instr.Units {
			p := m.deriveParams(&instr.Units[u], m.derived.patch[i].params[u])
			m.derived.patch[i].params[u] = p
			paramsWidth = max(paramsWidth, len(p))
		}
		m.derived.patch[i].paramsWidth = paramsWidth
	}
}

func (m *Model) deriveParams(unit *sointu.Unit, ret []Parameter) []Parameter {
	ret = ret[:0] // reset the slice
	unitType, ok := sointu.UnitTypes[unit.Type]
	if !ok {
		return ret
	}
	portIndex := 0
	for i, up := range unitType {
		if !up.CanSet && !up.CanModulate {
			continue // skip parameters that cannot be set or modulated
		}
		if unit.Type == "oscillator" && unit.Parameters["type"] != sointu.Sample && (up.Name == "samplestart" || up.Name == "loopstart" || up.Name == "looplength") {
			continue // don't show the sample related params unless necessary
		}
		if unit.Type == "send" && up.Name == "port" {
			continue
		}
		q := 0
		if up.CanModulate {
			portIndex++
			q = portIndex
		}
		ret = append(ret, Parameter{m: m, unit: unit, up: &unitType[i], vtable: &namedParameter{}, port: q})
	}
	if unit.Type == "oscillator" && unit.Parameters["type"] == sointu.Sample {
		ret = append(ret, Parameter{m: m, unit: unit, vtable: &gmDlsEntryParameter{}})
	}
	if unit.Type == "delay" {
		if unit.Parameters["stereo"] == 1 && len(unit.VarArgs)%2 == 1 {
			unit.VarArgs = append(unit.VarArgs, 1)
		}
		ret = append(ret,
			Parameter{m: m, unit: unit, vtable: &reverbParameter{}},
			Parameter{m: m, unit: unit, vtable: &delayLinesParameter{}})
		for i := range unit.VarArgs {
			ret = append(ret, Parameter{m: m, unit: unit, index: i, vtable: &delayTimeParameter{}})
		}
	}
	return ret
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

func (m *Model) buildTrackTitle(track int) string {
	if track < 0 || track >= len(m.d.Song.Score.Tracks) {
		return "?"
	}
	firstIndex, lastIndex, err := m.instrumentRangeFor(track)
	if err != nil {
		return "?"
	}
	switch diff := lastIndex - firstIndex; diff {
	case 0:
		return nilIsQuestionMark(m.d.Song.Patch[firstIndex].Name)
	case 1:
		return fmt.Sprintf("%s/%s",
			nilIsQuestionMark(m.d.Song.Patch[firstIndex].Name),
			nilIsQuestionMark(m.d.Song.Patch[firstIndex+1].Name))
	default:
		return fmt.Sprintf("%s/%s/...",
			nilIsQuestionMark(m.d.Song.Patch[firstIndex].Name),
			nilIsQuestionMark(m.d.Song.Patch[firstIndex+1].Name))
	}
}

func nilIsQuestionMark(s string) string {
	if len(s) == 0 {
		return "?"
	}
	return s
}

func (m *Model) buildPatternUseCounts(track sointu.Track) []int {
	result := make([]int, 0, len(track.Patterns))
	for j := range min(len(track.Order), m.d.Song.Score.Length) {
		if p := track.Order[j]; p >= 0 {
			for len(result) <= p {
				result = append(result, 0)
			}
			result[p]++
		}
	}
	return result
}

func (m *Model) updateRails() {
	type stackElem struct{ instr, unit int }
	scratchArray := [32]stackElem{}
	scratch := scratchArray[:0]
	m.derived.railError = RailError{}
	for i, instr := range m.d.Song.Patch {
		setSliceLength(&m.derived.patch[i].rails, len(instr.Units))
		start := len(scratch)
		maxWidth := 0
		for u, unit := range instr.Units {
			stackUse := unit.StackUse()
			numInputs := len(stackUse.Inputs)
			if len(scratch) < numInputs {
				if m.derived.railError == (RailError{}) {
					m.derived.railError = RailError{
						InstrIndex: i,
						UnitIndex:  u,
						Err:        fmt.Errorf("%s unit in instrument %d / %s needs %d inputs, but got only %d", unit.Type, i, instr.Name, numInputs, len(scratch)),
					}
				}
				scratch = scratch[:0]
			} else {
				scratch = scratch[:len(scratch)-numInputs]
			}
			m.derived.patch[i].rails[u] = Rail{
				PassThrough: len(scratch),
				StackUse:    stackUse,
				Send:        !unit.Disabled && unit.Type == "send",
			}
			maxWidth = max(maxWidth, len(scratch)+max(len(stackUse.Inputs), stackUse.NumOutputs))
			for range stackUse.NumOutputs {
				scratch = append(scratch, stackElem{instr: i, unit: u})
			}
		}
		m.derived.patch[i].railWidth = maxWidth
		diff := len(scratch) - start
		if instr.NumVoices > 1 && diff != 0 {
			if diff < 0 {
				morepop := (instr.NumVoices - 1) * diff
				if morepop > len(scratch) {
					if m.derived.railError == (RailError{}) {
						m.derived.railError = RailError{
							InstrIndex: i,
							UnitIndex:  -1,
							Err:        fmt.Errorf("each voice of instrument %d / %s consumes %d signals, but there was not enough signals available", i, instr.Name, -diff),
						}
					}
					scratch = scratch[:0]
				} else {
					scratch = scratch[:len(scratch)-morepop]
				}
			} else {
				for range (instr.NumVoices - 1) * diff {
					scratch = append(scratch, scratch[len(scratch)-diff])
				}
			}
		}
	}
	if len(scratch) > 0 && m.derived.railError == (RailError{}) {
		patch := m.d.Song.Patch
		m.derived.railError = RailError{
			InstrIndex: scratch[0].instr,
			UnitIndex:  scratch[0].unit,
			Err:        fmt.Errorf("instrument %d / %s unit %d / %s leaves a signal on stack", scratch[0].instr, patch[scratch[0].instr].Name, scratch[0].unit, patch[scratch[0].instr].Units[scratch[0].unit].Type),
		}
	}
	if m.derived.railError.Err != nil {
		m.Alerts().AddAlert(Alert{
			Name:     "RailError",
			Message:  m.derived.railError.Error(),
			Priority: Error,
			Duration: time.Minute,
		})
	} else { // clear the alert if it was set
		m.Alerts().AddAlert(Alert{Name: "RailError"})
	}
}

func (m *Model) updateWires() {
	for i := range m.d.Song.Patch {
		m.derived.patch[i].wires = m.derived.patch[i].wires[:0] // reset the wires
	}
	for i, instr := range m.d.Song.Patch {
		for u, unit := range instr.Units {
			if unit.Disabled || unit.Type != "send" {
				continue
			}
			tI, tU, err := m.d.Song.Patch.FindUnit(unit.Parameters["target"])
			if err != nil {
				continue
			}
			up, tX, ok := sointu.FindParamForModulationPort(m.d.Song.Patch[tI].Units[tU].Type, unit.Parameters["port"])
			if !ok {
				continue
			}
			if tI == i {
				// local send
				m.derived.patch[i].wires = append(m.derived.patch[i].wires, Wire{
					From:    u,
					FromSet: true,
					To:      Point{X: tX, Y: tU},
					ToSet:   true,
				})
			} else {
				// remote send
				m.derived.patch[i].wires = append(m.derived.patch[i].wires, Wire{
					From:    u,
					FromSet: true,
					Hint:    fmt.Sprintf("To instrument #%d (%s), unit #%d (%s), port %s", tI, m.d.Song.Patch[tI].Name, tU, m.d.Song.Patch[tI].Units[tU].Type, up.Name),
				})
				toPt := Point{X: tX, Y: tU}
				hint := fmt.Sprintf("From instrument #%d (%s), send #%d", i, m.d.Song.Patch[i].Name, u)
				for i, w := range m.derived.patch[tI].wires {
					if !w.FromSet && w.ToSet && w.To == toPt {
						m.derived.patch[tI].wires[i].Hint += "; " + hint
						goto skipAppend
					}
				}
				m.derived.patch[tI].wires = append(m.derived.patch[tI].wires, Wire{
					To:    toPt,
					ToSet: true,
					Hint:  hint,
				})
			skipAppend:
			}
		}
	}
}
