package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
	"golang.org/x/exp/slices"
)

// Model implements the mutable state for the tracker program GUI.
//
// Go does not have immutable slices, so there's no efficient way to guarantee
// accidental mutations in the song. But at least the value members are
// protected.
// It is owned by the GUI thread (goroutine), while the player is owned by
// by the audioprocessing thread. They communicate using the two channels
type (
	// modelData is the part of the model that gets save to recovery file
	modelData struct {
		Song                 sointu.Song
		SelectionCorner      ScorePoint
		Cursor               ScorePoint
		LowNibble            bool
		InstrIndex           int
		UnitIndex            int
		ParamIndex           int
		Octave               int
		NoteTracking         bool
		UsedIDs              map[int]bool
		MaxID                int
		FilePath             string
		ChangedSinceSave     bool
		PatternUseCount      [][]int
		Panic                bool
		Playing              bool
		Recording            bool
		PlayPosition         ScoreRow
		InstrEnlarged        bool
		RecoveryFilePath     string
		ChangedSinceRecovery bool

		PrevUndoType    string
		UndoSkipCounter int
		UndoStack       []sointu.Song
		RedoStack       []sointu.Song
	}

	Model struct {
		d              modelData
		PlayerMessages <-chan PlayerMessage
		modelMessages  chan<- interface{}
	}

	ModelPlayingChangedMessage struct {
		bool
	}

	ModelPlayFromPositionMessage struct {
		ScoreRow
	}

	ModelBPMChangedMessage struct {
		int
	}

	ModelRowsPerBeatChangedMessage struct {
		int
	}

	ModelPanicMessage struct {
		bool
	}

	ModelRecordingMessage struct {
		bool
	}

	ModelNoteOnMessage struct {
		NoteID
	}

	ModelNoteOffMessage struct {
		NoteID
	}
)

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

const maxUndo = 64
const RECOVERY_FILE = ".sointu_recovery"

func NewModel(modelMessages chan<- interface{}, playerMessages <-chan PlayerMessage, recoveryFilePath string) *Model {
	ret := new(Model)
	ret.modelMessages = modelMessages
	ret.PlayerMessages = playerMessages
	ret.setSongNoUndo(defaultSong.Copy())
	ret.d.Octave = 4
	ret.d.RecoveryFilePath = recoveryFilePath
	if recoveryFilePath != "" {
		if bytes2, err := os.ReadFile(ret.d.RecoveryFilePath); err == nil {
			json.Unmarshal(bytes2, &ret.d)
			ret.send(ret.d.Song.Copy())
		}
	}
	return ret
}

func (m *Model) MarshalRecovery() []byte {
	out, err := json.Marshal(m.d)
	if err != nil {
		return nil
	}
	if m.d.RecoveryFilePath != "" {
		os.Remove(m.d.RecoveryFilePath)
	}
	m.d.ChangedSinceRecovery = false
	return out
}

func (m *Model) SaveRecovery() error {
	if !m.d.ChangedSinceRecovery {
		return nil
	}
	if m.d.RecoveryFilePath == "" {
		return errors.New("no backup file path")
	}
	out, err := json.Marshal(m.d)
	if err != nil {
		return fmt.Errorf("could not marshal recovery data: %w", err)
	}
	dir := filepath.Dir(m.d.RecoveryFilePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, os.ModePerm)
	}
	file, err := os.Create(m.d.RecoveryFilePath)
	if err != nil {
		return fmt.Errorf("could not create recovery file: %w", err)
	}
	_, err = file.Write(out)
	if err != nil {
		return fmt.Errorf("could not write recovery file: %w", err)
	}
	m.d.ChangedSinceRecovery = false
	return nil
}

func (m *Model) UnmarshalRecovery(bytes []byte) {
	err := json.Unmarshal(bytes, &m.d)
	if err != nil {
		return
	}
	if m.d.RecoveryFilePath != "" { // check if there's a recovery file on disk and load it instead
		if bytes2, err := os.ReadFile(m.d.RecoveryFilePath); err == nil {
			json.Unmarshal(bytes2, &m.d)
		}
	}
	m.d.ChangedSinceRecovery = false
	m.send(m.d.Song.Copy())
}

func (m *Model) FilePath() string {
	return m.d.FilePath
}

func (m *Model) SetFilePath(value string) {
	m.d.FilePath = value
}

func (m *Model) ChangedSinceSave() bool {
	return m.d.ChangedSinceSave
}

func (m *Model) SetChangedSinceSave(value bool) {
	m.d.ChangedSinceSave = value
}

func (m *Model) ResetSong() {
	m.SetSong(defaultSong.Copy())
	m.d.FilePath = ""
	m.d.ChangedSinceSave = false
}

func (m *Model) SetSong(song sointu.Song) {
	// guard for malformed songs
	if len(song.Score.Tracks) == 0 || song.Score.Length <= 0 || len(song.Patch) == 0 {
		return
	}
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
	if m.d.Octave == value {
		return false
	}
	m.d.Octave = value
	return true
}

