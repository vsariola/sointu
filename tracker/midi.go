package tracker

import (
	"encoding/json"
	"fmt"
)

type MIDIModel Model

func (m *Model) MIDI() *MIDIModel { return (*MIDIModel)(m) }

type (
	midiState struct {
		noteEventsToGui bool
		binding         bool

		currentInput MIDIInputDevice
		context      MIDIContext
		inputs       []MIDIInputDevice
	}

	MIDIContext interface {
		Inputs(yield func(input MIDIInputDevice) bool)
		Close()
		Support() MIDISupport
	}

	MIDIInputDevice interface {
		Open() error
		Close() error
		IsOpen() bool
		String() string
	}

	MIDISupport int
)

const (
	MIDISupportNotCompiled MIDISupport = iota
	MIDISupportNoDriver
	MIDISupported
)

// Refresh
func (m *MIDIModel) Refresh() Action { return MakeAction((*midiRefresh)(m)) }

type midiRefresh MIDIModel

func (m *midiRefresh) Do() {
	if m.midi.context == nil {
		return
	}
	m.midi.inputs = m.midi.inputs[:0]
	for i := range m.midi.context.Inputs {
		m.midi.inputs = append(m.midi.inputs, i)
		if m.midi.currentInput != nil && i.String() == m.midi.currentInput.String() {
			m.midi.currentInput.Close()
			m.midi.currentInput = nil
			if err := i.Open(); err != nil {
				(*Model)(m).Alerts().Add(fmt.Sprintf("Failed to reopen MIDI input port: %s", err.Error()), Error)
				continue
			}
			m.midi.currentInput = i
		}
	}
}

// InputDevices can be iterated to get string names of all the MIDI input
// devices.
func (m *MIDIModel) Input() Int { return MakeInt((*midiInputDevices)(m)) }

type midiInputDevices MIDIModel

func (m *midiInputDevices) Value() int {
	if m.midi.currentInput == nil {
		return 0
	}
	for i, d := range m.midi.inputs {
		if d == m.midi.currentInput {
			return i + 1
		}
	}
	return 0
}
func (m *midiInputDevices) SetValue(val int) bool {
	if val < 0 || val > len(m.midi.inputs) {
		return false
	}
	if m.midi.currentInput != nil {
		if err := m.midi.currentInput.Close(); err != nil {
			(*Model)(m).Alerts().Add(fmt.Sprintf("Failed to close current MIDI input port: %s", err.Error()), Error)
		}
		m.midi.currentInput = nil
	}
	if val == 0 {
		return true
	}
	newInput := m.midi.inputs[val-1]
	if err := newInput.Open(); err != nil {
		(*Model)(m).Alerts().Add(fmt.Sprintf("Failed to open MIDI input port: %s", err.Error()), Error)
		return false
	}
	m.midi.currentInput = newInput
	(*Model)(m).Alerts().Add(fmt.Sprintf("Opened MIDI input port: %s", newInput.String()), Info)
	return true
}
func (m *midiInputDevices) Range() RangeInclusive {
	return RangeInclusive{Min: 0, Max: len(m.midi.inputs)}
}
func (m *midiInputDevices) StringOf(value int) string {
	if value < 0 || value > len(m.midi.inputs) {
		return ""
	}
	if value == 0 {
		switch m.midi.context.Support() {
		case MIDISupportNotCompiled:
			return "Not compiled"
		case MIDISupportNoDriver:
			return "No driver"
		default:
			return "Closed"
		}
	}
	return m.midi.inputs[value-1].String()
}

// InputtingNotes returns a Bool controlling whether the MIDI events are used
// just to trigger instruments, or if the note events are used to input notes to
// the note table.
func (m *MIDIModel) InputtingNotes() Bool { return MakeBool((*midiInputtingNotes)(m)) }

type midiInputtingNotes Model

