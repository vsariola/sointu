package sointu

import (
	"fmt"
	"math"
)

// UnitParameter documents one parameter that an unit takes
type UnitParameter struct {
	Name        string // thould be found with this name in the Unit.Parameters map
	MinValue    int    // minimum value of the parameter, inclusive
	MaxValue    int    // maximum value of the parameter, inclusive
	CanSet      bool   // if this parameter can be set before hand i.e. through the gui
	CanModulate bool   // if this parameter can be modulated i.e. has a port number in "send" unit
}

func engineeringTime(sec float64) string {
	if sec < 1e-3 {
		return fmt.Sprintf("%.2f us", sec*1e6)
	} else if sec < 1 {
		return fmt.Sprintf("%.2f ms", sec*1e3)
	}
	return fmt.Sprintf("%.2f s", sec)
}

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
		{Name: "notetracking", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
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
