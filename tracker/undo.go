package tracker

var undoSkip = map[string]int{
	"setNote": 10,
}

const maxUndo = 256

func (t *Tracker) SaveUndo() {
	if len(t.undoStack) >= maxUndo {
		t.undoStack = t.undoStack[1:]
	}
	t.undoStack = append(t.undoStack, t.song.Copy())
	t.redoStack = t.redoStack[:0]
}

func (t *Tracker) Undo() {
	if len(t.undoStack) > 0 {
		if len(t.redoStack) >= maxUndo {
			t.redoStack = t.redoStack[1:]
		}
		t.redoStack = append(t.redoStack, t.song.Copy())
		t.LoadSong(t.undoStack[len(t.undoStack)-1])
		t.undoStack = t.undoStack[:len(t.undoStack)-1]
	}
}

func (t *Tracker) Redo() {
	if len(t.redoStack) > 0 {
		if len(t.undoStack) >= maxUndo {
			t.undoStack = t.undoStack[1:]
		}
		t.undoStack = append(t.undoStack, t.song.Copy())
		t.LoadSong(t.redoStack[len(t.redoStack)-1])
		t.redoStack = t.redoStack[:len(t.redoStack)-1]
	}
}
