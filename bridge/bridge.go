package bridge

import (
	"errors"
)

// #cgo CFLAGS: -I"${SRCDIR}/../include"
// #cgo LDFLAGS: "${SRCDIR}/../build/src/libsointu.a"
// #include <sointu.h>
import "C"

// SynthState contains the entire state of sointu sound engine
type Synth C.Synth // hide C.Synth, explicit cast is still possible if needed

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
// Use this version if you are not interested in time modulation. Will always
// fill the buffer.
// Parameters:
//   buffer     float32 slice to fill with rendered samples. Stereo signal, so
//              should have even length.
// Returns an error if something went wrong.
func (synth *Synth) Render(state *SynthState, buffer []float32) error {
	if len(buffer)%1 == 1 {
		return errors.New("Render writes stereo signals, so buffer should have even length")
	}
	maxSamples := len(buffer) / 2
	state.RandSeed += 1 // if you initialize with empty struct, you will get randseed 1 which is sointu default behavior
	errcode := C.su_render((*C.Synth)(synth), (*C.SynthState)(state), (*C.float)(&buffer[0]), C.int(maxSamples))
	state.RandSeed -= 1
	if errcode > 0 {
		return errors.New("Render failed")
	}
	return nil
}

// RenderTime renders until the buffer is full or the modulated time is reached, whichever
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
func (synth *Synth) RenderTime(state *SynthState, buffer []float32, maxtime int) (int, int, error) {
	if len(buffer)%1 == 1 {
		return -1, -1, errors.New("RenderTime writes stereo signals, so buffer should have even length")
	}
	samples := C.int(len(buffer) / 2)
	time := C.int(maxtime)
	state.RandSeed += 1 // if you initialize with empty struct, you will get randseed 1 which is sointu default behavior
	errcode := int(C.su_render_time((*C.Synth)(synth), (*C.SynthState)(state), (*C.float)(&buffer[0]), &samples, &time))
	state.RandSeed -= 1
	if errcode > 0 {
		return -1, -1, errors.New("RenderTime failed")
	}
	return int(samples), int(time), nil
}

func Compile(patch Patch) (*Synth, error) {
	totalVoices := 0
	commands := make([]Opcode, 0)
	values := make([]byte, 0)
	polyphonyBitmask := 0
	for _, instr := range patch {
		if len(instr.Units) > 63 {
			return nil, errors.New("An instrument can have a maximum of 63 units")
		}
		if instr.NumVoices < 1 {
			return nil, errors.New("Each instrument must have at least 1 voice")
		}
		for _, unit := range instr.Units {
			commands = append(commands, unit.Command)
			values = append(values, unit.Params...)
		}
		commands = append(commands, Advance)
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
	s := new(Synth)
	for i := range commands {
		s.Commands[i] = (C.uchar)(commands[i])
	}
	for i := range values {
		s.Values[i] = (C.uchar)(values[i])
	}
	s.NumVoices = C.uint(totalVoices)
	s.Polyphony = C.uint(polyphonyBitmask)
	return s, nil
}

func (s *SynthState) Trigger(voice int, note byte) {
	cs := (*C.SynthState)(s)
	cs.SynthWrk.Voices[voice] = C.Voice{}
	cs.SynthWrk.Voices[voice].Note = C.int(note)
}

func (s *SynthState) Release(voice int) {
	cs := (*C.SynthState)(s)
	cs.SynthWrk.Voices[voice].Release = 1
}
