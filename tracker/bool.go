package tracker

import (
	"fmt"
)

type (
	Bool struct {
		value   BoolValue
		enabler Enabler
	}

	BoolValue interface {
		Value() bool
		SetValue(bool)
	}

	Panic          Model
	IsRecording    Model
	Playing        Model
	InstrEnlarged  Model
	Effect         Model
	TrackMidiIn    Model
	Follow         Model
	UnitSearching  Model
	UnitDisabled   Model
	LoopToggle     Model
	UniquePatterns Model
	Mute           Model
	Solo           Model
	LinkInstrTrack Model
	Oversampling   Model
	InstrEditor    Model
	InstrPresets   Model
	InstrComment   Model
	Thread1        Model
	Thread2        Model
	Thread3        Model
	Thread4        Model
)

func MakeBool(valueEnabler interface {
	BoolValue
	Enabler
}) Bool {
	return Bool{value: valueEnabler, enabler: valueEnabler}
}

func MakeEnabledBool(value BoolValue) Bool {
	return Bool{value: value, enabler: nil}
}

func (v Bool) Toggle() {
	v.SetValue(!v.Value())
}

func (v Bool) SetValue(value bool) {
	if v.Enabled() && v.Value() != value {
		v.value.SetValue(value)
	}
}

func (v Bool) Value() bool {
	if v.value == nil {
		return false
	}
	return v.value.Value()
}

func (v Bool) Enabled() bool {
	if v.enabler == nil {
		return true
	}
	return v.enabler.Enabled()
}

// Thread methods

func (m *Model) getThreadsBit(bit int) bool {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	mask := m.d.Song.Patch[m.d.InstrIndex].ThreadMaskM1 + 1
	return mask&(1<<bit) != 0
}

func (m *Model) setThreadsBit(bit int, value bool) {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return
	}
	defer (*Model)(m).change("ThreadBitMask", PatchChange, MinorChange)()
	mask := m.d.Song.Patch[m.d.InstrIndex].ThreadMaskM1 + 1
	if value {
		mask |= (1 << bit)
	} else {
		mask &^= (1 << bit)
	}
	m.d.Song.Patch[m.d.InstrIndex].ThreadMaskM1 = max(mask-1, 0) // -1 would have all threads disabled, so make that 0 i.e. use at least thread 1
	m.warnAboutCrossThreadSends()
	m.warnNoMultithreadSupport()
}

func (m *Model) warnAboutCrossThreadSends() {
	for i, instr := range m.d.Song.Patch {
		for _, unit := range instr.Units {
			if unit.Type == "send" {
				targetID, ok := unit.Parameters["target"]
				if !ok {
					continue
				}
				it, _, err := m.d.Song.Patch.FindUnit(targetID)
				if err != nil {
					continue
				}
				if instr.ThreadMaskM1 != m.d.Song.Patch[it].ThreadMaskM1 {
					m.Alerts().AddNamed("CrossThreadSend", fmt.Sprintf("Instrument %d '%s' has a send to instrument %d '%s' but they are not on the same threads, which may cause issues", i+1, instr.Name, it+1, m.d.Song.Patch[it].Name), Warning)
					return
				}
			}
		}
	}
}

func (m *Model) warnNoMultithreadSupport() {
	for _, instr := range m.d.Song.Patch {
		if instr.ThreadMaskM1 > 0 && !m.synthers[m.syntherIndex].SupportsMultithreading() {
			m.Alerts().AddNamed("NoMultithreadSupport", "The current synth does not support multithreading and the patch was configured to use more than one thread", Warning)
			return
		}
	}
}

func (m *Model) Thread1() Bool       { return MakeEnabledBool((*Thread1)(m)) }
func (m *Thread1) Value() bool       { return (*Model)(m).getThreadsBit(0) }
func (m *Thread1) SetValue(val bool) { (*Model)(m).setThreadsBit(0, val) }

func (m *Model) Thread2() Bool       { return MakeEnabledBool((*Thread2)(m)) }
func (m *Thread2) Value() bool       { return (*Model)(m).getThreadsBit(1) }
func (m *Thread2) SetValue(val bool) { (*Model)(m).setThreadsBit(1, val) }

func (m *Model) Thread3() Bool       { return MakeEnabledBool((*Thread3)(m)) }
func (m *Thread3) Value() bool       { return (*Model)(m).getThreadsBit(2) }
func (m *Thread3) SetValue(val bool) { (*Model)(m).setThreadsBit(2, val) }

