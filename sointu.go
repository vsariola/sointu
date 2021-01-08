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
type Patch struct {
	Instruments []Instrument
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

type Synth interface {
	Render(buffer []float32, maxtime int) (int, int, error)
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
		{Name: "samplestart", MinValue: 0, MaxValue: 3440659, CanSet: true, CanModulate: false},
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
	BPM    int
	Tracks []Track
	Patch  Patch
}

func (s *Song) PatternRows() int {
	return len(s.Tracks[0].Patterns[0])
}

func (s *Song) SequenceLength() int {
	return len(s.Tracks[0].Sequence)
}

func (s *Song) TotalRows() int {
	return s.PatternRows() * s.SequenceLength()
}

func (s *Song) SamplesPerRow() int {
	return 44100 * 60 / (s.BPM * 4)
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
		patternRow := row % song.PatternRows()
		pattern := row / song.PatternRows()
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
