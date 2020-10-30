package go4k

import (
	"errors"
	"math"
)

// Unit is e.g. a filter, oscillator, envelope and its parameters
type Unit struct {
	Type       string
	Stereo     bool
	Parameters map[string]int
}

// Instrument includes a list of units consisting of the instrument, and the number of polyphonic voices for this instrument
type Instrument struct {
	NumVoices int
	Units     []Unit
}

// Patch is simply a list of instruments used in a song
type Patch []Instrument

func (p Patch) TotalVoices() int {
	ret := 0
	for _, i := range p {
		ret += i.NumVoices
	}
	return ret
}

type Track struct {
	NumVoices int
	Sequence  []byte
}

type Synth interface {
	Render(buffer []float32, maxtime int) (int, int, error)
	Trigger(voice int, note byte)
	Release(voice int)
}

func Render(synth Synth, buffer []float32) error {
	s, _, err := synth.Render(buffer, math.MaxInt32)
	if s != len(buffer)/2 {
		return errors.New("synth.Render should have filled the whole buffer")
	}
	return err
}
