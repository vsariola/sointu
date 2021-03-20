package vm

import (
	"sort"

	"github.com/vsariola/sointu"
)

// FeatureSet defines what opcodes / parameters are included in the compiled virtual machine
// It is used by the compiler to decide how to encode opcodes
type FeatureSet interface {
	Opcode(unitType string) (int, bool)
	TransformCount(unitType string) int
	Instructions() []string
	InputNumber(unitType string, paramName string) int
	SupportsParamValue(unitType string, paramName string, value int) bool
	SupportsParamValueOtherThan(unitType string, paramName string, value int) bool
	SupportsModulation(unitType string, paramName string) bool
	SupportsPolyphony() bool
	SupportsGlobalSend() bool
}

type Instruction struct {
	Name           string
	TransformCount int
}

type paramKey struct {
	Unit  string
	Param string
}

type paramValueKey struct {
	Unit  string
	Param string
	Value int
}

// AllFeatures is used by the library compilation / bridging to configure a virtual machine
// that supports every conceivable parameter, so it needs no members and just returns "true" to all
// queries about what it supports. Contrast this NecessaryFeatures that only returns true if the patch
// needs support for that feature
type AllFeatures struct {
}

func (_ AllFeatures) SupportsParamValue(unit string, paramName string, value int) bool {
	return true
}

func (_ AllFeatures) SupportsParamValueOtherThan(unit string, paramName string, value int) bool {
	return true
}

func (_ AllFeatures) SupportsModulation(unit string, port string) bool {
	return true
}

func (_ AllFeatures) SupportsPolyphony() bool {
	return true
}

func (_ AllFeatures) SupportsGlobalSend() bool {
	return true
}

func (_ AllFeatures) Opcode(unitType string) (int, bool) {
	code, ok := allOpcodes[unitType]
	return code, ok
}

func (_ AllFeatures) TransformCount(unitType string) int {
	return allTransformCounts[unitType]
}

func (_ AllFeatures) Instructions() []string {
	return allInstructions
}

func (_ AllFeatures) InputNumber(unitType string, paramName string) int {
	return allInputs[paramKey{unitType, paramName}]
}

var allOpcodes map[string]int
var allInstructions []string
var allInputs map[paramKey]int
var allTransformCounts map[string]int

func init() {
	allInstructions = make([]string, len(sointu.UnitTypes))
	allOpcodes = map[string]int{}
	allTransformCounts = map[string]int{}
	allInputs = map[paramKey]int{}
	i := 0
	for k, v := range sointu.UnitTypes {
		inputCount := 0
		transformCount := 0
		for _, t := range v {
			if t.CanModulate {
				allInputs[paramKey{k, t.Name}] = inputCount
				inputCount++
			}
			if t.CanModulate && t.CanSet {
				transformCount++
			}
		}
		allInstructions[i] = k // Opcode 0 is reserved for instrument advance, so opcodes start from 1
		allTransformCounts[k] = transformCount
		i++
	}
	sort.Strings(allInstructions) // sort the opcodes to have predictable ordering, as maps don't guarantee the order the items
	for i, instruction := range allInstructions {
		allOpcodes[instruction] = (i + 1) * 2 // make a map to find out the opcode number based on the type
	}
}

// NecessaryFeatures returns true only if the patch actually needs the support for the feature
type NecessaryFeatures struct {
	opcodes            map[string]int
	instructions       []string
	supportsParamValue map[paramKey](map[int]bool)
	supportsModulation map[paramKey]bool
	globalSend         bool
	polyphony          bool
}

func NecessaryFeaturesFor(patch sointu.Patch) NecessaryFeatures {
	features := NecessaryFeatures{opcodes: map[string]int{}, supportsParamValue: map[paramKey](map[int]bool){}, supportsModulation: map[paramKey]bool{}}
	for instrIndex, instrument := range patch {
		for _, unit := range instrument.Units {
			if _, ok := features.opcodes[unit.Type]; !ok {
				features.instructions = append(features.instructions, unit.Type)
				features.opcodes[unit.Type] = len(features.instructions) * 2 // note that the first opcode gets value 1, as 0 is always reserved for advance
			}
			for _, paramType := range sointu.UnitTypes[unit.Type] {
				v := unit.Parameters[paramType.Name]
				key := paramKey{unit.Type, paramType.Name}
				if features.supportsParamValue[key] == nil {
					features.supportsParamValue[key] = map[int]bool{}
				}
				features.supportsParamValue[key][v] = true
			}
			if unit.Type == "send" {
				targetInstrIndex, targetUnitIndex, err := patch.FindSendTarget(unit.Parameters["target"])
				if err != nil {
					continue
				}
				targetUnit := patch[targetInstrIndex].Units[targetUnitIndex]
				portList := sointu.Ports[targetUnit.Type]
				portIndex := unit.Parameters["port"]
				if portIndex < 0 || portIndex >= len(portList) {
					continue
				}
				if targetInstrIndex != instrIndex || unit.Parameters["voice"] > 0 {
					features.globalSend = true
				}
				features.supportsModulation[paramKey{targetUnit.Type, portList[portIndex]}] = true
			}
		}
		if instrument.NumVoices > 1 {
			features.polyphony = true
		}
	}
	return features
}

func (n NecessaryFeatures) SupportsParamValue(unit string, paramName string, value int) bool {
	m, ok := n.supportsParamValue[paramKey{unit, paramName}]
	if !ok {
		return false
	}
	return m[value]
}

func (n NecessaryFeatures) SupportsParamValueOtherThan(unit string, paramName string, value int) bool {
	for paramValue := range n.supportsParamValue[paramKey{unit, paramName}] {
		if paramValue != value {
			return true
		}
	}
	return false
}

func (n NecessaryFeatures) SupportsModulation(unit string, param string) bool {
	return n.supportsModulation[paramKey{unit, param}]
}

func (n NecessaryFeatures) SupportsPolyphony() bool {
	return n.polyphony
}

func (n NecessaryFeatures) Opcode(unitType string) (int, bool) {
	code, ok := n.opcodes[unitType]
	return code, ok
}

func (n NecessaryFeatures) Instructions() []string {
	return n.instructions
}

func (n NecessaryFeatures) InputNumber(unitType string, paramName string) int {
	return allInputs[paramKey{unitType, paramName}]
}

func (_ NecessaryFeatures) TransformCount(unitType string) int {
	return allTransformCounts[unitType]
}

func (n NecessaryFeatures) SupportsGlobalSend() bool {
	return n.globalSend
}
