package gomidi

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vsariola/sointu/tracker"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
)

type (
	RTMIDIContext struct {
		driver             *rtmididrv.Driver
		currentIn          drivers.In
		inputDevices       []RTMIDIDevice
		devicesInitialized bool
		events             chan timestampedMsg
		eventsBuf          []timestampedMsg
		eventIndex         int
		startFrame         int
		startFrameSet      bool

		// qm210: this is my current solution for passing model information to the player
		// I do not completely love this, but improve at your own peril.
		currentConstraints tracker.PlayerProcessConstraints
	}

	RTMIDIDevice struct {
		context *RTMIDIContext
		in      drivers.In
	}

	timestampedMsg struct {
		frame int
		msg   midi.Message
	}
)

func (m *RTMIDIContext) InputDevices(yield func(tracker.MIDIDevice) bool) {
	if m.devicesInitialized {
		m.yieldCachedInputDevices(yield)
	} else {
		m.initInputDevices(yield)
	}
}

func (m *RTMIDIContext) yieldCachedInputDevices(yield func(tracker.MIDIDevice) bool) {
	for _, device := range m.inputDevices {
		if !yield(device) {
			break
		}
	}
}

func (m *RTMIDIContext) initInputDevices(yield func(tracker.MIDIDevice) bool) {
	if m.driver == nil {
		return
	}
	ins, err := m.driver.Ins()
	if err != nil {
		return
	}
	for i := 0; i < len(ins); i++ {
		device := RTMIDIDevice{context: m, in: ins[i]}
		m.inputDevices = append(m.inputDevices, device)
		if !yield(device) {
			break
		}
	}
	m.devicesInitialized = true
}

// Open the driver.
func NewContext() *RTMIDIContext {
	m := RTMIDIContext{events: make(chan timestampedMsg, 1024)}
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
		fmt.Errorf("Could not find any MIDI Input.\n")
	} else {
		fmt.Errorf("Could not find any default MIDI Input starting with \"%s\".\n", namePrefix)
	}
}

func (m *RTMIDIContext) HandleMessage(msg midi.Message, timestampms int32) {
	select {
	case m.events <- timestampedMsg{frame: int(int64(timestampms) * 44100 / 1000), msg: msg}: // if the channel is full, just drop the message
	default:
	}
}

func (c *RTMIDIContext) NextEvent(frame int) (event tracker.MIDINoteEvent, ok bool) {
F:
	for {
		select {
		case msg := <-c.events:
			c.eventsBuf = append(c.eventsBuf, msg)
			if !c.startFrameSet {
				c.startFrame = msg.frame
				c.startFrameSet = true
			}
		default:
			break F
		}
	}
	if c.eventIndex > 0 { // an event was consumed, check how badly we need to adjust the timing
		delta := frame + c.startFrame - c.eventsBuf[c.eventIndex-1].frame
		// delta should never be a negative number, because the renderer does
		// not consume an event until current frame is past the frame of the
		// event. However, if it's been a while since we consumed event, delta
		// may by *positive* i.e. we consume the event too late. So adjust the
		// internal clock in that case.
		c.startFrame -= delta / 5 // adjust the start frame towards the consumed event
	}
	for c.eventIndex < len(c.eventsBuf) {
		var channel uint8
		var velocity uint8
		var key uint8
		m := c.eventsBuf[c.eventIndex]
		f := m.frame - c.startFrame
		c.eventIndex++
		isNoteOn := m.msg.GetNoteOn(&channel, &key, &velocity)
		isNoteOff := !isNoteOn && m.msg.GetNoteOff(&channel, &key, &velocity)
		if isNoteOn || isNoteOff {
			return tracker.MIDINoteEvent{
				Frame:    f,
				On:       isNoteOn,
				Channel:  int(channel),
				Note:     key,
				Velocity: velocity,
			}, true
		}
	}
	c.eventIndex = len(c.eventsBuf) + 1
	return tracker.MIDINoteEvent{}, false
}

func (c *RTMIDIContext) FinishBlock(frame int) {
	c.startFrame += frame
	if c.eventIndex > 0 {
		copy(c.eventsBuf, c.eventsBuf[c.eventIndex-1:])
		c.eventsBuf = c.eventsBuf[:len(c.eventsBuf)-c.eventIndex+1]
		if len(c.eventsBuf) > 0 {
			// Events were not consumed this round; adjust the start frame
			// towards the future events. What this does is that it tries to
			// render the events at the same time as they were received here
			// delta will be always a negative number
			delta := c.startFrame - c.eventsBuf[0].frame
			c.startFrame -= delta / 5
		}
	}
	c.eventIndex = 0
}

func (c *RTMIDIContext) BPM() (bpm float64, ok bool) {
	return 0, false
}

func (c *RTMIDIContext) Constraints() tracker.PlayerProcessConstraints {
	return c.currentConstraints
}

func (c *RTMIDIContext) SetPlayerConstraints(constraints tracker.PlayerProcessConstraints) {
	c.currentConstraints = constraints
}
