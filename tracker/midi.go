package tracker

import (
	"fmt"
)

type MIDIModel Model

func (m *Model) MIDI() *MIDIModel { return (*MIDIModel)(m) }

type (
	midiState struct {
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

func (m *midiInputtingNotes) Value() bool       { return m.broker.mIDIEventsToGUI.Load() }
func (m *midiInputtingNotes) SetValue(val bool) { m.broker.mIDIEventsToGUI.Store(val) }

// NullMIDIContext is a mockup MIDIContext if you don't want to create a real
// one.
type NullMIDIContext struct{}

func (m NullMIDIContext) Inputs(yield func(input MIDIInputDevice) bool) {}
func (m NullMIDIContext) Close()                                        {}
func (m NullMIDIContext) Support() MIDISupport                          { return MIDISupportNotCompiled }
