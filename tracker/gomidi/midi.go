package gomidi

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vsariola/sointu/tracker"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
)

type (
	MIDIContext struct {
		driver          *rtmididrv.Driver
		inputAvailable  bool
		driverAvailable bool
		currentIn       MIDIDevicer

		events chan midi.Message
	}
	MIDIDevicer drivers.In
)

func (m *MIDIContext) ListInputDevices() <-chan tracker.MIDIDevicer {

	ins, err := m.driver.Ins()
	channel := make(chan tracker.MIDIDevicer, len(ins))
	if err != nil {
		m.driver.Close()
		m.driverAvailable = false
		return nil
	}
	go func() {
		for i := 0; i < len(ins); i++ {
			channel <- ins[i].(MIDIDevicer)
		}
		close(channel)
	}()
	return channel
}

// Open the driver.
func CreateContext() *MIDIContext {
	m := MIDIContext{}
	var err error
	m.driver, err = rtmididrv.New()
	m.driverAvailable = err == nil
	if m.driverAvailable {
		m.events = make(chan midi.Message)
	}
	return &m
}

func (m *MIDIContext) isCurrentInputOpen(in tracker.MIDIDevicer) bool {
	return (m.currentIn != nil &&
		m.currentIn.String() == in.String() &&
		m.currentIn.IsOpen())
}

// Open an input device while closing the currently open if necessary.
func (m *MIDIContext) OpenInputDevice(in tracker.MIDIDevicer) bool {
	if m.driverAvailable {
		if m.isCurrentInputOpen(in) {
			// "true" because the required input device is successfully open
			return true
		}

		if m.inputAvailable && m.currentIn.IsOpen() {
			m.currentIn.Close()
		}

		fmt.Printf("Opening midi device \"%s\".\n", in)

		m.currentIn = in.(MIDIDevicer)
		m.currentIn.Open()
		_, err := midi.ListenTo(m.currentIn, m.HandleMessage)

		if err != nil {
			m.inputAvailable = false
			return false
		}
	} else {
		fmt.Printf("Cannot Open Input Device %s: No MIDI driver available.\n", in)
	}
	return true
}

func (m *MIDIContext) OpenDefaultInputDevice(namePrefix string, takeFirst bool) bool {
	for input := range m.ListInputDevices() {
		if takeFirst || strings.HasPrefix(input.String(), namePrefix) {
			return m.OpenInputDevice(input)
		}
	}
	if takeFirst {
		fmt.Printf("Could not find any MIDI Input.\n")
	} else {
		fmt.Printf("Could not find any default MIDI Input starting with \"%s\".\n", namePrefix)
	}
	return false
}

func (m *MIDIContext) HandleMessage(msg midi.Message, timestampms int32) {
	go func() {
		m.events <- msg
		time.Sleep(time.Nanosecond)
	}()
}

func (c *MIDIContext) NextEvent() (event tracker.MIDINoteEvent, ok bool) {
	select {
	case msg := <-c.events:
		{
			var channel uint8
			var velocity uint8
			var key uint8
			var controller uint8
			var value uint8
			if msg.GetNoteOn(&channel, &key, &velocity) {
				return tracker.MIDINoteEvent{Frame: 0, On: true, Channel: int(channel), Note: key}, true
			} else if msg.GetNoteOff(&channel, &key, &velocity) {
				return tracker.MIDINoteEvent{Frame: 0, On: false, Channel: int(channel), Note: key}, true
			} else if msg.GetControlChange(&channel, &controller, &value) {
				fmt.Printf("CC @ Channel: %d, Controller: %d, Value: %d\n", channel, controller, value)
			} else {
				fmt.Printf("Unhandled MIDI message: %s\n", msg)
			}
		}
	default:
		// Note (@LeStahL): This empty select case is needed to make the implementation non-blocking.
	}
	return tracker.MIDINoteEvent{}, false
}

func (c *MIDIContext) BPM() (bpm float64, ok bool) {
	return 0, false
}

func (c *MIDIContext) DestroyContext() {
	close(c.events)
	c.currentIn.Close()
	c.driver.Close()
}
