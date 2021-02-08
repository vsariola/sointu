package sointu

import (
	"errors"
	"fmt"
	"math"
)

// Unit is e.g. a filter, oscillator, envelope and its parameters
type Unit struct {
	Type       string
	Parameters map[string]int `yaml:",flow"`
	VarArgs    []int          `yaml:",flow,omitempty"`
}

func (u *Unit) Copy() Unit {
	parameters := make(map[string]int)
	for k, v := range u.Parameters {
		parameters[k] = v
	}
	varArgs := make([]int, len(u.VarArgs))
	copy(varArgs, u.VarArgs)
	return Unit{Type: u.Type, Parameters: parameters, VarArgs: varArgs}
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
	Name      string
	NumVoices int
	Units     []Unit
}

func (instr *Instrument) Copy() Instrument {
	units := make([]Unit, len(instr.Units))
	for i, u := range instr.Units {
		units[i] = u.Copy()
	}
	return Instrument{Name: instr.Name, NumVoices: instr.NumVoices, Units: units}
}

// Patch is simply a list of instruments used in a song
type Patch struct {
	Instruments []Instrument
}

func (p *Patch) Copy() Patch {
	instruments := make([]Instrument, len(p.Instruments))
	for i, instr := range p.Instruments {
		instruments[i] = instr.Copy()
	}
	return Patch{Instruments: instruments}
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
	Sequence  []byte   `yaml:",flow"`
	Patterns  [][]byte `yaml:",flow"`
}

func (t *Track) Copy() Track {
	sequence := make([]byte, len(t.Sequence))
	copy(sequence, t.Sequence)
	patterns := make([][]byte, len(t.Patterns))
	for i, oldPat := range t.Patterns {
		newPat := make([]byte, len(oldPat))
		copy(newPat, oldPat)
		patterns[i] = newPat
	}
	return Track{
		NumVoices: t.NumVoices,
		Sequence:  sequence,
		Patterns:  patterns,
	}
}

type Synth interface {
	Render(buffer []float32, maxtime int) (int, int, error)
	Update(patch Patch) error
	Trigger(voice int, note byte)
	Release(voice int)
}

func Render(synth Synth, buffer []float32) error {
	s, _, err := synth.Render(buffer, math.MaxInt32)
	if err != nil {
		return fmt.Errorf("sointu.Render failed: %v", err)
	}
	if s != len(buffer)/2 {
		return errors.New("in sointu.Render, synth.Render should have filled the whole buffer but did not")
	}
	return nil
}

type SynthService interface {
	Compile(patch Patch) (Synth, error)
}

type AudioSink interface {
	WriteAudio(buffer []float32) (err error)
	Close() error
}

type AudioSource interface {
	ReadAudio(buffer []float32) (n int, err error)
	Close() error
}

type AudioContext interface {
	Output() AudioSink
	Close() error
}

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
}

type Song struct {
	BPM            int
	RowsPerPattern int
	RowsPerBeat    int
	Tracks         []Track
	Patch          Patch
}

func (s *Song) Copy() Song {
	tracks := make([]Track, len(s.Tracks))
	for i, t := range s.Tracks {
		tracks[i] = t.Copy()
	}
	return Song{BPM: s.BPM, RowsPerPattern: s.RowsPerPattern, RowsPerBeat: s.RowsPerBeat, Tracks: tracks, Patch: s.Patch.Copy()}
}

func (s *Song) SequenceLength() int {
	return len(s.Tracks[0].Sequence)
}

func (s *Song) TotalRows() int {
	return s.RowsPerPattern * s.SequenceLength()
}

func (s *Song) SamplesPerRow() int {
	return 44100 * 60 / (s.BPM * s.RowsPerBeat)
}

func (s *Song) FirstTrackVoice(track int) int {
	ret := 0
	for _, t := range s.Tracks[:track] {
		ret += t.NumVoices
	}
	return ret
}

func (s *Song) TotalTrackVoices() int {
	ret := 0
	for _, t := range s.Tracks {
		ret += t.NumVoices
	}
	return ret
}

