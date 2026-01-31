package gomidi

// These cgo linker flags tell mingw to link gcc_s_seh-1, stdc++-6 and
// winpthread-1 statically; otherwise they are needed as DLLs

// #cgo windows LDFLAGS: -static -static-libgcc -static-libstdc++
import "C"

import (
	"fmt"

	"github.com/vsariola/sointu/tracker"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
)

type (
	RTMIDIContext struct {
		driver *rtmididrv.Driver
		broker *tracker.Broker
	}

	RTMIDIInputDevice struct {
		broker *tracker.Broker
		drivers.In
	}
)

// Open the driver.
func NewContext(broker *tracker.Broker) *RTMIDIContext {
	m := RTMIDIContext{broker: broker}
	// there's not much we can do if this fails, so just use m.driver = nil to
	// indicate no driver available
	m.driver, _ = rtmididrv.New()
	return &m
}

func (m *RTMIDIContext) Inputs(yield func(input tracker.MIDIInputDevice) bool) {
	if m.driver == nil {
		return
	}
	ins, err := m.driver.Ins()
	if err != nil {
		return
	}
	for _, in := range ins {
		r := RTMIDIInputDevice{In: in, broker: m.broker}
		if !yield(r) {
			break
		}
	}
}

func (c *RTMIDIContext) Close() {
	if c.driver == nil {
		return
	}
	c.driver.Close()
}

func (c *RTMIDIContext) Support() tracker.MIDISupport {
	if c.driver == nil {
		return tracker.MIDISupportNoDriver
	}
	return tracker.MIDISupported
}

// Open an input device and starting the listener.
func (m RTMIDIInputDevice) Open() error {
	if err := m.In.Open(); err != nil {
		return fmt.Errorf("opening MIDI input failed: %w", err)
	}
	if _, err := midi.ListenTo(m.In, m.handleMessage); err != nil {
		m.In.Close()
		return fmt.Errorf("listening to MIDI input failed: %w", err)
	}
	return nil
}

func (m *RTMIDIInputDevice) handleMessage(msg midi.Message, timestampms int32) {
	var channel, key, velocity, controller, value uint8
	if msg.GetNoteOn(&channel, &key, &velocity) {
		ev := tracker.NoteEvent{Timestamp: int64(timestampms) * 441 / 10, On: true, Channel: int(channel), Note: key, Source: m}
		tracker.TrySend(m.broker.ToMIDIRouter, any(&ev))
	} else if msg.GetNoteOff(&channel, &key, &velocity) {
		ev := tracker.NoteEvent{Timestamp: int64(timestampms) * 441 / 10, On: false, Channel: int(channel), Note: key, Source: m}
		tracker.TrySend(m.broker.ToMIDIRouter, any(&ev))
	} else if msg.GetControlChange(&channel, &controller, &value) {
		ev := tracker.ControlChange{Channel: int(channel), Control: int(controller), Value: int(value)}
		tracker.TrySend(m.broker.ToMIDIRouter, any(&ev))
	}
}
