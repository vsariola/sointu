package tracker

import (
	"encoding/json"
	"fmt"
)

type MIDIModel Model

func (m *Model) MIDI() *MIDIModel { return (*MIDIModel)(m) }

type (
	midiState struct {
		binding bool
		router  midiRouter

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
		Open(func(msg *MIDIMessage)) error
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
			handler := func(msg *MIDIMessage) {
				TrySend(m.broker.ToMIDIHandler, any(msg))
			}
			if err := i.Open(handler); err != nil {
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
	handler := func(msg *MIDIMessage) {
		TrySend(m.broker.ToMIDIHandler, any(msg))
	}
	if err := newInput.Open(handler); err != nil {
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

func (m *midiInputtingNotes) Value() bool { return m.midi.router.sendNoteEventsToGUI }
func (m *midiInputtingNotes) SetValue(val bool) {
	m.midi.router.sendNoteEventsToGUI = val
	TrySend(m.broker.ToMIDIHandler, any(m.midi.router))
	TrySend(m.broker.ToPlayer, any(m.midi.router))
}

// MIDIMessage represents a MIDI message received from a MIDI input port or VST
// host.
type MIDIMessage struct {
	Timestamp int64 // in samples (at 44100 Hz)
	Data      [3]byte
	Source    any // tag to identify the source of the message; any unique pointer will do
}

func (m *MIDIMessage) isNoteOff() bool       { return m.Data[0]&0xF0 == 0x80 }
func (m *MIDIMessage) isNoteOn() bool        { return m.Data[0]&0xF0 == 0x90 }
func (m *MIDIMessage) isControlChange() bool { return m.Data[0]&0xF0 == 0xB0 }

func (m *MIDIMessage) getNoteOn() (channel, note, velocity byte, ok bool) {
	if !m.isNoteOn() {
		return 0, 0, 0, false
	}
	return m.Data[0] & 0x0F, m.Data[1], m.Data[2], true
}

func (m *MIDIMessage) getNoteOff() (channel, note, velocity byte, ok bool) {
	if !m.isNoteOff() {
		return 0, 0, 0, false
	}
	return m.Data[0] & 0x0F, m.Data[1], m.Data[2], true
}

func (m *MIDIMessage) getControlChange() (channel, controller, value byte, ok bool) {
	if !m.isControlChange() {
		return 0, 0, 0, false
	}
	return m.Data[0] & 0x0F, m.Data[1], m.Data[2], true
}

// midiRouter encompasses all the necessary information where MIDIMessages
// should be forwarded. MIDIHandler and Player have their own copies of the
// midiRouter so that the messages don't have to pass through other goroutines
// to be routed. Model has also a copy to display a gui to modify it.
type midiRouter struct {
	sendNoteEventsToGUI bool
}

func (r *midiRouter) route(b *Broker, msg *MIDIMessage) (ok bool) {
	switch {
	case msg.isNoteOn() || msg.isNoteOff():
		if r.sendNoteEventsToGUI {
			return TrySend(b.ToGUI, any(msg))
		} else {
			return TrySend(b.ToPlayer, any(msg))
		}
	case msg.isControlChange():
		return TrySend(b.ToModel, MsgToModel{Data: msg})
	}
	return false
}

func runMIDIHandler(b *Broker) {
	router := midiRouter{sendNoteEventsToGUI: false}
	for {
		select {
		case v := <-b.ToMIDIHandler:
			switch msg := v.(type) {
			case *MIDIMessage:
				router.route(b, msg)
			case midiRouter:
				router = msg
			}
		case <-b.CloseMIDIHandler:
			close(b.FinishedMIDIHandler)
			return
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

func (m *MIDIModel) handleControlEvent(channel, control, value int) {
	key := MIDIControl{Channel: channel, Control: control}
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
	newVal := (value*(t.Max-t.Min)+62)/127 + t.Min
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