func (m *Model) ProcessPlayerMessage(msg PlayerMessage) {
	m.d.PlayPosition = msg.SongRow
	m.d.Panic = msg.Panic
	switch e := msg.Inner.(type) {
	case Recording:
		if e.BPM == 0 {
			e.BPM = float64(m.d.Song.BPM)
		}
		song, err := e.Song(m.d.Song.Patch, m.d.Song.RowsPerBeat, m.d.Song.Score.RowsPerPattern)
		if err != nil {
			break
		}
		m.SetSong(song)
		m.d.InstrEnlarged = false
	default:
	}
}

func (m *Model) SetInstrument(instrument sointu.Instrument) bool {
	if len(instrument.Units) == 0 {
		return false
	}
	m.saveUndo("SetInstrument", 0)
	m.freeUnitIDs(m.d.Song.Patch[m.d.InstrIndex].Units)
	m.assignUnitIDs(instrument.Units)
	m.d.Song.Patch[m.d.InstrIndex] = instrument
	m.clampPositions()
	m.send(m.d.Song.Patch.Copy())
	return true
}

func (m *Model) SetInstrIndex(value int) {
	m.d.InstrIndex = value
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
	m.d.Song.Patch[m.d.InstrIndex].NumVoices = value
	m.send(m.d.Song.Patch.Copy())
}

func (m *Model) MaxInstrumentVoices() int {
	maxRemain := 32 - m.d.Song.Patch.NumVoices() + m.Instrument().NumVoices
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
	m.d.Song.Patch[m.d.InstrIndex].Name = name
}

func (m *Model) SetInstrumentComment(comment string) {
	if m.Instrument().Comment == comment {
		return
	}
	m.saveUndo("SetInstrumentComment", 10)
	m.d.Song.Patch[m.d.InstrIndex].Comment = comment
}

func (m *Model) SetBPM(value int) {
	if value < 1 {
		value = 1
	}
	if value > 999 {
		value = 999
	}
	if m.d.Song.BPM == value {
		return
	}
	m.saveUndo("SetBPM", 100)
	m.d.Song.BPM = value
	m.send(ModelBPMChangedMessage{value})
}

func (m *Model) SetRowsPerBeat(value int) {
	if value < 1 {
		value = 1
	}
	if value > 32 {
		value = 32
	}
	if m.d.Song.RowsPerBeat == value {
		return
	}
	m.saveUndo("SetRowsPerBeat", 10)
	m.d.Song.RowsPerBeat = value
	m.send(ModelRowsPerBeatChangedMessage{value})
}

func (m *Model) AddTrack(after bool) {
	if !m.CanAddTrack() {
		return
	}
	m.saveUndo("AddTrack", 0)
	newTracks := make([]sointu.Track, len(m.d.Song.Score.Tracks)+1)
	if after {
		m.d.Cursor.Track++
	}
	copy(newTracks, m.d.Song.Score.Tracks[:m.d.Cursor.Track])
	copy(newTracks[m.d.Cursor.Track+1:], m.d.Song.Score.Tracks[m.d.Cursor.Track:])
	newTracks[m.d.Cursor.Track] = sointu.Track{
		NumVoices: 1,
		Patterns:  []sointu.Pattern{},
	}
	m.d.Song.Score.Tracks = newTracks
	m.clampPositions()
	m.send(m.d.Song.Score.Copy())
}

func (m *Model) CanAddTrack() bool {
	return m.d.Song.Score.NumVoices() < 32
}

func (m *Model) DeleteTrack(forward bool) {
	if !m.CanDeleteTrack() {
		return
	}
	m.saveUndo("DeleteTrack", 0)
	newTracks := make([]sointu.Track, len(m.d.Song.Score.Tracks)-1)
	copy(newTracks, m.d.Song.Score.Tracks[:m.d.Cursor.Track])
	copy(newTracks[m.d.Cursor.Track:], m.d.Song.Score.Tracks[m.d.Cursor.Track+1:])
	m.d.Song.Score.Tracks = newTracks
	if !forward {
		m.d.Cursor.Track--
	}
	m.d.SelectionCorner = m.d.Cursor
	m.clampPositions()
	m.computePatternUseCounts()
	m.send(m.d.Song.Score.Copy())
}

func (m *Model) CanDeleteTrack() bool {
	return len(m.d.Song.Score.Tracks) > 1
}

func (m *Model) SwapTracks(i, j int) {
	if i < 0 || j < 0 || i >= len(m.d.Song.Score.Tracks) || j >= len(m.d.Song.Score.Tracks) || i == j {
		return
	}
	m.saveUndo("SwapTracks", 10)
	tracks := m.d.Song.Score.Tracks
	tracks[i], tracks[j] = tracks[j], tracks[i]
	m.clampPositions()
	m.send(m.d.Song.Score.Copy())
}

func (m *Model) SetTrackVoices(value int) {
	if value < 1 {
		value = 1
	}
	maxRemain := m.MaxTrackVoices()
	if value > maxRemain {
		value = maxRemain
	}
	if m.d.Song.Score.Tracks[m.d.Cursor.Track].NumVoices == value {
		return
	}
	m.saveUndo("SetTrackVoices", 10)
	m.d.Song.Score.Tracks[m.d.Cursor.Track].NumVoices = value
	m.send(m.d.Song.Score.Copy())
}

