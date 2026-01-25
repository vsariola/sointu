package tracker

import (
	"fmt"
	"strings"
)

type (
	MIDIModel   Model
	MIDIContext interface {
		InputDevices(yield func(deviceName string) bool)
		Open(deviceName string) error
		Close()
		IsOpen() bool
	}
)

func (m *Model) MIDI() *MIDIModel { return (*MIDIModel)(m) }

// InputDevices can be iterated to get string names of all the MIDI input
// devices.
func (m *MIDIModel) InputDevices(yield func(deviceName string) bool) { m.midi.InputDevices(yield) }

// IsOpen returns true if a midi device is currently open.
func (m *MIDIModel) IsOpen() bool { return m.midi.IsOpen() }

// InputtingNotes returns a Bool controlling whether the MIDI events are used
// just to trigger instruments, or if the note events are used to input notes to
// the note table.
func (m *MIDIModel) InputtingNotes() Bool { return MakeBool((*midiInputtingNotes)(m)) }

type midiInputtingNotes Model

func (m *midiInputtingNotes) Value() bool       { return m.broker.mIDIEventsToGUI.Load() }
func (m *midiInputtingNotes) SetValue(val bool) { m.broker.mIDIEventsToGUI.Store(val) }

// Open returns an Action to open the MIDI input device with a given name.
func (m *MIDIModel) Open(deviceName string) Action {
	return MakeAction(openMIDI{Item: deviceName, Model: (*Model)(m)})
}

type openMIDI struct {
	Item string
	*Model
}

func (s openMIDI) Do() {
	m := s.Model
	if err := s.Model.midi.Open(s.Item); err == nil {
		message := fmt.Sprintf("Opened MIDI device: %s", s.Item)
		m.Alerts().Add(message, Info)
	} else {
		message := fmt.Sprintf("Could not open MIDI device: %s", s.Item)
		m.Alerts().Add(message, Error)
	}
}

// FindMIDIDeviceByPrefix finds the MIDI input device whose name starts with the given
// prefix. It returns the full device name and true if found, or an empty string
// and false if not found.
func FindMIDIDeviceByPrefix(c MIDIContext, prefix string) (deviceName string, ok bool) {
	for input := range c.InputDevices {
		if strings.HasPrefix(input, prefix) {
			return input, true
		}
	}
	return "", false
}

// NullMIDIContext is a mockup MIDIContext if you don't want to create a real
// one.
type NullMIDIContext struct{}

func (m NullMIDIContext) InputDevices(yield func(string) bool) {}
func (m NullMIDIContext) Open(deviceName string) error         { return nil }
func (m NullMIDIContext) Close()                               {}
func (m NullMIDIContext) IsOpen() bool                         { return false }
