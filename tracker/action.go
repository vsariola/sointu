package tracker

import (
	"fmt"
	"math"
	"os"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
)

type (
	// Action describes a user action that can be performed on the model. It is
	// usually a button press or a menu item. Action advertises whether it is
	// allowed to be performed or not.
	Action struct {
		do      func()
		allowed func() bool
	}
)

// Action methods

func (e Action) Do() {
	if e.allowed != nil && e.allowed() {
		e.do()
	}
}

func (e Action) Allowed() bool {
	return e.allowed != nil && e.allowed()
}

func Allow(do func()) Action {
	return Action{do: do, allowed: func() bool { return true }}
}

func Check(do func(), allowed func() bool) Action {
	return Action{do: do, allowed: allowed}
}

// Model methods

func (m *Model) AddTrack() Action {
	return Action{
		allowed: func() bool { return m.d.Song.Score.NumVoices() < vm.MAX_VOICES },
		do: func() {
			defer (*Model)(m).change("AddTrack", SongChange, MajorChange)()
			voiceIndex := m.d.Song.Score.FirstVoiceForTrack(m.d.Cursor.Track)
			p := sointu.Patch{defaultInstrument.Copy()}
			t := []sointu.Track{sointu.Track{NumVoices: 1}}
			_, _, ok := m.addVoices(voiceIndex, p, t, (*Model)(m).linkInstrTrack, true)
			m.changeCancel = !ok
		},
	}
}

func (m *Model) DeleteTrack() Action {
	return Action{
		allowed: func() bool { return len(m.d.Song.Score.Tracks) > 0 },
		do:      func() { m.Tracks().List().DeleteElements(false) },
	}
}

func (m *Model) AddInstrument() Action {
	return Action{
		allowed: func() bool { return (*Model)(m).d.Song.Patch.NumVoices() < vm.MAX_VOICES },
		do: func() {
			defer (*Model)(m).change("AddInstrument", SongChange, MajorChange)()
			voiceIndex := m.d.Song.Patch.FirstVoiceForInstrument(m.d.InstrIndex)
			p := sointu.Patch{defaultInstrument.Copy()}
			t := []sointu.Track{sointu.Track{NumVoices: 1}}
			_, _, ok := m.addVoices(voiceIndex, p, t, true, (*Model)(m).linkInstrTrack)
			m.changeCancel = !ok
		},
	}
}

func (m *Model) DeleteInstrument() Action {
	return Action{
		allowed: func() bool { return len((*Model)(m).d.Song.Patch) > 0 },
		do:      func() { m.Instruments().List().DeleteElements(false) },
	}
}

func (m *Model) SplitTrack() Action {
	return Action{
		allowed: func() bool {
			return m.d.Cursor.Track >= 0 && m.d.Cursor.Track < len(m.d.Song.Score.Tracks) && m.d.Song.Score.Tracks[m.d.Cursor.Track].NumVoices > 1
		},
		do: func() {
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
		},
	}
}

func (m *Model) SplitInstrument() Action {
	return Action{
		allowed: func() bool {
			return m.d.InstrIndex >= 0 && m.d.InstrIndex < len(m.d.Song.Patch) && m.d.Song.Patch[m.d.InstrIndex].NumVoices > 1
		},
		do: func() {
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
			m.assignUnitIDs(newInstrument.Units)
			newInstrument.NumVoices = end - middle
			m.d.Song.Patch = append(left, newInstrument)
			m.d.Song.Patch = append(m.d.Song.Patch, right...)
		},
	}
}

