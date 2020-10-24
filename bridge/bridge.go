package bridge

import "fmt"
import "unsafe"
import "math"

// #cgo CFLAGS: -I"${SRCDIR}/../include"
// #cgo LDFLAGS: "${SRCDIR}/../build/src/libsointu.a"
// #include <sointu.h>
import "C"

type SynthState = C.SynthState

type Opcode byte

var ( // cannot be const as the rhs are not known at compile-time
  Add = Opcode(C.su_add_id)
  Addp = Opcode(C.su_addp_id)
  Pop = Opcode(C.su_pop_id)
  Loadnote = Opcode(C.su_loadnote_id)
  Mul = Opcode(C.su_mul_id)
  Mulp = Opcode(C.su_mulp_id)
  Push = Opcode(C.su_push_id)
  Xch = Opcode(C.su_xch_id)
  Distort = Opcode(C.su_distort_id)
  Hold = Opcode(C.su_hold_id)
  Crush = Opcode(C.su_crush_id)
  Gain = Opcode(C.su_gain_id)
  Invgain = Opcode(C.su_invgain_id)
  Filter = Opcode(C.su_filter_id)
  Clip = Opcode(C.su_clip_id)
  Pan = Opcode(C.su_pan_id)
  Delay = Opcode(C.su_delay_id)
  Compres = Opcode(C.su_compres_id)
  Advance = Opcode(C.su_advance_id)
  Speed = Opcode(C.su_speed_id)
  Out = Opcode(C.su_out_id)
  Outaux = Opcode(C.su_outaux_id)
  Aux = Opcode(C.su_aux_id)
  Send = Opcode(C.su_send_id)
  Envelope = Opcode(C.su_envelope_id)
  Noise = Opcode(C.su_noise_id)
  Oscillat = Opcode(C.su_oscillat_id)
  Loadval = Opcode(C.su_loadval_id)
  Receive = Opcode(C.su_receive_id)
  In = Opcode(C.su_in_id)
)

func (o Opcode) Stereo() Opcode {
  return Opcode(byte(o) | 1) // set lowest bit
}

func (o Opcode) Mono() Opcode {
  return Opcode(byte(o) & 0xFE) // clear lowest bit
}

func (s *SynthState) Render(buffer []float32) int {
  fmt.Printf("Calling Render...\n")
  var ret = C.su_render_samples(s, C.int(len(buffer))/2, (*C.float)(&buffer[0]))
  fmt.Printf("Returning from Render...\n")
  return int(ret)
}

func (s *SynthState) SetCommands(c [2048]Opcode) {
  pk := *((*[2048]C.uchar)(unsafe.Pointer(&c)))
  s.Commands = pk
}

func (s *SynthState) SetValues(c [16384]byte) {
  pk := *((*[16384]C.uchar)(unsafe.Pointer(&c)))
  s.Values = pk
}

func (s *SynthState) Trigger(voice int,note int) {
  fmt.Printf("Calling Trigger...\n")
  s.Synth.Voices[voice] = C.Voice{}
  s.Synth.Voices[voice].Note = C.int(note)
  fmt.Printf("Returning from Trigger...\n")
}

func (s *SynthState) Release(voice int) {
  fmt.Printf("Calling Release...\n")
  s.Synth.Voices[voice].Release = 1
  fmt.Printf("Returning from Release...\n")
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
