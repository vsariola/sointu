package tracker

import (
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
)

type (
	Parameter interface {
		IntData
		Type() ParameterType
		Name() string
		Hint() string
		LargeStep() int
		Reset()
	}

	parameter struct {
		m    *Model
		unit *sointu.Unit
	}

	NamedParameter struct {
		parameter
		up *sointu.UnitParameter
	}

	DelayTimeParameter struct {
		parameter
		index int
	}

	DelayLinesParameter struct{ parameter }
	GmDlsEntryParameter struct{ parameter }
	ReverbParameter     struct{ parameter }

	Params Model

	ParamYieldFunc func(Parameter)

	ParameterType int
)

const (
	IntegerParameter ParameterType = iota
	BoolParameter
	IDParameter
)

// Model methods

func (m *Model) Params() *Params { return (*Params)(m) }

// parameter methods

func (p parameter) change(kind string) func() {
	return p.m.change("Parameter."+kind, PatchChange, MinorChange)
}

// ParamList

func (pl *Params) List() List             { return List{pl} }
func (pl *Params) Selected() int          { return pl.d.ParamIndex }
func (pl *Params) Selected2() int         { return pl.Selected() }
func (pl *Params) SetSelected(value int)  { pl.d.ParamIndex = intMax(intMin(value, pl.Count()-1), 0) }
func (pl *Params) SetSelected2(value int) {}
func (pl *Params) cancel()                { (*Model)(pl).changeCancel = true }

func (pl *Params) change(n string, severity ChangeSeverity) func() {
	return (*Model)(pl).change("ParamList."+n, PatchChange, severity)
}

func (pl *Params) Count() int {
	count := 0
	pl.Iterate(func(p Parameter) {
		count++
	})
	return count
}

func (pl *Params) SelectedItem() (ret Parameter) {
	index := pl.Selected()
	pl.Iterate(func(param Parameter) {
		if index == 0 {
			ret = param
		}
		index--
	})
	return
}

func (pl *Params) Iterate(yield ParamYieldFunc) {
	if pl.d.InstrIndex < 0 || pl.d.InstrIndex >= len(pl.d.Song.Patch) {
		return
	}
	if pl.d.UnitIndex < 0 || pl.d.UnitIndex >= len(pl.d.Song.Patch[pl.d.InstrIndex].Units) {
		return
	}
	unit := &pl.d.Song.Patch[pl.d.InstrIndex].Units[pl.d.UnitIndex]
	unitType, ok := sointu.UnitTypes[unit.Type]
	if !ok {
		return
	}
	for i := range unitType {
		if !unitType[i].CanSet {
			continue
		}
		if unit.Type == "oscillator" && unit.Parameters["type"] != sointu.Sample && i >= 11 {
			break // don't show the sample related params unless necessary
		}
		yield(NamedParameter{
			parameter: parameter{m: (*Model)(pl), unit: unit},
			up:        &unitType[i],
		})
	}
	if unit.Type == "oscillator" && unit.Parameters["type"] == sointu.Sample {
		yield(GmDlsEntryParameter{parameter: parameter{m: (*Model)(pl), unit: unit}})
	}
	switch {
	case unit.Type == "delay":
		if unit.Parameters["stereo"] == 1 && len(unit.VarArgs)%2 == 1 {
			unit.VarArgs = append(unit.VarArgs, 1)
		}
		yield(ReverbParameter{parameter: parameter{m: (*Model)(pl), unit: unit}})
		yield(DelayLinesParameter{parameter: parameter{m: (*Model)(pl), unit: unit}})
		for i := range unit.VarArgs {
			yield(DelayTimeParameter{parameter: parameter{m: (*Model)(pl), unit: unit}, index: i})
		}
	}
}

// NamedParameter

func (p NamedParameter) Name() string       { return p.up.Name }
func (p NamedParameter) Range() intRange    { return intRange{Min: p.up.MinValue, Max: p.up.MaxValue} }
func (p NamedParameter) Value() int         { return p.unit.Parameters[p.up.Name] }
func (p NamedParameter) setValue(value int) { p.unit.Parameters[p.up.Name] = value }

