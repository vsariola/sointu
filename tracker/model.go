package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vsariola/sointu"
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
		Song                    sointu.Song
		Cursor, Cursor2         Cursor
		LowNibble               bool
		InstrIndex, InstrIndex2 int
		UnitIndex, UnitIndex2   int
		ParamIndex              int
		UnitSearchIndex         int
		UnitSearchString        string
		UnitSearching           bool
		Octave                  int
		Step                    int
		FilePath                string
		ChangedSinceSave        bool
		RecoveryFilePath        string
		ChangedSinceRecovery    bool
		SendSource              int
		InstrumentTab           InstrumentTab
		PresetSearchString      string
	}

	Model struct {
		d       modelData
		derived derivedModelData

		instrEnlarged bool

		prevUndoKind    string
		undoSkipCounter int
		undoStack       []modelData
		redoStack       []modelData

		changeLevel    int
		changeCancel   bool
		changeSeverity ChangeSeverity
		changeType     ChangeType

		panic          bool
		recording      bool
		playing        bool
		loop           Loop
		follow         bool
		quitted        bool
		uniquePatterns bool
		// when linkInstrTrack is false, editing an instrument does not change
		// the track. when true, editing an instrument changes the tracks (e.g.
		// reordering or deleting instrument can delete track)
		linkInstrTrack bool

		playerStatus PlayerStatus

		signalAnalyzer *ScopeModel
		detectorResult DetectorResult

		weightingType WeightingType
		oversampling  bool

		alerts []Alert
		dialog Dialog

		syntherIndex int              // the index of the synther used to create new synths
		synthers     []sointu.Synther // the synther used to create new synths

		broker *Broker

		MIDI MIDIContext

		presets     PresetSlice
		presetIndex int
	}

	// Cursor identifies a row and a track in a song score.
	Cursor struct {
		Track int
		sointu.SongPos
	}

	// Loop identifier the order rows, which are the loop positions
	// Length = 0 means no loop is chosen, regardless of start
	Loop struct {
		Start, Length int
	}

	Explore struct {
		IsSave       bool         // true if this is a save operation, false if open operation
		IsSong       bool         // true if this is a song, false if instrument
		Continuation func(string) // function to call with the selected file path
	}

	IsPlayingMsg   struct{ bool }
	StartPlayMsg   struct{ sointu.SongPos }
	BPMMsg         struct{ int }
	RowsPerBeatMsg struct{ int }
	PanicMsg       struct{ bool }
	RecordingMsg   struct{ bool }

	ChangeSeverity int
	ChangeType     int

	Dialog int

	MIDIContext interface {
		InputDevices(yield func(MIDIDevice) bool)
		Close()
		HasDeviceOpen() bool
		TryToOpenBy(name string, first bool)
	}

	NullMIDIContext struct{}

	MIDIDevice interface {
		String() string
		Open() error
	}

	InstrumentTab int
)

const (
	MajorChange ChangeSeverity = iota
	MinorChange
)

const (
	NoChange    ChangeType = 0
	PatchChange ChangeType = 1 << iota
	ScoreChange
	BPMChange
	RowsPerBeatChange
	SongChange ChangeType = PatchChange | ScoreChange | BPMChange | RowsPerBeatChange
)

const (
	NoDialog = iota
	SaveAsExplorer
	NewSongChanges
	NewSongSaveExplorer
	OpenSongChanges
	OpenSongSaveExplorer
	OpenSongOpenExplorer
	Export
	ExportFloatExplorer
	ExportInt16Explorer
	QuitChanges
	QuitSaveExplorer
	License
)

const (
	InstrumentEditorTab InstrumentTab = iota
	InstrumentPresetsTab
	InstrumentCommentTab
)

const maxUndo = 64

func (m *Model) PlayPosition() sointu.SongPos { return m.playerStatus.SongPos }
func (m *Model) Loop() Loop                   { return m.loop }
func (m *Model) PlaySongRow() int             { return m.d.Song.Score.SongRow(m.playerStatus.SongPos) }
func (m *Model) ChangedSinceSave() bool       { return m.d.ChangedSinceSave }
func (m *Model) Dialog() Dialog               { return m.dialog }
func (m *Model) Quitted() bool                { return m.quitted }

