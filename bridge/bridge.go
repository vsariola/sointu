package bridge

import (
	"errors"
	"math"
)

// #cgo CFLAGS: -I"${SRCDIR}/../include"
// #cgo LDFLAGS: "${SRCDIR}/../build/src/libsointu.a"
// #include <sointu.h>
import "C"

// SynthState contains the entire state of sointu sound engine
type SynthState C.SynthState // hide C.Synthstate, explicit cast is still possible if needed

// Opcode is a single byte, representing the virtual machine commands used in Sointu
type Opcode byte

// Unit includes command (filter, oscillator, envelope etc.) and its parameters
type Unit struct {
	Command Opcode
	Params  []byte
}

// Instrument includes a list of units consisting of the instrument, and the number of polyphonic voices for this instrument
type Instrument struct {
	NumVoices int
	Units     []Unit
}

type Patch []Instrument

func (p Patch) TotalVoices() int {
	ret := 0
	for _, i := range p {
		ret += i.NumVoices
	}
	return ret
}

var ( // cannot be const as the rhs are not known at compile-time
	Add      = Opcode(C.su_add_id)
	Addp     = Opcode(C.su_addp_id)
	Pop      = Opcode(C.su_pop_id)
	Loadnote = Opcode(C.su_loadnote_id)
	Mul      = Opcode(C.su_mul_id)
	Mulp     = Opcode(C.su_mulp_id)
	Push     = Opcode(C.su_push_id)
	Xch      = Opcode(C.su_xch_id)
	Distort  = Opcode(C.su_distort_id)
	Hold     = Opcode(C.su_hold_id)
	Crush    = Opcode(C.su_crush_id)
	Gain     = Opcode(C.su_gain_id)
	Invgain  = Opcode(C.su_invgain_id)
	Filter   = Opcode(C.su_filter_id)
	Clip     = Opcode(C.su_clip_id)
	Pan      = Opcode(C.su_pan_id)
	Delay    = Opcode(C.su_delay_id)
	Compres  = Opcode(C.su_compres_id)
	Advance  = Opcode(C.su_advance_id)
	Speed    = Opcode(C.su_speed_id)
	Out      = Opcode(C.su_out_id)
	Outaux   = Opcode(C.su_outaux_id)
	Aux      = Opcode(C.su_aux_id)
	Send     = Opcode(C.su_send_id)
	Envelope = Opcode(C.su_envelope_id)
	Noise    = Opcode(C.su_noise_id)
	Oscillat = Opcode(C.su_oscillat_id)
	Loadval  = Opcode(C.su_loadval_id)
	Receive  = Opcode(C.su_receive_id)
	In       = Opcode(C.su_in_id)
)

// Stereo returns the stereo version of any (mono or stereo) opcode
func (o Opcode) Stereo() Opcode {
	return Opcode(byte(o) | 1) // set lowest bit
}

// Mono returns the mono version of any (mono or stereo) opcode
func (o Opcode) Mono() Opcode {
	return Opcode(byte(o) & 0xFE) // clear lowest bit
}

// Render tries to fill the buffer with samples rendered by Sointu.
// Parameters:
//   buffer     float32 slice to fill with rendered samples. Stereo signal, so
//              should have even length.
// Returns a tuple (int, bool, error), consisting of the number samples
// rendered (len(buffer)/2 in the case where buffer was filled, less or equal
// if row end was reached before buffer was full), and bool indicating if row
// has ended
func (s *SynthState) Render(buffer []float32) (int, bool, error) {
	if len(buffer)%1 == 1 {
		return -1, false, errors.New("Render writes stereo signals, so buffer should have even length")
	}
	maxSamples := len(buffer) / 2
	retval := int(C.su_render_samples((*C.SynthState)(s), C.int(maxSamples), (*C.float)(&buffer[0])))
	if retval < 0 {
		return maxSamples, false, nil
	} else if retval == 0 {
		return maxSamples, true, nil
	} else {
		return maxSamples - retval, true, nil
	}
}

func (s *SynthState) SetPatch(patch Patch) error {
	totalVoices := 0
	commands := make([]Opcode, 0)
	values := make([]byte, 0)
	for _, instr := range patch {
		if len(instr.Units) > 63 {
			return errors.New("An instrument can have a maximum of 63 units")
		}
		if instr.NumVoices < 1 {
			return errors.New("Each instrument must have at least 1 voice")
		}
		for _, unit := range instr.Units {
			commands = append(commands, unit.Command)
			values = append(values, unit.Params...)
		}
		commands = append(commands, Advance)
		totalVoices += instr.NumVoices
	}
	if totalVoices > 32 {
		return errors.New("Sointu does not support more than 32 concurrent voices")
	}
	if len(commands) > 2048 { // TODO: 2048 could probably be pulled automatically from cgo
		return errors.New("The patch would result in more than 2048 commands")
	}
	if len(values) > 16384 { // TODO: 16384 could probably be pulled automatically from cgo
		return errors.New("The patch would result in more than 16384 values")
	}
	cs := (*C.SynthState)(s)
	for i := range commands {
		cs.Commands[i] = (C.uchar)(commands[i])
	}
	for i := range values {
		cs.Values[i] = (C.uchar)(values[i])
	}
	cs.NumVoices = C.uint(totalVoices)
	return nil
}

func (s *SynthState) Trigger(voice int, note byte) {
	cs := (*C.SynthState)(s)
	cs.Synth.Voices[voice] = C.Voice{}
	cs.Synth.Voices[voice].Note = C.int(note)
}

func (s *SynthState) Release(voice int) {
	cs := (*C.SynthState)(s)
	cs.Synth.Voices[voice].Release = 1
}

func (s *SynthState) SetSamplesPerRow(spr int) {
	s.SamplesPerRow = C.uint(spr)
}

func NewSynthState() *SynthState {
	s := new(SynthState)
	s.RandSeed = 1
	// The default behaviour will be to have rows/beats disabled i.e.
	// fill the whole buffer every call. This is a lot better default
	// behaviour than leaving this 0 (Render would never render anything)
	s.SamplesPerRow = math.MaxInt32
	return s
}