func (m *Model) MaxTrackVoices() int {
	maxRemain := 32 - m.d.Song.Score.NumVoices() + m.d.Song.Score.Tracks[m.d.Cursor.Track].NumVoices
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
	newInstruments := make([]sointu.Instrument, len(m.d.Song.Patch)+1)
	if after {
		m.d.InstrIndex++
	}
	copy(newInstruments, m.d.Song.Patch[:m.d.InstrIndex])
	copy(newInstruments[m.d.InstrIndex+1:], m.d.Song.Patch[m.d.InstrIndex:])
	newInstr := defaultInstrument.Copy()
	m.assignUnitIDs(newInstr.Units)
	newInstruments[m.d.InstrIndex] = newInstr
	m.d.UnitIndex = 0
	m.d.ParamIndex = 0
	m.d.Song.Patch = newInstruments
	m.send(m.d.Song.Patch.Copy())
}

func (m *Model) NoteOn(id NoteID) {
	m.send(ModelNoteOnMessage{id})
}

func (m *Model) NoteOff(id NoteID) {
	m.send(ModelNoteOffMessage{id})
}

func (m *Model) Playing() bool {
	return m.d.Playing
}

func (m *Model) SetPlaying(val bool) {
	if m.d.Playing != val {
		m.d.Playing = val
		m.send(ModelPlayingChangedMessage{val})
	}
}

func (m *Model) PlayPosition() ScoreRow {
	return m.d.PlayPosition
}

func (m *Model) CanAddInstrument() bool {
	return m.d.Song.Patch.NumVoices() < 32
}

func (m *Model) SwapInstruments(i, j int) {
	if i < 0 || j < 0 || i >= len(m.d.Song.Patch) || j >= len(m.d.Song.Patch) || i == j {
		return
	}
	m.saveUndo("SwapInstruments", 10)
	instruments := m.d.Song.Patch
	instruments[i], instruments[j] = instruments[j], instruments[i]
	m.clampPositions()
	m.send(m.d.Song.Patch.Copy())
}

func (m *Model) DeleteInstrument(forward bool) {
	if !m.CanDeleteInstrument() {
		return
	}
	m.saveUndo("DeleteInstrument", 0)
	m.freeUnitIDs(m.d.Song.Patch[m.d.InstrIndex].Units)
	m.d.Song.Patch = append(m.d.Song.Patch[:m.d.InstrIndex], m.d.Song.Patch[m.d.InstrIndex+1:]...)
	if (!forward && m.d.InstrIndex > 0) || m.d.InstrIndex >= len(m.d.Song.Patch) {
		m.d.InstrIndex--
	}
	m.clampPositions()
	m.send(m.d.Song.Patch.Copy())
}

func (m *Model) CanDeleteInstrument() bool {
	return len(m.d.Song.Patch) > 1
}

func (m *Model) Note() byte {
	trk := m.d.Song.Score.Tracks[m.d.Cursor.Track]
	pat := trk.Order.Get(m.d.Cursor.Pattern)
	if pat < 0 || pat >= len(trk.Patterns) {
		return 1
	}
	return trk.Patterns[pat].Get(m.d.Cursor.Row)
}

// SetCurrentNote sets the (note) value in current pattern under cursor to iv
func (m *Model) SetNote(iv byte) {
	m.saveUndo("SetNote", 10)
	tracks := m.d.Song.Score.Tracks
	if m.d.Cursor.Pattern < 0 || m.d.Cursor.Row < 0 {
		return
	}
	patIndex := tracks[m.d.Cursor.Track].Order.Get(m.d.Cursor.Pattern)
	if patIndex < 0 {
		patIndex = len(tracks[m.d.Cursor.Track].Patterns)
		for _, pi := range tracks[m.d.Cursor.Track].Order {
			if pi >= patIndex {
				patIndex = pi + 1 // we find a pattern that is not in the pattern table nor in the order list i.e. completely new pattern
			}
		}
		tracks[m.d.Cursor.Track].Order.Set(m.d.Cursor.Pattern, patIndex)
	}
	for len(tracks[m.d.Cursor.Track].Patterns) <= patIndex {
		tracks[m.d.Cursor.Track].Patterns = append(tracks[m.d.Cursor.Track].Patterns, nil)
	}
	tracks[m.d.Cursor.Track].Patterns[patIndex].Set(m.d.Cursor.Row, iv)
	m.send(m.d.Song.Score.Copy())
}

