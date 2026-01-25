package tracker

import (
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
	"gopkg.in/yaml.v3"
)

// Params returns the Param view of the Model, containing methods to manipulate
// the parameters.
func (m *Model) Params() *ParamModel { return (*ParamModel)(m) }

type ParamModel Model

// Wires returns the wires of the current instrument, telling which parameters
// are connected to which.
func (m *ParamModel) Wires(yield func(wire Wire) bool) {
	i := m.d.InstrIndex
	if i < 0 || i >= len(m.derived.patch) {
		return
	}
	for _, wire := range m.derived.patch[i].wires {
		wire.Highlight = (wire.FromSet && m.d.UnitIndex == wire.From) || (wire.ToSet && m.d.UnitIndex == wire.To.Y && m.d.ParamIndex == wire.To.X)
		if !yield(wire) {
			return
		}
	}
}

// chooseSendSource
type chooseSendSource struct {
	ID int
	*Model
}

func (m *ParamModel) IsChoosingSendTarget() bool {
	return m.d.SendSource > 0
}

func (m *ParamModel) ChooseSendSource(id int) Action {
	return MakeAction(chooseSendSource{ID: id, Model: (*Model)(m)})
}
func (s chooseSendSource) Do() {
	defer (*Model)(s.Model).change("ChooseSendSource", NoChange, MinorChange)()
	if s.Model.d.SendSource == s.ID {
		s.Model.d.SendSource = 0 // unselect
		return
	}
	s.Model.d.SendSource = s.ID
}

// chooseSendTarget
type chooseSendTarget struct {
	ID   int
	Port int
	*Model
}

func (m *ParamModel) ChooseSendTarget(id int, port int) Action {
	return MakeAction(chooseSendTarget{ID: id, Port: port, Model: (*Model)(m)})
}
func (s chooseSendTarget) Do() {
	defer (*Model)(s.Model).change("ChooseSendTarget", SongChange, MinorChange)()
	sourceID := (*Model)(s.Model).d.SendSource
	s.d.SendSource = 0
	if sourceID <= 0 || s.ID <= 0 || s.Port < 0 || s.Port > 7 {
		return
	}
	si, su, err := s.d.Song.Patch.FindUnit(sourceID)
	if err != nil {
		return
	}
	s.d.Song.Patch[si].Units[su].Parameters["target"] = s.ID
	s.d.Song.Patch[si].Units[su].Parameters["port"] = s.Port
}

// paramsColumns
type paramsColumns Model

func (m *ParamModel) Columns() List              { return List{(*paramsColumns)(m)} }
func (pt *paramsColumns) Selected() int          { return pt.d.ParamIndex }
func (pt *paramsColumns) Selected2() int         { return pt.d.ParamIndex }
func (pt *paramsColumns) SetSelected(index int)  { pt.d.ParamIndex = index }
func (pt *paramsColumns) SetSelected2(index int) {}
func (pt *paramsColumns) Count() int             { return (*ParamModel)(pt).Width() }

// Model and Params methods

