package tracker

import (
	"fmt"
	"math"
	"os"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
)

type (
	// Action describes a user action that can be performed on the model, which
	// can be initiated by calling the Do() method. It is usually initiated by a
	// button press or a menu item. Action advertises whether it is enabled, so
	// UI can e.g. gray out buttons when the underlying action is not allowed.
	// Action also implements the Doer and Enabler interfaces, but guarding that
	// the underlying doer will never get called if enabler return false. That's
	// why the doer and enabler are private fields, so that they cannot be
	// called directly from outside, circumventing the Enabled() check.
	Action struct {
		doer    Doer
		enabler Enabler
	}

	// Doer is an interface that defines a single Do() method, which is called
	// when an action is performed.
	Doer interface {
		Do()
	}

	// Enabler is an interface that defines a single Enabled() method, which
	// is used by the UI to check if UI Action/Bool/Int etc. is enabled or not.
	Enabler interface {
		Enabled() bool
	}

	DoFunc func()

	AddTrack         Model
	DeleteTrack      Model
	SplitTrack       Model
	AddSemitone      Model
	SubtractSemitone Model
	AddOctave        Model
	SubtractOctave   Model
	EditNoteOff      Model

	AddInstrument    Model
	DeleteInstrument Model
	SplitInstrument  Model

	AddUnit struct {
		Before bool
		*Model
	}
	DeleteUnit Model
	ClearUnit  Model

	Undo         Model
	Redo         Model
	RemoveUnused Model

	PlayCurrentPos    Model
	PlaySongStart     Model
	PlaySelected      Model
	PlayFromLoopStart Model
	StopPlaying       Model

	AddOrderRow struct {
		Before bool
		*Model
	}
	DeleteOrderRow struct {
		Backwards bool
		*Model
	}

	NewSong     Model
	OpenSong    Model
	SaveSong    Model
	RequestQuit Model
	ForceQuit   Model
	Cancel      Model

	DiscardSong     Model
	SaveSongAs      Model
	ExportAction    Model
	ExportFloat     Model
	ExportInt16     Model
	SelectMidiInput struct {
		Item MIDIDevice
		*Model
	}
	ShowLicense Model
)

// Action methods

// simple version for when both doer and enabler are the same
func MakeAction(doerEnabler interface {
	Doer
	Enabler
}) Action {
	return Action{doer: doerEnabler, enabler: doerEnabler}
}

// a version for cases where doer and enabler are different
func MakeAction2(doer Doer, enabler Enabler) Action {
	return Action{doer: doer, enabler: enabler}
}

func MakeEnabledAction(doer Doer) Action {
	return Action{doer: doer, enabler: nil}
}

func (a Action) Do() {
	if a.enabler != nil && !a.enabler.Enabled() {
		return
	}
	if a.doer != nil {
		a.doer.Do()
	}
}

func (a Action) Enabled() bool {
	if a.doer == nil {
		return false // no doer, not allowed
	}
	if a.enabler == nil {
		return true // no enabler, always allowed
	}
	return a.enabler.Enabled()
}

// DoFunc

func (d DoFunc) Do() { d() }

// AddTrack

func (m *Model) AddTrack() Action { return MakeAction((*AddTrack)(m)) }
func (m *AddTrack) Enabled() bool { return m.d.Song.Score.NumVoices() < vm.MAX_VOICES }
func (m *AddTrack) Do() {
	defer (*Model)(m).change("AddTrack", SongChange, MajorChange)()
	voiceIndex := m.d.Song.Score.FirstVoiceForTrack(m.d.Cursor.Track)
	p := sointu.Patch{defaultInstrument.Copy()}
	t := []sointu.Track{{NumVoices: 1}}
	_, _, ok := (*Model)(m).addVoices(voiceIndex, p, t, (*Model)(m).linkInstrTrack, true)
	m.changeCancel = !ok
}

// DeleteTrack

func (m *Model) DeleteTrack() Action { return MakeAction((*DeleteTrack)(m)) }
func (m *DeleteTrack) Enabled() bool { return len(m.d.Song.Score.Tracks) > 0 }
func (m *DeleteTrack) Do()           { (*Model)(m).Tracks().List().DeleteElements(false) }

// AddInstrument

