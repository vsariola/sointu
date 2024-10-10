package tracker

import (
	"fmt"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
)

type MIDIContext struct {
	driver          *rtmididrv.Driver
	inputAvailable  bool
	driverAvailable bool
	currentIn       drivers.In

	events     []midi.Message
	eventIndex int
}

func (m *MIDIContext) ListInputDevices() []drivers.In {
	if m.driverAvailable {
		midiIns, err := m.driver.Ins()
		if err != nil {
			m.driver.Close()
			m.inputAvailable = false
			return nil
		}
		return midiIns
	}
	return nil
}

// Open the driver.
func (m *MIDIContext) CreateContext() {
	var err error
	m.driver, err = rtmididrv.New()
	m.driverAvailable = err == nil
}

// Open an input device while closing the currently open if necessary.
func (m *MIDIContext) OpenInputDevice(in drivers.In) bool {
	if m.driverAvailable {
		if m.currentIn == in {
			return false
		}

		if m.currentIn != nil && m.currentIn.IsOpen() {
			m.currentIn.Close()
		}

		m.currentIn = in
		m.currentIn.Open()
		_, err := midi.ListenTo(m.currentIn, m.HandleMessage)

		if err != nil {
			m.currentIn = nil
			return false
		}
	}

	return true
}

func (m *MIDIContext) HandleMessage(msg midi.Message, timestampms int32) {
	m.events = append(m.events, msg)
}

func (c *MIDIContext) NextEvent() (event MIDINoteEvent, ok bool) {
	for c.eventIndex < len(c.events) {
		msg := c.events[c.eventIndex]
		c.eventIndex += 1

		var channel uint8
		var velocity uint8
		var key uint8
		var controller uint8
		var value uint8
		if msg.GetNoteOn(&channel, &key, &velocity) {
			c.events = c.events[c.eventIndex:]
			c.eventIndex = 0
			return MIDINoteEvent{Frame: 0, On: true, Channel: int(channel), Note: key}, true
		} else if msg.GetNoteOff(&channel, &key, &velocity) {
			c.events = c.events[c.eventIndex:]
			c.eventIndex = 0
			return MIDINoteEvent{Frame: 0, On: false, Channel: int(channel), Note: key}, true
		} else if msg.GetControlChange(&channel, &controller, &value) {
			fmt.Printf("CC @ Channel: %d, Controller: %d, Value: %d\n", channel, controller, value)
		} else {
			fmt.Printf("Unhandled MIDI message: %s\n", msg)
		}
		_ = channel
		_ = velocity
		_ = key
		_ = controller
		_ = value
	}

	return MIDINoteEvent{}, false
}

func (c *MIDIContext) BPM() (bpm float64, ok bool) {
	return 0, false
}
