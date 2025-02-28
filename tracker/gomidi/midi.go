package gomidi

// These cgo linker flags tell mingw to link gcc_s_seh-1, stdc++-6 and
// winpthread-1 statically; otherwise they are needed as DLLs

// #cgo windows LDFLAGS: -static -static-libgcc -static-libstdc++
import "C"

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
		driver        *rtmididrv.Driver
		currentIn     drivers.In
		events        chan timestampedMsg
		eventsBuf     []timestampedMsg
		eventIndex    int
		startFrame    int
		startFrameSet bool
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
	if m.driver == nil {
		return
	}
	ins, err := m.driver.Ins()
	if err != nil {
		return
	}
	for i := 0; i < len(ins); i++ {
		device := RTMIDIDevice{context: m, in: ins[i]}
		if !yield(device) {
			break
		}
	}
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
		if m.msg.GetNoteOn(&channel, &key, &velocity) {
			return tracker.MIDINoteEvent{Frame: f, On: true, Channel: int(channel), Note: key}, true
		} else if m.msg.GetNoteOff(&channel, &key, &velocity) {
			return tracker.MIDINoteEvent{Frame: f, On: false, Channel: int(channel), Note: key}, true
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
