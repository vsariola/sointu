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
	Name           string
	SupportsStereo bool
	Parameters     []UnitParameter
}

// UnitTypes documents all the available unit types and if they support stereo variant
// and what parameters they take.
var UnitTypes = []UnitType{
	{
		Name:           "add",
		SupportsStereo: true,
		Parameters:     []UnitParameter{}},
	{
		Name:           "addp",
		SupportsStereo: true,
		Parameters:     []UnitParameter{}},
	{
		Name:           "pop",
		SupportsStereo: true,
		Parameters:     []UnitParameter{}},
	{
		Name:           "loadnote",
		SupportsStereo: true,
		Parameters:     []UnitParameter{}},
	{
		Name:           "mul",
		SupportsStereo: true,
		Parameters:     []UnitParameter{}},
	{
		Name:           "mulp",
		SupportsStereo: true,
		Parameters:     []UnitParameter{}},
	{
		Name:           "push",
		SupportsStereo: true,
		Parameters:     []UnitParameter{}},
	{
		Name:           "xch",
		SupportsStereo: true,
		Parameters:     []UnitParameter{}},
	{
		Name:           "distort",
		SupportsStereo: true,
		Parameters:     []UnitParameter{{Name: "drive", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}}},
	{
		Name:           "hold",
		SupportsStereo: true,
		Parameters:     []UnitParameter{{Name: "holdfreq", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}}},
	{
		Name:           "crush",
		SupportsStereo: true,
		Parameters:     []UnitParameter{{Name: "resolution", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}}},
	{
		Name:           "gain",
		SupportsStereo: true,
		Parameters:     []UnitParameter{{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}}},
	{
		Name:           "invgain",
		SupportsStereo: true,
		Parameters:     []UnitParameter{{Name: "invgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}}},
	{
		Name:           "filter",
		SupportsStereo: true,
		Parameters: []UnitParameter{
			{Name: "frequency", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "resonance", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "lowpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "bandpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "highpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "negbandpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "neghighpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}}},
	{
		Name:           "clip",
		SupportsStereo: true,
		Parameters:     []UnitParameter{}},
	{
		Name:           "pan",
		SupportsStereo: true,
		Parameters:     []UnitParameter{{Name: "panning", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}}},
	{
		Name:           "delay",
		SupportsStereo: true,
		Parameters: []UnitParameter{
			{Name: "pregain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "dry", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "feedback", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "damp", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "notetracking", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
			{Name: "delay", MinValue: 0, MaxValue: -1, CanSet: false, CanModulate: true},
		}},
	{
		Name:           "compressor",
		SupportsStereo: true,
		Parameters: []UnitParameter{
			{Name: "attack", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "release", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "invgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "threshold", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "ratio", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		}},
	{
		Name:           "speed",
		SupportsStereo: false,
		Parameters:     []UnitParameter{}},
	{
		Name:           "out",
		SupportsStereo: true,
		Parameters:     []UnitParameter{{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}}},
	{
		Name:           "outaux",
		SupportsStereo: true,
		Parameters: []UnitParameter{
			{Name: "outgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "auxgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		}},
	{
		Name:           "aux",
		SupportsStereo: true,
		Parameters: []UnitParameter{
			{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "channel", MinValue: 0, MaxValue: 6, CanSet: true, CanModulate: false},
		}},
	{
		Name:           "send",
		SupportsStereo: true,
		Parameters: []UnitParameter{
			{Name: "amount", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "voice", MinValue: 0, MaxValue: 32, CanSet: true, CanModulate: false},
			{Name: "unit", MinValue: 0, MaxValue: 63, CanSet: true, CanModulate: false},
			{Name: "port", MinValue: 0, MaxValue: 7, CanSet: true, CanModulate: false},
			{Name: "sendpop", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		}},
	{
		Name:           "envelope",
		SupportsStereo: true,
		Parameters: []UnitParameter{
			{Name: "attack", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "decay", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "sustain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "release", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		}},
	{
		Name:           "noise",
		SupportsStereo: true,
		Parameters: []UnitParameter{
			{Name: "shape", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
			{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		}},
	{
		Name:           "oscillator",
		SupportsStereo: true,
		Parameters: []UnitParameter{
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
		Name:           "loadval",
		SupportsStereo: true,
		Parameters: []UnitParameter{
			{Name: "value", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		}},
	{
		Name:           "receive",
		SupportsStereo: true,
		Parameters: []UnitParameter{
			{Name: "left", MinValue: 0, MaxValue: -1, CanSet: false, CanModulate: true},
			{Name: "right", MinValue: 0, MaxValue: -1, CanSet: false, CanModulate: true},
		}},
	{
		Name:           "in",
		SupportsStereo: true,
		Parameters: []UnitParameter{
			{Name: "channel", MinValue: 0, MaxValue: 6, CanSet: true, CanModulate: false},
		}},
}