func (m *Model) DetectorResult() DetectorResult { return m.detectorResult }

// NewModelPlayer creates a new model and a player that communicates with it
func NewModel(broker *Broker, synthers []sointu.Synther, midiContext MIDIContext, recoveryFilePath string) *Model {
	m := new(Model)
	m.synthers = synthers
	m.MIDI = midiContext
	m.broker = broker
	m.d.Octave = 4
	m.linkInstrTrack = true
	m.d.RecoveryFilePath = recoveryFilePath
	m.resetSong()
	if recoveryFilePath != "" {
		if bytes2, err := os.ReadFile(m.d.RecoveryFilePath); err == nil {
			var data modelData
			if json.Unmarshal(bytes2, &data) == nil {
				m.d = data
			}
		}
	}
	TrySend(broker.ToPlayer, any(m.d.Song.Copy())) // we should be non-blocking in the constructor
	m.signalAnalyzer = NewScopeModel(broker, m.d.Song.BPM)
	m.updateDeriveData(SongChange)
	m.updateDerivedPresetSearch()
	m.loadPresets()
	return m
}

func (m *Model) change(kind string, t ChangeType, severity ChangeSeverity) func() {
	if m.changeLevel == 0 {
		m.changeType = NoChange
		m.undoStack = append(m.undoStack, m.d.Copy())
		m.changeCancel = false
		m.changeSeverity = severity
	} else {
		if m.changeSeverity < severity {
			m.changeSeverity = severity
		}
	}
	m.changeType |= t
	m.changeLevel++
	return func() {
		m.changeLevel--
		if m.changeLevel < 0 {
			panic("changeLevel < 0, mismatched change() calls")
		}
		if m.changeLevel == 0 {
			if m.changeCancel || m.d.Song.BPM <= 0 || m.d.Song.RowsPerBeat <= 0 || m.d.Song.Score.Length <= 0 {
				// the change was cancelled or put the song in invalid state, so we don't save it
				m.d = m.undoStack[len(m.undoStack)-1]
				m.undoStack = m.undoStack[:len(m.undoStack)-1]
				return
			}
			m.d.ChangedSinceSave = true
			m.d.ChangedSinceRecovery = true
			if m.changeType&ScoreChange != 0 {
				m.d.Cursor.SongPos = m.d.Song.Score.Clamp(m.d.Cursor.SongPos)
				m.d.Cursor2.SongPos = m.d.Song.Score.Clamp(m.d.Cursor2.SongPos)
				TrySend(m.broker.ToPlayer, any(m.d.Song.Score.Copy()))
			}
			if m.changeType&PatchChange != 0 {
				m.fixIDCollisions()
				m.fixUnitParams()
				m.d.InstrIndex = clamp(m.d.InstrIndex, 0, len(m.d.Song.Patch)-1)
				m.d.InstrIndex2 = clamp(m.d.InstrIndex2, 0, len(m.d.Song.Patch)-1)
				unitCount := 0
				if m.d.InstrIndex >= 0 && m.d.InstrIndex < len(m.d.Song.Patch) {
					unitCount = len(m.d.Song.Patch[m.d.InstrIndex].Units)
				}
				m.d.UnitIndex = clamp(m.d.UnitIndex, 0, unitCount-1)
				m.d.UnitIndex2 = clamp(m.d.UnitIndex2, 0, unitCount-1)
				m.d.UnitSearching = false // if we change anything in the patch, reset the unit searching
				m.d.UnitSearchString = ""
				m.d.SendSource = 0
				TrySend(m.broker.ToPlayer, any(m.d.Song.Patch.Copy()))
			}
			if m.changeType&BPMChange != 0 {
				TrySend(m.broker.ToPlayer, any(BPMMsg{m.d.Song.BPM}))
				m.signalAnalyzer.SetBpm(m.d.Song.BPM)
			}
			if m.changeType&RowsPerBeatChange != 0 {
				TrySend(m.broker.ToPlayer, any(RowsPerBeatMsg{m.d.Song.RowsPerBeat}))
			}
			m.updateDeriveData(m.changeType)
			m.undoSkipCounter++
			var limit int
			switch m.changeSeverity {
			default:
			case MajorChange:
				limit = 1
			case MinorChange:
				limit = 10
			}
			if m.prevUndoKind == kind && m.undoSkipCounter < limit {
				m.undoStack = m.undoStack[:len(m.undoStack)-1]
				return
			}
			m.undoSkipCounter = 0
			m.prevUndoKind = kind
			m.redoStack = m.redoStack[:0]
			if len(m.undoStack) > maxUndo {
				copy(m.undoStack, m.undoStack[len(m.undoStack)-maxUndo:])
				m.undoStack = m.undoStack[:maxUndo]
			}
		}
	}
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
	var data modelData
	err := json.Unmarshal(bytes, &data)
	if err != nil {
		return
	}
	m.d = data
	if m.d.RecoveryFilePath != "" { // check if there's a recovery file on disk and load it instead
		if bytes2, err := os.ReadFile(m.d.RecoveryFilePath); err == nil {
			var data modelData
			if json.Unmarshal(bytes2, &data) == nil {
				m.d = data
			}
		}
	}
	m.d.ChangedSinceRecovery = false
	TrySend(m.broker.ToPlayer, any(m.d.Song.Copy()))
	m.updateDeriveData(SongChange)
}

