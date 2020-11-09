package bridge

import (
	"errors"
	"fmt"

	"github.com/vsariola/sointu/go4k"
)

// #cgo CFLAGS: -I"${SRCDIR}/../../include/sointu"
// #cgo LDFLAGS: "${SRCDIR}/../../build/libsointu.a"
// #include <sointu.h>
import "C"

type opTableEntry struct {
	opcode        C.int
	parameterList []string
}

var opcodeTable = map[string]opTableEntry{
	"add":        opTableEntry{C.su_add_id, []string{}},
	"addp":       opTableEntry{C.su_addp_id, []string{}},
	"pop":        opTableEntry{C.su_pop_id, []string{}},
	"loadnote":   opTableEntry{C.su_loadnote_id, []string{}},
	"mul":        opTableEntry{C.su_mul_id, []string{}},
	"mulp":       opTableEntry{C.su_mulp_id, []string{}},
	"push":       opTableEntry{C.su_push_id, []string{}},
	"xch":        opTableEntry{C.su_xch_id, []string{}},
	"distort":    opTableEntry{C.su_distort_id, []string{"drive"}},
	"hold":       opTableEntry{C.su_hold_id, []string{"holdfreq"}},
	"crush":      opTableEntry{C.su_crush_id, []string{"resolution"}},
	"gain":       opTableEntry{C.su_gain_id, []string{"gain"}},
	"invgain":    opTableEntry{C.su_invgain_id, []string{"invgain"}},
	"filter":     opTableEntry{C.su_filter_id, []string{"frequency", "resonance"}},
	"clip":       opTableEntry{C.su_clip_id, []string{}},
	"pan":        opTableEntry{C.su_pan_id, []string{"panning"}},
	"delay":      opTableEntry{C.su_delay_id, []string{"pregain", "dry", "feedback", "damp", "delaycount"}},
	"compressor": opTableEntry{C.su_compres_id, []string{"attack", "release", "invgain", "threshold", "ratio"}},
	"speed":      opTableEntry{C.su_speed_id, []string{}},
	"out":        opTableEntry{C.su_out_id, []string{"gain"}},
	"outaux":     opTableEntry{C.su_outaux_id, []string{"outgain", "auxgain"}},
	"aux":        opTableEntry{C.su_aux_id, []string{"gain", "channel"}},
	"send":       opTableEntry{C.su_send_id, []string{"amount"}},
	"envelope":   opTableEntry{C.su_envelope_id, []string{"attack", "decay", "sustain", "release", "gain"}},
	"noise":      opTableEntry{C.su_noise_id, []string{"shape", "gain"}},
	"oscillator": opTableEntry{C.su_oscillat_id, []string{"transpose", "detune", "phase", "color", "shape", "gain"}},
	"loadval":    opTableEntry{C.su_loadval_id, []string{"value"}},
	"receive":    opTableEntry{C.su_receive_id, []string{}},
	"in":         opTableEntry{C.su_in_id, []string{"channel"}},
}

// Render renders until the buffer is full or the modulated time is reached, whichever
// happens first.
// Parameters:
//   buffer     float32 slice to fill with rendered samples. Stereo signal, so
//              should have even length.
//   maxtime    how long nominal time to render in samples. Speed unit might modulate time
//              so the actual number of samples rendered depends on the modulation and if
//              buffer is full before maxtime is reached.
// Returns a tuple (int, int, error), consisting of:
//   samples    number of samples rendered in the buffer
//   time       how much the time advanced
//   error      potential error
// In practice, if nsamples = len(buffer)/2, then time <= maxtime. If maxtime was reached
// first, then nsamples <= len(buffer)/2 and time >= maxtime. Note that it could happen that
// time > maxtime, as it is modulated and the time could advance by 2 or more, so the loop
// exit condition would fire when the time is already past maxtime.
// Under no conditions, nsamples >= len(buffer)/2 i.e. guaranteed to never overwrite the buffer.
func (synth *C.Synth) Render(buffer []float32, maxtime int) (int, int, error) {
	if len(buffer)%1 == 1 {
		return -1, -1, errors.New("RenderTime writes stereo signals, so buffer should have even length")
	}
	samples := C.int(len(buffer) / 2)
	time := C.int(maxtime)
	errcode := int(C.su_render(synth, (*C.float)(&buffer[0]), &samples, &time))
	if errcode > 0 {
		return -1, -1, errors.New("RenderTime failed")
	}
	return int(samples), int(time), nil
}

