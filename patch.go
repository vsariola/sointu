package sointu

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"

	"gopkg.in/yaml.v3"
)

type (
	// Patch is simply a list of instruments used in a song
	Patch []Instrument

	// Instrument includes a list of units consisting of the instrument, and the number of polyphonic voices for this instrument
	Instrument struct {
		Name        string `yaml:",omitempty"`
		Comment     string `yaml:",omitempty"`
		NumVoices   int
		Units       []Unit
		Mute        bool `yaml:",omitempty"` // Mute is only used in the tracker for soloing/muting instruments; the compiled player ignores this field
		CoreBitMask uint `yaml:",omitempty"` // CoreBitMask tells which cores this instrument can run on; 0 means all cores
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
		Parameters ParamMap `yaml:",flow"`

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

	ParamMap map[string]int

	// UnitParameter documents one parameter that an unit takes
	UnitParameter struct {
		Name        string // thould be found with this name in the Unit.Parameters map
		MinValue    int    // minimum value of the parameter, inclusive
		MaxValue    int    // maximum value of the parameter, inclusive
		Neutral     int    // neutral value of the parameter
		CanSet      bool   // if this parameter can be set before hand i.e. through the gui
		CanModulate bool   // if this parameter can be modulated i.e. has a port number in "send" unit
		DisplayFunc UnitParameterDisplayFunc
	}

	// StackUse documents how a unit will affect the signal stack.
	StackUse struct {
		Inputs     [][]int // Inputs documents which inputs contribute to which outputs. len(Inputs) is the number of inputs. Each input can contribute to multiple outputs, so its a slice.
		Modifies   []bool  // Modifies documents which of the (mixed) inputs are actually modified by the unit
		NumOutputs int     // NumOutputs is the number of outputs produced by the unit. This is used to determine how many outputs are needed for the unit.
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
		{Name: "drive", MinValue: 0, Neutral: 64, MaxValue: 128, CanSet: true, CanModulate: true}},
	"hold": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "holdfreq", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true}},
	"crush": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "resolution", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return formatFloat(24 * float64(v) / 128), "bits" }}},
	"gain": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return strconv.FormatFloat(toDecibel(float64(v)/128), 'g', 3, 64), "dB" }}},
	"invgain": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "invgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return strconv.FormatFloat(toDecibel(128/float64(v)), 'g', 3, 64), "dB" }}},
	"dbgain": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "decibels", MinValue: 0, Neutral: 64, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return formatFloat(40 * (float64(v)/64 - 1)), "dB" }}},
	"filter": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "frequency", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: filterFrequencyDispFunc},
		{Name: "resonance", MinValue: 0, Neutral: 128, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) {
			return strconv.FormatFloat(toDecibel(128/float64(v)), 'g', 3, 64), "Q dB"
		}},
		{Name: "lowpass", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "bandpass", MinValue: -1, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "highpass", MinValue: -1, MaxValue: 1, CanSet: true, CanModulate: false}},
	"clip": []UnitParameter{{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}},
	"pan": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "panning", MinValue: 0, Neutral: 64, MaxValue: 128, CanSet: true, CanModulate: true}},
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
			return strconv.FormatFloat(toDecibel(128/float64(v)), 'g', 3, 64), "dB"
		}},
		{Name: "threshold", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) {
			return strconv.FormatFloat(toDecibel(float64(v)/128), 'g', 3, 64), "dB"
		}},
		{Name: "ratio", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return formatFloat(1 - float64(v)/128), "" }}},
	"speed": []UnitParameter{},
	"out": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return strconv.FormatFloat(toDecibel(float64(v)/128), 'g', 3, 64), "dB" }}},
	"outaux": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "outgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return strconv.FormatFloat(toDecibel(float64(v)/128), 'g', 3, 64), "dB" }},
		{Name: "auxgain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return strconv.FormatFloat(toDecibel(float64(v)/128), 'g', 3, 64), "dB" }}},
	"aux": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return strconv.FormatFloat(toDecibel(float64(v)/128), 'g', 3, 64), "dB" }},
		{Name: "channel", MinValue: 0, MaxValue: 6, CanSet: true, CanModulate: false, DisplayFunc: arrDispFunc(channelNames[:])}},
	"send": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "amount", MinValue: 0, Neutral: 64, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return formatFloat(float64(v)/64 - 1), "" }},
		{Name: "voice", MinValue: 0, MaxValue: 32, CanSet: true, CanModulate: false, DisplayFunc: sendVoiceDispFunc},
		{Name: "target", MinValue: 0, MaxValue: math.MaxInt32, CanSet: true, CanModulate: false},
		{Name: "port", MinValue: 0, MaxValue: 7, CanSet: true, CanModulate: false},
		{Name: "sendpop", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false}},
	"envelope": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "attack", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return engineeringTime(math.Pow(2, 24*float64(v)/128) / 44100) }},
		{Name: "decay", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return engineeringTime(math.Pow(2, 24*float64(v)/128) / 44100) }},
		{Name: "sustain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return strconv.FormatFloat(toDecibel(float64(v)/128), 'g', 3, 64), "dB" }},
		{Name: "release", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return engineeringTime(math.Pow(2, 24*float64(v)/128) / 44100) }},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return strconv.FormatFloat(toDecibel(float64(v)/128), 'g', 3, 64), "dB" }}},
	"noise": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "shape", MinValue: 0, Neutral: 64, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return strconv.FormatFloat(toDecibel(float64(v)/128), 'g', 3, 64), "dB" }}},
	"oscillator": []UnitParameter{
		{Name: "stereo", MinValue: 0, MaxValue: 1, CanSet: true, CanModulate: false},
		{Name: "transpose", MinValue: 0, Neutral: 64, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: oscillatorTransposeDispFunc},
		{Name: "detune", MinValue: 0, Neutral: 64, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return formatFloat(float64(v-64) / 64), "st" }},
		{Name: "phase", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) {
			return strconv.FormatFloat(float64(v)/128*360, 'f', 1, 64), "°"
		}},
		{Name: "color", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "shape", MinValue: 0, Neutral: 64, MaxValue: 128, CanSet: true, CanModulate: true},
		{Name: "gain", MinValue: 0, MaxValue: 128, CanSet: true, CanModulate: true, DisplayFunc: func(v int) (string, string) { return strconv.FormatFloat(toDecibel(float64(v)/128), 'g', 3, 64), "dB" }},
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