func (m *Model) AddInstrument() Action { return MakeAction((*AddInstrument)(m)) }
func (m *AddInstrument) Enabled() bool { return (*Model)(m).d.Song.Patch.NumVoices() < vm.MAX_VOICES }
func (m *AddInstrument) Do() {
	defer (*Model)(m).change("AddInstrument", SongChange, MajorChange)()
	voiceIndex := m.d.Song.Patch.FirstVoiceForInstrument(m.d.InstrIndex)
	p := sointu.Patch{defaultInstrument.Copy()}
	t := []sointu.Track{{NumVoices: 1}}
	_, _, ok := (*Model)(m).addVoices(voiceIndex, p, t, true, (*Model)(m).linkInstrTrack)
	m.changeCancel = !ok
}

// DeleteInstrument

func (m *Model) DeleteInstrument() Action { return MakeAction((*DeleteInstrument)(m)) }
func (m *DeleteInstrument) Enabled() bool { return len((*Model)(m).d.Song.Patch) > 0 }
func (m *DeleteInstrument) Do()           { (*Model)(m).Instruments().List().DeleteElements(false) }

// SplitTrack

func (m *Model) SplitTrack() Action { return MakeAction((*SplitTrack)(m)) }
func (m *SplitTrack) Enabled() bool {
	return m.d.Cursor.Track >= 0 && m.d.Cursor.Track < len(m.d.Song.Score.Tracks) && m.d.Song.Score.Tracks[m.d.Cursor.Track].NumVoices > 1
}
func (m *SplitTrack) Do() {
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

// SplitInstrument

func (m *Model) SplitInstrument() Action { return MakeAction((*SplitInstrument)(m)) }
func (m *SplitInstrument) Enabled() bool {
	return m.d.InstrIndex >= 0 && m.d.InstrIndex < len(m.d.Song.Patch) && m.d.Song.Patch[m.d.InstrIndex].NumVoices > 1
}
func (m *SplitInstrument) Do() {
	defer (*Model)(m).change("SplitInstrument", SongChange, MajorChange)()
	voiceIndex := m.d.Song.Patch.Copy().FirstVoiceForInstrument(m.d.InstrIndex)
	middle := voiceIndex + (m.d.Song.Patch[m.d.InstrIndex].NumVoices+1)/2
	end := voiceIndex + m.d.Song.Patch[m.d.InstrIndex].NumVoices
	left, ok := VoiceSlice(m.d.Song.Patch, Range{math.MinInt, middle})
	if !ok {
		m.changeCancel = true
		return
	}
	right, ok := VoiceSlice(m.d.Song.Patch, Range{end, math.MaxInt})
	if !ok {
		m.changeCancel = true
		return
	}
	newInstrument := defaultInstrument.Copy()
	(*Model)(m).assignUnitIDs(newInstrument.Units)
	newInstrument.NumVoices = end - middle
	m.d.Song.Patch = append(left, newInstrument)
	m.d.Song.Patch = append(m.d.Song.Patch, right...)
}

// AddUnit

func (m *Model) AddUnit(before bool) Action {
	return MakeEnabledAction(AddUnit{Before: before, Model: m})
}
func (a AddUnit) Do() {
	m := (*Model)(a.Model)
	defer m.change("AddUnitAction", PatchChange, MajorChange)()
	if len(m.d.Song.Patch) == 0 { // no instruments, add one
		instr := sointu.Instrument{NumVoices: 1}
		instr.Units = make([]sointu.Unit, 0, 1)
		m.d.Song.Patch = append(m.d.Song.Patch, instr)
		m.d.UnitIndex = 0
	} else {
		if !a.Before {
			m.d.UnitIndex++
		}
	}
	m.d.InstrIndex = max(min(m.d.InstrIndex, len(m.d.Song.Patch)-1), 0)
	instr := m.d.Song.Patch[m.d.InstrIndex]
	newUnits := make([]sointu.Unit, len(instr.Units)+1)
	m.d.UnitIndex = clamp(m.d.UnitIndex, 0, len(newUnits)-1)
	m.d.UnitIndex2 = m.d.UnitIndex
	copy(newUnits, instr.Units[:m.d.UnitIndex])
	copy(newUnits[m.d.UnitIndex+1:], instr.Units[m.d.UnitIndex:])
	m.assignUnitIDs(newUnits[m.d.UnitIndex : m.d.UnitIndex+1])
	m.d.Song.Patch[m.d.InstrIndex].Units = newUnits
	m.d.ParamIndex = 0
}

// DeleteUnit

func (m *Model) DeleteUnit() Action { return MakeAction((*DeleteUnit)(m)) }
func (m *DeleteUnit) Enabled() bool {
	i := (*Model)(m).d.InstrIndex
	return i >= 0 && i < len((*Model)(m).d.Song.Patch) && len((*Model)(m).d.Song.Patch[i].Units) > 1
}
func (m *DeleteUnit) Do() {
	defer (*Model)(m).change("DeleteUnitAction", PatchChange, MajorChange)()
	(*Model)(m).Units().List().DeleteElements(true)
}

// ClearUnit

func (m *Model) ClearUnit() Action { return MakeAction((*ClearUnit)(m)) }
func (m *ClearUnit) Enabled() bool {
	i := (*Model)(m).d.InstrIndex
	return i >= 0 && i < len(m.d.Song.Patch) && len(m.d.Song.Patch[i].Units) > 0
}
func (m *ClearUnit) Do() {
	defer (*Model)(m).change("DeleteUnitAction", PatchChange, MajorChange)()
	m.d.UnitIndex = max(min(m.d.UnitIndex, len(m.d.Song.Patch[m.d.InstrIndex].Units)-1), 0)
	m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex] = sointu.Unit{}
	m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex].ID = (*Model)(m).maxID() + 1
}