func (m *Model) AdjustPatternNumber(delta int, swap bool) {
	r1, r2 := m.d.Cursor.Pattern, m.d.SelectionCorner.Pattern
	if r1 > r2 {
		r1, r2 = r2, r1
	}
	t1, t2 := m.d.Cursor.Track, m.d.SelectionCorner.Track
	if t1 > t2 {
		t1, t2 = t2, t1
	}
	type k = struct {
		track int
		pat   int
	}
	newIds := map[k]int{}
	usedIds := map[k]bool{}
	for t := t1; t <= t2; t++ {
		for r := r1; r <= r2; r++ {
			p := m.d.Song.Score.Tracks[t].Order.Get(r)
			if p < 0 {
				continue
			}
			if p+delta < 0 || p+delta > 35 {
				return // if any of the patterns would go out of range, abort
			}
			newIds[k{t, p}] = p + delta
			usedIds[k{t, p + delta}] = true
		}
	}
	m.saveUndo("AdjustPatternNumber", 10)
	for t := t1; t <= t2; t++ {
		if swap {
			maxId := len(m.d.Song.Score.Tracks[t].Patterns) - 1
			// check if song uses patterns that are not in the table yet
			for _, o := range m.d.Song.Score.Tracks[t].Order {
				if maxId < o {
					maxId = o
				}
			}
			for p := 0; p <= maxId; p++ {
				j := p
				if delta > 0 {
					j = maxId - p
				}
				if _, ok := newIds[k{t, j}]; ok {
					continue
				}
				nextId := j
				for used := usedIds[k{t, nextId}]; used; used = usedIds[k{t, nextId}] {
					if delta < 0 {
						nextId++
					} else {
						nextId--
					}
				}
				newIds[k{t, j}] = nextId
				usedIds[k{t, nextId}] = true
			}
			for i, o := range m.d.Song.Score.Tracks[t].Order {
				if o < 0 {
					continue
				}
				m.d.Song.Score.Tracks[t].Order[i] = newIds[k{t, o}]
			}
			newPatterns := make([]sointu.Pattern, len(m.d.Song.Score.Tracks[t].Patterns))
			for p, pat := range m.d.Song.Score.Tracks[t].Patterns {
				id := newIds[k{t, p}]
				for len(newPatterns) <= id {
					newPatterns = append(newPatterns, nil)
				}
				newPatterns[id] = pat
			}
			m.d.Song.Score.Tracks[t].Patterns = newPatterns
		} else {
			for r := r1; r <= r2; r++ {
				p := m.d.Song.Score.Tracks[t].Order.Get(r)
				if p < 0 {
					continue
				}
				m.d.Song.Score.Tracks[t].Order.Set(r, p+delta)
			}
		}
	}
	m.computePatternUseCounts()
	m.send(m.d.Song.Score.Copy())
}

func (m *Model) SetRecording(val bool) {
	if m.d.Recording != val {
		m.d.Recording = val
		m.d.InstrEnlarged = val
		m.send(ModelRecordingMessage{val})
	}
}

func (m *Model) Recording() bool {
	return m.d.Recording
}

func (m *Model) SetPanic(val bool) {
	if m.d.Panic != val {
		m.d.Panic = val
		m.send(ModelPanicMessage{val})
	}
}

func (m *Model) Panic() bool {
	return m.d.Panic
}

func (m *Model) SetInstrEnlarged(val bool) {
	m.d.InstrEnlarged = val
}

func (m *Model) InstrEnlarged() bool {
	return m.d.InstrEnlarged
}

func (m *Model) PlayFromPosition(sr ScoreRow) {
	m.d.Playing = true
	m.send(ModelPlayFromPositionMessage{sr})
}

func (m *Model) SetCurrentPattern(pat int) {
	m.saveUndo("SetCurrentPattern", 0)
	m.d.Song.Score.Tracks[m.d.Cursor.Track].Order.Set(m.d.Cursor.Pattern, pat)
	m.computePatternUseCounts()
	m.send(m.d.Song.Score.Copy())
}

func (m *Model) IsPatternUnique(track, pattern int) bool {
	if track < 0 || track >= len(m.d.PatternUseCount) {
		return false
	}
	p := m.d.PatternUseCount[track]
	if pattern < 0 || pattern >= len(p) {
		return false
	}
	return p[pattern] <= 1
}

func (m *Model) SetSongLength(value int) {
	if value < 1 {
		value = 1
	}
	if value == m.d.Song.Score.Length {
		return
	}
	m.saveUndo("SetSongLength", 10)
	m.d.Song.Score.Length = value
	m.clampPositions()
	m.computePatternUseCounts()
	m.send(m.d.Song.Score.Copy())
}

func (m *Model) SetRowsPerPattern(value int) {
	if value < 1 {
		value = 1
	}
	if value > 255 {
		value = 255
	}
	if value == m.d.Song.Score.RowsPerPattern {
		return
	}
	m.saveUndo("SetRowsPerPattern", 10)
	m.d.Song.Score.RowsPerPattern = value
	m.clampPositions()
	m.send(m.d.Song.Score.Copy())
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
	m.Instrument().Units[m.d.UnitIndex] = unit
	m.Instrument().Units[m.d.UnitIndex].ID = oldID // keep the ID of the replaced unit
	m.send(m.d.Song.Patch.Copy())
}

func (m *Model) PasteUnits(units []sointu.Unit) {
	m.saveUndo("PasteUnits", 0)
	newUnits := make([]sointu.Unit, len(m.Instrument().Units)+len(units))
	m.d.UnitIndex++
	copy(newUnits, m.Instrument().Units[:m.d.UnitIndex])
	copy(newUnits[m.d.UnitIndex+len(units):], m.Instrument().Units[m.d.UnitIndex:])
	for _, unit := range units {
		if _, ok := m.d.UsedIDs[unit.ID]; ok {
			m.d.MaxID++
			unit.ID = m.d.MaxID
		}
		m.d.UsedIDs[unit.ID] = true
	}
	copy(newUnits[m.d.UnitIndex:m.d.UnitIndex+len(units)], units)
	m.d.Song.Patch[m.d.InstrIndex].Units = newUnits
	m.d.ParamIndex = 0
	m.clampPositions()
	m.send(m.d.Song.Patch.Copy())
}

