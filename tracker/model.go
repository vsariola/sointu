package tracker

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
)

// Model implements the mutable state for the tracker program GUI.
//
// Go does not have immutable slices, so there's no efficient way to guarantee
// accidental mutations in the song. But at least the value members are
// protected.
type Model struct {
	song             sointu.Song
	selectionCorner  SongPoint
	cursor           SongPoint
	lowNibble        bool
	instrIndex       int
	unitIndex        int
	paramIndex       int
	octave           int
	noteTracking     bool
	usedIDs          map[int]bool
	maxID            int
	filePath         string
	changedSinceSave bool
	patternUseCount  [][]int

	prevUndoType    string
	undoSkipCounter int
	undoStack       []sointu.Song
	redoStack       []sointu.Song

	samplesPerRowObservers []chan<- int
	patchObservers         []chan<- sointu.Patch
	scoreObservers         []chan<- sointu.Score
	playingObservers       []chan<- bool
}

type Parameter struct {
	Type      ParameterType
	Name      string
	Hint      string
	Value     int
	Min       int
	Max       int
	LargeStep int
}

type ParameterType int

const (
	IntegerParameter ParameterType = iota
	BoolParameter
	IDParameter
)

const maxUndo = 256

func NewModel() *Model {
	ret := new(Model)
	ret.setSongNoUndo(defaultSong.Copy())
	return ret
}

func (m *Model) FilePath() string {
	return m.filePath
}

func (m *Model) SetFilePath(value string) {
	m.filePath = value
}

func (m *Model) ChangedSinceSave() bool {
	return m.changedSinceSave
}

func (m *Model) SetChangedSinceSave(value bool) {
	m.changedSinceSave = value
}

func (m *Model) ResetSong() {
	m.SetSong(defaultSong.Copy())
	m.filePath = ""
	m.changedSinceSave = false
}

func (m *Model) SetSong(song sointu.Song) {
	m.saveUndo("SetSong", 0)
	m.setSongNoUndo(song)
}

func (m *Model) SetOctave(value int) bool {
	if value < 0 {
		value = 0
	}
	if value > 9 {
		value = 9
	}
	if m.octave == value {
		return false
	}
	m.octave = value
	return true
}

func (m *Model) SetInstrument(instrument sointu.Instrument) bool {
	if len(instrument.Units) == 0 {
		return false
	}
	m.saveUndo("SetInstrument", 0)
	m.freeUnitIDs(m.song.Patch[m.instrIndex].Units)
	m.assignUnitIDs(instrument.Units)
	m.song.Patch[m.instrIndex] = instrument
	m.clampPositions()
	m.notifyPatchChange()
	return true
}

func (m *Model) SetInstrIndex(value int) {
	m.instrIndex = value
	m.clampPositions()
}

func (m *Model) SetInstrumentVoices(value int) {
	if value < 1 {
		value = 1
	}
	maxRemain := m.MaxInstrumentVoices()
	if value > maxRemain {
		value = maxRemain
	}
	if m.Instrument().NumVoices == value {
		return
	}
	m.saveUndo("SetInstrumentVoices", 10)
	m.song.Patch[m.instrIndex].NumVoices = value
	m.notifyPatchChange()
}

func (m *Model) MaxInstrumentVoices() int {
	maxRemain := 32 - m.song.Patch.NumVoices() + m.Instrument().NumVoices
	if maxRemain < 1 {
		return 1
	}
	return maxRemain
}

func (m *Model) SetInstrumentName(name string) {
	name = strings.TrimSpace(name)
	if m.Instrument().Name == name {
		return
	}
	m.saveUndo("SetInstrumentName", 10)
	m.song.Patch[m.instrIndex].Name = name
}

func (m *Model) SetInstrumentComment(comment string) {
	if m.Instrument().Comment == comment {
		return
	}
	m.saveUndo("SetInstrumentComment", 10)
	m.song.Patch[m.instrIndex].Comment = comment
}

func (m *Model) SetBPM(value int) {
	if value < 1 {
		value = 1
	}
	if value > 999 {
		value = 999
	}
	if m.song.BPM == value {
		return
	}
	m.saveUndo("SetBPM", 100)
	m.song.BPM = value
	m.notifySamplesPerRowChange()
}