func (m *Model) Thread4() Bool       { return MakeEnabledBool((*Thread4)(m)) }
func (m *Thread4) Value() bool       { return (*Model)(m).getThreadsBit(3) }
func (m *Thread4) SetValue(val bool) { (*Model)(m).setThreadsBit(3, val) }

// Panic methods

func (m *Model) Panic() Bool       { return MakeEnabledBool((*Panic)(m)) }
func (m *Panic) Value() bool       { return m.panic }
func (m *Panic) SetValue(val bool) { (*Model)(m).setPanic(val) }

// IsRecording methods

func (m *Model) IsRecording() Bool { return MakeEnabledBool((*IsRecording)(m)) }
func (m *IsRecording) Value() bool { return (*Model)(m).recording }
func (m *IsRecording) SetValue(val bool) {
	m.recording = val
	m.instrEnlarged = val
	TrySend(m.broker.ToPlayer, any(RecordingMsg{val}))
}

// Playing methods

func (m *Model) Playing() Bool { return MakeBool((*Playing)(m)) }
func (m *Playing) Value() bool { return m.playing }
func (m *Playing) SetValue(val bool) {
	m.playing = val
	if m.playing {
		(*Model)(m).setPanic(false)
		TrySend(m.broker.ToPlayer, any(StartPlayMsg{m.d.Cursor.SongPos}))
	} else {
		TrySend(m.broker.ToPlayer, any(IsPlayingMsg{val}))
	}
}
func (m *Playing) Enabled() bool { return m.playing || !m.instrEnlarged }

// InstrEnlarged methods

func (m *Model) InstrEnlarged() Bool       { return MakeEnabledBool((*InstrEnlarged)(m)) }
func (m *InstrEnlarged) Value() bool       { return m.instrEnlarged }
func (m *InstrEnlarged) SetValue(val bool) { m.instrEnlarged = val }

// InstrEditor methods

func (m *Model) InstrEditor() Bool { return MakeEnabledBool((*InstrEditor)(m)) }
func (m *InstrEditor) Value() bool { return m.d.InstrumentTab == InstrumentEditorTab }
func (m *InstrEditor) SetValue(val bool) {
	if val {
		m.d.InstrumentTab = InstrumentEditorTab
	}
}

func (m *Model) InstrComment() Bool { return MakeEnabledBool((*InstrComment)(m)) }
func (m *InstrComment) Value() bool { return m.d.InstrumentTab == InstrumentCommentTab }
func (m *InstrComment) SetValue(val bool) {
	if val {
		m.d.InstrumentTab = InstrumentCommentTab
	}
}

func (m *Model) InstrPresets() Bool { return MakeEnabledBool((*InstrPresets)(m)) }
func (m *InstrPresets) Value() bool { return m.d.InstrumentTab == InstrumentPresetsTab }
func (m *InstrPresets) SetValue(val bool) {
	if val {
		m.d.InstrumentTab = InstrumentPresetsTab
	}
}

// Follow methods

func (m *Model) Follow() Bool       { return MakeEnabledBool((*Follow)(m)) }
func (m *Follow) Value() bool       { return m.follow }
func (m *Follow) SetValue(val bool) { m.follow = val }

// TrackMidiIn (Midi Input for notes in the tracks)

func (m *Model) TrackMidiIn() Bool       { return MakeEnabledBool((*TrackMidiIn)(m)) }
func (m *TrackMidiIn) Value() bool       { return m.broker.mIDIEventsToGUI.Load() }
func (m *TrackMidiIn) SetValue(val bool) { m.broker.mIDIEventsToGUI.Store(val) }

// Effect methods

func (m *Model) Effect() Bool { return MakeEnabledBool((*Effect)(m)) }
func (m *Effect) Value() bool {
	if m.d.Cursor.Track < 0 || m.d.Cursor.Track >= len(m.d.Song.Score.Tracks) {
		return false
	}
	return m.d.Song.Score.Tracks[m.d.Cursor.Track].Effect
}
func (m *Effect) SetValue(val bool) {
	if m.d.Cursor.Track < 0 || m.d.Cursor.Track >= len(m.d.Song.Score.Tracks) {
		return
	}
	m.d.Song.Score.Tracks[m.d.Cursor.Track].Effect = val
}

// Oversampling methods

func (m *Model) Oversampling() Bool { return MakeEnabledBool((*Oversampling)(m)) }
func (m *Oversampling) Value() bool { return m.oversampling }
func (m *Oversampling) SetValue(val bool) {
	m.oversampling = val
	TrySend(m.broker.ToDetector, MsgToDetector{HasOversampling: true, Oversampling: val})
}