func (m *Model) SetUnitIndex(value int) {
	m.d.UnitIndex = value
	m.d.ParamIndex = 0
	m.clampPositions()
}

func (m *Model) AddUnit(after bool) {
	m.saveUndo("AddUnit", 10)
	newUnits := make([]sointu.Unit, len(m.Instrument().Units)+1)
	if after {
		m.d.UnitIndex++
	}
	copy(newUnits, m.Instrument().Units[:m.d.UnitIndex])
	copy(newUnits[m.d.UnitIndex+1:], m.Instrument().Units[m.d.UnitIndex:])
	m.assignUnitIDs(newUnits[m.d.UnitIndex : m.d.UnitIndex+1])
	m.d.Song.Patch[m.d.InstrIndex].Units = newUnits
	m.d.ParamIndex = 0
	m.clampPositions()
	m.send(m.d.Song.Patch.Copy())
}

func (m *Model) AddOrderRow(after bool) {
	m.saveUndo("AddOrderRow", 10)
	if after {
		m.d.Cursor.Pattern++
	}
	for i, trk := range m.d.Song.Score.Tracks {
		if l := len(trk.Order); l > m.d.Cursor.Pattern {
			newOrder := make([]int, l+1)
			copy(newOrder, trk.Order[:m.d.Cursor.Pattern])
			copy(newOrder[m.d.Cursor.Pattern+1:], trk.Order[m.d.Cursor.Pattern:])
			newOrder[m.d.Cursor.Pattern] = -1
			m.d.Song.Score.Tracks[i].Order = newOrder
		}
	}
	m.d.Song.Score.Length++
	m.d.SelectionCorner = m.d.Cursor
	m.clampPositions()
	m.computePatternUseCounts()
	m.send(m.d.Song.Score.Copy())
}

func (m *Model) DeleteOrderRow(forward bool) {
	if m.d.Song.Score.Length <= 1 {
		return
	}
	m.saveUndo("DeleteOrderRow", 0)
	for i, trk := range m.d.Song.Score.Tracks {
		if l := len(trk.Order); l > m.d.Cursor.Pattern {
			newOrder := make([]int, l-1)
			copy(newOrder, trk.Order[:m.d.Cursor.Pattern])
			copy(newOrder[m.d.Cursor.Pattern:], trk.Order[m.d.Cursor.Pattern+1:])
			m.d.Song.Score.Tracks[i].Order = newOrder
		}
	}
	if !forward && m.d.Cursor.Pattern > 0 {
		m.d.Cursor.Pattern--
	}
	m.d.Song.Score.Length--
	m.d.SelectionCorner = m.d.Cursor
	m.clampPositions()
	m.computePatternUseCounts()
	m.send(m.d.Song.Score.Copy())
}

func (m *Model) DeleteUnits(forward bool, a, b int) []sointu.Unit {
	instr := m.Instrument()
	m.saveUndo("DeleteUnits", 0)
	a, b = intMin(a, b), intMax(a, b)
	if a < 0 {
		a = 0
	}
	if b > len(instr.Units)-1 {
		b = len(instr.Units) - 1
	}
	for i := a; i <= b; i++ {
		delete(m.d.UsedIDs, instr.Units[i].ID)
	}
	var newUnits []sointu.Unit
	if a == 0 && b == len(instr.Units)-1 {
		newUnits = make([]sointu.Unit, 1)
		m.d.UnitIndex = 0
	} else {
		newUnits = make([]sointu.Unit, len(instr.Units)-(b-a+1))
		copy(newUnits, instr.Units[:a])
		copy(newUnits[a:], instr.Units[b+1:])
		m.d.UnitIndex = a
		if forward {
			m.d.UnitIndex--
		}
	}
	deletedUnits := instr.Units[a : b+1]
	m.d.Song.Patch[m.d.InstrIndex].Units = newUnits
	m.d.ParamIndex = 0
	m.clampPositions()
	m.send(m.d.Song.Patch.Copy())
	return deletedUnits
}

func (m *Model) CanDeleteUnit() bool {
	return len(m.Instrument().Units) > 1
}

func (m *Model) ResetParam() {
	p, err := m.Param(m.d.ParamIndex)
	if err != nil {
		return
	}
	unit := m.Unit()
	paramList, ok := sointu.UnitTypes[unit.Type]
	if !ok || m.d.ParamIndex < 0 || m.d.ParamIndex >= len(paramList) {
		return
	}
	paramType := paramList[m.d.ParamIndex]
	defaultValue, ok := defaultUnits[unit.Type].Parameters[paramType.Name]
	if unit.Parameters[p.Name] == defaultValue {
		return
	}
	m.saveUndo("ResetParam", 0)
	unit.Parameters[paramType.Name] = defaultValue
	m.clampPositions()
	m.send(m.d.Song.Patch.Copy())
}

func (m *Model) SetParamIndex(value int) {
	m.d.ParamIndex = value
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
	m.send(m.d.Song.Patch.Copy())
}

func (m *Model) setReverb(index int) {
	if index < 0 || index >= len(reverbs) {
		return
	}
	entry := reverbs[index]
	unit := &m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex]
	if unit.Type != "delay" {
		return
	}
	m.saveUndo("setReverb", 20)
	unit.Parameters["stereo"] = entry.stereo
	unit.Parameters["notetracking"] = 0
	unit.VarArgs = make([]int, len(entry.varArgs))
	copy(unit.VarArgs, entry.varArgs)
	m.send(m.d.Song.Patch.Copy())
}

