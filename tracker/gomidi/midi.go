package gomidi

import (
	"fmt"
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
		events          chan midi.Message
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

// Open an input device while closing the currently open if necessary.
func (m *MIDIContext) OpenInputDevice(in tracker.MIDIDevicer) bool {
	fmt.Printf("Opening midi device %s\n.", in)
	if m.driverAvailable {
		if m.currentIn == in {
			return false
		}
		if m.inputAvailable && m.currentIn.IsOpen() {
			m.currentIn.Close()
		}
		m.currentIn = in.(MIDIDevicer)
		m.currentIn.Open()
		_, err := midi.ListenTo(m.currentIn, m.HandleMessage)
		if err != nil {
			m.inputAvailable = false
			return false
		}
	}
	return true
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