func (m *Model) SetRowsPerBeat(value int) {
	if value < 1 {
		value = 1
	}
	if value > 32 {
		value = 32
	}
	if m.song.RowsPerBeat == value {
		return
	}
	m.saveUndo("SetRowsPerBeat", 10)
	m.song.RowsPerBeat = value
	m.notifySamplesPerRowChange()
}

func (m *Model) AddTrack(after bool) {
	if !m.CanAddTrack() {
		return
	}
	m.saveUndo("AddTrack", 0)
	newTracks := make([]sointu.Track, len(m.song.Score.Tracks)+1)
	if after {
		m.cursor.Track++
	}
	copy(newTracks, m.song.Score.Tracks[:m.cursor.Track])
	copy(newTracks[m.cursor.Track+1:], m.song.Score.Tracks[m.cursor.Track:])
	newTracks[m.cursor.Track] = sointu.Track{
		NumVoices: 1,
		Patterns:  []sointu.Pattern{},
	}
	m.song.Score.Tracks = newTracks
	m.clampPositions()
	m.notifyScoreChange()
}

func (m *Model) CanAddTrack() bool {
	return m.song.Score.NumVoices() < 32
}

func (m *Model) DeleteTrack(forward bool) {
	if !m.CanDeleteTrack() {
		return
	}
	m.saveUndo("DeleteTrack", 0)
	newTracks := make([]sointu.Track, len(m.song.Score.Tracks)-1)
	copy(newTracks, m.song.Score.Tracks[:m.cursor.Track])
	copy(newTracks[m.cursor.Track:], m.song.Score.Tracks[m.cursor.Track+1:])
	m.song.Score.Tracks = newTracks
	if !forward {
		m.cursor.Track--
	}
	m.selectionCorner = m.cursor
	m.clampPositions()
	m.computePatternUseCounts()
	m.notifyScoreChange()
}

func (m *Model) CanDeleteTrack() bool {
	return len(m.song.Score.Tracks) > 1
}

func (m *Model) SwapTracks(i, j int) {
	if i < 0 || j < 0 || i >= len(m.song.Score.Tracks) || j >= len(m.song.Score.Tracks) || i == j {
		return
	}
	m.saveUndo("SwapTracks", 10)
	tracks := m.song.Score.Tracks
	tracks[i], tracks[j] = tracks[j], tracks[i]
	m.clampPositions()
	m.notifyScoreChange()
}

func (m *Model) SetTrackVoices(value int) {
	if value < 1 {
		value = 1
	}
	maxRemain := m.MaxTrackVoices()
	if value > maxRemain {
		value = maxRemain
	}
	if m.song.Score.Tracks[m.cursor.Track].NumVoices == value {
		return
	}
	m.saveUndo("SetTrackVoices", 10)
	m.song.Score.Tracks[m.cursor.Track].NumVoices = value
	m.notifyScoreChange()
}

func (m *Model) MaxTrackVoices() int {
	maxRemain := 32 - m.song.Score.NumVoices() + m.song.Score.Tracks[m.cursor.Track].NumVoices
	if maxRemain < 1 {
		maxRemain = 1
	}
	return maxRemain
}

func (m *Model) AddInstrument(after bool) {
	if !m.CanAddInstrument() {
		return
	}
	m.saveUndo("AddInstrument", 0)
	newInstruments := make([]sointu.Instrument, len(m.song.Patch)+1)
	if after {
		m.instrIndex++
	}
	copy(newInstruments, m.song.Patch[:m.instrIndex])
	copy(newInstruments[m.instrIndex+1:], m.song.Patch[m.instrIndex:])
	newInstr := defaultInstrument.Copy()
	m.assignUnitIDs(newInstr.Units)
	newInstruments[m.instrIndex] = newInstr
	m.unitIndex = 0
	m.paramIndex = 0
	m.song.Patch = newInstruments
	m.notifyPatchChange()
}

func (m *Model) CanAddInstrument() bool {
	return m.song.Patch.NumVoices() < 32
}

func (m *Model) SwapInstruments(i, j int) {
	if i < 0 || j < 0 || i >= len(m.song.Patch) || j >= len(m.song.Patch) || i == j {
		return
	}
	m.saveUndo("SwapInstruments", 10)
	instruments := m.song.Patch
	instruments[i], instruments[j] = instruments[j], instruments[i]
	m.clampPositions()
	m.notifyPatchChange()
}

