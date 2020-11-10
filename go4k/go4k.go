package go4k

import (
	"errors"
	"math"
)

// Unit is e.g. a filter, oscillator, envelope and its parameters
type Unit struct {
	Type       string
	Parameters map[string]int
	DelayTimes []int
}

const (
	Sine   = iota
	Trisaw = iota
	Pulse  = iota
	Gate   = iota
	Sample = iota
)

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

func (patch Patch) InstrumentForVoice(voice int) (int, error) {
	if voice < 0 {
		return 0, errors.New("voice cannot be negative")
	}
	for i, instr := range patch {
		if voice < instr.NumVoices {
			return i, nil
		} else {
			voice -= instr.NumVoices
		}
	}
	return 0, errors.New("voice number is beyond the total voices of an instrument")
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

type SampleOffset struct {
	Start      int
	LoopStart  int
	LoopLength int
}

// UnitParameter documents one parameter that an unit takes
type UnitParameter struct {
	Name        string // thould be found with this name in the Unit.Parameters map
	MinValue    int    // minimum value of the parameter, inclusive
	MaxValue    int    // maximum value of the parameter, inclusive
	CanSet      bool   // if this parameter can be set before hand i.e. through the gui
	CanModulate bool   // if this parameter can be modulated i.e. has a port number in "send" unit
}

// UnitType documents the supported behaviour of one type of unit (oscillator, envelope etc.)
type UnitType struct {
	Name       string
	Parameters []UnitParameter
}

// UnitTypes documents all the available unit types and if they support stereo variant
// and what parameters they take.
var UnitTypes = []UnitType{
	{
		Name:       "add",
		Parameters: []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}}},
	{
		Name:       "addp",
		Parameters: []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}}},
	{
		Name:       "pop",
		Parameters: []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}}},
	{
		Name:       "loadnote",
		Parameters: []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}}},
	{
		Name:       "mul",
		Parameters: []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}}},
	{
		Name:       "mulp",
		Parameters: []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}}},
	{
		Name:       "push",
		Parameters: []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}}},
	{
		Name:       "xch",
		Parameters: []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}}},
	{
		Name: "distort",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "drive", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}}},
	{
		Name: "hold",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "holdfreq", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}}},
	{
		Name: "crush",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "resolution", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}}},
	{
		Name: "gain",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}}},
	{
		Name: "invgain",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "invgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}}},
	{
		Name: "filter",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "frequency", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "resonance", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "lowpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "bandpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "highpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "negbandpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "neghighpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}}},
	{
		Name:       "clip",
		Parameters: []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}}},
	{
		Name: "pan",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "panning", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}}},
	{
		Name: "delay",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "pregain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "dry", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "feedback", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "damp", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "notetracking", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "delay", MinValue: 0, MaxValue: -1, CanSet: false, CanModulate: true},
		}},
	{
		Name: "compressor",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "attack", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "release", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "invgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "threshold", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "ratio", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		}},
	{
		Name:       "speed",
		Parameters: []UnitParameter{}},
	{
		Name: "out",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}}},
	{
		Name: "outaux",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "outgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "auxgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		}},
	{
		Name: "aux",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "channel", MinValue: 0, MaxValue: 6, CanSet: true, CanModulate: false},
		}},
	{
		Name: "send",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "amount", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "voice", MinValue: 0, MaxValue: 32, CanSet: true, CanModulate: false},
			{Name: "unit", MinValue: 0, MaxValue: 63, CanSet: true, CanModulate: false},
			{Name: "port", MinValue: 0, MaxValue: 7, CanSet: true, CanModulate: false},
			{Name: "sendpop", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		}},
	{
		Name: "envelope",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "attack", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "decay", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "sustain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "release", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		}},
	{
		Name: "noise",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "shape", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		}},
	{
		Name: "oscillator",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "transpose", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "detune", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "phase", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "color", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "shape", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "type", MinValue: int(Sine), MaxValue: int(Sample), CanSet: true, CanModulate: false},
			{Name: "lfo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "unison", MinValue: 0, MaxValue: 3, CanSet: true, CanModulate: false},
			{Name: "start", MinValue: 0, MaxValue: 3440659, CanSet: true, CanModulate: false},    // if type is "sample", then the waveform starts at this position
			{Name: "loopstart", MinValue: 0, MaxValue: 65535, CanSet: true, CanModulate: false},  // if type is "sample", then the loop starts at this position, relative to "start"
			{Name: "looplength", MinValue: 0, MaxValue: 65535, CanSet: true, CanModulate: false}, // if type is "sample", then the loop length is this i.e. loop ends at "start" + "loopstart" + "looplength"
		}},
	{
		Name: "loadval",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "value", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		}},
	{
		Name: "receive",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "left", MinValue: 0, MaxValue: -1, CanSet: false, CanModulate: true},
			{Name: "right", MinValue: 0, MaxValue: -1, CanSet: false, CanModulate: true},
		}},
	{
		Name: "in",
		Parameters: []UnitParameter{
			{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "channel", MinValue: 0, MaxValue: 6, CanSet: true, CanModulate: false},
		}},
}