func (m *Model) ProcessMsg(msg MsgToModel) {
	if msg.HasPanicPlayerStatus {
		m.playerStatus = msg.PlayerStatus
		if m.playing && m.follow {
			m.d.Cursor.SongPos = msg.PlayerStatus.SongPos
			m.d.Cursor2.SongPos = msg.PlayerStatus.SongPos
			TrySend(m.broker.ToGUI, any(MsgToGUI{
				Kind:  GUIMessageCenterOnRow,
				Param: m.PlaySongRow(),
			}))
		}
		m.panic = msg.Panic
	}
	if msg.HasDetectorResult {
		m.detectorResult = msg.DetectorResult
	}
	if msg.TriggerChannel > 0 {
		m.signalAnalyzer.Trigger(msg.TriggerChannel)
	}
	if msg.Reset {
		m.signalAnalyzer.Reset()
	}
	switch e := msg.Data.(type) {
	case func():
		e()
	case Recording:
		if e.BPM == 0 {
			e.BPM = float64(m.d.Song.BPM)
		}
		score, err := e.Score(m.d.Song.Patch, m.d.Song.RowsPerBeat, m.d.Song.Score.RowsPerPattern)
		if err != nil || score.Length <= 0 {
			break
		}
		defer m.change("Recording", SongChange, MajorChange)()
		m.d.Song.Score = score
		m.d.Song.BPM = int(e.BPM + 0.5)
		m.instrEnlarged = false
	case Alert:
		m.Alerts().AddAlert(e)
	case IsPlayingMsg:
		m.playing = e.bool
	case *sointu.AudioBuffer:
		m.signalAnalyzer.ProcessAudioBuffer(e)
	}
}

func (m *Model) CPULoad() float64 { return m.playerStatus.CPULoad }

func (m *Model) SignalAnalyzer() *ScopeModel { return m.signalAnalyzer }
func (m *Model) Broker() *Broker             { return m.broker }

func (d *modelData) Copy() modelData {
	ret := *d
	ret.Song = d.Song.Copy()
	return ret
}

func (m NullMIDIContext) InputDevices(yield func(MIDIDevice) bool) {}
func (m NullMIDIContext) Close()                                   {}
func (m NullMIDIContext) HasDeviceOpen() bool                      { return false }
func (m NullMIDIContext) TryToOpenBy(name string, first bool)      {}

func (m *Model) resetSong() {
	m.d.Song = defaultSong.Copy()
	for _, instr := range m.d.Song.Patch {
		(*Model)(m).assignUnitIDs(instr.Units)
	}
	m.d.FilePath = ""
	m.d.ChangedSinceSave = false
}

func (m *Model) maxID() int {
	maxID := 0
	for _, instr := range m.d.Song.Patch {
		for _, unit := range instr.Units {
			if unit.ID > maxID {
				maxID = unit.ID
			}
		}
	}
	return maxID
}

