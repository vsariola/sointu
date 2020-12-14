package compiler

import (
	"errors"
	"fmt"

	"github.com/vsariola/sointu/go4k"
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

func Encode(patch *go4k.Patch, featureSet FeatureSet) (*EncodedPatch, error) {
	var c EncodedPatch
	sampleOffsetMap := map[SampleOffset]int{}
	for _, instr := range patch.Instruments {
		if len(instr.Units) > 63 {
			return nil, errors.New("An instrument can have a maximum of 63 units")
		}
		if instr.NumVoices < 1 {
			return nil, errors.New("Each instrument must have at least 1 voice")
		}
		for _, unit := range instr.Units {
			if unit.Type == "oscillator" && unit.Parameters["type"] == 4 {
				s := SampleOffset{Start: uint32(unit.Parameters["start"]), LoopStart: uint16(unit.Parameters["loopstart"]), LoopLength: uint16(unit.Parameters["looplength"])}
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
			command, values, err := EncodeUnit(unit, featureSet)
			if err != nil {
				return nil, fmt.Errorf(`encoding unit failed: %v`, err)
			}
			c.Commands = append(c.Commands, command)
			c.Values = append(c.Values, values...)
		}
		c.Commands = append(c.Commands, byte(0)) // advance
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

func EncodeUnit(unit go4k.Unit, featureSet FeatureSet) (byte, []byte, error) {
	opcode, ok := featureSet.Opcode(unit.Type)
	if !ok {
		return 0, nil, fmt.Errorf(`the targeted virtual machine is not configured to support unit type "%v"`, unit.Type)
	}
	var values []byte
	for _, v := range go4k.UnitTypes[unit.Type] {
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
		address := ((unit.Parameters["unit"] + 1) << 4) + unit.Parameters["port"] // each unit is 16 dwords, 8 workspace followed by 8 ports. +1 is for skipping the note/release/inputs
		if unit.Parameters["voice"] > 0 {
			address += 0x8000 + 16 + (unit.Parameters["voice"]-1)*1024 // global send, +16 is for skipping the out/aux ports
		}
		if unit.Parameters["sendpop"] == 1 {
			address += 0x8
		}
		values = append(values, byte(address&255), byte(address>>8))
	} else if unit.Type == "delay" {
		countTrack := (unit.Parameters["count"] << 1) - 1 + unit.Parameters["notetracking"] // 1 means no note tracking and 1 delay, 2 means notetracking with 1 delay, 3 means no note tracking and 2 delays etc.
		values = append(values, byte(unit.Parameters["delay"]), byte(countTrack))
	}
	return byte(opcode + unit.Parameters["stereo"]), values, nil
}
