package sointu

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
)

type (
	// Patch is simply a list of instruments used in a song
	Patch []Instrument

	// Instrument includes a list of units consisting of the instrument, and the number of polyphonic voices for this instrument
	Instrument struct {
		Name      string `yaml:",omitempty"`
		Comment   string `yaml:",omitempty"`
		NumVoices int
		Units     []Unit
	}

	// Unit is e.g. a filter, oscillator, envelope and its parameters
	Unit struct {
		// Type is the type of the unit, e.g. "add","oscillator" or "envelope".
		// Always in lowercase. "" type should be ignored, no invalid types should
		// be used.
		Type string `yaml:",omitempty"`

		// ID should be a unique ID for this unit, used by SEND units to target
		// specific units. ID = 0 means that no ID has been given to a unit and thus
		// cannot be targeted by SENDs. When possible, units that are not targeted
		// by any SENDs should be cleaned from having IDs, e.g. to keep the exported
		// data clean.
		ID int `yaml:",omitempty"`

		// Parameters is a map[string]int of parameters of a unit. For example, for
		// an oscillator, unit.Type == "oscillator" and unit.Parameters["attack"]
		// could be 64. Most parameters are either limites to 0 and 1 (e.g. stereo
		// parameters) or between 0 and 128, inclusive.
		Parameters map[string]int `yaml:",flow"`

		// VarArgs is a list containing the variable number arguments that some
		// units require, most notably the DELAY units. For example, for a DELAY
		// unit, VarArgs is the delaytimes, in samples, of the different delaylines
		// in the unit.
		VarArgs []int `yaml:",flow,omitempty"`

		// Disabled is a flag that can be set to true to disable the unit.
		// Disabled units are considered to be not present in the patch.
		Disabled bool `yaml:",omitempty"`

		// Comment is a free-form comment about the unit that can be displayed
		// instead of/besides the type of the unit in the GUI, to make it easier
		// to track what the unit is doing & to make it easier to target sends.
		Comment string `yaml:",omitempty"`
	}

	// UnitParameter documents one parameter that an unit takes
	UnitParameter struct {
		Name        string // thould be found with this name in the Unit.Parameters map
		MinValue    int    // minimum value of the parameter, inclusive
		MaxValue    int    // maximum value of the parameter, inclusive
		CanSet      bool   // if this parameter can be set before hand i.e. through the gui
		CanModulate bool   // if this parameter can be modulated i.e. has a port number in "send" unit
		DisplayFunc UnitParameterDisplayFunc
	}

	UnitParameterDisplayFunc func(int) (value string, unit string)
)