func (m *Model) AddUnit(before bool) Action {
	return Allow(func() {
		defer m.change("AddUnitAction", PatchChange, MajorChange)()
		if len(m.d.Song.Patch) == 0 { // no instruments, add one
			instr := sointu.Instrument{NumVoices: 1}
			instr.Units = make([]sointu.Unit, 0, 1)
			m.d.Song.Patch = append(m.d.Song.Patch, instr)
			m.d.UnitIndex = 0
		} else {
			if !before {
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
	})
}

func (m *Model) AddUnitAndThen(callback func()) Action {
	return Allow(func() {
		m.AddUnit(false).Do()
		callback()
	})
}

func (m *Model) DeleteUnit() Action {
	return Action{
		allowed: func() bool {
			return len((*Model)(m).d.Song.Patch) > 0 && len((*Model)(m).d.Song.Patch[(*Model)(m).d.InstrIndex].Units) > 1
		},
		do: func() {
			defer (*Model)(m).change("DeleteUnitAction", PatchChange, MajorChange)()
			m.Units().List().DeleteElements(true)
		},
	}
}

func (m *Model) ClearUnit() Action {
	return Action{
		do: func() {
			defer (*Model)(m).change("DeleteUnitAction", PatchChange, MajorChange)()
			m.d.UnitIndex = max(min(m.d.UnitIndex, len(m.d.Song.Patch[m.d.InstrIndex].Units)-1), 0)
			m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex] = sointu.Unit{}
			m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex].ID = m.maxID() + 1
		},
		allowed: func() bool {
			return m.d.InstrIndex >= 0 &&
				m.d.InstrIndex < len(m.d.Song.Patch) &&
				len(m.d.Song.Patch[m.d.InstrIndex].Units) > 0
		},
	}
}
func (m *Model) Undo() Action {
	return Action{
		allowed: func() bool { return len((*Model)(m).undoStack) > 0 },
		do: func() {
			m.redoStack = append(m.redoStack, m.d.Copy())
			if len(m.redoStack) >= maxUndo {
				copy(m.redoStack, m.redoStack[len(m.redoStack)-maxUndo:])
				m.redoStack = m.redoStack[:maxUndo]
			}
			m.d = m.undoStack[len(m.undoStack)-1]
			m.undoStack = m.undoStack[:len(m.undoStack)-1]
			m.prevUndoKind = ""
			TrySend(m.broker.ToPlayer, any(m.d.Song.Copy()))
		},
	}
}

func (m *Model) Redo() Action {
	return Action{
		allowed: func() bool { return len((*Model)(m).redoStack) > 0 },
		do: func() {
			m.undoStack = append(m.undoStack, m.d.Copy())
			if len(m.undoStack) >= maxUndo {
				copy(m.undoStack, m.undoStack[len(m.undoStack)-maxUndo:])
				m.undoStack = m.undoStack[:maxUndo]
			}
			m.d = m.redoStack[len(m.redoStack)-1]
			m.redoStack = m.redoStack[:len(m.redoStack)-1]
			m.prevUndoKind = ""
			TrySend(m.broker.ToPlayer, any(m.d.Song.Copy()))
		},
	}
}

func (m *Model) AddSemitone() Action {
	return Allow(func() { Table{(*Notes)(m)}.Add(1) })
}

func (m *Model) SubtractSemitone() Action {
	return Allow(func() { Table{(*Notes)(m)}.Add(-1) })
}

func (m *Model) AddOctave() Action {
	return Allow(func() { Table{(*Notes)(m)}.Add(12) })
}

func (m *Model) SubtractOctave() Action {
	return Allow(func() { Table{(*Notes)(m)}.Add(-12) })
}

func (m *Model) EditNoteOff() Action {
	return Allow(func() { Table{(*Notes)(m)}.Fill(0) })
}

func (m *Model) RemoveUnused() Action {
	return Allow(func() {
		defer m.change("RemoveUnusedAction", ScoreChange, MajorChange)()
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
	})
}

func (m *Model) PlayCurrentPos() Action {
	return Action{
		allowed: func() bool { return !m.instrEnlarged },
		do: func() {
			m.setPanic(false)
			m.setLoop(Loop{})
			m.playing = true
			TrySend(m.broker.ToPlayer, any(StartPlayMsg{m.d.Cursor.SongPos}))
		},
	}
}