// Undo

func (m *Model) Undo() Action { return MakeAction((*Undo)(m)) }
func (m *Undo) Enabled() bool { return len((*Model)(m).undoStack) > 0 }
func (m *Undo) Do() {
	m.redoStack = append(m.redoStack, m.d.Copy())
	if len(m.redoStack) >= maxUndo {
		copy(m.redoStack, m.redoStack[len(m.redoStack)-maxUndo:])
		m.redoStack = m.redoStack[:maxUndo]
	}
	m.d = m.undoStack[len(m.undoStack)-1]
	m.undoStack = m.undoStack[:len(m.undoStack)-1]
	m.prevUndoKind = ""
	(*Model)(m).updateDerivedScoreData()
	(*Model)(m).updateDerivedPatchData()
	TrySend(m.broker.ToPlayer, any(m.d.Song.Copy()))
}

// Redo

func (m *Model) Redo() Action { return MakeAction((*Redo)(m)) }
func (m *Redo) Enabled() bool { return len((*Model)(m).redoStack) > 0 }
func (m *Redo) Do() {
	m.undoStack = append(m.undoStack, m.d.Copy())
	if len(m.undoStack) >= maxUndo {
		copy(m.undoStack, m.undoStack[len(m.undoStack)-maxUndo:])
		m.undoStack = m.undoStack[:maxUndo]
	}
	m.d = m.redoStack[len(m.redoStack)-1]
	m.redoStack = m.redoStack[:len(m.redoStack)-1]
	m.prevUndoKind = ""
	(*Model)(m).updateDerivedScoreData()
	(*Model)(m).updateDerivedPatchData()
	TrySend(m.broker.ToPlayer, any(m.d.Song.Copy()))
}

// AddSemiTone

func (m *Model) AddSemitone() Action { return MakeEnabledAction((*AddSemitone)(m)) }
func (m *AddSemitone) Do()           { Table{(*Notes)(m)}.Add(1) }

// SubtractSemitone

func (m *Model) SubtractSemitone() Action { return MakeEnabledAction((*SubtractSemitone)(m)) }
func (m *SubtractSemitone) Do()           { Table{(*Notes)(m)}.Add(-1) }

// AddOctave

func (m *Model) AddOctave() Action { return MakeEnabledAction((*AddOctave)(m)) }
func (m *AddOctave) Do()           { Table{(*Notes)(m)}.Add(12) }

// SubtractOctave

func (m *Model) SubtractOctave() Action { return MakeEnabledAction((*SubtractOctave)(m)) }
func (m *SubtractOctave) Do()           { Table{(*Notes)(m)}.Add(-12) }

// EditNoteOff

func (m *Model) EditNoteOff() Action { return MakeEnabledAction((*EditNoteOff)(m)) }
func (m *EditNoteOff) Do()           { Table{(*Notes)(m)}.Fill(0) }

// RemoveUnused

func (m *Model) RemoveUnused() Action { return MakeEnabledAction((*RemoveUnused)(m)) }
func (m *RemoveUnused) Do() {
	defer (*Model)(m).change("RemoveUnusedAction", ScoreChange, MajorChange)()
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
}

// PlayCurrentPos

