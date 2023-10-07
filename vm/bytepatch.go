package vm

import (
	"errors"
	"fmt"

	"github.com/vsariola/sointu"
)

// BytePatch is the compiler Sointu VM bytecode & data (delay times, sample
// offsets) ready to interpret or from which the ASM/WASM code can be generate.
//
// PolyphonyBitmask is a rather peculiar bitmask used by Sointu VM to store the
// information about which voices use which instruments: bit MAXVOICES - n - 1
// corresponds to voice n. If the bit 1, the next voice uses the same
// instrument. If the bit 0, the next voice uses different instrument. For
// example, if first instrument has 3 voices, second instrument has 2 voices,
// and third instrument four voices, the PolyphonyBitmask is:
//
// (MSB) 110101110 (LSB)
type BytePatch struct {
	Commands         []byte
	Values           []byte
	DelayTimes       []uint16
	SampleOffsets    []SampleOffset
	PolyphonyBitmask uint32
	NumVoices        uint32
}

type SampleOffset struct {
	Start      uint32
	LoopStart  uint16
	LoopLength uint16
}

type bytePatchBuilder struct {
	sampleOffsetMap map[SampleOffset]int
	globalAddrs     map[int]uint16
	globalFixups    map[int]([]int)
	localAddrs      map[int]uint16
	localFixups     map[int]([]int)
	voiceNo         int
	delayIndices    [][]int
	unitNo          int
	BytePatch
}

func Encode(patch sointu.Patch, featureSet FeatureSet, bpm int) (*BytePatch, error) {
	if patch.NumVoices() > 32 {
		return nil, fmt.Errorf("Sointu does not support more than 32 concurrent voices; patch uses %v", patch.NumVoices())
	}
	b := newBytePatchBuilder(patch, bpm)
	for instrIndex, instr := range patch {
		if instr.NumVoices < 1 {
			return nil, errors.New("Each instrument must have at least 1 voice")
		}
		for unitIndex, unit := range instr.Units {
			if unit.Type == "" { // empty units are just ignored & skipped
				continue
			}
			opcode, ok := featureSet.Opcode(unit.Type)
			if !ok {
				return nil, fmt.Errorf(`VM is not configured to support unit type "%v"`, unit.Type)
			}
			if unit.ID != 0 {
				b.idLabel(unit.ID)
			}
			p := unit.Parameters
			switch unit.Type {
			case "oscillator":
				color := p["color"]
				if unit.Parameters["type"] == 4 {
					color = b.getSampleIndex(unit)
					if color > 255 {
						return nil, errors.New("Patch uses over 256 samples")
					}
				}
				flags := 0
				switch p["type"] {
				case sointu.Sine:
					flags = 0x40
				case sointu.Trisaw:
					flags = 0x20
				case sointu.Pulse:
					flags = 0x10
				case sointu.Gate:
					flags = 0x04
				case sointu.Sample:
					flags = 0x80
				}
				if p["lfo"] == 1 {
					flags += 0x08
				}
				flags += p["unison"]
				b.cmd(opcode + p["stereo"])
				b.vals(p["transpose"], p["detune"], p["phase"], color, p["shape"], p["gain"], flags)
			case "delay":
				count := len(unit.VarArgs)
				if unit.Parameters["stereo"] == 1 {
					count /= 2
				}
				if count == 0 {
					continue // skip encoding delays without any delay lines
				}
				countTrack := count*2 - 1 + (unit.Parameters["notetracking"] & 1) // 1 means no note tracking and 1 delay, 2 means notetracking with 1 delay, 3 means no note tracking and 2 delays etc.
				b.cmd(opcode + p["stereo"])
				b.defaultVals(unit)
				b.vals(b.delayIndices[instrIndex][unitIndex], countTrack)
			case "aux", "in":
				b.cmd(opcode + p["stereo"])
				b.defaultVals(unit)
				b.vals(unit.Parameters["channel"])
			case "filter":
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
				b.cmd(opcode + p["stereo"])
				b.defaultVals(unit)
				b.vals(flags)
			case "send":
				targetID := unit.Parameters["target"]
				targetInstrIndex, _, err := patch.FindSendTarget(targetID)
				targetVoice := unit.Parameters["voice"]
				addr := unit.Parameters["port"] & 7
				if err == nil {
					// local send is only possible if targetVoice is "auto" (0) and
					// the targeted unit is in the same instrument as send
					if targetInstrIndex == instrIndex && targetVoice == 0 {
						if unit.Parameters["sendpop"] == 1 {
							addr += 0x8
						}
						b.cmd(opcode + p["stereo"])
						b.defaultVals(unit)
						b.localIDRef(targetID, addr)
					} else {
						addr += 0x8000
						voiceStart := 0
						voiceEnd := patch[targetInstrIndex].NumVoices
						if targetVoice > 0 { // "all" (0) means for global send that it targets all voices of that instrument
							voiceStart = targetVoice - 1
							voiceEnd = targetVoice
						}
						addr += voiceStart * 0x400
						for i := voiceStart; i < voiceEnd; i++ {
							b.cmd(opcode + p["stereo"])
							b.defaultVals(unit)
							if i == voiceEnd-1 && unit.Parameters["sendpop"] == 1 {
								addr += 0x8 // when making multi unit send, only the last one should have POP bit set if popping
							}
							b.globalIDRef(targetID, addr)
							addr += 0x400
						}
					}
				} else {
					// if no target will be found, the send will trash some of
					// the last values of the last port of the last voice, which
					// is unlikely to cause issues. We still honor the POP bit.
					addr = 0xFFF7
					if unit.Parameters["sendpop"] == 1 {
						addr |= 0x8
					}
					b.cmd(opcode + p["stereo"])
					b.defaultVals(unit)
					b.Values = append(b.Values, byte(addr&255), byte(addr>>8))
				}
			default:
				b.cmd(opcode + p["stereo"])
				b.defaultVals(unit)
			}
			if b.unitNo > 63 {
				return nil, fmt.Errorf(`Instrument %v has over 63 units`, instrIndex)
			}
		}
		b.cmdFinish(instr)
	}
	return &b.BytePatch, nil
}

