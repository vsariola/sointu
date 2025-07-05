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
	// Parameter represents a parameter of a unit. To support polymorphism
	// without causing allocations, it has a vtable that defines the methods for
	// the specific parameter type, to which all the method calls are delegated.
	Parameter struct {
		m      *Model
		unit   *sointu.Unit
		up     *sointu.UnitParameter
		index  int
		vtable parameterVtable
		port   int
	}

	parameterVtable interface {
		Value(*Parameter) int
		SetValue(*Parameter, int) bool
		Range(*Parameter) IntRange
		Type(*Parameter) ParameterType
		Name(*Parameter) string
		Hint(*Parameter) ParameterHint
		LargeStep(*Parameter) int
		Reset(*Parameter)
	}

	Params        Model
	ParamVertList Model

	// different parameter vtables to handle different types of parameters.
	// Casting struct{} to interface does not cause allocations.
	namedParameter      struct{}
	delayTimeParameter  struct{}
	delayLinesParameter struct{}
	gmDlsEntryParameter struct{}
	reverbParameter     struct{}

	ParamYieldFunc func(param Parameter) bool

	ParameterType int

	ParameterHint struct {
		Label string
		Valid bool
	}
)

const (
	NoParameter ParameterType = iota
	IntegerParameter
	BoolParameter
	IDParameter
)

// Parameter methods

func (p *Parameter) Value() int {
	if p.vtable == nil {
		return 0
	}
	return p.vtable.Value(p)
}
func (p *Parameter) Port() (int, bool) {
	if p.port <= 0 {
		return 0, false
	}
	return p.port - 1, true
}
func (p *Parameter) SetValue(value int) bool {
	if p.vtable == nil {
		return false
	}
	r := p.Range()
	value = r.Clamp(value)
	if value == p.Value() || value < r.Min || value > r.Max {
		return false
	}
	return p.vtable.SetValue(p, value)
}
func (p *Parameter) Range() IntRange {
	if p.vtable == nil {
		return IntRange{}
	}
	return p.vtable.Range(p)
}
func (p *Parameter) Type() ParameterType {
	if p.vtable == nil {
		return NoParameter
	}
	return p.vtable.Type(p)
}
func (p *Parameter) Name() string {
	if p.vtable == nil {
		return ""
	}
	return p.vtable.Name(p)
}
func (p *Parameter) Hint() ParameterHint {
	if p.vtable == nil {
		return ParameterHint{}
	}
	return p.vtable.Hint(p)
}
func (p *Parameter) LargeStep() int {
	if p.vtable == nil {
		return 1
	}
	return p.vtable.LargeStep(p)
}
func (p *Parameter) Reset() {
	if p.vtable == nil {
		return
	}
	p.vtable.Reset(p)
}
func (p *Parameter) UnitID() int {
	if p.unit == nil {
		return 0
	}
	return p.unit.ID
}

//

func (m *Model) ParamVertList() *ParamVertList   { return (*ParamVertList)(m) }
func (pt *ParamVertList) List() List             { return List{pt} }
func (pt *ParamVertList) Selected() int          { return pt.d.ParamIndex }
func (pt *ParamVertList) Selected2() int         { return pt.d.ParamIndex }
func (pt *ParamVertList) SetSelected(index int)  { pt.d.ParamIndex = index }
func (pt *ParamVertList) SetSelected2(index int) {}
func (pt *ParamVertList) Count() int             { return (*Params)(pt).Width() }

// Model and Params methods

func (m *Model) Params() *Params  { return (*Params)(m) }
func (pt *Params) Table() Table   { return Table{pt} }
func (pt *Params) Cursor() Point  { return Point{pt.d.ParamIndex, pt.d.UnitIndex} }
func (pt *Params) Cursor2() Point { return pt.Cursor() }
func (pt *Params) SetCursor(p Point) {
	pt.d.ParamIndex = max(min(p.X, pt.Width()-1), 0)
	pt.d.UnitIndex = max(min(p.Y, pt.Height()-1), 0)
	pt.d.UnitIndex2 = pt.d.UnitIndex
}
func (pt *Params) SetCursor2(p Point) {}
func (pt *Params) Width() int {
	if pt.d.InstrIndex < 0 || pt.d.InstrIndex >= len(pt.derived.patch) {
		return 0
	}
	return pt.derived.patch[pt.d.InstrIndex].paramsWidth
}
func (pt *Params) Height() int { return (*Model)(pt).Units().Count() }
func (pt *Params) MoveCursor(dx, dy int) (ok bool) {
	p := pt.Cursor()
	p.X += dx
	p.Y += dy
	pt.SetCursor(p)
	return p == pt.Cursor()
}
func (pt *Params) Item(p Point) Parameter {
	if pt.d.InstrIndex < 0 || pt.d.InstrIndex >= len(pt.derived.patch) || p.Y < 0 || p.Y >= len(pt.derived.patch[pt.d.InstrIndex].params) || p.X < 0 || p.X >= len(pt.derived.patch[pt.d.InstrIndex].params[p.Y]) {
		return Parameter{}
	}
	return pt.derived.patch[pt.d.InstrIndex].params[p.Y][p.X]
}
func (pt *Params) clear(p Point) {
	panic("NOT IMPLEMENTED")
}
func (pt *Params) set(p Point, value int) {
	panic("NOT IMPLEMENTED")
}
func (pt *Params) add(rect Rect, delta int) (ok bool) {
	panic("NOT IMPLEMENTED")
}
func (pt *Params) marshal(rect Rect) (data []byte, ok bool) {
	panic("NOT IMPLEMENTED")
}
func (pt *Params) unmarshalAtCursor(data []byte) (ok bool) {
	panic("NOT IMPLEMENTED")
}
func (pt *Params) unmarshalRange(rect Rect, data []byte) (ok bool) {
	panic("NOT IMPLEMENTED")
}
func (pt *Params) change(kind string, severity ChangeSeverity) func() {
	panic("NOT IMPLEMENTED")
}
func (pt *Params) cancel() {
	panic("NOT IMPLEMENTED")
}