// compile errors if interface is not implemented.
var _ yaml.Unmarshaler = &ParamMap{}

func (a *ParamMap) UnmarshalYAML(value *yaml.Node) error {
	var m map[string]int
	if err := value.Decode(&m); err != nil {
		return err
	}
	// Backwards compatibility hack: if the patch was saved with an older
	// version of Sointu, it might have used the negbandpass and neghighpass
	// parameters, which now correspond to having bandpass as value -1 and
	// highpass as value -1.
	if n, ok := m["negbandpass"]; ok {
		m["bandpass"] = m["bandpass"] - n
		delete(m, "negbandpass")
	}
	if n, ok := m["neghighpass"]; ok {
		m["highpass"] = m["highpass"] - n
		delete(m, "neghighpass")
	}
	*a = m
	return nil
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
	if relvalue%12 == 0 {
		return strconv.Itoa(relvalue / 12), "oct"
	}
	return strconv.Itoa(relvalue), "st"
}

func sendVoiceDispFunc(v int) (string, string) {
	if v == 0 {
		return "default", ""
	}
	return strconv.Itoa(v), ""
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

func toDecibel(amplitude float64) float64 {
	if amplitude <= 0 {
		return math.Inf(-1)
	}
	// Decibels are defined as 20 * log10(amplitude)
	// https://en.wikipedia.org/wiki/Decibel#Sound_pressure
	return 20 * math.Log10(amplitude)
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
	ret := *u
	ret.Parameters = make(map[string]int, len(u.Parameters))
	for k, v := range u.Parameters {
		ret.Parameters[k] = v
	}
	ret.VarArgs = make([]int, len(u.VarArgs))
	copy(ret.VarArgs, u.VarArgs)
	return ret
}

var stackUseSource = [2]StackUse{
	{Inputs: [][]int{}, Modifies: []bool{true}, NumOutputs: 1},       // mono
	{Inputs: [][]int{}, Modifies: []bool{true, true}, NumOutputs: 2}, // stereo
}
var stackUseSink = [2]StackUse{
	{Inputs: [][]int{{0}}, Modifies: []bool{true}, NumOutputs: 0},            // mono
	{Inputs: [][]int{{0}, {1}}, Modifies: []bool{true, true}, NumOutputs: 0}, // stereo
}
var stackUseEffect = [2]StackUse{
	{Inputs: [][]int{{0}}, Modifies: []bool{true}, NumOutputs: 1},            // mono
	{Inputs: [][]int{{0}, {1}}, Modifies: []bool{true, true}, NumOutputs: 2}, // stereo
}
var stackUseMonoStereo = map[string][2]StackUse{
	"add": {
		{Inputs: [][]int{{0, 1}, {1}}, Modifies: []bool{false, true}, NumOutputs: 2},
		{Inputs: [][]int{{0, 2}, {1, 3}, {2}, {3}}, Modifies: []bool{false, false, true, true}, NumOutputs: 4},
	},
	"mul": {
		{Inputs: [][]int{{0, 1}, {1}}, Modifies: []bool{false, true}, NumOutputs: 2},
		{Inputs: [][]int{{0, 2}, {1, 3}, {2}, {3}}, Modifies: []bool{false, false, true, true}, NumOutputs: 4},
	},
	"addp": {
		{Inputs: [][]int{{0}, {0}}, Modifies: []bool{true}, NumOutputs: 1},
		{Inputs: [][]int{{0}, {1}, {0}, {1}}, Modifies: []bool{true, true}, NumOutputs: 2},
	},
	"mulp": {
		{Inputs: [][]int{{0}, {0}}, Modifies: []bool{true}, NumOutputs: 1},
		{Inputs: [][]int{{0}, {1}, {0}, {1}}, Modifies: []bool{true, true}, NumOutputs: 2},
	},
	"xch": {
		{Inputs: [][]int{{1}, {0}}, Modifies: []bool{false, false}, NumOutputs: 2},
		{Inputs: [][]int{{2}, {3}, {0}, {1}}, Modifies: []bool{false, false, false, false}, NumOutputs: 4},
	},
	"push": {
		{Inputs: [][]int{{0, 1}}, Modifies: []bool{false, false}, NumOutputs: 2},
		{Inputs: [][]int{{0, 2}, {1, 3}}, Modifies: []bool{false, false, false, false}, NumOutputs: 4},
	},
	"pop":        stackUseSink,
	"envelope":   stackUseSource,
	"oscillator": stackUseSource,
	"noise":      stackUseSource,
	"loadnote":   stackUseSource,
	"loadval":    stackUseSource,
	"receive":    stackUseSource,
	"in":         stackUseSource,
	"out":        stackUseSink,
	"outaux":     stackUseSink,
	"aux":        stackUseSink,
	"distort":    stackUseEffect,
	"hold":       stackUseEffect,
	"crush":      stackUseEffect,
	"gain":       stackUseEffect,
	"invgain":    stackUseEffect,
	"dbgain":     stackUseEffect,
	"filter":     stackUseEffect,
	"clip":       stackUseEffect,
	"delay":      stackUseEffect,
	"compressor": {
		{Inputs: [][]int{{0, 1}}, Modifies: []bool{false, true}, NumOutputs: 2},                            // mono
		{Inputs: [][]int{{0, 2, 3}, {1, 2, 3}}, Modifies: []bool{false, false, true, true}, NumOutputs: 4}, // stereo
	},
	"pan": {
		{Inputs: [][]int{{0, 1}}, Modifies: []bool{true, true}, NumOutputs: 2},   // mono
		{Inputs: [][]int{{0}, {1}}, Modifies: []bool{true, true}, NumOutputs: 2}, // mono
	},
	"speed": {
		{Inputs: [][]int{{0}}, Modifies: []bool{true}, NumOutputs: 0},
		{},
	},
	"sync": {
		{Inputs: [][]int{{0}}, Modifies: []bool{false}, NumOutputs: 1},
		{},
	},
}
var stackUseSendNoPop = [2]StackUse{
	{Inputs: [][]int{{0}}, Modifies: []bool{true}, NumOutputs: 1},
	{Inputs: [][]int{{0}, {1}}, Modifies: []bool{true, true}, NumOutputs: 2},
}
var stackUseSendPop = [2]StackUse{
	{Inputs: [][]int{{0}}, Modifies: []bool{true}, NumOutputs: 0},            // mono
	{Inputs: [][]int{{0}, {1}}, Modifies: []bool{true, true}, NumOutputs: 0}, // stereo
}

func (u *Unit) StackUse() StackUse {
	if u.Disabled {
		return StackUse{}
	}
	if u.Type == "send" {
		// "send" unit is special, it has a different stack use depending on sendpop
		if u.Parameters["sendpop"] == 0 {
			return stackUseSendNoPop[u.Parameters["stereo"]]
		}
		return stackUseSendPop[u.Parameters["stereo"]]
	}
	return stackUseMonoStereo[u.Type][u.Parameters["stereo"]]
}

// StackChange returns how this unit will affect the signal stack. "pop" and
// "addp" and such will consume the topmost signal, and thus return -1 (or -2,
// if the unit is a stereo unit). On the other hand, "oscillator" and "envelope"
// will produce a signal, and thus return 1 (or 2, if the unit is a stereo
// unit). Effects that just change the topmost signal and will not change the
// number of signals on the stack and thus return 0.
func (u *Unit) StackChange() int {
	s := u.StackUse()
	return s.NumOutputs - len(s.Inputs)
}

// StackNeed returns the number of signals that should be on the stack before
// this unit is executed. Used to prevent stack underflow. Units producing
// signals do not care what is on the stack before and will return 0.
func (u *Unit) StackNeed() int {
	return len(u.StackUse().Inputs)
}

// Copy makes a deep copy of an Instrument
func (instr *Instrument) Copy() Instrument {
	ret := *instr
	ret.Units = make([]Unit, len(instr.Units))
	for i, u := range instr.Units {
		ret.Units[i] = u.Copy()
	}
	return ret
}

// Implement the counter interface
func (i *Instrument) GetNumVoices() int {
	return i.NumVoices
}

func (i *Instrument) SetNumVoices(count int) {
	i.NumVoices = count
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
	if instrIndex < 0 {
		return 0
	}
	instrIndex = min(instrIndex, len(p))
	ret := 0
	for i := 0; i < instrIndex; i++ {
		ret += p[i].NumVoices
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

func FindParamForModulationPort(unitName string, index int) (up UnitParameter, upIndex int, ok bool) {
	unitType, ok := UnitTypes[unitName]
	if !ok {
		return UnitParameter{}, 0, false
	}
	for i, param := range unitType {
		if !param.CanModulate {
			continue
		}
		if index == 0 {
			return param, i, true
		}
		index--
	}
	return UnitParameter{}, 0, false
}