func newBytePatchBuilder(patch sointu.Patch, bpm int) *bytePatchBuilder {
	var polyphonyBitmask uint32 = 0
	for _, instr := range patch {
		for j := 0; j < instr.NumVoices-1; j++ {
			polyphonyBitmask = (polyphonyBitmask << 1) + 1 // for each instrument, NumVoices - 1 bits are ones
		}
		polyphonyBitmask <<= 1 // ...and the last bit is zero, to denote "change instrument"
	}
	delayTimesInt, delayIndices := constructDelayTimeTable(patch, bpm)
	delayTimesU16 := make([]uint16, len(delayTimesInt))
	for i, d := range delayTimesInt {
		delayTimesU16[i] = uint16(d)
	}
	c := bytePatchBuilder{
		BytePatch:       BytePatch{PolyphonyBitmask: polyphonyBitmask, NumVoices: uint32(patch.NumVoices()), DelayTimes: delayTimesU16},
		sampleOffsetMap: map[SampleOffset]int{},
		globalAddrs:     map[int]uint16{},
		globalFixups:    map[int]([]int){},
		localAddrs:      map[int]uint16{},
		localFixups:     map[int]([]int){},
		delayIndices:    delayIndices}
	return &c
}

// cmd adds a command to the bytecode, and increments the unit number
func (b *bytePatchBuilder) cmd(opcode int) {
	b.Commands = append(b.Commands, byte(opcode))
	b.unitNo++
}

// cmdFinish adds a command to the bytecode that marks the end of an instrument, resets the unit number and increments the voice number
// local addresses are forgotten when instrument ends
func (b *bytePatchBuilder) cmdFinish(instr sointu.Instrument) {
	b.Commands = append(b.Commands, 0)
	b.unitNo = 0
	b.voiceNo += instr.NumVoices
	b.localAddrs = map[int]uint16{}
	b.localFixups = map[int]([]int){}
}

// vals appends values to the value stream
func (b *bytePatchBuilder) vals(values ...int) {
	for _, v := range values {
		b.Values = append(b.Values, byte(v))
	}
}

// defaultVals appends the values to the value stream for all parameters that can be modulated and set
func (b *bytePatchBuilder) defaultVals(unit sointu.Unit) {
	for _, v := range sointu.UnitTypes[unit.Type] {
		if v.CanModulate && v.CanSet {
			b.Values = append(b.Values, byte(unit.Parameters[v.Name]))
		}
	}
}

// localIDRef adds a reference to a local id label to the value stream; if the targeted ID has not been seen yet, it is added to the fixup list
func (b *bytePatchBuilder) localIDRef(id int, addr int) {
	if v, ok := b.localAddrs[id]; ok {
		addr += int(v)
	} else {
		b.localFixups[id] = append(b.localFixups[id], len(b.Values))
	}
	b.Values = append(b.Values, byte(addr&255), byte(addr>>8))
}

// globalIDRef adds a reference to a global id label to the value stream; if the targeted ID has not been seen yet, it is added to the fixup list
func (b *bytePatchBuilder) globalIDRef(id int, addr int) {
	if v, ok := b.globalAddrs[id]; ok {
		addr += int(v)
	} else {
		b.globalFixups[id] = append(b.globalFixups[id], len(b.Values))
	}
	b.Values = append(b.Values, byte(addr&255), byte(addr>>8))
}

// idLabel adds a label to the value stream for the given id; all earlier references to the id are fixed up
func (b *bytePatchBuilder) idLabel(id int) {
	localAddr := uint16((b.unitNo + 1) << 4)
	b.fixUp(b.localFixups[id], localAddr)
	b.localFixups[id] = nil
	b.localAddrs[id] = localAddr
	globalAddr := localAddr + 16 + uint16(b.voiceNo)*1024
	b.fixUp(b.globalFixups[id], globalAddr)
	b.globalFixups[id] = nil
	b.globalAddrs[id] = globalAddr
}

// fixUp fixes up the references to the given id with the given delta
func (b *bytePatchBuilder) fixUp(positions []int, delta uint16) {
	for _, pos := range positions {
		orig := (uint16(b.Values[pos+1]) << 8) + uint16(b.Values[pos])
		new := orig + delta
		b.Values[pos] = byte(new & 255)
		b.Values[pos+1] = byte(new >> 8)
	}
}

// getSampleIndex returns the index of the sample in the sample offset table; if the sample has not been seen yet, it is added to the table
func (b *bytePatchBuilder) getSampleIndex(unit sointu.Unit) int {
	s := SampleOffset{Start: uint32(unit.Parameters["samplestart"]), LoopStart: uint16(unit.Parameters["loopstart"]), LoopLength: uint16(unit.Parameters["looplength"])}
	if s.LoopLength == 0 {
		// hacky quick fix: looplength 0 causes div by zero so avoid crashing
		s.LoopLength = 1
	}
	index, ok := b.sampleOffsetMap[s]
	if !ok {
		index = len(b.SampleOffsets)
		b.sampleOffsetMap[s] = index
		b.SampleOffsets = append(b.SampleOffsets, s)
	}
	return index
}