func (p NamedParameter) Reset() {
	v, ok := defaultUnits[p.unit.Type].Parameters[p.up.Name]
	if !ok || p.unit.Parameters[p.up.Name] == v {
		return
	}
	defer p.parameter.change("Reset")()
	p.unit.Parameters[p.up.Name] = v
}

func (p NamedParameter) Type() ParameterType {
	if p.unit.Type == "send" && p.up.Name == "target" {
		return IDParameter
	}
	if p.up.MinValue == 0 && p.up.MaxValue == 1 {
		return BoolParameter
	}
	return IntegerParameter
}

func (p NamedParameter) Hint() string {
	val := p.Value()
	if p.up.DisplayFunc != nil {
		valueInUnits, units := p.up.DisplayFunc(val)
		return fmt.Sprintf("%d / %s %s", val, valueInUnits, units)
	}
	if p.unit.Type == "send" && p.up.Name == "voice" && val == 0 {
		targetIndex, _, err := p.m.FindUnit(p.unit.Parameters["target"])
		if err == nil && targetIndex != p.m.d.InstrIndex {
			return "all"
		}
		return "self"
	}
	if p.unit.Type == "send" && p.up.Name == "port" {
		instrIndex, unitIndex, err := p.m.FindUnit(p.unit.Parameters["target"])
		if err != nil {
			return strconv.Itoa(val)
		}
		portList := sointu.Ports[p.m.d.Song.Patch[instrIndex].Units[unitIndex].Type]
		if val < 0 || val >= len(portList) {
			return strconv.Itoa(val)
		}
		return fmt.Sprintf(portList[val])
	}
	return strconv.Itoa(val)
}

func (p NamedParameter) LargeStep() int {
	if p.up.Name == "transpose" {
		return 12
	}
	return 16
}

// GmDlsEntryParameter

func (p GmDlsEntryParameter) Name() string        { return "sample" }
func (p GmDlsEntryParameter) Type() ParameterType { return IntegerParameter }
func (p GmDlsEntryParameter) Range() intRange     { return intRange{Min: 0, Max: len(GmDlsEntries)} }
func (p GmDlsEntryParameter) LargeStep() int      { return 16 }
func (p GmDlsEntryParameter) Reset()              { return }

func (p GmDlsEntryParameter) Value() int {
	key := vm.SampleOffset{Start: uint32(p.unit.Parameters["samplestart"]), LoopStart: uint16(p.unit.Parameters["loopstart"]), LoopLength: uint16(p.unit.Parameters["looplength"])}
	if v, ok := gmDlsEntryMap[key]; ok {
		return v + 1
	}
	return 0
}

func (p GmDlsEntryParameter) setValue(v int) {
	if v < 1 || v > len(GmDlsEntries) {
		return
	}
	e := GmDlsEntries[v-1]
	p.unit.Parameters["samplestart"] = e.Start
	p.unit.Parameters["loopstart"] = e.LoopStart
	p.unit.Parameters["looplength"] = e.LoopLength
	p.unit.Parameters["transpose"] = 64 + e.SuggestedTranspose
}

func (p GmDlsEntryParameter) Hint() string {
	if v := p.Value(); v > 0 {
		return fmt.Sprintf("%v / %v", v, GmDlsEntries[v-1].Name)
	}
	return "0 / custom"
}

// DelayTimeParameter

func (p DelayTimeParameter) Name() string        { return "delaytime" }
func (p DelayTimeParameter) Type() ParameterType { return IntegerParameter }
func (p DelayTimeParameter) LargeStep() int      { return 16 }
func (p DelayTimeParameter) Reset()              { return }

func (p DelayTimeParameter) Value() int {
	if p.index < 0 || p.index >= len(p.unit.VarArgs) {
		return 1
	}
	return p.unit.VarArgs[p.index]
}

func (p DelayTimeParameter) setValue(v int) {
	p.unit.VarArgs[p.index] = v
}