func (m *Model) PlaySongStart() Action {
	return Action{
		allowed: func() bool { return !m.instrEnlarged },
		do: func() {
			m.setPanic(false)
			m.setLoop(Loop{})
			m.playing = true
			TrySend(m.broker.ToPlayer, any(StartPlayMsg{}))
		},
	}
}

func (m *Model) PlaySelected() Action {
	return Action{
		allowed: func() bool { return !m.instrEnlarged },
		do: func() {
			m.setPanic(false)
			m.playing = true
			l := m.OrderRows().List()
			r := l.listRange()
			newLoop := Loop{r.Start, r.End - r.Start}
			m.setLoop(newLoop)
			TrySend(m.broker.ToPlayer, any(StartPlayMsg{sointu.SongPos{OrderRow: r.Start, PatternRow: 0}}))
		},
	}
}

func (m *Model) PlayFromLoopStart() Action {
	return Action{
		allowed: func() bool { return !m.instrEnlarged },
		do: func() {
			m.setPanic(false)
			if m.loop == (Loop{}) {
				m.PlaySelected().Do()
				return
			}
			m.playing = true
			TrySend(m.broker.ToPlayer, any(StartPlayMsg{sointu.SongPos{OrderRow: m.loop.Start, PatternRow: 0}}))
		},
	}
}

func (m *Model) StopPlaying() Action {
	return Action{
		allowed: func() bool { return true },
		do: func() {
			if !m.playing {
				m.setPanic(true)
				m.setLoop(Loop{})
				return
			}
			m.playing = false
			TrySend(m.broker.ToPlayer, any(IsPlayingMsg{false}))
		},
	}
}

func (m *Model) AddOrderRow(before bool) Action {
	return Allow(func() {
		defer m.change("AddOrderRowAction", ScoreChange, MinorChange)()
		if !before {
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
	})
}

func (m *Model) DeleteOrderRow(backwards bool) Action {
	return Allow(func() {
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
		if backwards {
			if m.d.Cursor.OrderRow > 0 {
				m.d.Cursor.OrderRow--
			}
		}
		m.d.Cursor2.OrderRow = m.d.Cursor.OrderRow
		return
	})
}

func (m *Model) NewSong() Action {
	return Allow(func() {
		m.dialog = NewSongChanges
		m.completeAction(true)
	})
}

func (m *Model) OpenSong() Action {
	return Allow(func() {
		m.dialog = OpenSongChanges
		m.completeAction(true)
	})
}

func (m *Model) RequestQuit() Action {
	return Allow(func() {
		if !m.quitted {
			m.dialog = QuitChanges
			m.completeAction(true)
		}
	})
}

func (m *Model) ForceQuit() Action {
	return Allow(func() {
		m.quitted = true
	})
}

func (m *Model) SaveSong() Action {
	return Allow(func() {
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
			m.Alerts().Add("Error creating file: "+err.Error(), Error)
			return
		}
		m.WriteSong(f)
		m.d.ChangedSinceSave = false
	})
}

func (m *Model) DiscardSong() Action { return Allow(func() { m.completeAction(false) }) }
func (m *Model) SaveSongAs() Action  { return Allow(func() { m.dialog = SaveAsExplorer }) }
func (m *Model) Cancel() Action      { return Allow(func() { m.dialog = NoDialog }) }
func (m *Model) Export() Action      { return Allow(func() { m.dialog = Export }) }
func (m *Model) ExportFloat() Action { return Allow(func() { m.dialog = ExportFloatExplorer }) }
func (m *Model) ExportInt16() Action { return Allow(func() { m.dialog = ExportInt16Explorer }) }
func (m *Model) SelectMidiInput(item MIDIDevice) Action {
	return Allow(func() {
		if err := item.Open(); err == nil {
			message := fmt.Sprintf("Opened MIDI device: %s", item)
			m.Alerts().Add(message, Info)
		} else {
			message := fmt.Sprintf("Could not open MIDI device: %s", item)
			m.Alerts().Add(message, Error)
		}
	})
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