func (pt *ParamModel) Table() Table   { return Table{pt} }
func (pt *ParamModel) Cursor() Point  { return Point{pt.d.ParamIndex, pt.d.UnitIndex} }
func (pt *ParamModel) Cursor2() Point { return Point{pt.d.ParamIndex, pt.d.UnitIndex2} }
func (pt *ParamModel) SetCursor(p Point) {
	pt.d.ParamIndex = max(min(p.X, pt.Width()-1), 0)
	pt.d.UnitIndex = max(min(p.Y, pt.Height()-1), 0)
}
func (pt *ParamModel) SetCursor2(p Point) {
	pt.d.ParamIndex = max(min(p.X, pt.Width()-1), 0)
	pt.d.UnitIndex2 = max(min(p.Y, pt.Height()-1), 0)
}
func (pt *ParamModel) Width() int {
	if pt.d.InstrIndex < 0 || pt.d.InstrIndex >= len(pt.derived.patch) {
		return 0
	}
	// TODO: we hack the +1 so that we always have one extra cell to draw the
	// comments. Refactor the gioui side so that we can specify the width and
	// height regardless of the underlying table size
	return pt.derived.patch[pt.d.InstrIndex].paramsWidth + 1
}
func (pt *ParamModel) RowWidth(y int) int {
	if pt.d.InstrIndex < 0 || pt.d.InstrIndex >= len(pt.derived.patch) || y < 0 || y >= len(pt.derived.patch[pt.d.InstrIndex].params) {
		return 0
	}
	return len(pt.derived.patch[pt.d.InstrIndex].params[y])
}
func (pt *ParamModel) Height() int { return (*Model)(pt).Unit().List().Count() }
func (pt *ParamModel) MoveCursor(dx, dy int) (ok bool) {
	p := pt.Cursor()
	p.X += dx
	p.Y += dy
	pt.SetCursor(p)
	return p == pt.Cursor()
}
func (pt *ParamModel) Item(p Point) Parameter {
	if pt.d.InstrIndex < 0 || pt.d.InstrIndex >= len(pt.derived.patch) || p.Y < 0 || p.Y >= len(pt.derived.patch[pt.d.InstrIndex].params) || p.X < 0 || p.X >= len(pt.derived.patch[pt.d.InstrIndex].params[p.Y]) {
		return Parameter{}
	}
	return pt.derived.patch[pt.d.InstrIndex].params[p.Y][p.X]
}
func (pt *ParamModel) clear(p Point) {
	q := pt.Item(p)
	q.Reset()
}
func (pt *ParamModel) set(p Point, value int) {
	q := pt.Item(p)
	q.SetValue(value)
}
func (pt *ParamModel) add(rect Rect, delta int, largeStep bool) (ok bool) {
	for y := rect.TopLeft.Y; y <= rect.BottomRight.Y; y++ {
		for x := rect.TopLeft.X; x <= rect.BottomRight.X; x++ {
			p := Point{x, y}
			q := pt.Item(p)
			if !q.Add(delta, largeStep) {
				return false
			}
		}
	}
	return true
}

type paramsTable struct {
	Params [][]int `yaml:",flow"`
}

func (pt *ParamModel) marshal(rect Rect) (data []byte, ok bool) {
	width := rect.BottomRight.X - rect.TopLeft.X + 1
	height := rect.BottomRight.Y - rect.TopLeft.Y + 1
	var table = paramsTable{Params: make([][]int, 0, width)}
	for x := 0; x < width; x++ {
		table.Params = append(table.Params, make([]int, 0, rect.BottomRight.Y-rect.TopLeft.Y+1))
		for y := 0; y < height; y++ {
			p := pt.Item(Point{x + rect.TopLeft.X, y + rect.TopLeft.Y})
			table.Params[x] = append(table.Params[x], p.Value())
		}
	}
	ret, err := yaml.Marshal(table)
	if err != nil {
		return nil, false
	}
	return ret, true
}
func (pt *ParamModel) unmarshal(data []byte) (paramsTable, bool) {
	var table paramsTable
	yaml.Unmarshal(data, &table)
	if len(table.Params) == 0 {
		return paramsTable{}, false
	}
	for i := 0; i < len(table.Params); i++ {
		if len(table.Params[i]) > 0 {
			return table, true
		}
	}
	return paramsTable{}, false
}

func (pt *ParamModel) unmarshalAtCursor(data []byte) (ret bool) {
	table, ok := pt.unmarshal(data)
	if !ok {
		return false
	}
	for i := 0; i < len(table.Params); i++ {
		for j, q := range table.Params[i] {
			x := i + pt.Cursor().X
			y := j + pt.Cursor().Y
			p := pt.Item(Point{x, y})
			ret = p.SetValue(q) || ret
		}
	}
	return ret
}
func (pt *ParamModel) unmarshalRange(rect Rect, data []byte) (ret bool) {
	table, ok := pt.unmarshal(data)
	if !ok {
		return false
	}
	if len(table.Params) == 0 || len(table.Params[0]) == 0 {
		return false
	}
	width := rect.BottomRight.X - rect.TopLeft.X + 1
	height := rect.BottomRight.Y - rect.TopLeft.Y + 1
	if len(table.Params) < width {
		return false
	}
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			if len(table.Params[0]) < height {
				return false
			}
			p := pt.Item(Point{x + rect.TopLeft.X, y + rect.TopLeft.Y})
			ret = p.SetValue(table.Params[x][y]) || ret
		}
	}
	return ret
}
func (pt *ParamModel) change(kind string, severity ChangeSeverity) func() {
	return (*Model)(pt).change(kind, PatchChange, severity)
}
func (pt *ParamModel) cancel() {
	pt.changeCancel = true
}

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
		Range(*Parameter) RangeInclusive
		Type(*Parameter) ParameterType
		Name(*Parameter) string
		Hint(*Parameter) ParameterHint
		Reset(*Parameter)
		RoundToGrid(*Parameter, int, bool) int
	}

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
func (p *Parameter) Add(delta int, snapToGrid bool) bool {
	if p.vtable == nil {
		return false
	}
	newVal := p.Value() + delta
	if snapToGrid && p.vtable != nil {
		newVal = p.vtable.RoundToGrid(p, newVal, delta > 0)
	}
	return p.SetValue(newVal)
}