func (p DelayTimeParameter) Range() intRange {
	if p.unit.Parameters["notetracking"] == 2 {
		return intRange{Min: 1, Max: 576}
	}
	return intRange{Min: 1, Max: 65535}
}

func (p DelayTimeParameter) Hint() string {
	val := p.Value()
	var text string
	switch p.unit.Parameters["notetracking"] {
	default:
	case 0:
		text = fmt.Sprintf("%v / %.3f rows", val, float32(val)/float32(p.m.d.Song.SamplesPerRow()))
	case 1:
		relPitch := float64(val) / 10787
		semitones := -math.Log2(relPitch) * 12
		text = fmt.Sprintf("%v / %.3f st", val, semitones)
	case 2:
		k := 0
		v := val
		for v&1 == 0 { // divide val by 2 until it is odd
			v >>= 1
			k++
		}
		switch v {
		case 1:
			if k <= 7 {
				text = fmt.Sprintf(" (1/%d triplet)", 1<<(7-k))
			}
		case 3:
			if k <= 6 {
				text = fmt.Sprintf(" (1/%d)", 1<<(6-k))
			}
			break
		case 9:
			if k <= 5 {
				text = fmt.Sprintf(" (1/%d dotted)", 1<<(5-k))
			}
		}
		text = fmt.Sprintf("%v / %.3f beats%s", val, float32(val)/48.0, text)
	}
	if p.unit.Parameters["stereo"] == 1 {
		if p.index < len(p.unit.VarArgs)/2 {
			text += " R"
		} else {
			text += " L"
		}
	}
	return text
}

// DelayLinesParameter

func (p DelayLinesParameter) Name() string        { return "delaylines" }
func (p DelayLinesParameter) Type() ParameterType { return IntegerParameter }
func (p DelayLinesParameter) Range() intRange     { return intRange{Min: 1, Max: 32} }
func (p DelayLinesParameter) LargeStep() int      { return 4 }
func (p DelayLinesParameter) Reset()              { return }
func (p DelayLinesParameter) Hint() string        { return strconv.Itoa(p.Value()) }

func (p DelayLinesParameter) Value() int {
	val := len(p.unit.VarArgs)
	if p.unit.Parameters["stereo"] == 1 {
		val /= 2
	}
	return val
}

func (p DelayLinesParameter) setValue(v int) {
	targetLines := v
	if p.unit.Parameters["stereo"] == 1 {
		targetLines *= 2
	}
	for len(p.unit.VarArgs) < targetLines {
		p.unit.VarArgs = append(p.unit.VarArgs, 1)
	}
	p.unit.VarArgs = p.unit.VarArgs[:targetLines]
}

// ReverbParameter

func (p ReverbParameter) Name() string        { return "reverb" }
func (p ReverbParameter) Type() ParameterType { return IntegerParameter }
func (p ReverbParameter) Range() intRange     { return intRange{Min: 0, Max: len(reverbs)} }
func (p ReverbParameter) LargeStep() int      { return 1 }
func (p ReverbParameter) Reset()              { return }

func (p ReverbParameter) Value() int {
	i := slices.IndexFunc(reverbs, func(d delayPreset) bool {
		return d.stereo == p.unit.Parameters["stereo"] && p.unit.Parameters["notetracking"] == 0 && slices.Equal(d.varArgs, p.unit.VarArgs)
	})
	return i + 1
}

func (p ReverbParameter) setValue(v int) {
	if v < 1 || v > len(reverbs) {
		return
	}
	entry := reverbs[v-1]
	p.unit.Parameters["stereo"] = entry.stereo
	p.unit.Parameters["notetracking"] = 0
	p.unit.VarArgs = make([]int, len(entry.varArgs))
	copy(p.unit.VarArgs, entry.varArgs)
}

func (p ReverbParameter) Hint() string {
	i := p.Value()
	if i > 0 {
		return fmt.Sprintf("%v / %v", i, reverbs[i-1].name)
	}
	return "0 / custom"
}