func (m *Model) DeleteInstrument(forward bool) {
	if !m.CanDeleteInstrument() {
		return
	}
	m.saveUndo("DeleteInstrument", 0)
	m.freeUnitIDs(m.song.Patch[m.instrIndex].Units)
	m.song.Patch = append(m.song.Patch[:m.instrIndex], m.song.Patch[m.instrIndex+1:]...)
	if (!forward && m.instrIndex > 0) || m.instrIndex >= len(m.song.Patch) {
		m.instrIndex--
	}
	m.clampPositions()
	m.notifyPatchChange()
}

func (m *Model) CanDeleteInstrument() bool {
	return len(m.song.Patch) > 1
}

func (m *Model) Note() byte {
	trk := m.song.Score.Tracks[m.cursor.Track]
	pat := trk.Order.Get(m.cursor.Pattern)
	if pat < 0 || pat >= len(trk.Patterns) {
		return 1
	}
	return trk.Patterns[pat].Get(m.cursor.Row)
}

// SetCurrentNote sets the (note) value in current pattern under cursor to iv
func (m *Model) SetNote(iv byte) {
	m.saveUndo("SetNote", 10)
	tracks := m.song.Score.Tracks
	if m.cursor.Pattern < 0 || m.cursor.Row < 0 {
		return
	}
	patIndex := tracks[m.cursor.Track].Order.Get(m.cursor.Pattern)
	if patIndex < 0 {
		patIndex = len(tracks[m.cursor.Track].Patterns)
		for _, pi := range tracks[m.cursor.Track].Order {
			if pi >= patIndex {
				patIndex = pi + 1 // we find a pattern that is not in the pattern table nor in the order list i.e. completely new pattern
			}
		}
		tracks[m.cursor.Track].Order.Set(m.cursor.Pattern, patIndex)
	}
	for len(tracks[m.cursor.Track].Patterns) <= patIndex {
		tracks[m.cursor.Track].Patterns = append(tracks[m.cursor.Track].Patterns, nil)
	}
	tracks[m.cursor.Track].Patterns[patIndex].Set(m.cursor.Row, iv)
	m.notifyScoreChange()
}

func (m *Model) SetCurrentPattern(pat int) {
	m.saveUndo("SetCurrentPattern", 0)
	m.song.Score.Tracks[m.cursor.Track].Order.Set(m.cursor.Pattern, pat)
	m.computePatternUseCounts()
	m.notifyScoreChange()
}

func (m *Model) IsPatternUnique(track, pattern int) bool {
	if track < 0 || track >= len(m.patternUseCount) {
		return false
	}
	p := m.patternUseCount[track]
	if pattern < 0 || pattern >= len(p) {
		return false
	}
	return p[pattern] <= 1
}

func (m *Model) SetSongLength(value int) {
	if value < 1 {
		value = 1
	}
	if value == m.song.Score.Length {
		return
	}
	m.saveUndo("SetSongLength", 10)
	m.song.Score.Length = value
	m.clampPositions()
	m.computePatternUseCounts()
	m.notifyScoreChange()
}

func (m *Model) SetRowsPerPattern(value int) {
	if value < 1 {
		value = 1
	}
	if value > 255 {
		value = 255
	}
	if value == m.song.Score.RowsPerPattern {
		return
	}
	m.saveUndo("SetRowsPerPattern", 10)
	m.song.Score.RowsPerPattern = value
	m.clampPositions()
	m.notifyScoreChange()
}

func (m *Model) SetUnitType(t string) {
	unit, ok := defaultUnits[t]
	if !ok { // if the type is invalid, we just set it to empty unit
		unit = sointu.Unit{Parameters: make(map[string]int)}
	} else {
		unit = unit.Copy()
	}
	if m.Unit().Type == unit.Type {
		return
	}
	m.saveUndo("SetUnitType", 0)
	oldID := m.Unit().ID
	m.Instrument().Units[m.unitIndex] = unit
	m.Instrument().Units[m.unitIndex].ID = oldID // keep the ID of the replaced unit
	m.notifyPatchChange()
}

func (m *Model) SetUnitIndex(value int) {
	m.unitIndex = value
	m.paramIndex = 0
	m.clampPositions()
}

func (m *Model) AddUnit(after bool) {
	m.saveUndo("AddUnit", 10)
	newUnits := make([]sointu.Unit, len(m.Instrument().Units)+1)
	if after {
		m.unitIndex++
	}
	copy(newUnits, m.Instrument().Units[:m.unitIndex])
	copy(newUnits[m.unitIndex+1:], m.Instrument().Units[m.unitIndex:])
	m.assignUnitIDs(newUnits[m.unitIndex : m.unitIndex+1])
	m.song.Patch[m.instrIndex].Units = newUnits
	m.paramIndex = 0
	m.clampPositions()
	m.notifyPatchChange()
}

