package sointu

import (
	"errors"
	"fmt"
	"math"
)

// Patch is simply a list of instruments used in a song
type Patch []Instrument

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
	case "send":
		switch param {
		case "amount":
			return fmt.Sprintf("%.2f", float32(value)/64-1)
		case "voice":
			if value == 0 {
				return "auto"
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