// UnitTypes documents all the available unit types and if they support stereo variant
// and what parameters they take.
var UnitTypes = map[string]([]UnitParameter){
	"add":      []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}},
	"addp":     []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}},
	"pop":      []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}},
	"loadnote": []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}},
	"mul":      []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}},
	"mulp":     []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}},
	"push":     []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}},
	"xch":      []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}},
	"distort": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "drive", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"hold": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "holdfreq", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"crush": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "resolution", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return formatFloat(24 * float64(v) / 128), "bits" }}},
	"gain": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"invgain": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "invgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"dbgain": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "decibels", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return formatFloat(40 * (float64(v)/64 - 1)), "dB" }}},
	"filter": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "frequency", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: filterFrequencyDispFunc},
		{Name: "resonance", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "lowpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "bandpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "highpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "negbandpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "neghighpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}},
	"clip": []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}},
	"pan": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "panning", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"delay": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "pregain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "dry", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "feedback", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "damp", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "notetracking", MinValue: 0, MaxValue: 2, CanSet: true, CanModulate: false, DisplayFunc: arrDispFunc(noteTrackingNames[:])},
		{Name: "delaytime", MinValue: 0, MaxValue: -1, CanSet: false, CanModulate: true}},
	"compressor": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "attack", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: compressorTimeDispFunc},
		{Name: "release", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: compressorTimeDispFunc},
		{Name: "invgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) {
			return strconv.FormatFloat(20*math.Log10(128/float64(v)), 'f', 2, 64), "dB"
		}},
		{Name: "threshold", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) {
			return strconv.FormatFloat(20*math.Log10(float64(v)/128), 'f', 2, 64), "dB"
		}},
		{Name: "ratio", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return formatFloat(1 - float64(v)/128), "" }}},
	"speed": []UnitParameter{},
	"out": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"outaux": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "outgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "auxgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"aux": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "channel", MinValue: 0, MaxValue: 6, CanSet: true, CanModulate: false, DisplayFunc: arrDispFunc(channelNames[:])}},
	"send": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "amount", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return formatFloat(float64(v)/64 - 1), "" }},
		{Name: "voice", MinValue: 0, MaxValue: 32, CanSet: true, CanModulate: false},
		{Name: "target", MinValue: 0, MaxValue: math.MaxInt32, CanSet: true, CanModulate: false},
		{Name: "port", MinValue: 0, MaxValue: 7, CanSet: true, CanModulate: false},
		{Name: "sendpop", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}},
	"envelope": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "attack", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return engineeringTime(math.Pow(2, 24*float64(v)/128) / 44100) }},
		{Name: "decay", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return engineeringTime(math.Pow(2, 24*float64(v)/128) / 44100) }},
		{Name: "sustain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "release", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return engineeringTime(math.Pow(2, 24*float64(v)/128) / 44100) }},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"noise": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "shape", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"oscillator": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "transpose", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: oscillatorTransposeDispFunc},
		{Name: "detune", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return formatFloat(float64(v-64) / 64), "st" }},
		{Name: "phase", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "color", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "shape", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "frequency", MinValue: 0, MaxValue: -1, CanSet: false, CanModulate: true},
		{Name: "type", MinValue: int(Sine), MaxValue: int(Sample), CanSet: true, CanModulate: false, DisplayFunc: arrDispFunc(oscTypes[:])},
		{Name: "lfo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "unison", MinValue: 0, MaxValue: 3, CanSet: true, CanModulate: false},
		{Name: "samplestart", MinValue: 0, MaxValue: 1720329, CanSet: true, CanModulate: false},
		{Name: "loopstart", MinValue: 0, MaxValue: 65535, CanSet: true, CanModulate: false},
		{Name: "looplength", MinValue: 0, MaxValue: 65535, CanSet: true, CanModulate: false}},
	"loadval": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "value", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return formatFloat(float64(v)/64 - 1), "" }}},
	"receive": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "left", MinValue: 0, MaxValue: -1, CanSet: false, CanModulate: true},
		{Name: "right", MinValue: 0, MaxValue: -1, CanSet: false, CanModulate: true}},
	"in": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "channel", MinValue: 0, MaxValue: 6, CanSet: true, CanModulate: false, DisplayFunc: arrDispFunc(channelNames[:])}},
	"sync": []UnitParameter{},
}

var channelNames = [...]string{"left", "right", "aux1 left", "aux1 right", "aux2 left", "aux2 right", "aux3 left", "aux3 right"}
var noteTrackingNames = [...]string{"fixed", "pitch", "BPM"}
var oscTypes = [...]string{"sine", "trisaw", "pulse", "gate", "sample"}

func arrDispFunc(arr []string) UnitParameterDisplayFunc {
	return func(v int) (string, string) {
		if v < 0 || v >= len(arr) {
			return "???", ""
		}
		return arr[v], ""
	}
}

func filterFrequencyDispFunc(v int) (string, string) {
	// Matlab was used to find the frequency for the singularity when r = 0:
	// % p is the frequency parameter squared, p = freq * freq
	// % We assume the singular case r = 0.
	// syms p z s T
	// A = [1 p;-p 1-p*p]; % discrete state-space matrix x(k+1)=A*x(k) + ...
	// pol = det(z*eye(2)-A) % characteristic discrete polynomial
	// spol = simplify(subs(pol,z,(1+s*T/2)/(1-s*T/2))) % Tustin approximation
	// % where T = 1/(44100 Hz) is the sample period
	// % spol is of the form N(s)/D(s), where N(s)=(-T^2*p^2*s^2+4*T^2*s^2+4*p^2)
	// % We are interested in the roots i.e. when spol == 0 <=> N(s)==0
	// simplify(solve((-T^2*p^2*s^2+4*T^2*s^2+4*p^2)==0,s))
	// % Answer: s=±2*p/(T*(p^2-4)^(1/2)). For small p, this simplifies to:
	// % s=±p*j/T. Thus, s=j*omega=j*2*pi*f => f=±p/(2*pi*T).
	// So the singularity is when f = p / (2*pi*T) Hz.
	freq := float64(v) / 128
	p := freq * freq
	f := 44100 * p / math.Pi / 2
	return strconv.FormatFloat(f, 'f', 0, 64), "Hz"
}

