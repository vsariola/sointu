package gomidi

import (
	"errors"
	"fmt"

	"github.com/vsariola/sointu/tracker"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
	"strings"
)

type (
	RTMIDIContext struct {
		driver    *rtmididrv.Driver
		currentIn drivers.In
		events    chan midi.Message
	}

	RTMIDIDevice struct {
		context *RTMIDIContext
		in      drivers.In
	}
)

func (c *RTMIDIContext) InputDevices(yield func(tracker.MIDIDevice) bool) {
	if c.driver == nil {
		return
	}
	ins, err := c.driver.Ins()
	if err != nil {
		return
	}
	for i := 0; i < len(ins); i++ {
		device := RTMIDIDevice{context: c, in: ins[i]}
		if !yield(device) {
			break
		}
	}
}

func NewContext() *RTMIDIContext {
	m := RTMIDIContext{events: make(chan midi.Message, 1024)}
	// there's not much we can do if this fails, so just use m.driver = nil to
	// indicate no driver available
	m.driver, _ = rtmididrv.New()
	return &m
}

// Open an input device while closing the currently open if necessary.
func (m RTMIDIDevice) Open() error {
	if m.context.currentIn == m.in {
		return nil
	}
	if m.context.driver == nil {
		return errors.New("no driver available")
	}
	if m.context.HasDeviceOpen() {
		m.context.currentIn.Close()
	}
	m.context.currentIn = m.in
	err := m.in.Open()
	if err != nil {
		m.context.currentIn = nil
		return fmt.Errorf("opening MIDI input failed: %W", err)
	}
	_, err = midi.ListenTo(m.in, m.context.HandleMessage)
	if err != nil {
		m.in.Close()
		m.context.currentIn = nil
	}
	return nil
}

func (d RTMIDIDevice) String() string {
	return d.in.String()
}

func (c *RTMIDIContext) HandleMessage(msg midi.Message, timestampms int32) {
	select {
	case c.events <- msg: // if the channel is full, just drop the message
	default:
	}
}

func (c *RTMIDIContext) NextEvent() (event tracker.MIDINoteEvent, ok bool) {
	for {
		select {
		case msg := <-c.events:
			{
				var channel uint8
				var velocity uint8
				var key uint8
				if msg.GetNoteOn(&channel, &key, &velocity) {
					return tracker.MIDINoteEvent{Frame: 0, On: true, Channel: int(channel), Note: key}, true
				} else if msg.GetNoteOff(&channel, &key, &velocity) {
					return tracker.MIDINoteEvent{Frame: 0, On: false, Channel: int(channel), Note: key}, true
				}
				// TODO: handle control messages with something like:
				// if msg.GetControlChange(&channel, &controller, &value) {
				//	....
				// if the message is not any recognized type, ignore it and continue looping
			}
		default:
			return tracker.MIDINoteEvent{}, false
		}
	}
}

func (c *RTMIDIContext) BPM() (bpm float64, ok bool) {
	return 0, false
}

func (c *RTMIDIContext) Close() {
	if c.driver == nil {
		return
	}
	if c.currentIn != nil && c.currentIn.IsOpen() {
		c.currentIn.Close()
	}
	c.driver.Close()
}

func (c *RTMIDIContext) HasDeviceOpen() bool {
	return c.currentIn != nil && c.currentIn.IsOpen()
}

func (c *RTMIDIContext) TryToOpenBy(namePrefix string, takeFirst bool) {
	if namePrefix == "" && !takeFirst {
		return
	}
	for input := range c.InputDevices {
		if takeFirst || strings.HasPrefix(input.String(), namePrefix) {
			input.Open()
			return
		}
	}
	if takeFirst {
		fmt.Printf("Could not find any MIDI Input.\n")
	} else {
		fmt.Printf("Could not find any default MIDI Input starting with \"%s\".\n", namePrefix)
	}
}
