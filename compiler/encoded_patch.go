package compiler

import (
	"errors"
	"fmt"

	"github.com/vsariola/sointu"
)

type EncodedPatch struct {
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

func Encode(patch sointu.Patch, featureSet FeatureSet) (*EncodedPatch, error) {
	var c EncodedPatch
	sampleOffsetMap := map[SampleOffset]int{}
	globalAddrs := map[int]uint16{}
	globalFixups := map[int]([]int){}
	voiceNo := 0
	for instrIndex, instr := range patch {
		if len(instr.Units) > 63 {
			return nil, errors.New("An instrument can have a maximum of 63 units")
		}
		if instr.NumVoices < 1 {
			return nil, errors.New("Each instrument must have at least 1 voice")
		}
		localAddrs := map[int]uint16{}
		localFixups := map[int]([]int){}
		localUnitNo := 0
		for _, unit := range instr.Units {
			if unit.Type == "" { // empty units are just ignored & skipped
				continue
			}
			if unit.Type == "oscillator" && unit.Parameters["type"] == 4 {
				s := SampleOffset{Start: uint32(unit.Parameters["samplestart"]), LoopStart: uint16(unit.Parameters["loopstart"]), LoopLength: uint16(unit.Parameters["looplength"])}
				if s.LoopLength == 0 {
					// hacky quick fix: looplength 0 causes div by zero so avoid crashing
					s.LoopLength = 1
				}
				index, ok := sampleOffsetMap[s]
				if !ok {
					index = len(c.SampleOffsets)
					sampleOffsetMap[s] = index
					c.SampleOffsets = append(c.SampleOffsets, s)
				}
				unit.Parameters["color"] = index
			}
			if unit.Type == "delay" {
				unit.Parameters["delay"] = len(c.DelayTimes)
				if unit.Parameters["stereo"] == 1 {
					unit.Parameters["count"] = len(unit.VarArgs) / 2
				} else {
					unit.Parameters["count"] = len(unit.VarArgs)
				}
				for _, v := range unit.VarArgs {
					c.DelayTimes = append(c.DelayTimes, uint16(v))
				}
			}
			opcode, ok := featureSet.Opcode(unit.Type)
			if !ok {
				return nil, fmt.Errorf(`the targeted virtual machine is not configured to support unit type "%v"`, unit.Type)
			}
			var values []byte
			for _, v := range sointu.UnitTypes[unit.Type] {
				if v.CanModulate && v.CanSet {
					values = append(values, byte(unit.Parameters[v.Name]))
				}
			}
			if unit.Type == "aux" {
				values = append(values, byte(unit.Parameters["channel"]))
			} else if unit.Type == "in" {
				values = append(values, byte(unit.Parameters["channel"]))
			} else if unit.Type == "oscillator" {
				flags := 0
				switch unit.Parameters["type"] {
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
				targetID := unit.Parameters["target"]
				targetInstrIndex, _, err := patch.FindSendTarget(targetID)
				targetVoice := unit.Parameters["voice"]
				var addr uint16 = uint16(unit.Parameters["port"]) & 7
				if unit.Parameters["sendpop"] == 1 {
					addr += 0x8
				}

				if err == nil {
					// local send is only possible if targetVoice is "auto" (0) and
					// the targeted unit is in the same instrument as send
					if targetInstrIndex == instrIndex && targetVoice == 0 {
						if v, ok := localAddrs[targetID]; ok {
							addr += v
						} else {
							localFixups[targetID] = append(localFixups[targetID], len(c.Values)+len(values))
						}
					} else {
						addr += 0x8000
						if targetVoice > 0 { // "auto" (0) means for global send that it targets voice 0 of that instrument
							addr += uint16((targetVoice - 1) * 0x400)
						}
						if v, ok := globalAddrs[targetID]; ok {
							addr += v
						} else {
							globalFixups[targetID] = append(globalFixups[targetID], len(c.Values)+len(values))
						}
					}
				} else {
					// if no target will be found, the send will trash some of
					// the last values of the last port of the last voice, which
					// is unlikely to cause issues. We still honor the POP bit.
					addr &= 0x8
					addr |= 0xFFF7
				}
				values = append(values, byte(addr&255), byte(addr>>8))
			} else if unit.Type == "delay" {
				countTrack := (unit.Parameters["count"] << 1) - 1 + unit.Parameters["notetracking"] // 1 means no note tracking and 1 delay, 2 means notetracking with 1 delay, 3 means no note tracking and 2 delays etc.
				values = append(values, byte(unit.Parameters["delay"]), byte(countTrack))
			}
			c.Commands = append(c.Commands, byte(opcode+unit.Parameters["stereo"]))
			c.Values = append(c.Values, values...)
			if unit.ID != 0 {
				localAddr := uint16((localUnitNo + 1) << 4)
				fixUp(c.Values, localFixups[unit.ID], localAddr)
				localFixups[unit.ID] = nil
				localAddrs[unit.ID] = localAddr
				globalAddr := localAddr + 16 + uint16(voiceNo)*1024
				fixUp(c.Values, globalFixups[unit.ID], globalAddr)
				globalFixups[unit.ID] = nil
				globalAddrs[unit.ID] = globalAddr
			}
			localUnitNo++ // a command in command stream means the wrkspace addr gets also increased
		}
		c.Commands = append(c.Commands, byte(0)) // advance
		voiceNo += instr.NumVoices
		c.NumVoices += uint32(instr.NumVoices)
		for k := 0; k < instr.NumVoices-1; k++ {
			c.PolyphonyBitmask = (c.PolyphonyBitmask << 1) + 1
		}
		c.PolyphonyBitmask <<= 1
	}
	if c.NumVoices > 32 {
		return nil, fmt.Errorf("Sointu does not support more than 32 concurrent voices; patch uses %v", c.NumVoices)
	}

	return &c, nil
}

func fixUp(values []byte, positions []int, delta uint16) {
	for _, pos := range positions {
		orig := (uint16(values[pos+1]) << 8) + uint16(values[pos])
		new := orig + delta
		values[pos] = byte(new & 255)
		values[pos+1] = byte(new >> 8)
	}
}