func compressorTimeDispFunc(v int) (string, string) {
	alpha := math.Pow(2, -24*float64(v)/128) // alpha is the "smoothing factor" of first order low pass iir
	sec := -1 / (44100 * math.Log(1-alpha))  // from smoothing factor to time constant, https://en.wikipedia.org/wiki/Exponential_smoothing
	return engineeringTime(sec)
}

func oscillatorTransposeDispFunc(v int) (string, string) {
	relvalue := v - 64
	octaves := relvalue / 12
	semitones := relvalue % 12
	if semitones == 0 {
		return strconv.Itoa(octaves), "oct"
	}
	return strconv.Itoa(semitones), "st"
}

func engineeringTime(sec float64) (string, string) {
	if sec < 1e-3 {
		return fmt.Sprintf("%.2f", sec*1e6), "us"
	} else if sec < 1 {
		return fmt.Sprintf("%.2f", sec*1e3), "ms"
	}
	return fmt.Sprintf("%.2f", sec), "s"
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// When unit.Type = "oscillator", its unit.Parameter["Type"] tells the type of
// the oscillator. There is five different oscillator types, so these consts
// just enumerate them.
const (
	Sine   = iota
	Trisaw = iota
	Pulse  = iota
	Gate   = iota
	Sample = iota
)

// UnitNames is a list of all the names of units, sorted
// alphabetically.
var UnitNames []string

func init() {
	UnitNames = make([]string, 0, len(UnitTypes))
	for k := range UnitTypes {
		UnitNames = append(UnitNames, k)
	}
	sort.Strings(UnitNames)
}

// Ports is static map allowing quickly finding the parameters of a unit that
// can be modulated. This is populated based on the UnitTypes list during
// init(). Thus, should be immutable, but Go not supporting that, then this will
// have to suffice: DO NOT EVER CHANGE THIS MAP.
var Ports = make(map[string]([]string))

func init() {
	for name, unitType := range UnitTypes {
		unitPorts := make([]string, 0)
		for _, param := range unitType {
			if param.CanModulate {
				unitPorts = append(unitPorts, param.Name)
			}
		}
		Ports[name] = unitPorts
	}
}

// Copy makes a deep copy of a unit.
func (u *Unit) Copy() Unit {
	parameters := make(map[string]int)
	for k, v := range u.Parameters {
		parameters[k] = v
	}
	varArgs := make([]int, len(u.VarArgs))
	copy(varArgs, u.VarArgs)
	return Unit{Type: u.Type, Parameters: parameters, VarArgs: varArgs, ID: u.ID, Disabled: u.Disabled, Comment: u.Comment}
}

// StackChange returns how this unit will affect the signal stack. "pop" and
// "addp" and such will consume the topmost signal, and thus return -1 (or -2,
// if the unit is a stereo unit). On the other hand, "oscillator" and "envelope"
// will produce a signal, and thus return 1 (or 2, if the unit is a stereo
// unit). Effects that just change the topmost signal and will not change the
// number of signals on the stack and thus return 0.
func (u *Unit) StackChange() int {
	if u.Disabled {
		return 0
	}
	switch u.Type {
	case "addp", "mulp", "pop", "out", "outaux", "aux":
		return -1 - u.Parameters["stereo"]
	case "envelope", "oscillator", "push", "noise", "receive", "loadnote", "loadval", "in", "compressor":
		return 1 + u.Parameters["stereo"]
	case "pan":
		return 1 - u.Parameters["stereo"]
	case "speed":
		return -1
	case "send":
		return (-1 - u.Parameters["stereo"]) * u.Parameters["sendpop"]
	}
	return 0
}

// StackNeed returns the number of signals that should be on the stack before
// this unit is executed. Used to prevent stack underflow. Units producing
// signals do not care what is on the stack before and will return 0.
func (u *Unit) StackNeed() int {
	if u.Disabled {
		return 0
	}
	switch u.Type {
	case "", "envelope", "oscillator", "noise", "receive", "loadnote", "loadval", "in":
		return 0
	case "mulp", "mul", "add", "addp", "xch":
		return 2 * (1 + u.Parameters["stereo"])
	case "speed":
		return 1
	}
	return 1 + u.Parameters["stereo"]
}

// Copy makes a deep copy of an Instrument
func (instr *Instrument) Copy() Instrument {
	units := make([]Unit, len(instr.Units))
	for i, u := range instr.Units {
		units[i] = u.Copy()
	}
	return Instrument{Name: instr.Name, Comment: instr.Comment, NumVoices: instr.NumVoices, Units: units}
}

// Copy makes a deep copy of a Patch.
func (p Patch) Copy() Patch {
	instruments := make([]Instrument, len(p))
	for i, instr := range p {
		instruments[i] = instr.Copy()
	}
	return instruments
}

// NumVoices returns the total number of voices used in the patch; summing the
// voices of every instrument
func (p Patch) NumVoices() int {
	ret := 0
	for _, i := range p {
		ret += i.NumVoices
	}
	return ret
}

// NumDelayLines return the total number of delay lines used in the patch;
// summing the number of delay lines of every delay unit in every instrument
func (p Patch) NumDelayLines() int {
	total := 0
	for _, instr := range p {
		for _, unit := range instr.Units {
			if unit.Type == "delay" {
				total += len(unit.VarArgs) * instr.NumVoices
			}
		}
	}
	return total
}

// NumSyns return the total number of sync outputs used in the patch; summing
// the number of sync outputs of every sync unit in every instrument
func (p Patch) NumSyncs() int {
	total := 0
	for _, instr := range p {
		for _, unit := range instr.Units {
			if unit.Type == "sync" {
				total += instr.NumVoices
			}
		}
	}
	return total
}

// FirstVoiceForInstrument returns the index of the first voice of given
// instrument. For example, if the Patch has three instruments (0, 1 and 2),
// with 1, 3, 2 voices, respectively, then FirstVoiceForInstrument(0) returns 0,
// FirstVoiceForInstrument(1) returns 1 and FirstVoiceForInstrument(2) returns
// 4. Essentially computes just the cumulative sum.
func (p Patch) FirstVoiceForInstrument(instrIndex int) int {
	ret := 0
	for _, t := range p[:instrIndex] {
		ret += t.NumVoices
	}
	return ret
}

// InstrumentForVoice returns the instrument number for the given voice index.
// For example, if the Patch has three instruments (0, 1 and 2), with 1, 3, 2
// voices, respectively, then InstrumentForVoice(0) returns 0,
// InstrumentForVoice(1) returns 1 and InstrumentForVoice(3) returns 1.
func (p Patch) InstrumentForVoice(voice int) (int, error) {
	if voice < 0 {
		return 0, errors.New("voice cannot be negative")
	}
	for i, instr := range p {
		if voice < instr.NumVoices {
			return i, nil
		}
		voice -= instr.NumVoices
	}
	return 0, errors.New("voice number is beyond the total voices of an instrument")
}

// FindUnit searches the instrument index and unit index for a unit with the
// given id. Two units should never have the same id, but if they do, then the
// first match is returned. Id 0 is interpreted as "no id", thus searching for
// id 0 returns an error. Error is also returned if the searched id is not
// found. FindUnit considers disabled units as non-existent.
func (p Patch) FindUnit(id int) (instrIndex int, unitIndex int, err error) {
	if id == 0 {
		return 0, 0, errors.New("FindUnit called with id 0")
	}
	for i, instr := range p {
		for u, unit := range instr.Units {
			if unit.ID == id && !unit.Disabled {
				return i, u, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("could not find a unit with id %v", id)
}
