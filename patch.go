package sointu

import (
	"errors"
	"fmt"
	"math"
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
	}

	// UnitParameter documents one parameter that an unit takes
	UnitParameter struct {
		Name        string // thould be found with this name in the Unit.Parameters map
		MinValue    int    // minimum value of the parameter, inclusive
		MaxValue    int    // maximum value of the parameter, inclusive
		CanSet      bool   // if this parameter can be set before hand i.e. through the gui
		CanModulate bool   // if this parameter can be modulated i.e. has a port number in "send" unit
	}
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
		{Name: "resolution", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"gain": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"invgain": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "invgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"filter": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "frequency", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
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
		{Name: "notetracking", MinValue: 0, MaxValue: 2, CanSet: true, CanModulate: false},
		{Name: "delaytime", MinValue: 0, MaxValue: -1, CanSet: false, CanModulate: true}},
	"compressor": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "attack", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "release", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "invgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "threshold", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "ratio", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
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
		{Name: "channel", MinValue: 0, MaxValue: 6, CanSet: true, CanModulate: false}},
	"send": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "amount", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "voice", MinValue: 0, MaxValue: 32, CanSet: true, CanModulate: false},
		{Name: "target", MinValue: 0, MaxValue: math.MaxInt32, CanSet: true, CanModulate: false},
		{Name: "port", MinValue: 0, MaxValue: 7, CanSet: true, CanModulate: false},
		{Name: "sendpop", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}},
	"envelope": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "attack", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "decay", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "sustain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "release", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"noise": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "shape", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"oscillator": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "transpose", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "detune", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "phase", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "color", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "shape", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "frequency", MinValue: 0, MaxValue: -1, CanSet: false, CanModulate: true},
		{Name: "type", MinValue: int(Sine), MaxValue: int(Sample), CanSet: true, CanModulate: false},
		{Name: "lfo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "unison", MinValue: 0, MaxValue: 3, CanSet: true, CanModulate: false},
		{Name: "samplestart", MinValue: 0, MaxValue: 1720329, CanSet: true, CanModulate: false},
		{Name: "loopstart", MinValue: 0, MaxValue: 65535, CanSet: true, CanModulate: false},
		{Name: "looplength", MinValue: 0, MaxValue: 65535, CanSet: true, CanModulate: false}},
	"loadval": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "value", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"receive": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "left", MinValue: 0, MaxValue: -1, CanSet: false, CanModulate: true},
		{Name: "right", MinValue: 0, MaxValue: -1, CanSet: false, CanModulate: true}},
	"in": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "channel", MinValue: 0, MaxValue: 6, CanSet: true, CanModulate: false}},
	"sync": []UnitParameter{},
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
	return Unit{Type: u.Type, Parameters: parameters, VarArgs: varArgs, ID: u.ID}
}