func (m *midiInputtingNotes) Value() bool { return m.midi.noteEventsToGui }
func (m *midiInputtingNotes) SetValue(val bool) {
	m.midi.noteEventsToGui = val
	TrySend(m.broker.ToMIDIRouter, any(setNoteEventsToGUI(val)))
}

type setNoteEventsToGUI bool

func runMIDIRouter(broker *Broker) {
	noteEventsToGUI := false
	for {
		select {
		case <-broker.CloseMIDIRouter:
			close(broker.FinishedMIDIRouter)
			return
		case msg := <-broker.ToMIDIRouter:
			switch m := msg.(type) {
			case setNoteEventsToGUI:
				noteEventsToGUI = bool(m)
			case *NoteEvent:
				if noteEventsToGUI {
					TrySend(broker.ToGUI, msg)
					continue
				}
				TrySend(broker.ToPlayer, msg)
			case *ControlChange:
				TrySend(broker.ToModel, MsgToModel{Data: msg})
			}
		}
	}
}

// Binding returns a Bool controlling whether the next received MIDI controller
// event is used to bind a parameter.
func (m *MIDIModel) Binding() Bool { return MakeBool((*midiBinding)(m)) }

type midiBinding MIDIModel

func (m *midiBinding) Value() bool { return m.midi.binding }
func (m *midiBinding) SetValue(val bool) {
	m.midi.binding = val
	if val {
		(*Model)(m).Alerts().Add("Move a MIDI controller to bind it to the selected parameter", Info)
	}
}

// Unbind removes the MIDI binding for the currently selected parameter.
func (m *MIDIModel) Unbind() Action { return MakeAction((*midiUnbind)(m)) }

type midiUnbind MIDIModel

func (m *midiUnbind) Enabled() bool {
	p, ok := (*MIDIModel)(m).selectedParam()
	if !ok {
		return false
	}
	_, ok = m.d.MIDIBindings.GetControl(p)
	return ok
}
func (m *midiUnbind) Do() {
	p, _ := (*MIDIModel)(m).selectedParam()
	m.d.MIDIBindings.UnlinkParam(p)
	(*Model)(m).Alerts().Add("Removed MIDI controller bindings for the selected parameter", Info)
}

// UnbindAll removes all MIDI bindings.
func (m *MIDIModel) UnbindAll() Action { return MakeAction((*midiUnbindAll)(m)) }

type midiUnbindAll MIDIModel

func (m *midiUnbindAll) Enabled() bool { return len(m.d.MIDIBindings.ControlBindings) > 0 }
func (m *midiUnbindAll) Do() {
	m.d.MIDIBindings = MIDIBindings{}
	(*Model)(m).Alerts().Add("Removed all MIDI controller bindings", Info)
}

func (m *MIDIModel) selectedParam() (MIDIParam, bool) {
	point := (*Model)(m).Params().Table().Cursor()
	item := (*Model)(m).Params().Item(point)
	if _, ok := item.vtable.(*namedParameter); !ok {
		return MIDIParam{}, false
	}
	r := item.Range()
	value := MIDIParam{
		Id:    item.unit.ID,
		Param: item.up.Name,
		Min:   r.Min,
		Max:   r.Max,
	}
	return value, true
}

func (m *MIDIModel) handleControlEvent(e ControlChange) {
	key := MIDIControl{Channel: e.Channel, Control: e.Control}
	if m.midi.binding {
		m.midi.binding = false
		value, ok := m.selectedParam()
		if !ok {
			(*Model)(m).Alerts().Add("Cannot bind MIDI controller to this parameter type", Warning)
			return
		}
		m.d.MIDIBindings.Link(key, value)
		(*Model)(m).Alerts().Add(fmt.Sprintf("Bound MIDI CC %d on channel %d to %s", key.Control, key.Channel+1, value.Param), Info)
	}
	t, ok := m.d.MIDIBindings.GetParam(key)
	if !ok {
		return
	}
	i, u, err := m.d.Song.Patch.FindUnit(t.Id)
	if err != nil {
		return
	}
	// +62 is chose so that the center position of a typical MIDI controller,
	// which is 64, maps to 64 of a 0..128 range Sointu parameter. From there
	// on, 65 maps to 66 and, importantly, 127 maps to 128.
	newVal := (e.Value*(t.Max-t.Min)+62)/127 + t.Min
	if m.d.Song.Patch[i].Units[u].Parameters[t.Param] == newVal {
		return
	}
	defer (*Model)(m).change("MIDIControlChange", PatchChange, MinorChange)()
	m.d.Song.Patch[i].Units[u].Parameters[t.Param] = newVal
}