func (m *Model) PlayCurrentPos() Action { return MakeAction((*PlayCurrentPos)(m)) }
func (m *PlayCurrentPos) Enabled() bool { return !m.instrEnlarged }
func (m *PlayCurrentPos) Do() {
	(*Model)(m).setPanic(false)
	(*Model)(m).setLoop(Loop{})
	m.playing = true
	TrySend(m.broker.ToPlayer, any(StartPlayMsg{m.d.Cursor.SongPos}))
}

// PlaySongStart

func (m *Model) PlaySongStart() Action { return MakeAction((*PlaySongStart)(m)) }
func (m *PlaySongStart) Enabled() bool { return !m.instrEnlarged }
func (m *PlaySongStart) Do() {
	(*Model)(m).setPanic(false)
	(*Model)(m).setLoop(Loop{})
	m.playing = true
	TrySend(m.broker.ToPlayer, any(StartPlayMsg{}))
}

// PlaySelected

func (m *Model) PlaySelected() Action { return MakeAction((*PlaySelected)(m)) }
func (m *PlaySelected) Enabled() bool { return !m.instrEnlarged }
func (m *PlaySelected) Do() {
	(*Model)(m).setPanic(false)
	m.playing = true
	l := (*Model)(m).OrderRows().List()
	r := l.listRange()
	newLoop := Loop{r.Start, r.End - r.Start}
	(*Model)(m).setLoop(newLoop)
	TrySend(m.broker.ToPlayer, any(StartPlayMsg{sointu.SongPos{OrderRow: r.Start, PatternRow: 0}}))
}

// PlayFromLoopStart

func (m *Model) PlayFromLoopStart() Action { return MakeAction((*PlayFromLoopStart)(m)) }
func (m *PlayFromLoopStart) Enabled() bool { return !m.instrEnlarged }
func (m *PlayFromLoopStart) Do() {
	(*Model)(m).setPanic(false)
	if m.loop == (Loop{}) {
		(*Model)(m).PlaySelected().Do()
		return
	}
	m.playing = true
	TrySend(m.broker.ToPlayer, any(StartPlayMsg{sointu.SongPos{OrderRow: m.loop.Start, PatternRow: 0}}))
}

// StopPlaying

func (m *Model) StopPlaying() Action { return MakeEnabledAction((*StopPlaying)(m)) }
func (m *StopPlaying) Do() {
	if !m.playing {
		(*Model)(m).setPanic(true)
		(*Model)(m).setLoop(Loop{})
		return
	}
	m.playing = false
	TrySend(m.broker.ToPlayer, any(IsPlayingMsg{false}))
}

// AddOrderRow

func (m *Model) AddOrderRow(before bool) Action {
	return MakeEnabledAction(AddOrderRow{Before: before, Model: m})
}
func (a AddOrderRow) Do() {
	m := a.Model
	defer m.change("AddOrderRowAction", ScoreChange, MinorChange)()
	if !a.Before {
		m.d.Cursor.OrderRow++
	}
	m.d.Cursor2.OrderRow = m.d.Cursor.OrderRow
	from := m.d.Cursor.OrderRow
	m.d.Song.Score.Length++
	for i := range m.d.Song.Score.Tracks {
		order := &m.d.Song.Score.Tracks[i].Order
		if len(*order) > from {
			*order = append(*order, -1)
			copy((*order)[from+1:], (*order)[from:])
			(*order)[from] = -1
		}
	}
}

// DeleteOrderRow

func (m *Model) DeleteOrderRow(backwards bool) Action {
	return MakeEnabledAction(DeleteOrderRow{Backwards: backwards, Model: m})
}
func (d DeleteOrderRow) Do() {
	m := d.Model
	defer m.change("AddOrderRowAction", ScoreChange, MinorChange)()
	from := m.d.Cursor.OrderRow
	m.d.Song.Score.Length--
	for i := range m.d.Song.Score.Tracks {
		order := &m.d.Song.Score.Tracks[i].Order
		if len(*order) > from {
			copy((*order)[from:], (*order)[from+1:])
			*order = (*order)[:len(*order)-1]
		}
	}
	if d.Backwards {
		if m.d.Cursor.OrderRow > 0 {
			m.d.Cursor.OrderRow--
		}
	}
	m.d.Cursor2.OrderRow = m.d.Cursor.OrderRow
}

// NewSong

func (m *Model) NewSong() Action { return MakeEnabledAction((*NewSong)(m)) }
func (m *NewSong) Do() {
	m.dialog = NewSongChanges
	(*Model)(m).completeAction(true)
}