func (m *Model) AddOrderRow(after bool) {
	m.saveUndo("AddOrderRow", 10)
	if after {
		m.cursor.Pattern++
	}
	for i, trk := range m.song.Score.Tracks {
		if l := len(trk.Order); l > m.cursor.Pattern {
			newOrder := make([]int, l+1)
			copy(newOrder, trk.Order[:m.cursor.Pattern])
			copy(newOrder[m.cursor.Pattern+1:], trk.Order[m.cursor.Pattern:])
			newOrder[m.cursor.Pattern] = -1
			m.song.Score.Tracks[i].Order = newOrder
		}
	}
	m.song.Score.Length++
	m.selectionCorner = m.cursor
	m.clampPositions()
	m.computePatternUseCounts()
	m.notifyScoreChange()
}

func (m *Model) DeleteOrderRow(forward bool) {
	if m.song.Score.Length <= 1 {
		return
	}
	m.saveUndo("DeleteOrderRow", 0)
	for i, trk := range m.song.Score.Tracks {
		if l := len(trk.Order); l > m.cursor.Pattern {
			newOrder := make([]int, l-1)
			copy(newOrder, trk.Order[:m.cursor.Pattern])
			copy(newOrder[m.cursor.Pattern:], trk.Order[m.cursor.Pattern+1:])
			m.song.Score.Tracks[i].Order = newOrder
		}
	}
	if !forward && m.cursor.Pattern > 0 {
		m.cursor.Pattern--
	}
	m.song.Score.Length--
	m.selectionCorner = m.cursor
	m.clampPositions()
	m.computePatternUseCounts()
	m.notifyScoreChange()
}

func (m *Model) DeleteUnit(forward bool) {
	if !m.CanDeleteUnit() {
		return
	}
	instr := m.Instrument()
	m.saveUndo("DeleteUnit", 0)
	delete(m.usedIDs, instr.Units[m.unitIndex].ID)
	newUnits := make([]sointu.Unit, len(instr.Units)-1)
	copy(newUnits, instr.Units[:m.unitIndex])
	copy(newUnits[m.unitIndex:], instr.Units[m.unitIndex+1:])
	m.song.Patch[m.instrIndex].Units = newUnits
	if !forward && m.unitIndex > 0 {
		m.unitIndex--
	}
	m.paramIndex = 0
	m.clampPositions()
	m.notifyPatchChange()
}

func (m *Model) CanDeleteUnit() bool {
	return len(m.Instrument().Units) > 1
}

func (m *Model) ResetParam() {
	p, err := m.Param(m.paramIndex)
	if err != nil {
		return
	}
	unit := m.Unit()
	paramList, ok := sointu.UnitTypes[unit.Type]
	if !ok || m.paramIndex < 0 || m.paramIndex >= len(paramList) {
		return
	}
	paramType := paramList[m.paramIndex]
	defaultValue, ok := defaultUnits[unit.Type].Parameters[paramType.Name]
	if unit.Parameters[p.Name] == defaultValue {
		return
	}
	m.saveUndo("ResetParam", 0)
	unit.Parameters[paramType.Name] = defaultValue
	m.clampPositions()
	m.notifyPatchChange()
}

func (m *Model) SetParamIndex(value int) {
	m.paramIndex = value
	m.clampPositions()
}

func (m *Model) setGmDlsEntry(index int) {
	if index < 0 || index >= len(GmDlsEntries) {
		return
	}
	entry := GmDlsEntries[index]
	unit := m.Unit()
	if unit.Type != "oscillator" || unit.Parameters["type"] != sointu.Sample {
		return
	}
	if unit.Parameters["samplestart"] == entry.Start && unit.Parameters["loopstart"] == entry.LoopStart && unit.Parameters["looplength"] == entry.LoopLength {
		return
	}
	m.saveUndo("SetGmDlsEntry", 20)
	unit.Parameters["samplestart"] = entry.Start
	unit.Parameters["loopstart"] = entry.LoopStart
	unit.Parameters["looplength"] = entry.LoopLength
	unit.Parameters["transpose"] = 64 + entry.SuggestedTranspose
	m.notifyPatchChange()
}

