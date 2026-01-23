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
	// The underlying Doer can optionally implement the Enabler interface to
	// decide if the action is enabled or not; if it does not implement the
	// Enabler interface, the action is always allowed.
	Action struct {
		doer Doer
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
)

// Action methods

func MakeAction(doer Doer) Action {
	return Action{doer: doer}
}

func (a Action) Do() {
	e, ok := a.doer.(Enabler)
	if ok && !e.Enabled() {
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
	e, ok := a.doer.(Enabler)
	if !ok {
		return true // not enabler, always allowed
	}
	return e.Enabled()
}

// addTrack
type addTrack Model

func (m *Model) AddTrack() Action { return MakeAction((*addTrack)(m)) }
func (m *addTrack) Enabled() bool { return m.d.Song.Score.NumVoices() < vm.MAX_VOICES }
func (m *addTrack) Do() {
	defer (*Model)(m).change("AddTrack", SongChange, MajorChange)()
	voiceIndex := m.d.Song.Score.FirstVoiceForTrack(m.d.Cursor.Track)
	p := sointu.Patch{defaultInstrument.Copy()}
	t := []sointu.Track{{NumVoices: 1}}
	_, _, ok := (*Model)(m).addVoices(voiceIndex, p, t, (*Model)(m).linkInstrTrack, true)
	m.changeCancel = !ok
}

// deleteTrack
type deleteTrack Model

func (m *Model) DeleteTrack() Action { return MakeAction((*deleteTrack)(m)) }
func (m *deleteTrack) Enabled() bool { return len(m.d.Song.Score.Tracks) > 0 }
func (m *deleteTrack) Do()           { (*Model)(m).Tracks().DeleteElements(false) }

// addInstrument
type addInstrument Model

func (m *Model) AddInstrument() Action { return MakeAction((*addInstrument)(m)) }
func (m *addInstrument) Enabled() bool { return (*Model)(m).d.Song.Patch.NumVoices() < vm.MAX_VOICES }
func (m *addInstrument) Do() {
	defer (*Model)(m).change("AddInstrument", SongChange, MajorChange)()
	voiceIndex := m.d.Song.Patch.FirstVoiceForInstrument(m.d.InstrIndex)
	p := sointu.Patch{defaultInstrument.Copy()}
	t := []sointu.Track{{NumVoices: 1}}
	_, _, ok := (*Model)(m).addVoices(voiceIndex, p, t, true, (*Model)(m).linkInstrTrack)
	m.changeCancel = !ok
}

// deleteInstrument
type deleteInstrument Model

func (m *Model) DeleteInstrument() Action { return MakeAction((*deleteInstrument)(m)) }
func (m *deleteInstrument) Enabled() bool { return len((*Model)(m).d.Song.Patch) > 0 }
func (m *deleteInstrument) Do()           { (*Model)(m).Instruments().DeleteElements(false) }

// splitTrack
type splitTrack Model

func (m *Model) SplitTrack() Action { return MakeAction((*splitTrack)(m)) }
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

// splitInstrument
type splitInstrument Model

func (m *Model) SplitInstrument() Action { return MakeAction((*splitInstrument)(m)) }
func (m *splitInstrument) Enabled() bool {
	return m.d.InstrIndex >= 0 && m.d.InstrIndex < len(m.d.Song.Patch) && m.d.Song.Patch[m.d.InstrIndex].NumVoices > 1
}
func (m *splitInstrument) Do() {
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

// addUnit
type addUnit struct {
	Before bool
	*Model
}

func (m *Model) AddUnit(before bool) Action {
	return MakeAction(addUnit{Before: before, Model: m})
}
func (a addUnit) Do() {
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

// deleteUnit
type deleteUnit Model

func (m *Model) DeleteUnit() Action { return MakeAction((*deleteUnit)(m)) }
func (m *deleteUnit) Enabled() bool {
	i := (*Model)(m).d.InstrIndex
	return i >= 0 && i < len((*Model)(m).d.Song.Patch) && len((*Model)(m).d.Song.Patch[i].Units) > 1
}
func (m *deleteUnit) Do() {
	defer (*Model)(m).change("DeleteUnitAction", PatchChange, MajorChange)()
	(*Model)(m).Units().DeleteElements(true)
}

// clearUnit
type clearUnit Model

func (m *Model) ClearUnit() Action { return MakeAction((*clearUnit)(m)) }
func (m *clearUnit) Enabled() bool {
	i := (*Model)(m).d.InstrIndex
	return i >= 0 && i < len(m.d.Song.Patch) && len(m.d.Song.Patch[i].Units) > 0
}
func (m *clearUnit) Do() {
	defer (*Model)(m).change("DeleteUnitAction", PatchChange, MajorChange)()
	l := ((*Model)(m)).Units()
	r := l.listRange()
	for i := r.Start; i < r.End; i++ {
		m.d.Song.Patch[m.d.InstrIndex].Units[i] = sointu.Unit{}
		m.d.Song.Patch[m.d.InstrIndex].Units[i].ID = (*Model)(m).maxID() + 1
	}
}

// undo
type undo Model

func (m *Model) Undo() Action { return MakeAction((*undo)(m)) }
func (m *undo) Enabled() bool { return len((*Model)(m).undoStack) > 0 }
func (m *undo) Do() {
	m.redoStack = append(m.redoStack, m.d.Copy())
	if len(m.redoStack) >= maxUndo {
		copy(m.redoStack, m.redoStack[len(m.redoStack)-maxUndo:])
		m.redoStack = m.redoStack[:maxUndo]
	}
	m.d = m.undoStack[len(m.undoStack)-1]
	m.undoStack = m.undoStack[:len(m.undoStack)-1]
	m.prevUndoKind = ""
	(*Model)(m).updateDeriveData(SongChange)
	TrySend(m.broker.ToPlayer, any(m.d.Song.Copy()))
}

// redo
type redo Model

func (m *Model) Redo() Action { return MakeAction((*redo)(m)) }
func (m *redo) Enabled() bool { return len((*Model)(m).redoStack) > 0 }
func (m *redo) Do() {
	m.undoStack = append(m.undoStack, m.d.Copy())
	if len(m.undoStack) >= maxUndo {
		copy(m.undoStack, m.undoStack[len(m.undoStack)-maxUndo:])
		m.undoStack = m.undoStack[:maxUndo]
	}
	m.d = m.redoStack[len(m.redoStack)-1]
	m.redoStack = m.redoStack[:len(m.redoStack)-1]
	m.prevUndoKind = ""
	(*Model)(m).updateDeriveData(SongChange)
	TrySend(m.broker.ToPlayer, any(m.d.Song.Copy()))
}

// AddSemiTone
type addSemitone Model

func (m *Model) AddSemitone() Action { return MakeAction((*addSemitone)(m)) }
func (m *addSemitone) Do()           { Table{(*Notes)(m)}.Add(1, false) }

// subtractSemitone
type subtractSemitone Model

func (m *Model) SubtractSemitone() Action { return MakeAction((*subtractSemitone)(m)) }
func (m *subtractSemitone) Do()           { Table{(*Notes)(m)}.Add(-1, false) }

// addOctave
type addOctave Model

func (m *Model) AddOctave() Action { return MakeAction((*addOctave)(m)) }
func (m *addOctave) Do()           { Table{(*Notes)(m)}.Add(1, true) }

// subtractOctave
type subtractOctave Model

func (m *Model) SubtractOctave() Action { return MakeAction((*subtractOctave)(m)) }
func (m *subtractOctave) Do()           { Table{(*Notes)(m)}.Add(-1, true) }

// editNoteOff
type editNoteOff Model

func (m *Model) EditNoteOff() Action { return MakeAction((*editNoteOff)(m)) }
func (m *editNoteOff) Do()           { Table{(*Notes)(m)}.Fill(0) }

// removeUnused
type removeUnused Model

func (m *Model) RemoveUnused() Action { return MakeAction((*removeUnused)(m)) }
func (m *removeUnused) Do() {
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

// playCurrentPos
type playCurrentPos Model

func (m *Model) PlayCurrentPos() Action { return MakeAction((*playCurrentPos)(m)) }
func (m *playCurrentPos) Enabled() bool { return !m.instrEnlarged }
func (m *playCurrentPos) Do() {
	(*Model)(m).setPanic(false)
	(*Model)(m).setLoop(Loop{})
	m.playing = true
	TrySend(m.broker.ToPlayer, any(StartPlayMsg{m.d.Cursor.SongPos}))
}

// playSongStart
type playSongStart Model

func (m *Model) PlaySongStart() Action { return MakeAction((*playSongStart)(m)) }
func (m *playSongStart) Enabled() bool { return !m.instrEnlarged }
func (m *playSongStart) Do() {
	(*Model)(m).setPanic(false)
	(*Model)(m).setLoop(Loop{})
	m.playing = true
	TrySend(m.broker.ToPlayer, any(StartPlayMsg{}))
}

// playSelected
type playSelected Model

func (m *Model) PlaySelected() Action { return MakeAction((*playSelected)(m)) }
func (m *playSelected) Enabled() bool { return !m.instrEnlarged }
func (m *playSelected) Do() {
	(*Model)(m).setPanic(false)
	m.playing = true
	l := (*Model)(m).OrderRows()
	r := l.listRange()
	newLoop := Loop{r.Start, r.End - r.Start}
	(*Model)(m).setLoop(newLoop)
	TrySend(m.broker.ToPlayer, any(StartPlayMsg{sointu.SongPos{OrderRow: r.Start, PatternRow: 0}}))
}

// playFromLoopStart
type playFromLoopStart Model

func (m *Model) PlayFromLoopStart() Action { return MakeAction((*playFromLoopStart)(m)) }
func (m *playFromLoopStart) Enabled() bool { return !m.instrEnlarged }
func (m *playFromLoopStart) Do() {
	(*Model)(m).setPanic(false)
	if m.loop == (Loop{}) {
		(*Model)(m).PlaySelected().Do()
		return
	}
	m.playing = true
	TrySend(m.broker.ToPlayer, any(StartPlayMsg{sointu.SongPos{OrderRow: m.loop.Start, PatternRow: 0}}))
}

// stopPlaying
type stopPlaying Model

func (m *Model) StopPlaying() Action { return MakeAction((*stopPlaying)(m)) }
func (m *stopPlaying) Do() {
	if !m.playing {
		(*Model)(m).setPanic(true)
		(*Model)(m).setLoop(Loop{})
		return
	}
	m.playing = false
	TrySend(m.broker.ToPlayer, any(IsPlayingMsg{false}))
}

// addOrderRow
type addOrderRow struct {
	Before bool
	*Model
}

func (m *Model) AddOrderRow(before bool) Action {
	return MakeAction(addOrderRow{Before: before, Model: m})
}
func (a addOrderRow) Do() {
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

// deleteOrderRow
type deleteOrderRow struct {
	Backwards bool
	*Model
}

func (m *Model) DeleteOrderRow(backwards bool) Action {
	return MakeAction(deleteOrderRow{Backwards: backwards, Model: m})
}
func (d deleteOrderRow) Do() {
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

// chooseSendSource
type chooseSendSource struct {
	ID int
	*Model
}

func (m *Model) IsChoosingSendTarget() bool {
	return m.d.SendSource > 0
}

func (m *Model) ChooseSendSource(id int) Action {
	return MakeAction(chooseSendSource{ID: id, Model: m})
}
func (s chooseSendSource) Do() {
	defer (*Model)(s.Model).change("ChooseSendSource", NoChange, MinorChange)()
	if s.Model.d.SendSource == s.ID {
		s.Model.d.SendSource = 0 // unselect
		return
	}
	s.Model.d.SendSource = s.ID
}

// chooseSendTarget
type chooseSendTarget struct {
	ID   int
	Port int
	*Model
}

func (m *Model) ChooseSendTarget(id int, port int) Action {
	return MakeAction(chooseSendTarget{ID: id, Port: port, Model: m})
}
func (s chooseSendTarget) Do() {
	defer (*Model)(s.Model).change("ChooseSendTarget", SongChange, MinorChange)()
	sourceID := (*Model)(s.Model).d.SendSource
	s.d.SendSource = 0
	if sourceID <= 0 || s.ID <= 0 || s.Port < 0 || s.Port > 7 {
		return
	}
	si, su, err := s.d.Song.Patch.FindUnit(sourceID)
	if err != nil {
		return
	}
	s.d.Song.Patch[si].Units[su].Parameters["target"] = s.ID
	s.d.Song.Patch[si].Units[su].Parameters["port"] = s.Port
}

// newSong
type newSong Model

func (m *Model) NewSong() Action { return MakeAction((*newSong)(m)) }
func (m *newSong) Do() {
	m.dialog = NewSongChanges
	(*Model)(m).completeAction(true)
}

// openSong
type openSong Model

func (m *Model) OpenSong() Action { return MakeAction((*openSong)(m)) }
func (m *openSong) Do() {
	m.dialog = OpenSongChanges
	(*Model)(m).completeAction(true)
}

// requestQuit
type requestQuit Model

func (m *Model) RequestQuit() Action { return MakeAction((*requestQuit)(m)) }
func (m *requestQuit) Do() {
	if !m.quitted {
		m.dialog = QuitChanges
		(*Model)(m).completeAction(true)
	}
}

// forceQuit
type forceQuit Model

func (m *Model) ForceQuit() Action { return MakeAction((*forceQuit)(m)) }
func (m *forceQuit) Do()           { m.quitted = true }

// saveSong
type saveSong Model

func (m *Model) SaveSong() Action { return MakeAction((*saveSong)(m)) }
func (m *saveSong) Do() {
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

type discardSong Model

func (m *Model) DiscardSong() Action { return MakeAction((*discardSong)(m)) }
func (m *discardSong) Do()           { (*Model)(m).completeAction(false) }

type saveSongAs Model

func (m *Model) SaveSongAs() Action { return MakeAction((*saveSongAs)(m)) }
func (m *saveSongAs) Do()           { m.dialog = SaveAsExplorer }

type cancel Model

func (m *Model) Cancel() Action { return MakeAction((*cancel)(m)) }
func (m *cancel) Do()           { m.dialog = NoDialog }

type exportAction Model

func (m *Model) Export() Action { return MakeAction((*exportAction)(m)) }
func (m *exportAction) Do()     { m.dialog = Export }

type exportFloat Model

func (m *Model) ExportFloat() Action { return MakeAction((*exportFloat)(m)) }
func (m *exportFloat) Do()           { m.dialog = ExportFloatExplorer }

type ExportInt16 Model

func (m *Model) ExportInt16() Action { return MakeAction((*ExportInt16)(m)) }
func (m *ExportInt16) Do()           { m.dialog = ExportInt16Explorer }

type showLicense Model

func (m *Model) ShowLicense() Action { return MakeAction((*showLicense)(m)) }
func (m *showLicense) Do()           { m.dialog = License }

type selectMidiInput struct {
	Item string
	*Model
}

func (m *Model) SelectMidiInput(item string) Action {
	return MakeAction(selectMidiInput{Item: item, Model: m})
}
func (s selectMidiInput) Do() {
	m := s.Model
	if err := s.Model.MIDI.Open(s.Item); err == nil {
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
