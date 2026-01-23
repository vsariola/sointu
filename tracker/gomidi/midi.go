package gomidi

// These cgo linker flags tell mingw to link gcc_s_seh-1, stdc++-6 and
// winpthread-1 statically; otherwise they are needed as DLLs

// #cgo windows LDFLAGS: -static -static-libgcc -static-libstdc++
import "C"

import (
	"errors"
	"fmt"

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
)

func (m *RTMIDIContext) InputDevices(yield func(string) bool) {
	if m.driver == nil {
		return
	}
	ins, err := m.driver.Ins()
	if err != nil {
		return
	}
	for _, in := range ins {
		if !yield(in.String()) {
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
func (m *RTMIDIContext) Open(name string) error {
	if m.currentIn != nil && m.currentIn.String() == name {
		return nil
	}
	if m.driver == nil {
		return errors.New("no driver available")
	}
	if m.IsOpen() {
		m.currentIn.Close()
	}
	m.currentIn = nil
	ins, err := m.driver.Ins()
	if err != nil {
		return fmt.Errorf("retrieving MIDI inputs failed: %w", err)
	}
	for _, in := range ins {
		if in.String() == name {
			m.currentIn = in
		}
	}
	if m.currentIn == nil {
		return fmt.Errorf("MIDI input device not found: %s", name)
	}
	err = m.currentIn.Open()
	if err != nil {
		m.currentIn = nil
		return fmt.Errorf("opening MIDI input failed: %w", err)
	}
	_, err = midi.ListenTo(m.currentIn, m.HandleMessage)
	if err != nil {
		m.currentIn.Close()
		m.currentIn = nil
		return fmt.Errorf("listening to MIDI input failed: %w", err)
	}
	return nil
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

func (c *RTMIDIContext) IsOpen() bool {
	return c.currentIn != nil && c.currentIn.IsOpen()
}
