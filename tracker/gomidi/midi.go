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
		driver    *rtmididrv.Driver
		currentIn drivers.In
		broker    *tracker.Broker
	}

	RTMIDIDevice struct {
		context *RTMIDIContext
		in      drivers.In
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
func NewContext(broker *tracker.Broker) *RTMIDIContext {
	m := RTMIDIContext{broker: broker}
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
	var channel, key, velocity uint8
	if msg.GetNoteOn(&channel, &key, &velocity) {
		ev := tracker.NoteEvent{Timestamp: int64(timestampms) * 441 / 10, On: true, Channel: int(channel), Note: key, Source: m}
		tracker.TrySend(m.broker.MIDIChannel(), any(ev))
	} else if msg.GetNoteOff(&channel, &key, &velocity) {
		ev := tracker.NoteEvent{Timestamp: int64(timestampms) * 441 / 10, On: false, Channel: int(channel), Note: key, Source: m}
		tracker.TrySend(m.broker.MIDIChannel(), any(ev))
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
		fmt.Errorf("Could not find any MIDI Input.\n")
	} else {
		fmt.Errorf("Could not find any default MIDI Input starting with \"%s\".\n", namePrefix)
	}
}
