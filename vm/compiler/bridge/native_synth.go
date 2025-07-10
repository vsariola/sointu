package bridge

// #cgo CFLAGS: -I"${SRCDIR}/../../../build/"
// #cgo LDFLAGS: "${SRCDIR}/../../../build/libsointu.a"
// #include <sointu.h>
import "C"
import (
	"errors"
	"fmt"
	"strings"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
)

type NativeSynther struct {
}

type NativeSynth C.Synth

func (s NativeSynther) Name() string { return "Native" }

func (s NativeSynther) Synth(patch sointu.Patch, bpm int) (sointu.Synth, error) {
	synth, err := Synth(patch, bpm)
	return synth, err
}

func Synth(patch sointu.Patch, bpm int) (*NativeSynth, error) {
	s := new(C.Synth)
	if n := patch.NumDelayLines(); n > 128 {
		return nil, fmt.Errorf("native bridge has currently a hard limit of 128 delaylines; patch uses %v", n)
	}
	comPatch, err := vm.NewBytecode(patch, vm.AllFeatures{}, bpm)
	if err != nil {
		return nil, fmt.Errorf("error compiling patch: %v", err)
	}
	if len(comPatch.Opcodes) > 2048 { // TODO: 2048 could probably be pulled automatically from cgo
		return nil, errors.New("bridge supports at most 2048 opcodes; the compiled patch has more")
	}
	if len(comPatch.Operands) > 16384 { // TODO: 16384 could probably be pulled automatically from cgo
		return nil, errors.New("bridge supports at most 16384 operands; the compiled patch has more")
	}
	// if the patch is empty, we still need to initialize the synth with a single opcode
	if len(comPatch.Opcodes) == 0 {
		s.Opcodes[0] = 0
		s.NumVoices = 1
		s.Polyphony = 0
		return (*NativeSynth)(s), nil
	}
	for i, v := range comPatch.Opcodes {
		s.Opcodes[i] = (C.uchar)(v)
	}
	for i, v := range comPatch.Operands {
		s.Operands[i] = (C.uchar)(v)
	}
	for i, v := range comPatch.DelayTimes {
		s.DelayTimes[i] = (C.ushort)(v)
	}
	for i, v := range comPatch.SampleOffsets {
		s.SampleOffsets[i].Start = (C.uint)(v.Start)
		s.SampleOffsets[i].LoopStart = (C.ushort)(v.LoopStart)
		s.SampleOffsets[i].LoopLength = (C.ushort)(v.LoopLength)
	}
	s.NumVoices = C.uint(comPatch.NumVoices)
	s.Polyphony = C.uint(comPatch.PolyphonyBitmask)
	s.RandSeed = 1
	return (*NativeSynth)(s), nil
}

// Render renders until the buffer is full or the modulated time is reached, whichever
// happens first.
// Parameters:
//
//	buffer     float32 slice to fill with rendered samples. Stereo signal, so
//	           should have even length.
//	maxtime    how long nominal time to render in samples. Speed unit might modulate time
//	           so the actual number of samples rendered depends on the modulation and if
//	           buffer is full before maxtime is reached.
//
// Returns a tuple (int, int, error), consisting of:
//
//	samples    number of samples rendered in the buffer
//	time       how much the time advanced
//	error      potential error
//
// In practice, if nsamples = len(buffer)/2, then time <= maxtime. If maxtime was reached
// first, then nsamples <= len(buffer)/2 and time >= maxtime. Note that it could happen that
// time > maxtime, as it is modulated and the time could advance by 2 or more, so the loop
// exit condition would fire when the time is already past maxtime.
// Under no conditions, nsamples >= len(buffer)/2 i.e. guaranteed to never overwrite the buffer.
func (bridgesynth *NativeSynth) Render(buffer sointu.AudioBuffer, maxtime int) (int, int, error) {
	synth := (*C.Synth)(bridgesynth)
	// TODO: syncBuffer is not getting passed to cgo; do we want to even try to support the syncing with the native bridge
	if len(buffer)%1 == 1 {
		return -1, -1, errors.New("RenderTime writes stereo signals, so buffer should have even length")
	}
	samples := C.int(len(buffer))
	time := C.int(maxtime)
	errcode := int(C.su_render(synth, (*C.float)(&buffer[0][0]), &samples, &time))
	if errcode > 0 {
		return int(samples), int(time), &RenderError{errcode: errcode}
	}
	return int(samples), int(time), nil
}

