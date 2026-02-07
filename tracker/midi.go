package tracker

import (
	"encoding/json"
	"fmt"

	"github.com/vsariola/sointu"
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

// Transpose returns an Int controlling the MIDI transpose value of the
// currently selected instrument.
func (m *MIDIModel) Transpose() Int { return MakeInt((*midiTranspose)(m)) }

type midiTranspose MIDIModel

func (m *midiTranspose) Value() int {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.d.Song.Patch) {
		return 0
	}
	return m.d.Song.Patch[i].MIDI.Transpose
}
func (m *midiTranspose) SetValue(val int) bool {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.d.Song.Patch) {
		return false
	}
	defer (*Model)(m).change("MIDITranspose", PatchChange, MinorChange)()
	m.d.Song.Patch[i].MIDI.Transpose = val
	return true
}
func (m *midiTranspose) Range() RangeInclusive { return RangeInclusive{-127, 127} }

// NoteStart returns an Int controlling the MIDI note start value of the
// currently selected instrument.
func (m *MIDIModel) NoteStart() Int { return MakeInt((*midiNoteStart)(m)) }

type midiNoteStart MIDIModel

func (m *midiNoteStart) Value() int {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.d.Song.Patch) {
		return 0
	}
	return m.d.Song.Patch[i].MIDI.Start
}
func (m *midiNoteStart) SetValue(val int) bool {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.d.Song.Patch) {
		return false
	}
	defer (*Model)(m).change("MIDINoteStart", PatchChange, MinorChange)()
	m.d.Song.Patch[i].MIDI.Start = val
	return true
}
func (m *midiNoteStart) Range() RangeInclusive { return RangeInclusive{0, 127} }

// NoteEnd returns an Int controlling the MIDI note end value of the
// currently selected instrument.
func (m *MIDIModel) NoteEnd() Int { return MakeInt((*midiNoteEnd)(m)) }

type midiNoteEnd MIDIModel

func (m *midiNoteEnd) Value() int {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.d.Song.Patch) {
		return 0
	}
	return 127 - m.d.Song.Patch[i].MIDI.End
}
func (m *midiNoteEnd) SetValue(val int) bool {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.d.Song.Patch) {
		return false
	}
	defer (*Model)(m).change("MIDINoteEnd", PatchChange, MinorChange)()
	m.d.Song.Patch[i].MIDI.End = 127 - val
	return true
}
func (m *midiNoteEnd) Range() RangeInclusive { return RangeInclusive{0, 127} }

// Velocity returns a Bool controlling whether the velocity value from MIDI
// event is used instead of the normal note value
func (m *MIDIModel) Velocity() Bool { return MakeBool((*midiVelocity)(m)) }

type midiVelocity MIDIModel

func (m *midiVelocity) Value() bool {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.d.Song.Patch) {
		return false
	}
	return m.d.Song.Patch[i].MIDI.Velocity
}
func (m *midiVelocity) SetValue(val bool) {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.d.Song.Patch) {
		return
	}
	defer (*Model)(m).change("MIDIVelocity", PatchChange, MinorChange)()
	m.d.Song.Patch[i].MIDI.Velocity = val
}

// Change returns a Bool controlling whether only the change in note or velocity value is used
func (m *MIDIModel) Change() Bool { return MakeBool((*midiChange)(m)) }

type midiChange MIDIModel

func (m *midiChange) Value() bool {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.d.Song.Patch) {
		return false
	}
	return m.d.Song.Patch[i].MIDI.NoRetrigger
}
func (m *midiChange) SetValue(val bool) {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.d.Song.Patch) {
		return
	}
	defer (*Model)(m).change("MIDIChange", PatchChange, MinorChange)()
	m.d.Song.Patch[i].MIDI.NoRetrigger = val
}

// IgnoreNoteOff returns a Bool controlling whether note off events are ignored
func (m *MIDIModel) IgnoreNoteOff() Bool { return MakeBool((*midiIgnoreNoteOff)(m)) }

type midiIgnoreNoteOff MIDIModel