func (m *Model) maxIDandUsed() (maxID int, usedIDs map[int]bool) {
	usedIDs = make(map[int]bool)
	for _, instr := range m.d.Song.Patch {
		for _, unit := range instr.Units {
			usedIDs[unit.ID] = true
			if maxID < unit.ID {
				maxID = unit.ID
			}
		}
	}
	return
}

func (m *Model) assignUnitIDsForPatch(patch sointu.Patch) {
	maxId, usedIds := m.maxIDandUsed()
	rewrites := map[int]int{}
	for _, instr := range patch {
		rewriteUnitIds(instr.Units, &maxId, usedIds, rewrites)
	}
	for _, instr := range patch {
		rewriteSendTargets(instr.Units, rewrites)
	}
}

func (m *Model) assignUnitIDs(units []sointu.Unit) {
	maxID, usedIds := m.maxIDandUsed()
	rewrites := map[int]int{}
	rewriteUnitIds(units, &maxID, usedIds, rewrites)
	rewriteSendTargets(units, rewrites)
}

func rewriteUnitIds(units []sointu.Unit, maxId *int, usedIds map[int]bool, rewrites map[int]int) {
	for i := range units {
		if id := units[i].ID; id == 0 || usedIds[id] {
			*maxId++
			if id > 0 {
				rewrites[id] = *maxId
			}
			units[i].ID = *maxId
		}
		usedIds[units[i].ID] = true
		if *maxId < units[i].ID {
			*maxId = units[i].ID
		}
	}
}

func rewriteSendTargets(units []sointu.Unit, rewrites map[int]int) {
	for i := range units {
		if target, ok := units[i].Parameters["target"]; units[i].Type == "send" && ok {
			if newId, ok := rewrites[target]; ok {
				units[i].Parameters["target"] = newId
			}
		}
	}
}

func (m *Model) fixIDCollisions() {
	// loop over all instruments and units and check if two units have the same
	// ID. If so, give the later units new IDs.
	usedIDs := map[int]bool{}
	needsFix := false
	maxID := 0
	for i, instr := range m.d.Song.Patch {
		for j, unit := range instr.Units {
			if usedIDs[unit.ID] {
				m.d.Song.Patch[i].Units[j].ID = 0
				needsFix = true
			}
			if unit.ID > maxID {
				maxID = unit.ID
			}
			usedIDs[unit.ID] = true
		}
	}
	if needsFix {
		m.Alerts().AddNamed("IDCollision", "Some units had duplicate IDs, they were fixed", Error)
		for i, instr := range m.d.Song.Patch {
			for j, unit := range instr.Units {
				if unit.ID == 0 {
					maxID++
					m.d.Song.Patch[i].Units[j].ID = maxID
				}
			}
		}
	}
}

var validParameters = map[string](map[string]bool){}

func init() {
	for name, unitType := range sointu.UnitTypes {
		validParameters[name] = map[string]bool{}
		for _, param := range unitType {
			validParameters[name][param.Name] = true
		}
	}
}

func (m *Model) fixUnitParams() {
	// loop over all instruments and units and check that unit parameter table
	// only has the parameters that are defined in the unit type
	fixed := false
	for i := range m.d.Song.Patch {
		fixed = RemoveUnusedUnitParameters(&m.d.Song.Patch[i]) || fixed
	}
	if fixed {
		m.Alerts().AddNamed("InvalidUnitParameters", "Some units had invalid parameters, they were removed", Error)
	}
}

// RemoveUnusedUnitParameters removes any parameters from the instrument that are not valid for the unit type.
// It returns true if any parameters were removed.
func RemoveUnusedUnitParameters(instr *sointu.Instrument) bool {
	fixed := false
	for _, unit := range instr.Units {
		for paramName := range unit.Parameters {
			if !validParameters[unit.Type][paramName] {
				delete(unit.Parameters, paramName)
				fixed = true
			}
		}
	}
	return fixed
}

func clamp(a, min, max int) int {
	if a > max {
		return max
	}
	if a < min {
		return min
	}
	return a
}