func (m *Model) SwapUnits(i, j int) {
	units := m.Instrument().Units
	if i < 0 || j < 0 || i >= len(units) || j >= len(units) || i == j {
		return
	}
	m.saveUndo("SwapUnits", 10)
	units[i], units[j] = units[j], units[i]
	m.clampPositions()
	m.notifyPatchChange()
}

func (m *Model) getSelectionRange() (int, int, int, int) {
	r1 := m.cursor.Pattern*m.song.Score.RowsPerPattern + m.cursor.Row
	r2 := m.selectionCorner.Pattern*m.song.Score.RowsPerPattern + m.selectionCorner.Row
	if r2 < r1 {
		r1, r2 = r2, r1
	}
	t1 := m.cursor.Track
	t2 := m.selectionCorner.Track
	if t2 < t1 {
		t1, t2 = t2, t1
	}
	return r1, r2, t1, t2
}

func (m *Model) AdjustSelectionPitch(delta int) {
	m.saveUndo("AdjustSelectionPitch", 10)
	r1, r2, t1, t2 := m.getSelectionRange()
	for c := t1; c <= t2; c++ {
		adjustedNotes := map[struct {
			Pat int
			Row int
		}]bool{}
		for r := r1; r <= r2; r++ {
			s := SongRow{Row: r}.Wrap(m.song.Score)
			if s.Pattern >= len(m.song.Score.Tracks[c].Order) {
				break
			}
			p := m.song.Score.Tracks[c].Order[s.Pattern]
			if p < 0 {
				continue
			}
			noteIndex := struct {
				Pat int
				Row int
			}{p, s.Row}
			if !adjustedNotes[noteIndex] {
				patterns := m.song.Score.Tracks[c].Patterns
				if p >= len(patterns) {
					continue
				}
				pattern := patterns[p]
				if s.Row >= len(pattern) {
					continue
				}
				if val := pattern[s.Row]; val > 1 {
					newVal := int(val) + delta
					if newVal < 2 {
						newVal = 2
					} else if newVal > 255 {
						newVal = 255
					}
					pattern[s.Row] = byte(newVal)
				}
				adjustedNotes[noteIndex] = true
			}
		}
	}
	m.notifyScoreChange()
}

func (m *Model) DeleteSelection() {
	m.saveUndo("DeleteSelection", 0)
	r1, r2, t1, t2 := m.getSelectionRange()
	for r := r1; r <= r2; r++ {
		s := SongRow{Row: r}.Wrap(m.song.Score)
		for c := t1; c <= t2; c++ {
			if len(m.song.Score.Tracks[c].Order) <= s.Pattern {
				continue
			}
			p := m.song.Score.Tracks[c].Order[s.Pattern]
			if p < 0 {
				continue
			}
			patterns := m.song.Score.Tracks[c].Patterns
			if p >= len(patterns) {
				continue
			}
			pattern := patterns[p]
			if s.Row >= len(pattern) {
				continue
			}
			m.song.Score.Tracks[c].Patterns[p][s.Row] = 1
		}
	}
	m.notifyScoreChange()
}

func (m *Model) DeletePatternSelection() {
	m.saveUndo("DeletePatternSelection", 0)
	r1, r2, t1, t2 := m.getSelectionRange()
	p1 := SongRow{Row: r1}.Wrap(m.song.Score).Pattern
	p2 := SongRow{Row: r2}.Wrap(m.song.Score).Pattern
	for p := p1; p <= p2; p++ {
		for c := t1; c <= t2; c++ {
			if p < len(m.song.Score.Tracks[c].Order) {
				m.song.Score.Tracks[c].Order[p] = -1
			}
		}
	}
	m.computePatternUseCounts()
	m.notifyScoreChange()
}

func (m *Model) Undo() {
	if !m.CanUndo() {
		return
	}
	if len(m.redoStack) >= maxUndo {
		m.redoStack = m.redoStack[1:]
	}
	m.redoStack = append(m.redoStack, m.song.Copy())
	m.setSongNoUndo(m.undoStack[len(m.undoStack)-1])
	m.undoStack = m.undoStack[:len(m.undoStack)-1]
}

func (m *Model) CanUndo() bool {
	return len(m.undoStack) > 0
}

func (m *Model) ClearUndoHistory() {
	if len(m.undoStack) > 0 {
		m.undoStack = m.undoStack[:0]
	}
	if len(m.redoStack) > 0 {
		m.redoStack = m.redoStack[:0]
	}
}