// OpenSong

func (m *Model) OpenSong() Action { return MakeEnabledAction((*OpenSong)(m)) }
func (m *OpenSong) Do() {
	m.dialog = OpenSongChanges
	(*Model)(m).completeAction(true)
}

// RequestQuit

func (m *Model) RequestQuit() Action { return MakeEnabledAction((*RequestQuit)(m)) }
func (m *RequestQuit) Do() {
	if !m.quitted {
		m.dialog = QuitChanges
		(*Model)(m).completeAction(true)
	}
}

// ForceQuit

func (m *Model) ForceQuit() Action { return MakeEnabledAction((*ForceQuit)(m)) }
func (m *ForceQuit) Do()           { m.quitted = true }

// SaveSong

func (m *Model) SaveSong() Action { return MakeEnabledAction((*SaveSong)(m)) }
func (m *SaveSong) Do() {
	if m.d.FilePath == "" {
		switch m.dialog {
		case NoDialog:
			m.dialog = SaveAsExplorer
		case NewSongChanges:
			m.dialog = NewSongSaveExplorer
		case OpenSongChanges:
			m.dialog = OpenSongSaveExplorer
		case QuitChanges:
			m.dialog = QuitSaveExplorer
		}
		return
	}
	f, err := os.Create(m.d.FilePath)
	if err != nil {
		(*Model)(m).Alerts().Add("Error creating file: "+err.Error(), Error)
		return
	}
	(*Model)(m).WriteSong(f)
	m.d.ChangedSinceSave = false
}

func (m *Model) DiscardSong() Action { return MakeEnabledAction((*DiscardSong)(m)) }
func (m *DiscardSong) Do()           { (*Model)(m).completeAction(false) }

func (m *Model) SaveSongAs() Action { return MakeEnabledAction((*SaveSongAs)(m)) }
func (m *SaveSongAs) Do()           { m.dialog = SaveAsExplorer }

func (m *Model) Cancel() Action { return MakeEnabledAction((*Cancel)(m)) }
func (m *Cancel) Do()           { m.dialog = NoDialog }

func (m *Model) Export() Action { return MakeEnabledAction((*ExportAction)(m)) }
func (m *ExportAction) Do()     { m.dialog = Export }

func (m *Model) ExportFloat() Action { return MakeEnabledAction((*ExportFloat)(m)) }
func (m *ExportFloat) Do()           { m.dialog = ExportFloatExplorer }

func (m *Model) ExportInt16() Action { return MakeEnabledAction((*ExportInt16)(m)) }
func (m *ExportInt16) Do()           { m.dialog = ExportInt16Explorer }

func (m *Model) ShowLicense() Action { return MakeEnabledAction((*ShowLicense)(m)) }
func (m *ShowLicense) Do()           { m.dialog = License }

func (m *Model) SelectMidiInput(item MIDIDevice) Action {
	return MakeEnabledAction(SelectMidiInput{Item: item, Model: m})
}
func (s SelectMidiInput) Do() {
	m := s.Model
	if err := s.Item.Open(); err == nil {
		message := fmt.Sprintf("Opened MIDI device: %s", s.Item)
		m.Alerts().Add(message, Info)
	} else {
		message := fmt.Sprintf("Could not open MIDI device: %s", s.Item)
		m.Alerts().Add(message, Error)
	}
}

func (m *Model) completeAction(checkSave bool) {
	if checkSave && m.d.ChangedSinceSave {
		return
	}
	switch m.dialog {
	case NewSongChanges, NewSongSaveExplorer:
		c := m.change("NewSong", SongChange, MajorChange)
		m.resetSong()
		m.setLoop(Loop{})
		c()
		m.d.ChangedSinceSave = false
		m.dialog = NoDialog
	case OpenSongChanges, OpenSongSaveExplorer:
		m.dialog = OpenSongOpenExplorer
	case QuitChanges, QuitSaveExplorer:
		m.quitted = true
		m.dialog = NoDialog
	default:
		m.dialog = NoDialog
	}
}

func (m *Model) setPanic(val bool) {
	if m.panic != val {
		m.panic = val
		TrySend(m.broker.ToPlayer, any(PanicMsg{val}))
	}
}

func (m *Model) setLoop(newLoop Loop) {
	if m.loop != newLoop {
		m.loop = newLoop
		TrySend(m.broker.ToPlayer, any(newLoop))
	}
}
