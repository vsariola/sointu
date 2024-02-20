package tracker

type (
	Bool struct {
		BoolData
	}

	BoolData interface {
		Value() bool
		Enabled() bool
		setValue(bool)
	}

	Panic           Model
	IsRecording     Model
	Playing         Model
	InstrEnlarged   Model
	Effect          Model
	CommentExpanded Model
	NoteTracking    Model
	UnitSearching   Model
	UnitDisabled    Model
	LoopToggle      Model
)

func (v Bool) Toggle() {
	v.Set(!v.Value())
}

func (v Bool) Set(value bool) {
	if v.Enabled() && v.Value() != value {
		v.setValue(value)
	}
}

// Model methods

func (m *Model) Panic() *Panic                     { return (*Panic)(m) }
func (m *Model) IsRecording() *IsRecording         { return (*IsRecording)(m) }
func (m *Model) Playing() *Playing                 { return (*Playing)(m) }
func (m *Model) InstrEnlarged() *InstrEnlarged     { return (*InstrEnlarged)(m) }
func (m *Model) Effect() *Effect                   { return (*Effect)(m) }
func (m *Model) CommentExpanded() *CommentExpanded { return (*CommentExpanded)(m) }
func (m *Model) NoteTracking() *NoteTracking       { return (*NoteTracking)(m) }
func (m *Model) UnitSearching() *UnitSearching     { return (*UnitSearching)(m) }
func (m *Model) UnitDisabled() *UnitDisabled       { return (*UnitDisabled)(m) }
func (m *Model) LoopToggle() *LoopToggle           { return (*LoopToggle)(m) }

// Panic methods

func (m *Panic) Bool() Bool  { return Bool{m} }
func (m *Panic) Value() bool { return m.panic }
func (m *Panic) setValue(val bool) {
	m.panic = val
	(*Model)(m).send(PanicMsg{val})
}
func (m *Panic) Enabled() bool { return true }

// IsRecording methods

func (m *IsRecording) Bool() Bool  { return Bool{m} }
func (m *IsRecording) Value() bool { return (*Model)(m).recording }
func (m *IsRecording) setValue(val bool) {
	m.recording = val
	m.instrEnlarged = val
	(*Model)(m).send(RecordingMsg{val})
}
func (m *IsRecording) Enabled() bool { return true }

// Playing methods

func (m *Playing) Bool() Bool  { return Bool{m} }
func (m *Playing) Value() bool { return m.playing }
func (m *Playing) setValue(val bool) {
	m.playing = val
	if m.playing {
		(*Model)(m).send(StartPlayMsg{m.d.Cursor.SongPos})
	} else {
		(*Model)(m).send(IsPlayingMsg{val})
	}
}
func (m *Playing) Enabled() bool { return m.playing || !m.instrEnlarged }

// InstrEnlarged methods

func (m *InstrEnlarged) Bool() Bool        { return Bool{m} }
func (m *InstrEnlarged) Value() bool       { return m.instrEnlarged }
func (m *InstrEnlarged) setValue(val bool) { m.instrEnlarged = val }
func (m *InstrEnlarged) Enabled() bool     { return true }

// CommentExpanded methods

func (m *CommentExpanded) Bool() Bool        { return Bool{m} }
func (m *CommentExpanded) Value() bool       { return m.commentExpanded }
func (m *CommentExpanded) setValue(val bool) { m.commentExpanded = val }
func (m *CommentExpanded) Enabled() bool     { return true }

// NoteTracking methods

func (m *NoteTracking) Bool() Bool        { return Bool{m} }
func (m *NoteTracking) Value() bool       { return m.playing && m.noteTracking }
func (m *NoteTracking) setValue(val bool) { m.noteTracking = val }
func (m *NoteTracking) Enabled() bool     { return m.playing }

// Effect methods

func (m *Effect) Bool() Bool { return Bool{m} }
func (m *Effect) Value() bool {
	if m.d.Cursor.Track < 0 || m.d.Cursor.Track >= len(m.d.Song.Score.Tracks) {
		return false
	}
	return m.d.Song.Score.Tracks[m.d.Cursor.Track].Effect
}
func (m *Effect) setValue(val bool) {
	if m.d.Cursor.Track < 0 || m.d.Cursor.Track >= len(m.d.Song.Score.Tracks) {
		return
	}
	m.d.Song.Score.Tracks[m.d.Cursor.Track].Effect = val
}
func (m *Effect) Enabled() bool { return true }

// UnitSearching methods

func (m *UnitSearching) Bool() Bool  { return Bool{m} }
func (m *UnitSearching) Value() bool { return m.d.UnitSearching }
func (m *UnitSearching) setValue(val bool) {
	m.d.UnitSearching = val
	if !val {
		m.d.UnitSearchString = ""
	}
}
func (m *UnitSearching) Enabled() bool { return true }

// UnitDisabled methods

func (m *UnitDisabled) Bool() Bool { return Bool{m} }
func (m *UnitDisabled) Value() bool {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	if m.d.UnitIndex < 0 || m.d.UnitIndex >= len(m.d.Song.Patch[m.d.InstrIndex].Units) {
		return false
	}
	return m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex].Disabled
}
func (m *UnitDisabled) setValue(val bool) {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return
	}
	l := ((*Model)(m)).Units().List()
	a, b := l.listRange()
	defer (*Model)(m).change("UnitDisabledSet", PatchChange, MajorChange)()
	for i := a; i <= b; i++ {
		m.d.Song.Patch[m.d.InstrIndex].Units[i].Disabled = val
	}
}
func (m *UnitDisabled) Enabled() bool {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	if len(m.d.Song.Patch[m.d.InstrIndex].Units) == 0 {
		return false
	}
	return true
}

// LoopToggle methods

func (m *LoopToggle) Bool() Bool  { return Bool{m} }
func (m *LoopToggle) Value() bool { return m.d.Loop.Length > 0 }
func (t *LoopToggle) setValue(val bool) {
	m := (*Model)(t)
	defer m.change("SetLoopAction", LoopChange, MinorChange)()
	if !val {
		m.d.Loop = Loop{}
		return
	}
	l := m.OrderRows().List()
	a, b := l.listRange()
	m.d.Loop = Loop{a, b - a + 1}
}
func (m *LoopToggle) Enabled() bool { return true }