// namedParameter vtable

func (n *namedParameter) Value(p *Parameter) int { return p.unit.Parameters[p.up.Name] }
func (n *namedParameter) SetValue(p *Parameter, value int) bool {
	defer p.m.change("Parameter"+p.Name(), PatchChange, MinorChange)()
	p.unit.Parameters[p.up.Name] = value
	return true
}
func (n *namedParameter) Range(p *Parameter) IntRange {
	return IntRange{Min: p.up.MinValue, Max: p.up.MaxValue}
}
func (n *namedParameter) Type(p *Parameter) ParameterType {
	if p.up == nil || !p.up.CanSet {
		return NoParameter
	}
	if p.unit.Type == "send" && p.up.Name == "target" {
		return IDParameter
	}
	if p.up.MinValue == 0 && p.up.MaxValue == 1 {
		return BoolParameter
	}
	return IntegerParameter
}
func (n *namedParameter) Name(p *Parameter) string {
	return p.up.Name
}
func (n *namedParameter) Hint(p *Parameter) ParameterHint {
	val := p.Value()
	label := strconv.Itoa(val)
	if p.up.DisplayFunc != nil {
		valueInUnits, units := p.up.DisplayFunc(val)
		label = fmt.Sprintf("%s %s", valueInUnits, units)
	}
	return ParameterHint{label, true}
}
func (n *namedParameter) LargeStep(p *Parameter) int {
	if p.up.Name == "transpose" {
		return 12
	}
	return 16
}
func (n *namedParameter) Reset(p *Parameter) {
	v, ok := defaultUnits[p.unit.Type].Parameters[p.up.Name]
	if !ok || p.unit.Parameters[p.up.Name] == v {
		return
	}
	defer p.m.change("Reset"+p.Name(), PatchChange, MinorChange)()
	p.unit.Parameters[p.up.Name] = v
}

// gmDlsEntryParameter vtable

func (g *gmDlsEntryParameter) Value(p *Parameter) int {
	key := vm.SampleOffset{
		Start:      uint32(p.unit.Parameters["samplestart"]),
		LoopStart:  uint16(p.unit.Parameters["loopstart"]),
		LoopLength: uint16(p.unit.Parameters["looplength"]),
	}
	if v, ok := gmDlsEntryMap[key]; ok {
		return v + 1
	}
	return 0
}
func (g *gmDlsEntryParameter) SetValue(p *Parameter, v int) bool {
	if v < 1 || v > len(GmDlsEntries) {
		return false
	}
	defer p.m.change("GmDlsEntryParameter", PatchChange, MinorChange)()
	e := GmDlsEntries[v-1]
	p.unit.Parameters["samplestart"] = e.Start
	p.unit.Parameters["loopstart"] = e.LoopStart
	p.unit.Parameters["looplength"] = e.LoopLength
	p.unit.Parameters["transpose"] = 64 + e.SuggestedTranspose
	return true
}
func (g *gmDlsEntryParameter) Range(p *Parameter) IntRange {
	return IntRange{Min: 0, Max: len(GmDlsEntries)}
}
func (g *gmDlsEntryParameter) Type(p *Parameter) ParameterType {
	return IntegerParameter
}
func (g *gmDlsEntryParameter) Name(p *Parameter) string {
	return "sample"
}
func (g *gmDlsEntryParameter) Hint(p *Parameter) ParameterHint {
	label := "custom"
	if v := g.Value(p); v > 0 {
		label = GmDlsEntries[v-1].Name
	}
	return ParameterHint{label, true}
}
func (g *gmDlsEntryParameter) LargeStep(p *Parameter) int {
	return 16
}
func (g *gmDlsEntryParameter) Reset(p *Parameter) {}

