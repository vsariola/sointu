package vm

import (
	"errors"
	"fmt"

	"github.com/vsariola/sointu"
)

type (
	// Bytecode is the Sointu VM bytecode & data (delay times, sample offsets)
	// which is executed by the synthesizer. It is generated from a Sointu patch.
	Bytecode struct {
		// Opcodes is the bytecode, which is a sequence of opcode bytes, one
		// per unit in the patch. A byte of 0 denotes the end of an instrument,
		// at which point if that instrument has more than one voice, the
		// opcodes are repeated for each voice.
		Opcodes []byte

		// Operands are the operands of the opcodes. When executing the
		// bytecodes, every opcode reads 0 or more operands from it and advances
		// in the sequence.
		Operands []byte

		// DelayTimes is a table of delay times in samples. The delay times are
		// used by the delay units in the patch. The delay unit only stores
		// index and count of delay lines, and the delay times are looked up
		// from this table. This way multiple reverb units do not have to repeat
		// the same delay times.
		DelayTimes []uint16

		// SampleOffsets is a table of sample offsets, which tell where to find
		// a particular sample in the sample data loaded from gm.dls. The sample
		// offsets are used by the oscillator units that are configured to use
		// samples. The unit only stores the index pointing to this table.
		SampleOffsets []SampleOffset

		// PolyphonyBitmask is a rather peculiar bitmask used by Sointu VM to store
		// the information about which voices use which instruments: bit MAXVOICES -
		// n - 1 corresponds to voice n. If the bit 1, the next voice uses the same
		// instrument. If the bit 0, the next voice uses different instrument. For
		// example, if first instrument has 3 voices, second instrument has 2
		// voices, and third instrument four voices, the PolyphonyBitmask is: (MSB)
		// 110101110 (LSB)
		PolyphonyBitmask uint32

		// NumVoices is the total number of voices in the patch
		NumVoices uint32
	}

	// SampleOffset is an entry in the sample offset table
	SampleOffset struct {
		Start      uint32 // start offset in words (1 word = 2 bytes)
		LoopStart  uint16 // loop start offset in words, relative to Start
		LoopLength uint16 // loop length in words
	}
)

type bytecodeBuilder struct {
	sampleOffsetMap map[SampleOffset]int
	globalAddrs     map[int]uint16
	globalFixups    map[int]([]int)
	localAddrs      map[int]uint16
	localFixups     map[int]([]int)
	voiceNo         int
	delayIndices    [][]int
	unitNo          int
	Bytecode
}

