package go4k

import (
	"errors"
	"math"
)

// Unit is e.g. a filter, oscillator, envelope and its parameters
type Unit struct {
	Type       string
	Parameters map[string]int `yaml:",flow"`
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

type SampleOffset struct {
	Start      int
	LoopStart  int
	LoopLength int
}

// Patch is simply a list of instruments used in a song
type Patch struct {
	Instruments   []Instrument
	DelayTimes    []int `yaml:",flow"`
	SampleOffsets []SampleOffset
}

func (p Patch) TotalVoices() int {
	ret := 0
	for _, i := range p.Instruments {
		ret += i.NumVoices
	}
	return ret
}

func (patch Patch) InstrumentForVoice(voice int) (int, error) {
	if voice < 0 {
		return 0, errors.New("voice cannot be negative")
	}
	for i, instr := range patch.Instruments {
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
	Sequence  []byte `yaml:",flow"`
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

func (p *Patch) Encode() ([]string, []byte, []byte) {
	var code []byte
	var values []byte
	var jumpTable []string
	assignedIds := map[string]byte{}
	for _, instr := range p.Instruments {
		for _, unit := range instr.Units {
			if _, ok := assignedIds[unit.Type]; !ok {
				jumpTable = append(jumpTable, unit.Type)
				assignedIds[unit.Type] = byte(len(jumpTable) * 2)
			}
			stereo, unitValues := Encode(unit)
			code = append(code, stereo+assignedIds[unit.Type])
			values = append(values, unitValues...)
		}
		code = append(code, 0)
	}
	return jumpTable, code, values
}

// UnitParameter documents one parameter that an unit takes
type UnitParameter struct {
	Name        string // thould be found with this name in the Unit.Parameters map
	MinValue    int    // minimum value of the parameter, inclusive
	MaxValue    int    // maximum value of the parameter, inclusive
	CanSet      bool   // if this parameter can be set before hand i.e. through the gui
	CanModulate bool   // if this parameter can be modulated i.e. has a port number in "send" unit
}

func Encode(unit Unit) (byte, []byte) {
	var values []byte
	for _, v := range UnitTypes[unit.Type] {
		if v.CanSet && v.CanModulate {
			values = append(values, byte(unit.Parameters[v.Name]))
		}
	}
	if unit.Type == "aux" {
		values = append(values, byte(unit.Parameters["channel"]))
	} else if unit.Type == "in" {
		values = append(values, byte(unit.Parameters["channel"]))
	} else if unit.Type == "oscillator" {
		flags := 0
		switch unit.Parameters["type"] {
		case Sine:
			flags = 0x40
		case Trisaw:
			flags = 0x20
		case Pulse:
			flags = 0x10
		case Gate:
			flags = 0x04
		case Sample:
			flags = 0x80
		}
		if unit.Parameters["lfo"] == 1 {
			flags += 0x08
		}
		flags += unit.Parameters["unison"]
		values = append(values, byte(flags))
	} else if unit.Type == "filter" {
		flags := 0
		if unit.Parameters["lowpass"] == 1 {
			flags += 0x40
		}
		if unit.Parameters["bandpass"] == 1 {
			flags += 0x20
		}
		if unit.Parameters["highpass"] == 1 {
			flags += 0x10
		}
		if unit.Parameters["negbandpass"] == 1 {
			flags += 0x08
		}
		if unit.Parameters["neghighpass"] == 1 {
			flags += 0x04
		}
		values = append(values, byte(flags))
	} else if unit.Type == "send" {
		address := ((unit.Parameters["unit"] + 1) << 4) + unit.Parameters["port"] // each unit is 16 dwords, 8 workspace followed by 8 ports. +1 is for skipping the note/release/inputs
		if unit.Parameters["voice"] > 0 {
			address += 0x8000 + 16 + (unit.Parameters["voice"]-1)*1024 // global send, +16 is for skipping the out/aux ports
		}
		if unit.Parameters["sendpop"] == 1 {
			address += 0x8
		}
		values = append(values, byte(address&255), byte(address>>8))
	} else if unit.Type == "delay" {
		countTrack := (unit.Parameters["count"] << 1) - 1 + unit.Parameters["notetracking"] // 1 means no note tracking and 1 delay, 2 means notetracking with 1 delay, 3 means no note tracking and 2 delays etc.
		values = append(values, byte(unit.Parameters["delay"]), byte(countTrack))
	}
	return byte(unit.Parameters["stereo"]), values
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
		{Name: "delay", MinValue: 0, MaxValue: 255, CanSet: true, CanModulate: false},
		{Name: "count", MinValue: 0, MaxValue: 255, CanSet: true, CanModulate: false},
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
		{Name: "unit", MinValue: 0, MaxValue: 63, CanSet: true, CanModulate: false},
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
		{Name: "unison", MinValue: 0, MaxValue: 3, CanSet: true, CanModulate: false}},
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
}