func (m *Model) Redo() {
	if !m.CanRedo() {
		return
	}
	if len(m.undoStack) >= maxUndo {
		m.undoStack = m.undoStack[1:]
	}
	m.undoStack = append(m.undoStack, m.song.Copy())
	m.setSongNoUndo(m.redoStack[len(m.redoStack)-1])
	m.redoStack = m.redoStack[:len(m.redoStack)-1]
}

func (m *Model) CanRedo() bool {
	return len(m.redoStack) > 0
}

func (m *Model) SetNoteTracking(value bool) {
	m.noteTracking = value
}

func (m *Model) NoteTracking() bool {
	return m.noteTracking
}

func (m *Model) Octave() int {
	return m.octave
}

func (m *Model) Song() sointu.Song {
	return m.song
}

func (m *Model) SelectionCorner() SongPoint {
	return m.selectionCorner
}

func (m *Model) SetSelectionCorner(value SongPoint) {
	m.selectionCorner = value
	m.clampPositions()
}

func (m *Model) Cursor() SongPoint {
	return m.cursor
}

func (m *Model) SetCursor(value SongPoint) {
	m.cursor = value
	m.clampPositions()
}

func (m *Model) LowNibble() bool {
	return m.lowNibble
}

func (m *Model) SetLowNibble(value bool) {
	m.lowNibble = value
}

func (m *Model) InstrIndex() int {
	return m.instrIndex
}

func (m *Model) Track() sointu.Track {
	return m.song.Score.Tracks[m.cursor.Track]
}

func (m *Model) Instrument() sointu.Instrument {
	return m.song.Patch[m.instrIndex]
}

func (m *Model) Unit() sointu.Unit {
	return m.song.Patch[m.instrIndex].Units[m.unitIndex]
}

func (m *Model) UnitIndex() int {
	return m.unitIndex
}

func (m *Model) ParamIndex() int {
	return m.paramIndex
}

func (m *Model) clampPositions() {
	m.cursor = m.cursor.Wrap(m.song.Score)
	m.selectionCorner = m.selectionCorner.Wrap(m.song.Score)
	if !m.Track().Effect {
		m.lowNibble = false
	}
	m.instrIndex = clamp(m.instrIndex, 0, len(m.song.Patch)-1)
	m.unitIndex = clamp(m.unitIndex, 0, len(m.Instrument().Units)-1)
	m.paramIndex = clamp(m.paramIndex, 0, m.NumParams()-1)
}

func (m *Model) NumParams() int {
	unit := m.Unit()
	if unit.Type == "oscillator" {
		if unit.Parameters["type"] != sointu.Sample {
			return 10
		}
		return 14
	}
	numSettableParams := 0
	for _, t := range sointu.UnitTypes[m.Unit().Type] {
		if t.CanSet {
			numSettableParams++
		}
	}
	if numSettableParams == 0 {
		numSettableParams = 1
	}
	if unit.Type == "delay" {
		numSettableParams += 1 + len(unit.VarArgs)
		if len(unit.VarArgs)%2 == 1 && unit.Parameters["stereo"] == 1 {
			numSettableParams++
		}
	}
	return numSettableParams
}