func (m *Model) SwapUnits(i, j int) {
	units := m.Instrument().Units
	if i < 0 || j < 0 || i >= len(units) || j >= len(units) || i == j {
		return
	}
	m.saveUndo("SwapUnits", 10)
	units[i], units[j] = units[j], units[i]
	m.clampPositions()
	m.send(m.d.Song.Patch.Copy())
}

func (m *Model) getSelectionRange() (int, int, int, int) {
	r1 := m.d.Cursor.Pattern*m.d.Song.Score.RowsPerPattern + m.d.Cursor.Row
	r2 := m.d.SelectionCorner.Pattern*m.d.Song.Score.RowsPerPattern + m.d.SelectionCorner.Row
	if r2 < r1 {
		r1, r2 = r2, r1
	}
	t1 := m.d.Cursor.Track
	t2 := m.d.SelectionCorner.Track
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
			s := ScoreRow{Row: r}.Wrap(m.d.Song.Score)
			if s.Pattern >= len(m.d.Song.Score.Tracks[c].Order) {
				break
			}
			p := m.d.Song.Score.Tracks[c].Order[s.Pattern]
			if p < 0 {
				continue
			}
			noteIndex := struct {
				Pat int
				Row int
			}{p, s.Row}
			if !adjustedNotes[noteIndex] {
				patterns := m.d.Song.Score.Tracks[c].Patterns
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
	m.send(m.d.Song.Score.Copy())
}

func (m *Model) DeleteSelection() {
	m.saveUndo("DeleteSelection", 0)
	r1, r2, t1, t2 := m.getSelectionRange()
	for r := r1; r <= r2; r++ {
		s := ScoreRow{Row: r}.Wrap(m.d.Song.Score)
		for c := t1; c <= t2; c++ {
			if len(m.d.Song.Score.Tracks[c].Order) <= s.Pattern {
				continue
			}
			p := m.d.Song.Score.Tracks[c].Order[s.Pattern]
			if p < 0 {
				continue
			}
			patterns := m.d.Song.Score.Tracks[c].Patterns
			if p >= len(patterns) {
				continue
			}
			pattern := patterns[p]
			if s.Row >= len(pattern) {
				continue
			}
			m.d.Song.Score.Tracks[c].Patterns[p][s.Row] = 1
		}
	}
	m.send(m.d.Song.Score.Copy())
}

func (m *Model) DeletePatternSelection() {
	m.saveUndo("DeletePatternSelection", 0)
	r1, r2, t1, t2 := m.getSelectionRange()
	p1 := ScoreRow{Row: r1}.Wrap(m.d.Song.Score).Pattern
	p2 := ScoreRow{Row: r2}.Wrap(m.d.Song.Score).Pattern
	for p := p1; p <= p2; p++ {
		for c := t1; c <= t2; c++ {
			if p < len(m.d.Song.Score.Tracks[c].Order) {
				m.d.Song.Score.Tracks[c].Order[p] = -1
			}
		}
	}
	m.computePatternUseCounts()
	m.send(m.d.Song.Score.Copy())
}

func (m *Model) Undo() {
	if !m.CanUndo() {
		return
	}
	m.d.RedoStack = append(m.d.RedoStack, m.d.Song.Copy())
	m.setSongNoUndo(m.d.UndoStack[len(m.d.UndoStack)-1])
	m.d.UndoStack = m.d.UndoStack[:len(m.d.UndoStack)-1]
	m.limitUndoRedoLengths()
	m.d.PrevUndoType = ""
}

func (m *Model) CanUndo() bool {
	return len(m.d.UndoStack) > 0
}

func (m *Model) ClearUndoHistory() {
	if len(m.d.UndoStack) > 0 {
		m.d.UndoStack = m.d.UndoStack[:0]
	}
	if len(m.d.RedoStack) > 0 {
		m.d.RedoStack = m.d.RedoStack[:0]
	}
	m.d.PrevUndoType = ""
}

func (m *Model) Redo() {
	if !m.CanRedo() {
		return
	}
	m.d.UndoStack = append(m.d.UndoStack, m.d.Song.Copy())
	m.setSongNoUndo(m.d.RedoStack[len(m.d.RedoStack)-1])
	m.d.RedoStack = m.d.RedoStack[:len(m.d.RedoStack)-1]
	m.limitUndoRedoLengths()
	m.d.PrevUndoType = ""
}

func (m *Model) CanRedo() bool {
	return len(m.d.RedoStack) > 0
}

func (m *Model) SetNoteTracking(value bool) {
	m.d.NoteTracking = value
}

func (m *Model) NoteTracking() bool {
	return m.d.NoteTracking
}

func (m *Model) Octave() int {
	return m.d.Octave
}

func (m *Model) Song() sointu.Song {
	return m.d.Song
}

func (m *Model) SelectionCorner() ScorePoint {
	return m.d.SelectionCorner
}

func (m *Model) SetSelectionCorner(value ScorePoint) {
	m.d.SelectionCorner = value
	m.clampPositions()
}

func (m *Model) Cursor() ScorePoint {
	return m.d.Cursor
}

func (m *Model) SetCursor(value ScorePoint) {
	m.d.Cursor = value
	m.clampPositions()
}

func (m *Model) LowNibble() bool {
	return m.d.LowNibble
}

func (m *Model) SetLowNibble(value bool) {
	m.d.LowNibble = value
}

func (m *Model) InstrIndex() int {
	return m.d.InstrIndex
}

func (m *Model) Track() sointu.Track {
	return m.d.Song.Score.Tracks[m.d.Cursor.Track]
}

func (m *Model) Instrument() sointu.Instrument {
	return m.d.Song.Patch[m.d.InstrIndex]
}

func (m *Model) Unit() sointu.Unit {
	return m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex]
}