func (p *Parameter) Range() RangeInclusive {
	if p.vtable == nil {
		return RangeInclusive{}
	}
	return p.vtable.Range(p)
}
func (p *Parameter) Neutral() int {
	if p.vtable == nil {
		return 0
	}
	if p.up != nil {
		return p.up.Neutral
	}
	return 0
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

// namedParameter vtable

func (n *namedParameter) Value(p *Parameter) int { return p.unit.Parameters[p.up.Name] }
func (n *namedParameter) SetValue(p *Parameter, value int) bool {
	defer p.m.change("Parameter"+p.Name(), PatchChange, MinorChange)()
	p.unit.Parameters[p.up.Name] = value
	return true
}
func (n *namedParameter) Range(p *Parameter) RangeInclusive {
	return RangeInclusive{Min: p.up.MinValue, Max: p.up.MaxValue}
}
func (n *namedParameter) Type(p *Parameter) ParameterType {
	if p.up == nil || !p.up.CanSet {
		return NoParameter
	}
	if p.unit.Type == "send" && p.up.Name == "target" {
		return IDParameter
	}
	if p.up.MinValue >= -1 && p.up.MaxValue <= 1 {
		return BoolParameter
	}
	return IntegerParameter
}
func (n *namedParameter) Name(p *Parameter) string {
	if p.up.Name == "notetracking" {
		return "tracking" // notetracking does not fit in the UI
	}
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
func (n *namedParameter) RoundToGrid(p *Parameter, val int, up bool) int {
	if p.up.Name == "transpose" {
		return roundToGrid(val-64, 12, up) + 64
	}
	return roundToGrid(val, 8, up)
}
func (n *namedParameter) Reset(p *Parameter) {
	v, ok := defaultUnits[p.unit.Type].Parameters[p.up.Name]
	if !ok || p.unit.Parameters[p.up.Name] == v {
		return
	}
	defer p.m.change("Reset"+p.Name(), PatchChange, MinorChange)()
	p.unit.Parameters[p.up.Name] = v
}

// GmDlsEntry is a single sample entry from the gm.dls file
type GmDlsEntry struct {
	Start              int    // sample start offset in words
	LoopStart          int    // loop start offset in words
	LoopLength         int    // loop length in words
	SuggestedTranspose int    // suggested transpose in semitones, so that all samples play at same pitch
	Name               string // sample Name
}

// gmDlsEntryMap is a reverse map, to find the index of the GmDlsEntry in the
var gmDlsEntryMap = make(map[vm.SampleOffset]int)

func init() {
	for i, e := range GmDlsEntries {
		key := vm.SampleOffset{Start: uint32(e.Start), LoopStart: uint16(e.LoopStart), LoopLength: uint16(e.LoopLength)}
		gmDlsEntryMap[key] = i
	}
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
func (g *gmDlsEntryParameter) Range(p *Parameter) RangeInclusive {
	return RangeInclusive{Min: 0, Max: len(GmDlsEntries)}
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
func (g *gmDlsEntryParameter) RoundToGrid(p *Parameter, val int, up bool) int {
	return roundToGrid(val, 16, up)
}
func (g *gmDlsEntryParameter) Reset(p *Parameter) {}

// delayTimeParameter vtable

var delayNoteTrackGrid, delayBpmTrackGrid []int

func init() {
	for st := -30; st <= 30; st++ {
		gridVal := int(math.Exp2(float64(st)/12)*10787 + 0.5)
		delayNoteTrackGrid = append(delayNoteTrackGrid, gridVal)
	}
	for i := 0; i < 16; i++ {
		delayBpmTrackGrid = append(delayBpmTrackGrid, 1<<i)
		delayBpmTrackGrid = append(delayBpmTrackGrid, 3<<i)
		delayBpmTrackGrid = append(delayBpmTrackGrid, 9<<i)
	}
	slices.Sort(delayBpmTrackGrid)
}

func (d *delayTimeParameter) Type(p *Parameter) ParameterType { return IntegerParameter }
func (d *delayTimeParameter) Name(p *Parameter) string        { return "delaytime" }
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
func (d *delayTimeParameter) Range(p *Parameter) RangeInclusive {
	if p.unit.Parameters["notetracking"] == 2 {
		return RangeInclusive{Min: 1, Max: 576}
	}
	return RangeInclusive{Min: 1, Max: 65535}
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
		text = fmt.Sprintf("%.3f beats%s", float32(val)/48.0, text)
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
func (d *delayTimeParameter) RoundToGrid(p *Parameter, val int, up bool) int {
	switch p.unit.Parameters["notetracking"] {
	default:
		return roundToGrid(val, 16, up)
	case 1:
		return roundToSliceGrid(val, delayNoteTrackGrid, up)
	case 2:
		return roundToSliceGrid(val, delayBpmTrackGrid, up)
	}
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
func (d *delayLinesParameter) Range(p *Parameter) RangeInclusive {
	return RangeInclusive{Min: 1, Max: 32}
}
func (d *delayLinesParameter) Type(p *Parameter) ParameterType                { return IntegerParameter }
func (d *delayLinesParameter) Name(p *Parameter) string                       { return "delaylines" }
func (r *delayLinesParameter) RoundToGrid(p *Parameter, val int, up bool) int { return val }
func (d *delayLinesParameter) Hint(p *Parameter) ParameterHint {
	return ParameterHint{strconv.Itoa(d.Value(p)), true}
}
func (d *delayLinesParameter) LargeStep(p *Parameter) int {
	return 4
}
func (d *delayLinesParameter) Reset(p *Parameter) {}

// reverbParameter vtable

type delayPreset struct {
	name    string
	stereo  int
	varArgs []int
}

var reverbs = []delayPreset{
	{"stereo", 1, []int{1116, 1188, 1276, 1356, 1422, 1492, 1556, 1618,
		1140, 1212, 1300, 1380, 1446, 1516, 1580, 1642,
	}},
	{"left", 0, []int{1116, 1188, 1276, 1356, 1422, 1492, 1556, 1618}},
	{"right", 0, []int{1140, 1212, 1300, 1380, 1446, 1516, 1580, 1642}},
}

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
func (r *reverbParameter) Range(p *Parameter) RangeInclusive {
	return RangeInclusive{Min: 0, Max: len(reverbs)}
}
func (r *reverbParameter) Type(p *Parameter) ParameterType                { return IntegerParameter }
func (r *reverbParameter) Name(p *Parameter) string                       { return "reverb" }
func (r *reverbParameter) RoundToGrid(p *Parameter, val int, up bool) int { return val }
func (r *reverbParameter) Reset(p *Parameter)                             {}
func (r *reverbParameter) Hint(p *Parameter) ParameterHint {
	i := r.Value(p)
	label := "custom"
	if i > 0 {
		label = reverbs[i-1].name
	}
	return ParameterHint{label, true}
}

func roundToGrid(value, grid int, up bool) int {
	if up {
		return value + mod(-value, grid)
	}
	return value - mod(value, grid)
}

func mod(a, b int) int {
	m := a % b
	if a < 0 && b < 0 {
		m -= b
	}
	if a < 0 && b > 0 {
		m += b
	}
	return m
}

func roundToSliceGrid(value int, grid []int, up bool) int {
	if up {
		for _, v := range grid {
			if value < v {
				return v
			}
		}
	} else {
		for i := len(grid) - 1; i >= 0; i-- {
			if value > grid[i] {
				return grid[i]
			}
		}
	}
	return value
}