func (m *Model) Param(index int) (Parameter, error) {
	unit := m.Unit()
	for _, t := range sointu.UnitTypes[unit.Type] {
		if !t.CanSet {
			continue
		}
		if index != 0 {
			index--
			continue
		}
		typ := IntegerParameter
		if t.MaxValue == t.MinValue+1 {
			typ = BoolParameter
		}
		val := m.Unit().Parameters[t.Name]
		name := t.Name
		hint := m.song.Patch.ParamHintString(m.instrIndex, m.unitIndex, name)
		var text string
		if hint != "" {
			text = fmt.Sprintf("%v / %v", val, hint)
		} else {
			text = strconv.Itoa(val)
		}
		min, max := t.MinValue, t.MaxValue
		if unit.Type == "send" {
			if t.Name == "voice" {
				i, _, err := m.song.Patch.FindSendTarget(unit.Parameters["target"])
				if err == nil {
					max = m.song.Patch[i].NumVoices
				}
			} else if t.Name == "target" {
				typ = IDParameter
			}
		}
		largeStep := 16
		if unit.Type == "oscillator" && t.Name == "transpose" {
			largeStep = 12
		}
		return Parameter{Type: typ, Min: min, Max: max, Name: name, Hint: text, Value: val, LargeStep: largeStep}, nil
	}
	if unit.Type == "oscillator" && index == 0 {
		key := vm.SampleOffset{Start: uint32(unit.Parameters["samplestart"]), LoopStart: uint16(unit.Parameters["loopstart"]), LoopLength: uint16(unit.Parameters["looplength"])}
		val := 0
		hint := "0 / custom"
		if v, ok := GmDlsEntryMap[key]; ok {
			val = v + 1
			hint = fmt.Sprintf("%v / %v", val, GmDlsEntries[v].Name)
		}
		return Parameter{Type: IntegerParameter, Min: 0, Max: len(GmDlsEntries), Name: "sample", Hint: hint, Value: val}, nil
	}
	if unit.Type == "delay" {
		if index == 0 {
			l := len(unit.VarArgs)
			if unit.Parameters["stereo"] == 1 {
				l = (l + 1) / 2
			}
			return Parameter{Type: IntegerParameter, Min: 1, Max: 32, Name: "delaylines", Hint: strconv.Itoa(l), Value: l}, nil
		}
		index--
		if index < len(unit.VarArgs) {
			val := unit.VarArgs[index]
			var text string
			if unit.Parameters["notetracking"] == 1 {
				relPitch := float64(val) / 10787
				semitones := -math.Log2(relPitch) * 12
				text = fmt.Sprintf("%v / %.3f st", val, semitones)
			} else {
				text = fmt.Sprintf("%v / %.3f rows", val, float32(val)/float32(m.song.SamplesPerRow()))
			}
			return Parameter{Type: IntegerParameter, Min: 1, Max: 65535, Name: "delaytime", Hint: text, Value: val, LargeStep: 256}, nil
		}
	}
	return Parameter{}, errors.New("invalid parameter")
}

func (m *Model) RemoveUnusedData() {
	m.saveUndo("RemoveUnusedData", 0)
	for trkIndex, trk := range m.song.Score.Tracks {
		// assign new indices to patterns
		newIndex := map[int]int{}
		runningIndex := 0
		length := 0
		if len(trk.Order) > m.song.Score.Length {
			trk.Order = trk.Order[:m.song.Score.Length]
		}
		for i, p := range trk.Order {
			// if the pattern hasn't been considered and is within limits
			if _, ok := newIndex[p]; !ok && p >= 0 && p < len(trk.Patterns) {
				pat := trk.Patterns[p]
				useful := false
				for _, n := range pat { // patterns that have anything else than all holds are useful and to be kept
					if n != 1 {
						useful = true
						break
					}
				}
				if useful {
					newIndex[p] = runningIndex
					runningIndex++
				} else {
					newIndex[p] = -1
				}
			}
			if ind, ok := newIndex[p]; ok && ind > -1 {
				length = i + 1
				trk.Order[i] = ind
			} else {
				trk.Order[i] = -1
			}
		}
		trk.Order = trk.Order[:length]
		newPatterns := make([]sointu.Pattern, runningIndex)
		for i, pat := range trk.Patterns {
			if ind, ok := newIndex[i]; ok && ind > -1 {
				patLength := 0
				for j, note := range pat { // find last note that is something else that hold
					if note != 1 {
						patLength = j + 1
					}
				}
				if patLength > m.song.Score.RowsPerPattern {
					patLength = m.song.Score.RowsPerPattern
				}
				newPatterns[ind] = pat[:patLength] // crop to either RowsPerPattern or last row having something else than hold
			}
		}
		trk.Patterns = newPatterns
		m.song.Score.Tracks[trkIndex] = trk
	}
	m.computePatternUseCounts()
	m.notifyScoreChange()
}