func Synth(patch go4k.Patch) (*C.Synth, error) {
	s := new(C.Synth)
	sampleno := 0
	totalVoices := 0
	commands := make([]byte, 0)
	values := make([]byte, 0)
	polyphonyBitmask := 0
	delayTable, delayIndices := go4k.ConstructDelayTimeTable(patch)
	for i, v := range delayTable {
		s.DelayTimes[i] = C.ushort(v)
	}
	for insid, instr := range patch {
		if len(instr.Units) > 63 {
			return nil, errors.New("An instrument can have a maximum of 63 units")
		}
		if instr.NumVoices < 1 {
			return nil, errors.New("Each instrument must have at least 1 voice")
		}
		for unitid, unit := range instr.Units {
			if val, ok := opcodeTable[unit.Type]; ok {
				opCode := val.opcode
				if unit.Stereo {
					opCode++
				}
				commands = append(commands, byte(opCode))
				for _, paramname := range val.parameterList {
					if unit.Type == "delay" && paramname == "delaycount" {
						if unit.Stereo && len(unit.DelayTimes)%2 != 0 {
							return nil, errors.New("Stereo delays should have even number of delaytimes")
						}
						values = append(values, byte(delayIndices[insid][unitid]))
						count := len(unit.DelayTimes)
						if unit.Stereo {
							count /= 2
						}
						count = count*2 - 1
						if unit.Parameters["notetracking"] == 1 {
							count++
						}
						values = append(values, byte(count))
					} else if unit.Type == "oscillator" && unit.Parameters["type"] == go4k.Sample && paramname == "color" {
						values = append(values, byte(sampleno))
						s.SampleOffsets[sampleno].Start = (C.uint)(unit.Parameters["start"])
						s.SampleOffsets[sampleno].LoopStart = (C.ushort)(unit.Parameters["loopstart"])
						s.SampleOffsets[sampleno].LoopLength = (C.ushort)(unit.Parameters["looplength"])
						sampleno++
					} else if pval, ok := unit.Parameters[paramname]; ok {
						values = append(values, byte(pval))
					} else {
						return nil, fmt.Errorf("Unit parameter undefined: %v (at instrument %v, unit %v)", paramname, insid, unitid)
					}
				}
				if unit.Type == "oscillator" {
					flags := 0
					switch unit.Parameters["type"] {
					case go4k.Sine:
						flags = 0x40
					case go4k.Trisaw:
						flags = 0x20
					case go4k.Pulse:
						flags = 0x10
					case go4k.Gate:
						flags = 0x04
					case go4k.Sample:
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
					address := unit.Parameters["unit"]*16 + 24 + unit.Parameters["port"]
					if unit.Parameters["voice"] != -1 {
						address += 0x4000 + 16 + unit.Parameters["voice"]*1024 // global send, address is computed relative to synthworkspace
					}
					if unit.Parameters["pop"] == 1 {
						address += 0x8000
					}
					values = append(values, byte(address&255), byte(address>>8))
				}
			} else {
				return nil, fmt.Errorf("Unknown unit type: %v (at instrument %v, unit %v)", unit.Type, insid, unitid)
			}
		}
		commands = append(commands, byte(C.su_advance_id))
		totalVoices += instr.NumVoices
		for k := 0; k < instr.NumVoices-1; k++ {
			polyphonyBitmask = (polyphonyBitmask << 1) + 1
		}
		polyphonyBitmask <<= 1
	}
	if totalVoices > 32 {
		return nil, errors.New("Sointu does not support more than 32 concurrent voices")
	}
	if len(commands) > 2048 { // TODO: 2048 could probably be pulled automatically from cgo
		return nil, errors.New("The patch would result in more than 2048 commands")
	}
	if len(values) > 16384 { // TODO: 16384 could probably be pulled automatically from cgo
		return nil, errors.New("The patch would result in more than 16384 values")
	}
	for i := range commands {
		s.Commands[i] = (C.uchar)(commands[i])
	}
	for i := range values {
		s.Values[i] = (C.uchar)(values[i])
	}
	s.NumVoices = C.uint(totalVoices)
	s.Polyphony = C.uint(polyphonyBitmask)
	s.RandSeed = 1
	return s, nil
}

func (s *C.Synth) Trigger(voice int, note byte) {
	s.SynthWrk.Voices[voice] = C.Voice{}
	s.SynthWrk.Voices[voice].Note = C.int(note)
}

func (s *C.Synth) Release(voice int) {
	s.SynthWrk.Voices[voice].Release = 1
}