// Trigger is part of C.Synths' implementation of sointu.Synth interface
func (bridgesynth *NativeSynth) Trigger(voice int, note byte) {
	s := (*C.Synth)(bridgesynth)
	if voice < 0 || voice >= len(s.SynthWrk.Voices) {
		return
	}
	s.SynthWrk.Voices[voice] = C.Voice{}
	s.SynthWrk.Voices[voice].Note = C.int(note)
	s.SynthWrk.Voices[voice].Sustain = 1
}

// Release is part of C.Synths' implementation of sointu.Synth interface
func (bridgesynth *NativeSynth) Release(voice int) {
	s := (*C.Synth)(bridgesynth)
	if voice < 0 || voice >= len(s.SynthWrk.Voices) {
		return
	}
	s.SynthWrk.Voices[voice].Sustain = 0
}

// Update
func (bridgesynth *NativeSynth) Update(patch sointu.Patch, bpm int) error {
	s := (*C.Synth)(bridgesynth)
	if n := patch.NumDelayLines(); n > 128 {
		return fmt.Errorf("native bridge has currently a hard limit of 128 delaylines; patch uses %v", n)
	}
	comPatch, err := vm.NewBytecode(patch, vm.AllFeatures{}, bpm)
	if err != nil {
		return fmt.Errorf("error compiling patch: %v", err)
	}
	if len(comPatch.Opcodes) > 2048 { // TODO: 2048 could probably be pulled automatically from cgo
		return errors.New("bridge supports at most 2048 opcodes; the compiled patch has more")
	}
	if len(comPatch.Operands) > 16384 { // TODO: 16384 could probably be pulled automatically from cgo
		return errors.New("bridge supports at most 16384 operands; the compiled patch has more")
	}
	// if the patch is empty, we still need to initialize the synth with a single opcode
	if len(comPatch.Opcodes) == 0 {
		s.Opcodes[0] = 0
		s.NumVoices = 1
		s.Polyphony = 0
		return nil
	}
	needsRefresh := false
	for i, v := range comPatch.Opcodes {
		if cmdChar := (C.uchar)(v); s.Opcodes[i] != cmdChar {
			s.Opcodes[i] = cmdChar
			needsRefresh = true // if any of the opcodes change, we retrigger all units
		}
	}
	for i, v := range comPatch.Operands {
		s.Operands[i] = (C.uchar)(v)
	}
	for i, v := range comPatch.DelayTimes {
		s.DelayTimes[i] = (C.ushort)(v)
	}
	for i, v := range comPatch.SampleOffsets {
		s.SampleOffsets[i].Start = (C.uint)(v.Start)
		s.SampleOffsets[i].LoopStart = (C.ushort)(v.LoopStart)
		s.SampleOffsets[i].LoopLength = (C.ushort)(v.LoopLength)
	}
	s.NumVoices = C.uint(comPatch.NumVoices)
	s.Polyphony = C.uint(comPatch.PolyphonyBitmask)
	if needsRefresh {
		for i := range s.SynthWrk.Voices {
			// if any of the opcodes change, we retrigger all units
			// note that we don't change the notes or release states, just the units
			for j := range s.SynthWrk.Voices[i].Units {
				s.SynthWrk.Voices[i].Units[j] = C.Unit{}
			}
		}
	}
	return nil
}

// Render error stores the exact errorcode, which is actually just the x87 FPU flags,
// with only the critical failure flags masked. Useful if you are interested exactly
// what went wrong with the patch.
type RenderError struct {
	errcode int
}

func (e *RenderError) Error() string {
	var reasons []string
	if e.errcode&0x40 != 0 {
		reasons = append(reasons, "FPU stack over/underflow")
	}
	if e.errcode&0x04 != 0 {
		reasons = append(reasons, "FPU divide by zero")
	}
	if e.errcode&0x01 != 0 {
		reasons = append(reasons, "FPU invalid operation")
	}
	if e.errcode&0x3800 != 0 {
		reasons = append(reasons, "FPU stack push/pops are not balanced")
	}
	return "RenderError: " + strings.Join(reasons, ", ")
}
