//go:build !cgo

package cmd

import (
	"github.com/vsariola/sointu/tracker"
)

func NewMidiContext(broker *tracker.Broker) tracker.MIDIContext {
	// with no cgo, we cannot use MIDI, so return a null context
	return tracker.NullMIDIContext{}
}