// StackChange returns how this unit will affect the signal stack. "pop" and
// "addp" and such will consume the topmost signal, and thus return -1 (or -2,
// if the unit is a stereo unit). On the other hand, "oscillator" and "envelope"
// will produce a signal, and thus return 1 (or 2, if the unit is a stereo
// unit). Effects that just change the topmost signal and will not change the
// number of signals on the stack and thus return 0.
func (u *Unit) StackChange() int {
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

// FindSendTarget searches the instrument number and unit index for a unit with
// the given id. Two units should never have the same id, but if they do, then
// the first match is returned. Id 0 is interpreted as "no id", thus searching
// for id 0 returns an error. Error is also returned if the searched id is not
// found.
func (p Patch) FindSendTarget(id int) (int, int, error) {
	if id == 0 {
		return 0, 0, errors.New("send targets unit id 0")
	}
	for i, instr := range p {
		for u, unit := range instr.Units {
			if unit.ID == id {
				return i, u, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("send targets an unit with id %v, could not find a unit with such an ID in the patch", id)
}

// ParamHintString returns a human readable string representing the current
// value of a given unit parameter.
func (p Patch) ParamHintString(instrIndex, unitIndex int, param string) string {
	if instrIndex < 0 || instrIndex >= len(p) {
		return ""
	}
	instr := p[instrIndex]
	if unitIndex < 0 || unitIndex >= len(instr.Units) {
		return ""
	}
	unit := instr.Units[unitIndex]
	value := unit.Parameters[param]
	switch unit.Type {
	case "envelope":
		switch param {
		case "attack":
			return engineeringTime(math.Pow(2, 24*float64(value)/128) / 44100)
		case "decay":
			return engineeringTime(math.Pow(2, 24*float64(value)/128) / 44100 * (1 - float64(unit.Parameters["sustain"])/128))
		case "release":
			return engineeringTime(math.Pow(2, 24*float64(value)/128) / 44100 * float64(unit.Parameters["sustain"]) / 128)
		}
	case "oscillator":
		switch param {
		case "type":
			switch value {
			case Sine:
				return "Sine"
			case Trisaw:
				return "Trisaw"
			case Pulse:
				return "Pulse"
			case Gate:
				return "Gate"
			case Sample:
				return "Sample"
			default:
				return "Unknown"
			}
		case "transpose":
			relvalue := value - 64
			octaves := relvalue / 12
			semitones := relvalue % 12
			if octaves != 0 {
				return fmt.Sprintf("%v oct, %v st", octaves, semitones)
			}
			return fmt.Sprintf("%v st", semitones)
		case "detune":
			return fmt.Sprintf("%v st", float32(value-64)/64.0)
		}
	case "compressor":
		switch param {
		case "attack":
			fallthrough
		case "release":
			alpha := math.Pow(2, -24*float64(value)/128) // alpha is the "smoothing factor" of first order low pass iir
			sec := -1 / (44100 * math.Log(1-alpha))      // from smoothing factor to time constant, https://en.wikipedia.org/wiki/Exponential_smoothing
			return engineeringTime(sec)
		case "ratio":
			return fmt.Sprintf("1 : %.3f", 1-float64(value)/128)
		}
	case "loadval":
		switch param {
		case "value":
			return fmt.Sprintf("%.2f", float32(value)/64-1)
		}
	case "send":
		switch param {
		case "amount":
			return fmt.Sprintf("%.2f", float32(value)/64-1)
		case "voice":
			if value == 0 {
				targetIndex, _, err := p.FindSendTarget(unit.Parameters["target"])
				if err == nil && targetIndex != instrIndex {
					return "all"
				}
				return "self"
			}
			return fmt.Sprintf("%v", value)
		case "target":
			instrIndex, unitIndex, err := p.FindSendTarget(unit.Parameters["target"])
			if err != nil {
				return "invalid target"
			}
			instr := p[instrIndex]
			unit := instr.Units[unitIndex]
			return fmt.Sprintf("%v / %v%v", instr.Name, unit.Type, unitIndex)
		case "port":
			instrIndex, unitIndex, err := p.FindSendTarget(unit.Parameters["target"])
			if err != nil {
				return fmt.Sprintf("%v ???", value)
			}
			portList := Ports[p[instrIndex].Units[unitIndex].Type]
			if value < 0 || value >= len(portList) {
				return fmt.Sprintf("%v ???", value)
			}
			return fmt.Sprintf(portList[value])
		}
	case "delay":
		switch param {
		case "notetracking":
			switch value {
			case 0:
				return "fixed"
			case 1:
				return "tracks pitch"
			case 2:
				return "tracks BPM"
			}
		}
	case "in", "aux":
		switch param {
		case "channel":
			switch value {
			case 0:
				return "left"
			case 1:
				return "right"
			case 2:
				return "aux1 left"
			case 3:
				return "aux1 right"
			case 4:
				return "aux2 left"
			case 5:
				return "aux2 right"
			case 6:
				return "aux3 left"
			case 7:
				return "aux3 right"
			}
		}
	case "crush":
		switch param {
		case "resolution":
			return fmt.Sprintf("%v bits", 24*float32(value)/128)
		}
	}
	return ""
}

func engineeringTime(sec float64) string {
	if sec < 1e-3 {
		return fmt.Sprintf("%.2f us", sec*1e6)
	} else if sec < 1 {
		return fmt.Sprintf("%.2f ms", sec*1e3)
	}
	return fmt.Sprintf("%.2f s", sec)
}