type (
	// Two-way map between MIDI controls and parameters that makes sure only one control channel is linked to only one parameter and vice versa.
	MIDIBindings struct {
		ControlBindings map[MIDIControl]MIDIParam
		ParamBindings   map[MIDIParam]MIDIControl
	}

	MIDIParam struct {
		Id       int
		Param    string
		Min, Max int
	}

	MIDIControl struct{ Channel, Control int }

	midiControlParam struct {
		Control MIDIControl
		Param   MIDIParam
	}
)

// marshal as slice of bindings cause json doesn't support marshaling maps with
// struct keys
func (t *MIDIBindings) UnmarshalJSON(data []byte) error {
	var bindings []midiControlParam
	err := json.Unmarshal(data, &bindings)
	if err != nil {
		return err
	}
	for _, b := range bindings {
		t.Link(b.Control, b.Param)
	}
	return nil
}

func (t MIDIBindings) MarshalJSON() ([]byte, error) {
	var bindings []midiControlParam
	for k, v := range t.ControlBindings {
		bindings = append(bindings, midiControlParam{Control: k, Param: v})
	}
	return json.Marshal(bindings)
}

func (t *MIDIBindings) GetParam(m MIDIControl) (MIDIParam, bool) {
	if t.ControlBindings == nil {
		return MIDIParam{}, false
	}
	p, ok := t.ControlBindings[m]
	return p, ok
}

func (t *MIDIBindings) GetControl(p MIDIParam) (MIDIControl, bool) {
	if t.ParamBindings == nil {
		return MIDIControl{}, false
	}
	c, ok := t.ParamBindings[p]
	return c, ok
}

func (t *MIDIBindings) Link(m MIDIControl, p MIDIParam) {
	if t.ControlBindings == nil {
		t.ControlBindings = make(map[MIDIControl]MIDIParam)
	}
	if t.ParamBindings == nil {
		t.ParamBindings = make(map[MIDIParam]MIDIControl)
	}
	if p, ok := t.ControlBindings[m]; ok {
		delete(t.ParamBindings, p)
	}
	if m, ok := t.ParamBindings[p]; ok {
		delete(t.ControlBindings, m)
	}
	t.ControlBindings[m] = p
	t.ParamBindings[p] = m
}

func (t *MIDIBindings) UnlinkParam(p MIDIParam) {
	if t.ParamBindings == nil {
		return
	}
	if c, ok := t.ParamBindings[p]; ok {
		delete(t.ParamBindings, p)
		delete(t.ControlBindings, c)
	}
}

func (t *MIDIBindings) Copy() MIDIBindings {
	ret := MIDIBindings{
		ControlBindings: make(map[MIDIControl]MIDIParam, len(t.ControlBindings)),
		ParamBindings:   make(map[MIDIParam]MIDIControl, len(t.ParamBindings)),
	}
	for k, v := range t.ControlBindings {
		ret.ControlBindings[k] = v
	}
	for k, v := range t.ParamBindings {
		ret.ParamBindings[k] = v
	}
	return ret
}

// NullMIDIContext is a mockup MIDIContext if you don't want to create a real
// one.
type NullMIDIContext struct{}

func (m NullMIDIContext) Inputs(yield func(input MIDIInputDevice) bool) {}
func (m NullMIDIContext) Close()                                        {}
func (m NullMIDIContext) Support() MIDISupport                          { return MIDISupportNotCompiled }
