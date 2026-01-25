package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// History returns the History view of the model, containing methods to manipulate
// the undo/redo history and saving recovery files.
func (m *Model) History() *HistoryModel { return (*HistoryModel)(m) }

type HistoryModel Model

// Undo returns an Action to undo the last change.
func (m *HistoryModel) Undo() Action { return MakeAction((*historyUndo)(m)) }

type historyUndo HistoryModel

func (m *historyUndo) Enabled() bool { return len((*Model)(m).undoStack) > 0 }
func (m *historyUndo) Do() {
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

// Redo returns an Action to redo the last undone change.
func (m *HistoryModel) Redo() Action { return MakeAction((*historyRedo)(m)) }

type historyRedo HistoryModel

func (m *historyRedo) Enabled() bool { return len((*Model)(m).redoStack) > 0 }
func (m *historyRedo) Do() {
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

// MarshalRecovery marshals the current model data to a byte slice for recovery
// saving.
func (m *HistoryModel) MarshalRecovery() []byte {
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

// SaveRecovery saves the current model data to the recovery file on disk if
// there are unsaved changes.
func (m *HistoryModel) SaveRecovery() error {
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

// UnmarshalRecovery unmarshals the model data from a byte slice, then checking
// if a recovery file exists on disk and loading it instead.
func (m *HistoryModel) UnmarshalRecovery(bytes []byte) {
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
	(*Model)(m).updateDeriveData(SongChange)
}