// TBD: Where shall we put methods that work on pure domain types and have no dependencies
// e.g. Validate here
func (s *Song) Validate() error {
	if s.BPM < 1 {
		return errors.New("BPM should be > 0")
	}
	var patternLen int
	for i, t := range s.Tracks {
		for j, pat := range t.Patterns {
			if i == 0 && j == 0 {
				patternLen = len(pat)
			} else {
				if len(pat) != patternLen {
					return errors.New("Every pattern should have the same length")
				}
			}
		}
	}
	for i := range s.Tracks[:len(s.Tracks)-1] {
		if len(s.Tracks[i].Sequence) != len(s.Tracks[i+1].Sequence) {
			return errors.New("Every track should have the same sequence length")
		}
	}
	totalTrackVoices := 0
	for _, track := range s.Tracks {
		totalTrackVoices += track.NumVoices
		for _, p := range track.Sequence {
			if p < 0 || int(p) >= len(track.Patterns) {
				return errors.New("Tracks use a non-existing pattern")
			}
		}
	}
	if totalTrackVoices > s.Patch.TotalVoices() {
		return errors.New("Tracks use too many voices")
	}
	return nil
}

func (s *Song) ParamHintString(instrIndex, unitIndex int, param string) string {
	if instrIndex < 0 || instrIndex >= len(s.Patch.Instruments) {
		return ""
	}
	instr := s.Patch.Instruments[instrIndex]
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
	case "send":
		if param == "voice" || param == "unit" || param == "port" {
			targetVoice := unit.Parameters["voice"]
			if param == "voice" && targetVoice == 0 {
				return "self"
			}
			targetInstrument := instrIndex
			if targetVoice > 0 { // global send, find the instrument
				if targetVoice > s.Patch.TotalVoices() {
					return ""
				}
				targetVoice--
				targetInstrument = 0
				for targetVoice >= s.Patch.Instruments[targetInstrument].NumVoices {
					targetVoice -= s.Patch.Instruments[targetInstrument].NumVoices
					targetInstrument++
				}
			}
			if param == "voice" {
				return fmt.Sprintf("%v (voice %v)", s.Patch.Instruments[targetInstrument].Name, targetVoice)
			}
			targetUnitIndex := unit.Parameters["unit"]
			units := s.Patch.Instruments[targetInstrument].Units
			if targetUnitIndex < 0 || targetUnitIndex >= len(units) {
				return ""
			}
			if param == "unit" {
				return fmt.Sprintf("%v#%v", units[targetUnitIndex].Type, targetUnitIndex)
			}
			port := value
			for _, param := range UnitTypes[units[targetUnitIndex].Type] {
				if param.CanModulate {
					port--
					if port < 0 {
						return param.Name
					}
				}
			}
		}
	}
	return ""
}

func Play(synth Synth, song Song) ([]float32, error) {
	err := song.Validate()
	if err != nil {
		return nil, err
	}
	curVoices := make([]int, len(song.Tracks))
	for i := range curVoices {
		curVoices[i] = song.FirstTrackVoice(i)
	}
	initialCapacity := song.TotalRows() * song.SamplesPerRow() * 2
	buffer := make([]float32, 0, initialCapacity)
	rowbuffer := make([]float32, song.SamplesPerRow()*2)
	for row := 0; row < song.TotalRows(); row++ {
		patternRow := row % song.RowsPerPattern
		pattern := row / song.RowsPerPattern
		for t := range song.Tracks {
			patternIndex := song.Tracks[t].Sequence[pattern]
			note := song.Tracks[t].Patterns[patternIndex][patternRow]
			if note > 0 && note <= 1 { // anything but hold causes an action.
				continue
			}
			synth.Release(curVoices[t])
			if note > 1 {
				curVoices[t]++
				first := song.FirstTrackVoice(t)
				if curVoices[t] >= first+song.Tracks[t].NumVoices {
					curVoices[t] = first
				}
				synth.Trigger(curVoices[t], note)
			}
		}
		tries := 0
		for rowtime := 0; rowtime < song.SamplesPerRow(); {
			samples, time, _ := synth.Render(rowbuffer, song.SamplesPerRow()-rowtime)
			rowtime += time
			buffer = append(buffer, rowbuffer[:samples*2]...)
			if tries > 100 {
				return nil, fmt.Errorf("Song speed modulation likely so slow that row never advances; error at pattern %v, row %v", pattern, patternRow)
			}
		}
	}
	return buffer, nil
}