func NewBytecode(patch sointu.Patch, featureSet FeatureSet, bpm int) (*Bytecode, error) {
	if patch.NumVoices() > 32 {
		return nil, fmt.Errorf("Sointu does not support more than 32 concurrent voices; patch uses %v", patch.NumVoices())
	}
	b := newBytecodeBuilder(patch, bpm)
	for instrIndex, instr := range patch {
		if instr.NumVoices < 1 {
			return nil, errors.New("Each instrument must have at least 1 voice")
		}
		for unitIndex, unit := range instr.Units {
			if unit.Type == "" || unit.Disabled { // empty units are just ignored & skipped
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
				b.op(opcode + p["stereo"])
				b.operand(p["transpose"], p["detune"], p["phase"], color, p["shape"], p["gain"], flags)
			case "delay":
				count := len(unit.VarArgs)
				if unit.Parameters["stereo"] == 1 {
					count /= 2
				}
				if count == 0 {
					continue // skip encoding delays without any delay lines
				}
				countTrack := count*2 - 1 + (unit.Parameters["notetracking"] & 1) // 1 means no note tracking and 1 delay, 2 means notetracking with 1 delay, 3 means no note tracking and 2 delays etc.
				b.op(opcode + p["stereo"])
				b.defOperands(unit)
				b.operand(b.delayIndices[instrIndex][unitIndex], countTrack)
			case "aux", "in":
				b.op(opcode + p["stereo"])
				b.defOperands(unit)
				b.operand(unit.Parameters["channel"])
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
				if unit.Parameters["bandpass"] == -1 {
					flags += 0x08
				}
				if unit.Parameters["highpass"] == -1 {
					flags += 0x04
				}
				b.op(opcode + p["stereo"])
				b.defOperands(unit)
				b.operand(flags)
			case "send":
				targetID := unit.Parameters["target"]
				targetInstrIndex, _, err := patch.FindUnit(targetID)
				targetVoice := unit.Parameters["voice"]
				addr := unit.Parameters["port"] & 7
				if err == nil {
					// local send is only possible if targetVoice is "auto" (0) and
					// the targeted unit is in the same instrument as send
					if targetInstrIndex == instrIndex && targetVoice == 0 {
						if unit.Parameters["sendpop"] == 1 {
							addr += 0x8
						}
						b.op(opcode + p["stereo"])
						b.defOperands(unit)
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
							b.op(opcode + p["stereo"])
							b.defOperands(unit)
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
					b.op(opcode + p["stereo"])
					b.defOperands(unit)
					b.Operands = append(b.Operands, byte(addr&255), byte(addr>>8))
				}
			default:
				b.op(opcode + p["stereo"])
				b.defOperands(unit)
			}
			if b.unitNo > 63 {
				return nil, fmt.Errorf(`Instrument %v has over 63 units`, instrIndex)
			}
		}
		b.opFinish(instr)
	}
	return &b.Bytecode, nil
}

func newBytecodeBuilder(patch sointu.Patch, bpm int) *bytecodeBuilder {
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
	c := bytecodeBuilder{
		Bytecode:        Bytecode{PolyphonyBitmask: polyphonyBitmask, NumVoices: uint32(patch.NumVoices()), DelayTimes: delayTimesU16},
		sampleOffsetMap: map[SampleOffset]int{},
		globalAddrs:     map[int]uint16{},
		globalFixups:    map[int]([]int){},
		localAddrs:      map[int]uint16{},
		localFixups:     map[int]([]int){},
		delayIndices:    delayIndices}
	return &c
}

// op adds a command to the bytecode, and increments the unit number
func (b *bytecodeBuilder) op(opcode int) {
	b.Opcodes = append(b.Opcodes, byte(opcode))
	b.unitNo++
}

// opFinish adds a command to the bytecode that marks the end of an instrument, resets the unit number and increments the voice number
// local addresses are forgotten when instrument ends
func (b *bytecodeBuilder) opFinish(instr sointu.Instrument) {
	b.Opcodes = append(b.Opcodes, 0)
	b.unitNo = 0
	b.voiceNo += instr.NumVoices
	b.localAddrs = map[int]uint16{}
	b.localFixups = map[int]([]int){}
}

// operand appends operands to the operand stream
func (b *bytecodeBuilder) operand(operands ...int) {
	for _, v := range operands {
		b.Operands = append(b.Operands, byte(v))
	}
}

// defOperands appends the operands to the stream for all parameters that can be
// modulated and set
func (b *bytecodeBuilder) defOperands(unit sointu.Unit) {
	for _, v := range sointu.UnitTypes[unit.Type] {
		if v.CanModulate && v.CanSet {
			b.Operands = append(b.Operands, byte(unit.Parameters[v.Name]))
		}
	}
}

// localIDRef adds a reference to a local id label to the value stream; if the targeted ID has not been seen yet, it is added to the fixup list
func (b *bytecodeBuilder) localIDRef(id int, addr int) {
	if v, ok := b.localAddrs[id]; ok {
		addr += int(v)
	} else {
		b.localFixups[id] = append(b.localFixups[id], len(b.Operands))
	}
	b.Operands = append(b.Operands, byte(addr&255), byte(addr>>8))
}

// globalIDRef adds a reference to a global id label to the value stream; if the targeted ID has not been seen yet, it is added to the fixup list
func (b *bytecodeBuilder) globalIDRef(id int, addr int) {
	if v, ok := b.globalAddrs[id]; ok {
		addr += int(v)
	} else {
		b.globalFixups[id] = append(b.globalFixups[id], len(b.Operands))
	}
	b.Operands = append(b.Operands, byte(addr&255), byte(addr>>8))
}

// idLabel adds a label to the value stream for the given id; all earlier references to the id are fixed up
func (b *bytecodeBuilder) idLabel(id int) {
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
func (b *bytecodeBuilder) fixUp(positions []int, delta uint16) {
	for _, pos := range positions {
		orig := (uint16(b.Operands[pos+1]) << 8) + uint16(b.Operands[pos])
		new := orig + delta
		b.Operands[pos] = byte(new & 255)
		b.Operands[pos+1] = byte(new >> 8)
	}
}

// getSampleIndex returns the index of the sample in the sample offset table; if the sample has not been seen yet, it is added to the table
func (b *bytecodeBuilder) getSampleIndex(unit sointu.Unit) int {
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