func (m *midiIgnoreNoteOff) Value() bool {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.d.Song.Patch) {
		return false
	}
	return m.d.Song.Patch[i].MIDI.IgnoreNoteOff
}
func (m *midiIgnoreNoteOff) SetValue(val bool) {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.d.Song.Patch) {
		return
	}
	defer (*Model)(m).change("MIDIIgnoreNoteOff", PatchChange, MinorChange)()
	m.d.Song.Patch[i].MIDI.IgnoreNoteOff = val
}

// Channel returns an Int controlling the MIDI channel of the currently selected
// instrument. 0 = automatically selected, 1-16 fixed to specific MIDI channel
func (m *MIDIModel) Channel() Int { return MakeInt((*midiChannel)(m)) }

type midiChannel MIDIModel

func (m *midiChannel) Value() int {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.d.Song.Patch) {
		return 0
	}
	return m.d.Song.Patch[i].MIDI.Channel
}
func (m *midiChannel) SetValue(val int) bool {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.d.Song.Patch) {
		return false
	}
	defer (*Model)(m).change("MIDIChannel", PatchChange, MinorChange)()
	m.d.Song.Patch[i].MIDI.Channel = val
	return true
}
func (m *midiChannel) Range() RangeInclusive { return RangeInclusive{0, 16} }

type (
	midiAssigns struct {
		ctoi map[midiAssignKey][]midiAssignRange // map to quickly find which instruments to trigger
		itoc []int                               // slice to quickly find which MIDI channel was assigned for a given instrument
	}
	midiAssignKey struct {
		Channel  int
		Velocity bool
	}
	midiAssignRange struct {
		Start, End byte
		Instr      int
	}
)

const MAX_MIDI_CHANNELS = 16

// update tries to assign MIDI channels to instruments that have MIDI channel 0
// (automatic) in a way that minimizes the number of channels used. It also
// updates the ctoi and itoc maps for quick lookup when routing MIDI messages.
// The algorithm iterates through the instruments and, for those with automatic
// channel, it tries to find the lowest MIDI channel that doesn't have an
// overlapping note range with any of the already assigned instruments with the
// same velocity setting. If it runs out of channels, it leaves the rest of the
// instruments unassigned (channel 0).
func (a *midiAssigns) update(p sointu.Patch) {
	for k := range a.ctoi {
		a.ctoi[k] = a.ctoi[k][:0] // clear all slices, keeping their allocated memory
	}
	a.itoc = a.itoc[:0]
	for i, instr := range p {
		if instr.MIDI.Channel != 0 {
			k := midiAssignKey{Channel: instr.MIDI.Channel, Velocity: instr.MIDI.Velocity}
			v := midiAssignRange{Start: byte(max(instr.MIDI.Start, 0)), End: byte(min(127-instr.MIDI.End, 127)), Instr: i}
			a.ctoi[k] = append(a.ctoi[k], v)
		}
	}
	k := midiAssignKey{Channel: 1, Velocity: false}
outer:
	for i, e := range p {
		if e.MIDI.Channel > 0 { // already assigned to a specific channel, so skip automatic assignment
			a.itoc = append(a.itoc, e.MIDI.Channel)
			continue
		}
		k.Velocity = e.MIDI.Velocity
		x := midiAssignRange{Start: byte(max(e.MIDI.Start, 0)), End: byte(min(127-e.MIDI.End, 127)), Instr: i}
	inner:
		for {
			for _, y := range a.ctoi[k] {
				if max(x.Start, y.Start) <= min(x.End, y.End) {
					if k.Channel >= MAX_MIDI_CHANNELS {
						break outer // we've ran out of channels, leave the rest unassigned
					}
					k.Channel++
					continue inner // this channel is already taken for the overlapping range, try next channel
				}
			}
			break // this channel had no overlaps with the already assigned ranges, so we can use it
		}
		a.ctoi[k] = append(a.ctoi[k], x)
		a.itoc = append(a.itoc, k.Channel)
	}
}

func (a *midiAssigns) forEach(chn int, vel bool, val byte, cb func(instr int, val byte)) {
	k := midiAssignKey{Channel: chn, Velocity: vel}
	for _, r := range a.ctoi[k] {
		if r.Start <= val && val <= r.End {
			cb(r.Instr, val)
		}
	}
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