func (m *Model) UnitIndex() int {
	return m.d.UnitIndex
}

func (m *Model) ParamIndex() int {
	return m.d.ParamIndex
}

func (m *Model) limitUndoRedoLengths() {
	if len(m.d.UndoStack) >= maxUndo {
		m.d.UndoStack = m.d.UndoStack[len(m.d.UndoStack)-maxUndo:]
	}
	if len(m.d.RedoStack) >= maxUndo {
		m.d.RedoStack = m.d.RedoStack[len(m.d.RedoStack)-maxUndo:]
	}
}

func (m *Model) clampPositions() {
	m.d.Cursor = m.d.Cursor.Wrap(m.d.Song.Score)
	m.d.SelectionCorner = m.d.SelectionCorner.Wrap(m.d.Song.Score)
	if !m.Track().Effect {
		m.d.LowNibble = false
	}
	m.d.InstrIndex = clamp(m.d.InstrIndex, 0, len(m.d.Song.Patch)-1)
	m.d.UnitIndex = clamp(m.d.UnitIndex, 0, len(m.Instrument().Units)-1)
	m.d.ParamIndex = clamp(m.d.ParamIndex, 0, m.NumParams()-1)
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
		numSettableParams += 2 + len(unit.VarArgs)
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
		hint := m.d.Song.Patch.ParamHintString(m.d.InstrIndex, m.d.UnitIndex, name)
		var text string
		if hint != "" {
			text = fmt.Sprintf("%v / %v", val, hint)
		} else {
			text = strconv.Itoa(val)
		}
		min, max := t.MinValue, t.MaxValue
		if unit.Type == "send" {
			if t.Name == "voice" {
				i, _, err := m.d.Song.Patch.FindUnit(unit.Parameters["target"])
				if err == nil {
					max = m.d.Song.Patch[i].NumVoices
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
			i := slices.IndexFunc(reverbs, func(p delayPreset) bool {
				return p.stereo == unit.Parameters["stereo"] && unit.Parameters["notetracking"] == 0 && slices.Equal(p.varArgs, unit.VarArgs)
			})
			hint := "0 / custom"
			if i >= 0 {
				hint = fmt.Sprintf("%v / %v", i+1, reverbs[i].name)
			}
			return Parameter{Type: IntegerParameter, Min: 0, Max: len(reverbs), Name: "reverb", Hint: hint, Value: i + 1}, nil
		}
		if index == 1 {
			l := len(unit.VarArgs)
			if unit.Parameters["stereo"] == 1 {
				l = (l + 1) / 2
			}
			return Parameter{Type: IntegerParameter, Min: 1, Max: 32, Name: "delaylines", Hint: strconv.Itoa(l), Value: l}, nil
		}
		index -= 2
		if index < len(unit.VarArgs) {
			val := unit.VarArgs[index]
			var text string
			switch unit.Parameters["notetracking"] {
			default:
			case 0:
				text = fmt.Sprintf("%v / %.3f rows", val, float32(val)/float32(m.d.Song.SamplesPerRow()))
				return Parameter{Type: IntegerParameter, Min: 1, Max: 65535, Name: "delaytime", Hint: text, Value: val, LargeStep: 256}, nil
			case 1:
				relPitch := float64(val) / 10787
				semitones := -math.Log2(relPitch) * 12
				text = fmt.Sprintf("%v / %.3f st", val, semitones)
				return Parameter{Type: IntegerParameter, Min: 1, Max: 65535, Name: "delaytime", Hint: text, Value: val, LargeStep: 256}, nil
			case 2:
				k := 0
				v := val
				for v&1 == 0 { // divide val by 2 until it is odd
					v >>= 1
					k++
				}
				text := ""
				switch v {
				case 1:
					if k <= 7 {
						text = fmt.Sprintf(" (1/%d triplet)", 1<<(7-k))
					}
				case 3:
					if k <= 6 {
						text = fmt.Sprintf(" (1/%d)", 1<<(6-k))
					}
					break
				case 9:
					if k <= 5 {
						text = fmt.Sprintf(" (1/%d dotted)", 1<<(5-k))
					}
				}
				text = fmt.Sprintf("%v / %.3f beats%s", val, float32(val)/48.0, text)
				return Parameter{Type: IntegerParameter, Min: 1, Max: 576, Name: "delaytime", Hint: text, Value: val, LargeStep: 16}, nil
			}

		}
	}
	return Parameter{}, errors.New("invalid parameter")
}

func (m *Model) RemoveUnusedData() {
	m.saveUndo("RemoveUnusedData", 0)
	for trkIndex, trk := range m.d.Song.Score.Tracks {
		// assign new indices to patterns
		newIndex := map[int]int{}
		runningIndex := 0
		length := 0
		if len(trk.Order) > m.d.Song.Score.Length {
			trk.Order = trk.Order[:m.d.Song.Score.Length]
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
				if patLength > m.d.Song.Score.RowsPerPattern {
					patLength = m.d.Song.Score.RowsPerPattern
				}
				newPatterns[ind] = pat[:patLength] // crop to either RowsPerPattern or last row having something else than hold
			}
		}
		trk.Patterns = newPatterns
		m.d.Song.Score.Tracks[trkIndex] = trk
	}
	m.computePatternUseCounts()
	m.send(m.d.Song.Score.Copy())
}