// UnitSearching methods

func (m *Model) UnitSearching() Bool { return MakeEnabledBool((*UnitSearching)(m)) }
func (m *UnitSearching) Value() bool { return m.d.UnitSearching }
func (m *UnitSearching) SetValue(val bool) {
	m.d.UnitSearching = val
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		m.d.UnitSearchString = ""
		return
	}
	if m.d.UnitIndex < 0 || m.d.UnitIndex >= len(m.d.Song.Patch[m.d.InstrIndex].Units) {
		m.d.UnitSearchString = ""
		return
	}
	m.d.UnitSearchString = m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex].Type
}

// UnitDisabled methods

func (m *Model) UnitDisabled() Bool { return MakeBool((*UnitDisabled)(m)) }
func (m *UnitDisabled) Value() bool {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	if m.d.UnitIndex < 0 || m.d.UnitIndex >= len(m.d.Song.Patch[m.d.InstrIndex].Units) {
		return false
	}
	return m.d.Song.Patch[m.d.InstrIndex].Units[m.d.UnitIndex].Disabled
}
func (m *UnitDisabled) SetValue(val bool) {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return
	}
	l := ((*Model)(m)).Units().List()
	r := l.listRange()
	defer (*Model)(m).change("UnitDisabledSet", PatchChange, MajorChange)()
	for i := r.Start; i < r.End; i++ {
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

func (m *Model) LoopToggle() Bool { return MakeEnabledBool((*LoopToggle)(m)) }
func (m *LoopToggle) Value() bool { return m.loop.Length > 0 }
func (t *LoopToggle) SetValue(val bool) {
	m := (*Model)(t)
	newLoop := Loop{}
	if val {
		l := m.OrderRows().List()
		r := l.listRange()
		newLoop = Loop{r.Start, r.End - r.Start}
	}
	m.setLoop(newLoop)
}

// UniquePatterns methods

func (m *Model) UniquePatterns() Bool       { return MakeEnabledBool((*UniquePatterns)(m)) }
func (m *UniquePatterns) Value() bool       { return m.uniquePatterns }
func (m *UniquePatterns) SetValue(val bool) { m.uniquePatterns = val }

// Mute methods
func (m *Model) Mute() Bool { return MakeBool((*Mute)(m)) }
func (m *Mute) Value() bool {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	return m.d.Song.Patch[m.d.InstrIndex].Mute
}
func (m *Mute) SetValue(val bool) {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return
	}
	defer (*Model)(m).change("Mute", PatchChange, MinorChange)()
	a, b := min(m.d.InstrIndex, m.d.InstrIndex2), max(m.d.InstrIndex, m.d.InstrIndex2)
	for i := a; i <= b; i++ {
		if i < 0 || i >= len(m.d.Song.Patch) {
			continue
		}
		m.d.Song.Patch[i].Mute = val
	}
}
func (m *Mute) Enabled() bool { return m.d.InstrIndex >= 0 && m.d.InstrIndex < len(m.d.Song.Patch) }

// Solo methods

func (m *Model) Solo() Bool { return MakeBool((*Solo)(m)) }
func (m *Solo) Value() bool {
	a, b := min(m.d.InstrIndex, m.d.InstrIndex2), max(m.d.InstrIndex, m.d.InstrIndex2)
	for i := range m.d.Song.Patch {
		if i < 0 || i >= len(m.d.Song.Patch) {
			continue
		}
		if (i >= a && i <= b) == m.d.Song.Patch[i].Mute {
			return false
		}
	}
	return true
}
func (m *Solo) SetValue(val bool) {
	defer (*Model)(m).change("Solo", PatchChange, MinorChange)()
	a, b := min(m.d.InstrIndex, m.d.InstrIndex2), max(m.d.InstrIndex, m.d.InstrIndex2)
	for i := range m.d.Song.Patch {
		if i < 0 || i >= len(m.d.Song.Patch) {
			continue
		}
		m.d.Song.Patch[i].Mute = !(i >= a && i <= b) && val
	}
}
func (m *Solo) Enabled() bool { return m.d.InstrIndex >= 0 && m.d.InstrIndex < len(m.d.Song.Patch) }

// LinkInstrTrack methods

func (m *Model) LinkInstrTrack() Bool       { return MakeEnabledBool((*LinkInstrTrack)(m)) }
func (m *LinkInstrTrack) Value() bool       { return m.linkInstrTrack }
func (m *LinkInstrTrack) SetValue(val bool) { m.linkInstrTrack = val }