func (m *Model) SetParam(value int) {
	p, err := m.Param(m.paramIndex)
	if err != nil {
		return
	}
	if value < p.Min {
		value = p.Min
	} else if value > p.Max {
		value = p.Max
	}
	if p.Name == "sample" {
		m.setGmDlsEntry(value - 1)
		return
	}
	unit := m.Unit()
	if p.Name == "delaylines" {
		m.saveUndo("SetParam", 20)
		targetLines := value
		if unit.Parameters["stereo"] == 1 {
			targetLines *= 2
		}
		for len(m.Instrument().Units[m.unitIndex].VarArgs) < targetLines {
			m.Instrument().Units[m.unitIndex].VarArgs = append(m.Instrument().Units[m.unitIndex].VarArgs, 1)
		}
		m.Instrument().Units[m.unitIndex].VarArgs = m.Instrument().Units[m.unitIndex].VarArgs[:targetLines]
	} else if p.Name == "delaytime" {
		m.saveUndo("SetParam", 20)
		index := m.paramIndex - 7
		for len(m.Instrument().Units[m.unitIndex].VarArgs) <= index {
			m.Instrument().Units[m.unitIndex].VarArgs = append(m.Instrument().Units[m.unitIndex].VarArgs, 1)
		}
		m.Instrument().Units[m.unitIndex].VarArgs[index] = value
	} else {
		if unit.Parameters[p.Name] == value {
			return
		}
		m.saveUndo("SetParam", 20)
		unit.Parameters[p.Name] = value
	}
	m.clampPositions()
	m.notifyPatchChange()
}

func (m *Model) AddPatchObserver(observer chan<- sointu.Patch) {
	m.patchObservers = append(m.patchObservers, observer)
}

func (m *Model) AddScoreObserver(observer chan<- sointu.Score) {
	m.scoreObservers = append(m.scoreObservers, observer)
}

func (m *Model) AddSamplesPerRowObserver(observer chan<- int) {
	m.samplesPerRowObservers = append(m.samplesPerRowObservers, observer)
}

func (m *Model) AddPlayingObserver(observer chan<- bool) {
	m.playingObservers = append(m.playingObservers, observer)
}

func (m *Model) setSongNoUndo(song sointu.Song) {
	m.song = song
	m.usedIDs = make(map[int]bool)
	m.maxID = 0
	for _, instr := range m.song.Patch {
		for _, unit := range instr.Units {
			if m.maxID < unit.ID {
				m.maxID = unit.ID
			}
		}
	}
	for _, instr := range m.song.Patch {
		m.assignUnitIDs(instr.Units)
	}
	m.clampPositions()
	m.computePatternUseCounts()
	m.notifySamplesPerRowChange()
	m.notifyPatchChange()
	m.notifyScoreChange()
}

func (m *Model) notifyPatchChange() {
	for _, channel := range m.patchObservers {
		channel <- m.song.Patch.Copy()
	}
}

func (m *Model) notifyScoreChange() {
	for _, channel := range m.scoreObservers {
		channel <- m.song.Score.Copy()
	}
}

func (m *Model) notifySamplesPerRowChange() {
	for _, channel := range m.samplesPerRowObservers {
		channel <- m.song.SamplesPerRow()
	}
}

func (m *Model) saveUndo(undoType string, undoSkipping int) {
	if m.prevUndoType == undoType && m.undoSkipCounter < undoSkipping {
		m.undoSkipCounter++
		return
	}
	m.changedSinceSave = true
	m.prevUndoType = undoType
	m.undoSkipCounter = 0
	if len(m.undoStack) >= maxUndo {
		m.undoStack = m.undoStack[1:]
	}
	m.undoStack = append(m.undoStack, m.song.Copy())
	m.redoStack = m.redoStack[:0]
}

func (m *Model) freeUnitIDs(units []sointu.Unit) {
	for _, u := range units {
		delete(m.usedIDs, u.ID)
	}
}

func (m *Model) assignUnitIDs(units []sointu.Unit) {
	for i := range units {
		if units[i].ID == 0 || m.usedIDs[units[i].ID] {
			m.maxID++
			units[i].ID = m.maxID
		}
		m.usedIDs[units[i].ID] = true
		if m.maxID < units[i].ID {
			m.maxID = units[i].ID
		}
	}
}

func (m *Model) computePatternUseCounts() {
	for i, track := range m.song.Score.Tracks {
		for len(m.patternUseCount) <= i {
			m.patternUseCount = append(m.patternUseCount, nil)
		}
		for j := range m.patternUseCount[i] {
			m.patternUseCount[i][j] = 0
		}
		for j := 0; j < m.song.Score.Length; j++ {
			if j >= len(track.Order) {
				break
			}
			p := track.Order[j]
			for len(m.patternUseCount[i]) <= p {
				m.patternUseCount[i] = append(m.patternUseCount[i], 0)
			}
			if p < 0 {
				continue
			}
			m.patternUseCount[i][p]++
		}
	}
}

func clamp(a, min, max int) int {
	if a < min {
		return min
	}
	if a > max {
		return max
	}
	return a
}