// delayTimeParameter vtable

func (d *delayTimeParameter) Value(p *Parameter) int {
	if p.index < 0 || p.index >= len(p.unit.VarArgs) {
		return 1
	}
	return p.unit.VarArgs[p.index]
}
func (d *delayTimeParameter) SetValue(p *Parameter, v int) bool {
	defer p.m.change("DelayTimeParameter", PatchChange, MinorChange)()
	p.unit.VarArgs[p.index] = v
	return true
}
func (d *delayTimeParameter) Range(p *Parameter) IntRange {
	if p.unit.Parameters["notetracking"] == 2 {
		return IntRange{Min: 1, Max: 576}
	}
	return IntRange{Min: 1, Max: 65535}
}
func (d *delayTimeParameter) Type(p *Parameter) ParameterType {
	return IntegerParameter
}
func (d *delayTimeParameter) Name(p *Parameter) string {
	return "delaytime"
}
func (d *delayTimeParameter) Hint(p *Parameter) ParameterHint {
	val := d.Value(p)
	var text string
	switch p.unit.Parameters["notetracking"] {
	default:
	case 0:
		text = fmt.Sprintf("%.3f rows", float32(val)/float32(p.m.d.Song.SamplesPerRow()))
	case 1:
		relPitch := float64(val) / 10787
		semitones := -math.Log2(relPitch) * 12
		text = fmt.Sprintf("%.3f st", semitones)
	case 2:
		k := 0
		v := val
		for v&1 == 0 {
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
	return ParameterHint{text, true}
}
func (d *delayTimeParameter) LargeStep(p *Parameter) int {
	return 16
}
func (d *delayTimeParameter) Reset(p *Parameter) {}

// delayLinesParameter vtable

func (d *delayLinesParameter) Value(p *Parameter) int {
	val := len(p.unit.VarArgs)
	if p.unit.Parameters["stereo"] == 1 {
		val /= 2
	}
	return val
}
func (d *delayLinesParameter) SetValue(p *Parameter, v int) bool {
	defer p.m.change("DelayLinesParameter", PatchChange, MinorChange)()
	targetLines := v
	if p.unit.Parameters["stereo"] == 1 {
		targetLines *= 2
	}
	for len(p.unit.VarArgs) < targetLines {
		p.unit.VarArgs = append(p.unit.VarArgs, 1)
	}
	p.unit.VarArgs = p.unit.VarArgs[:targetLines]
	return true
}
func (d *delayLinesParameter) Range(p *Parameter) IntRange {
	return IntRange{Min: 1, Max: 32}
}
func (d *delayLinesParameter) Type(p *Parameter) ParameterType {
	return IntegerParameter
}
func (d *delayLinesParameter) Name(p *Parameter) string {
	return "delaylines"
}
func (d *delayLinesParameter) Hint(p *Parameter) ParameterHint {
	return ParameterHint{strconv.Itoa(d.Value(p)), true}
}
func (d *delayLinesParameter) LargeStep(p *Parameter) int {
	return 4
}
func (d *delayLinesParameter) Reset(p *Parameter) {}

// reverbParameter vtable

func (r *reverbParameter) Value(p *Parameter) int {
	i := slices.IndexFunc(reverbs, func(d delayPreset) bool {
		return d.stereo == p.unit.Parameters["stereo"] && p.unit.Parameters["notetracking"] == 0 && slices.Equal(d.varArgs, p.unit.VarArgs)
	})
	return i + 1
}
func (r *reverbParameter) SetValue(p *Parameter, v int) bool {
	if v < 1 || v > len(reverbs) {
		return false
	}
	defer p.m.change("ReverbParameter", PatchChange, MinorChange)()
	entry := reverbs[v-1]
	p.unit.Parameters["stereo"] = entry.stereo
	p.unit.Parameters["notetracking"] = 0
	p.unit.VarArgs = make([]int, len(entry.varArgs))
	copy(p.unit.VarArgs, entry.varArgs)
	return true
}
func (r *reverbParameter) Range(p *Parameter) IntRange {
	return IntRange{Min: 0, Max: len(reverbs)}
}
func (r *reverbParameter) Type(p *Parameter) ParameterType {
	return IntegerParameter
}
func (r *reverbParameter) Name(p *Parameter) string {
	return "reverb"
}
func (r *reverbParameter) Hint(p *Parameter) ParameterHint {
	i := r.Value(p)
	label := "custom"
	if i > 0 {
		label = reverbs[i-1].name
	}
	return ParameterHint{label, true}
}
func (r *reverbParameter) LargeStep(p *Parameter) int {
	return 1
}
func (r *reverbParameter) Reset(p *Parameter) {}
