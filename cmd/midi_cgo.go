//go:build cgo

package cmd

import (
	"github.com/vsariola/sointu/tracker"
	"github.com/vsariola/sointu/tracker/gomidi"
)

func NewMidiContext(broker *tracker.Broker) tracker.MIDIContext {
	return gomidi.NewContext(broker)
}