func (m *Model) SetParam(value int) {
	p, err := m.Param(m.d.ParamIndex)
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
	if p.Name == "reverb" {
		m.setReverb(value - 1)
		return
	}
	unit := m.Unit()
	if p.Name == "delaylines" {
		m.saveUndo("SetParam", 20)
		targetLines := value
		if unit.Parameters["stereo"] == 1 {
			targetLines *= 2
		}
		for len(m.Instrument().Units[m.d.UnitIndex].VarArgs) < targetLines {
			m.Instrument().Units[m.d.UnitIndex].VarArgs = append(m.Instrument().Units[m.d.UnitIndex].VarArgs, 1)
		}
		m.Instrument().Units[m.d.UnitIndex].VarArgs = m.Instrument().Units[m.d.UnitIndex].VarArgs[:targetLines]
	} else if p.Name == "delaytime" {
		m.saveUndo("SetParam", 20)
		index := m.d.ParamIndex - 8
		for len(m.Instrument().Units[m.d.UnitIndex].VarArgs) <= index {
			m.Instrument().Units[m.d.UnitIndex].VarArgs = append(m.Instrument().Units[m.d.UnitIndex].VarArgs, 1)
		}
		m.Instrument().Units[m.d.UnitIndex].VarArgs[index] = value
	} else {
		if unit.Parameters[p.Name] == value {
			return
		}
		m.saveUndo("SetParam", 20)
		unit.Parameters[p.Name] = value
	}
	m.clampPositions()
	m.send(m.d.Song.Patch.Copy())
}

func (m *Model) setSongNoUndo(song sointu.Song) {
	m.d.Song = song
	m.d.UsedIDs = make(map[int]bool)
	m.d.MaxID = 0
	for _, instr := range m.d.Song.Patch {
		for _, unit := range instr.Units {
			if m.d.MaxID < unit.ID {
				m.d.MaxID = unit.ID
			}
		}
	}
	for _, instr := range m.d.Song.Patch {
		m.assignUnitIDs(instr.Units)
	}
	m.clampPositions()
	m.computePatternUseCounts()
	m.send(m.d.Song.Copy())
}

// send sends a message to the player
func (m *Model) send(message interface{}) {
	m.modelMessages <- message
}

func (m *Model) saveUndo(undoType string, undoSkipping int) {
	m.d.ChangedSinceSave = true
	m.d.ChangedSinceRecovery = true
	if m.d.PrevUndoType == undoType && m.d.UndoSkipCounter < undoSkipping {
		m.d.UndoSkipCounter++
		return
	}
	m.d.PrevUndoType = undoType
	m.d.UndoSkipCounter = 0
	m.d.UndoStack = append(m.d.UndoStack, m.d.Song.Copy())
	m.d.RedoStack = m.d.RedoStack[:0]
	m.limitUndoRedoLengths()
}

func (m *Model) freeUnitIDs(units []sointu.Unit) {
	for _, u := range units {
		delete(m.d.UsedIDs, u.ID)
	}
}

func (m *Model) assignUnitIDs(units []sointu.Unit) {
	rewrites := map[int]int{}
	for i := range units {
		if id := units[i].ID; id == 0 || m.d.UsedIDs[id] {
			m.d.MaxID++
			if id > 0 {
				rewrites[id] = m.d.MaxID
			}
			units[i].ID = m.d.MaxID
		}
		m.d.UsedIDs[units[i].ID] = true
		if m.d.MaxID < units[i].ID {
			m.d.MaxID = units[i].ID
		}
	}
	for i, u := range units {
		if target, ok := u.Parameters["target"]; u.Type == "send" && ok {
			if newId, ok := rewrites[target]; ok {
				units[i].Parameters["target"] = newId
			}
		}
	}
}

func (m *Model) computePatternUseCounts() {
	for i, track := range m.d.Song.Score.Tracks {
		for len(m.d.PatternUseCount) <= i {
			m.d.PatternUseCount = append(m.d.PatternUseCount, nil)
		}
		for j := range m.d.PatternUseCount[i] {
			m.d.PatternUseCount[i][j] = 0
		}
		for j := 0; j < m.d.Song.Score.Length; j++ {
			if j >= len(track.Order) {
				break
			}
			p := track.Order[j]
			for len(m.d.PatternUseCount[i]) <= p {
				m.d.PatternUseCount[i] = append(m.d.PatternUseCount[i], 0)
			}
			if p < 0 {
				continue
			}
			m.d.PatternUseCount[i][p]++
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

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
