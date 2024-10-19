package tracker

import (
	"iter"
	"math"

	"github.com/vsariola/sointu"
	"gopkg.in/yaml.v3"
)

type (
	Table struct {
		TableData
	}

	TableData interface {
		Cursor() Point
		Cursor2() Point
		SetCursor(Point)
		SetCursor2(Point)
		SetCursorFloat(x, y float32)
		Width() int
		Height() int
		MoveCursor(dx, dy int) (ok bool)

		clear(p Point)
		set(p Point, value int)
		add(rect Rect, delta int) (ok bool)
		marshal(rect Rect) (data []byte, ok bool)
		unmarshalAtCursor(data []byte) (ok bool)
		unmarshalRange(rect Rect, data []byte) (ok bool)
		change(kind string, severity ChangeSeverity) func()
		cancel()
	}

	Point struct {
		X, Y int
	}

	Rect struct {
		TopLeft, BottomRight Point
	}

	Order Model
	Notes Model
)

// Model methods

func (m *Model) Order() *Order { return (*Order)(m) }
func (m *Model) Notes() *Notes { return (*Notes)(m) }

// Rect methods

func (r *Rect) Contains(p Point) bool {
	return r.TopLeft.X <= p.X && p.X <= r.BottomRight.X &&
		r.TopLeft.Y <= p.Y && p.Y <= r.BottomRight.Y
}

func (r *Rect) Width() int {
	return r.BottomRight.X - r.TopLeft.X + 1
}

func (r *Rect) Height() int {
	return r.BottomRight.Y - r.TopLeft.Y + 1
}

func (r *Rect) Limit(width, height int) {
	if r.TopLeft.X < 0 {
		r.TopLeft.X = 0
	}
	if r.TopLeft.Y < 0 {
		r.TopLeft.Y = 0
	}
	if r.BottomRight.X >= width {
		r.BottomRight.X = width - 1
	}
	if r.BottomRight.Y >= height {
		r.BottomRight.Y = height - 1
	}
}

// Table methods

func (v Table) Range() (rect Rect) {
	rect.TopLeft.X = intMin(v.Cursor().X, v.Cursor2().X)
	rect.TopLeft.Y = intMin(v.Cursor().Y, v.Cursor2().Y)
	rect.BottomRight.X = intMax(v.Cursor().X, v.Cursor2().X)
	rect.BottomRight.Y = intMax(v.Cursor().Y, v.Cursor2().Y)
	return
}

func (v Table) Copy() ([]byte, bool) {
	ret, ok := v.marshal(v.Range())
	if !ok {
		return nil, false
	}
	return ret, true
}

func (v Table) Paste(data []byte) bool {
	defer v.change("Paste", MajorChange)()
	if v.Cursor() == v.Cursor2() {
		return v.unmarshalAtCursor(data)
	} else {
		return v.unmarshalRange(v.Range(), data)
	}
}

func (v Table) Clear() {
	defer v.change("Clear", MajorChange)()
	rect := v.Range()
	rect.Limit(v.Width(), v.Height())
	for y := rect.TopLeft.Y; y <= rect.BottomRight.Y; y++ {
		for x := rect.TopLeft.X; x <= rect.BottomRight.X; x++ {
			v.clear(Point{x, y})
		}
	}
}

func (v Table) Set(value byte) {
	defer v.change("Set", MajorChange)()
	cursor := v.Cursor()
	// TODO: might check for visibility
	v.set(cursor, int(value))
}

func (v Table) Fill(value int) {
	defer v.change("Fill", MajorChange)()
	rect := v.Range()
	rect.Limit(v.Width(), v.Height())
	for y := rect.TopLeft.Y; y <= rect.BottomRight.Y; y++ {
		for x := rect.TopLeft.X; x <= rect.BottomRight.X; x++ {
			v.set(Point{x, y}, value)
		}
	}
}

func (v Table) Add(delta int) {
	defer v.change("Add", MinorChange)()
	if !v.add(v.Range(), delta) {
		v.cancel()
	}
}

func (v Table) SetCursorX(x int) {
	p := v.Cursor()
	p.X = x
	v.SetCursor(p)
}

func (v Table) SetCursorY(y int) {
	p := v.Cursor()
	p.Y = y
	v.SetCursor(p)
}

// Order methods

func (m *Order) Table() Table {
	return Table{m}
}

func (m *Order) Cursor() Point {
	t := intMax(intMin(m.d.Cursor.Track, len(m.d.Song.Score.Tracks)-1), 0)
	p := intMax(intMin(m.d.Cursor.OrderRow, m.d.Song.Score.Length-1), 0)
	return Point{t, p}
}

func (m *Order) Cursor2() Point {
	t := intMax(intMin(m.d.Cursor2.Track, len(m.d.Song.Score.Tracks)-1), 0)
	p := intMax(intMin(m.d.Cursor2.OrderRow, m.d.Song.Score.Length-1), 0)
	return Point{t, p}
}

func (m *Order) SetCursor(p Point) {
	m.d.Cursor.Track = intMax(intMin(p.X, len(m.d.Song.Score.Tracks)-1), 0)
	y := intMax(intMin(p.Y, m.d.Song.Score.Length-1), 0)
	if y != m.d.Cursor.OrderRow {
		m.follow = false
	}
	m.d.Cursor.OrderRow = y
	m.updateCursorRows()
}

func (m *Order) SetCursor2(p Point) {
	m.d.Cursor2.Track = intMax(intMin(p.X, len(m.d.Song.Score.Tracks)-1), 0)
	m.d.Cursor2.OrderRow = intMax(intMin(p.Y, m.d.Song.Score.Length-1), 0)
	m.updateCursorRows()
}

func (m *Order) SetCursorFloat(x, y float32) {
	m.SetCursor(Point{int(x), int(y)})
}

func (m *Order) updateCursorRows() {
	if m.Cursor() == m.Cursor2() {
		m.d.Cursor.PatternRow = 0
		m.d.Cursor2.PatternRow = 0
		return
	}
	if m.d.Cursor.OrderRow > m.d.Cursor2.OrderRow {
		m.d.Cursor.PatternRow = m.d.Song.Score.RowsPerPattern - 1
		m.d.Cursor2.PatternRow = 0
	} else {
		m.d.Cursor.PatternRow = 0
		m.d.Cursor2.PatternRow = m.d.Song.Score.RowsPerPattern - 1
	}
}

func (m *Order) Width() int {
	return len((*Model)(m).d.Song.Score.Tracks)
}

func (m *Order) Height() int {
	return (*Model)(m).d.Song.Score.Length
}

func (m *Order) MoveCursor(dx, dy int) (ok bool) {
	p := m.Cursor()
	p.X += dx
	p.Y += dy
	m.SetCursor(p)
	return p == m.Cursor()
}

func (m *Order) clear(p Point) {
	m.d.Song.Score.Tracks[p.X].Order.Set(p.Y, -1)
}

func (m *Order) set(p Point, value int) {
	m.d.Song.Score.Tracks[p.X].Order.Set(p.Y, value)
}

func (m *Order) add(rect Rect, delta int) (ok bool) {
	for x := rect.TopLeft.X; x <= rect.BottomRight.X; x++ {
		for y := rect.TopLeft.Y; y <= rect.BottomRight.Y; y++ {
			if !m.add1(Point{x, y}, delta) {
				return false
			}
		}
	}
	return true
}

func (m *Order) add1(p Point, delta int) (ok bool) {
	if p.X < 0 || p.X >= len(m.d.Song.Score.Tracks) {
		return true
	}
	val := m.d.Song.Score.Tracks[p.X].Order.Get(p.Y)
	if val < 0 {
		return true
	}
	val += delta
	if val < 0 || val > 36 {
		return false
	}
	m.d.Song.Score.Tracks[p.X].Order.Set(p.Y, val)
	return true
}

type marshalOrder struct {
	Order []int `yaml:",flow"`
}

type marshalTracks struct {
	Tracks []marshalOrder
}

func (m *Order) marshal(rect Rect) (data []byte, ok bool) {
	width := rect.BottomRight.X - rect.TopLeft.X + 1
	height := rect.BottomRight.Y - rect.TopLeft.Y + 1
	var table = marshalTracks{Tracks: make([]marshalOrder, 0, width)}
	for x := 0; x < width; x++ {
		ax := x + rect.TopLeft.X
		if ax < 0 || ax >= len(m.d.Song.Score.Tracks) {
			continue
		}
		table.Tracks = append(table.Tracks, marshalOrder{Order: make([]int, 0, rect.BottomRight.Y-rect.TopLeft.Y+1)})
		for y := 0; y < height; y++ {
			table.Tracks[x].Order = append(table.Tracks[x].Order, m.d.Song.Score.Tracks[ax].Order.Get(y+rect.TopLeft.Y))
		}
	}
	ret, err := yaml.Marshal(table)
	if err != nil {
		return nil, false
	}
	return ret, true
}

func (m *Order) unmarshal(data []byte) (marshalTracks, bool) {
	var table marshalTracks
	yaml.Unmarshal(data, &table)
	if len(table.Tracks) == 0 {
		return marshalTracks{}, false
	}
	for i := 0; i < len(table.Tracks); i++ {
		if len(table.Tracks[i].Order) > 0 {
			return table, true
		}
	}
	return marshalTracks{}, false
}

func (m *Order) unmarshalAtCursor(data []byte) bool {
	table, ok := m.unmarshal(data)
	if !ok {
		return false
	}
	for i := 0; i < len(table.Tracks); i++ {
		for j, q := range table.Tracks[i].Order {
			if table.Tracks[i].Order[j] < -1 || table.Tracks[i].Order[j] > 36 {
				continue
			}
			x := i + m.Cursor().X
			y := j + m.Cursor().Y
			if x < 0 || x >= len(m.d.Song.Score.Tracks) || y < 0 || y >= m.d.Song.Score.Length {
				continue
			}
			m.d.Song.Score.Tracks[x].Order.Set(y, q)
		}
	}
	return true
}

func (m *Order) unmarshalRange(rect Rect, data []byte) bool {
	table, ok := m.unmarshal(data)
	if !ok {
		return false
	}
	for i := 0; i < rect.Width(); i++ {
		for j := 0; j < rect.Height(); j++ {
			k := i % len(table.Tracks)
			l := j % len(table.Tracks[k].Order)
			a := table.Tracks[k].Order[l]
			if a < -1 || a > 36 {
				continue
			}
			x := i + rect.TopLeft.X
			y := j + rect.TopLeft.Y
			if x < 0 || x >= len(m.d.Song.Score.Tracks) || y < 0 || y >= m.d.Song.Score.Length {
				continue
			}
			m.d.Song.Score.Tracks[x].Order.Set(y, a)
		}
	}
	return true
}

func (m *Order) change(kind string, severity ChangeSeverity) func() {
	return (*Model)(m).change("OrderTableView."+kind, ScoreChange, severity)
}

func (m *Order) cancel() {
	m.changeCancel = true
}

func (m *Order) Value(p Point) int {
	if p.X < 0 || p.X >= len(m.d.Song.Score.Tracks) {
		return -1
	}
	return m.d.Song.Score.Tracks[p.X].Order.Get(p.Y)
}

func (m *Order) SetValue(p Point, val int) {
	defer (*Model)(m).change("OrderElement.SetValue", ScoreChange, MinorChange)()
	m.d.Song.Score.Tracks[p.X].Order.Set(p.Y, val)
}

func (e *Order) Title(x int) (title string) {
	title = "?"
	if x < 0 || x >= len(e.d.Song.Score.Tracks) {
		return
	}
	firstIndex, lastIndex, err := e.instrumentListFor(x)
	if err != nil {
		return
	}
	switch diff := lastIndex - firstIndex; diff {
	case 0:
		title = e.d.Song.Patch[firstIndex].Name
	default:
		n1 := e.d.Song.Patch[firstIndex].Name
		n2 := e.d.Song.Patch[firstIndex+1].Name
		if len(n1) > 0 {
			n1 = string(n1[0])
		} else {
			n1 = "?"
		}
		if len(n2) > 0 {
			n2 = string(n2[0])
		} else {
			n2 = "?"
		}
		if diff > 1 {
			title = n1 + "/" + n2 + "..."
		} else {
			title = n1 + "/" + n2
		}
	}
	return
}

func (e *Order) instrumentListFor(trackIndex int) (int, int, error) {
	track := e.d.Song.Score.Tracks[trackIndex]
	firstVoice := e.d.Song.Score.FirstVoiceForTrack(trackIndex)
	lastVoice := firstVoice + track.NumVoices - 1
	firstIndex, err1 := e.d.Song.Patch.InstrumentForVoice(firstVoice)
	if err1 != nil {
		return trackIndex, trackIndex, err1
	}
	lastIndex, err2 := e.d.Song.Patch.InstrumentForVoice(lastVoice)
	if err2 != nil {
		return trackIndex, trackIndex, err2
	}
	return firstIndex, lastIndex, nil
}

func (e *Order) TrackIndicesForCurrentInstrument() iter.Seq[int] {
	currentTrack := e.d.Cursor.Track
	return e.d.Song.AllTracksWithSameInstrument(currentTrack)
}

func (e *Order) CountNextTracksForCurrentInstrument() int {
	currentTrack := e.d.Cursor.Track
	count := 0
	for t := range e.TrackIndicesForCurrentInstrument() {
		if t > currentTrack {
			count++
		}
	}
	return count
}

// NoteTable

func (v *Notes) Table() Table {
	return Table{v}
}

func (m *Notes) Cursor() Point {
	t := intMax(intMin(m.d.Cursor.Track, len(m.d.Song.Score.Tracks)-1), 0)
	p := intMax(intMin(m.d.Song.Score.SongRow(m.d.Cursor.SongPos), m.d.Song.Score.LengthInRows()-1), 0)
	return Point{t, p}
}

func (m *Notes) Cursor2() Point {
	t := intMax(intMin(m.d.Cursor2.Track, len(m.d.Song.Score.Tracks)-1), 0)
	p := intMax(intMin(m.d.Song.Score.SongRow(m.d.Cursor2.SongPos), m.d.Song.Score.LengthInRows()-1), 0)
	return Point{t, p}
}

func (v *Notes) SetCursor(p Point) {
	v.d.Cursor.Track = intMax(intMin(p.X, len(v.d.Song.Score.Tracks)-1), 0)
	newPos := v.d.Song.Score.Clamp(sointu.SongPos{PatternRow: p.Y})
	if newPos != v.d.Cursor.SongPos {
		v.follow = false
	}
	v.d.Cursor.SongPos = newPos
}

func (v *Notes) SetCursor2(p Point) {
	v.d.Cursor2.Track = intMax(intMin(p.X, len(v.d.Song.Score.Tracks)-1), 0)
	v.d.Cursor2.SongPos = v.d.Song.Score.Clamp(sointu.SongPos{PatternRow: p.Y})
}

func (m *Notes) SetCursorFloat(x, y float32) {
	m.SetCursor(Point{int(x), int(y)})
	m.d.LowNibble = math.Mod(float64(x), 1.0) > 0.5
}

func (v *Notes) Width() int {
	return len((*Model)(v).d.Song.Score.Tracks)
}

func (v *Notes) Height() int {
	return (*Model)(v).d.Song.Score.Length * (*Model)(v).d.Song.Score.RowsPerPattern
}

func (v *Notes) MoveCursor(dx, dy int) (ok bool) {
	p := v.Cursor()
	for dx < 0 {
		if v.Effect(p.X) && v.d.LowNibble {
			v.d.LowNibble = false
		} else {
			p.X--
			v.d.LowNibble = true
		}
		dx++
	}
	for dx > 0 {
		if v.Effect(p.X) && !v.d.LowNibble {
			v.d.LowNibble = true
		} else {
			p.X++
			v.d.LowNibble = false
		}
		dx--
	}
	p.Y += dy
	v.SetCursor(p)
	return p == v.Cursor()
}

func (v *Notes) clear(p Point) {
	v.SetValue(p, 1)
}

func (v *Notes) set(p Point, value int) {
	v.SetValue(p, byte(value))
}

func (v *Notes) add(rect Rect, delta int) (ok bool) {
	for x := rect.BottomRight.X; x >= rect.TopLeft.X; x-- {
		for y := rect.BottomRight.Y; y >= rect.TopLeft.Y; y-- {
			if x < 0 || x >= len(v.d.Song.Score.Tracks) || y < 0 || y >= v.d.Song.Score.LengthInRows() {
				continue
			}
			pos := v.d.Song.Score.SongPos(y)
			note := v.d.Song.Score.Tracks[x].Note(pos)
			if note <= 1 {
				continue
			}
			newVal := int(note) + delta
			if newVal < 2 {
				newVal = 2
			} else if newVal > 255 {
				newVal = 255
			}
			// only do all sets after all gets, so we don't accidentally adjust single note multiple times
			defer v.d.Song.Score.Tracks[x].SetNote(pos, byte(newVal), v.uniquePatterns)
		}
	}
	return true
}

type noteTable struct {
	Notes [][]byte `yaml:",flow"`
}

func (m *Notes) marshal(rect Rect) (data []byte, ok bool) {
	width := rect.BottomRight.X - rect.TopLeft.X + 1
	height := rect.BottomRight.Y - rect.TopLeft.Y + 1
	var table = noteTable{Notes: make([][]byte, 0, width)}
	for x := 0; x < width; x++ {
		table.Notes = append(table.Notes, make([]byte, 0, rect.BottomRight.Y-rect.TopLeft.Y+1))
		for y := 0; y < height; y++ {
			pos := m.d.Song.Score.SongPos(y + rect.TopLeft.Y)
			ax := x + rect.TopLeft.X
			if ax < 0 || ax >= len(m.d.Song.Score.Tracks) {
				continue
			}
			table.Notes[x] = append(table.Notes[x], m.d.Song.Score.Tracks[ax].Note(pos))
		}
	}
	ret, err := yaml.Marshal(table)
	if err != nil {
		return nil, false
	}
	return ret, true
}

func (v *Notes) unmarshal(data []byte) (noteTable, bool) {
	var table noteTable
	yaml.Unmarshal(data, &table)
	if len(table.Notes) == 0 {
		return noteTable{}, false
	}
	for i := 0; i < len(table.Notes); i++ {
		if len(table.Notes[i]) > 0 {
			return table, true
		}
	}
	return noteTable{}, false
}

func (v *Notes) unmarshalAtCursor(data []byte) bool {
	table, ok := v.unmarshal(data)
	if !ok {
		return false
	}
	for i := 0; i < len(table.Notes); i++ {
		for j, q := range table.Notes[i] {
			x := i + v.Cursor().X
			y := j + v.Cursor().Y
			if x < 0 || x >= len(v.d.Song.Score.Tracks) || y < 0 || y >= v.d.Song.Score.LengthInRows() {
				continue
			}
			pos := v.d.Song.Score.SongPos(y)
			v.d.Song.Score.Tracks[x].SetNote(pos, q, v.uniquePatterns)
		}
	}
	return true
}

func (v *Notes) unmarshalRange(rect Rect, data []byte) bool {
	table, ok := v.unmarshal(data)
	if !ok {
		return false
	}
	for i := 0; i < rect.Width(); i++ {
		for j := 0; j < rect.Height(); j++ {
			k := i % len(table.Notes)
			l := j % len(table.Notes[k])
			a := table.Notes[k][l]
			x := i + rect.TopLeft.X
			y := j + rect.TopLeft.Y
			if x < 0 || x >= len(v.d.Song.Score.Tracks) || y < 0 || y >= v.d.Song.Score.LengthInRows() {
				continue
			}
			pos := v.d.Song.Score.SongPos(y)
			v.d.Song.Score.Tracks[x].SetNote(pos, a, v.uniquePatterns)
		}
	}
	return true
}

func (v *Notes) change(kind string, severity ChangeSeverity) func() {
	return (*Model)(v).change("OrderTableView."+kind, ScoreChange, severity)
}

func (v *Notes) cancel() {
	v.changeCancel = true
}

func (m *Notes) Value(p Point) byte {
	if p.Y < 0 || p.X < 0 || p.X >= len(m.d.Song.Score.Tracks) {
		return 1
	}
	pos := m.d.Song.Score.SongPos(p.Y)
	return m.d.Song.Score.Tracks[p.X].Note(pos)
}

func (m *Notes) Effect(x int) bool {
	if x < 0 || x >= len(m.d.Song.Score.Tracks) {
		return false
	}
	return m.d.Song.Score.Tracks[x].Effect
}

func (m *Notes) LowNibble() bool {
	return m.d.LowNibble
}

func (m *Notes) Unique(t, p int) bool {
	if t < 0 || t >= len(m.cachePatternUseCount) || p < 0 || p >= len(m.cachePatternUseCount[t]) {
		return false
	}
	return m.cachePatternUseCount[t][p] == 1
}

func (m *Notes) SetValue(p Point, val byte) {
	defer m.change("SetValue", MinorChange)()
	if p.Y < 0 || p.X < 0 || p.X >= len(m.d.Song.Score.Tracks) {
		return
	}
	track := &(m.d.Song.Score.Tracks[p.X])
	pos := m.d.Song.Score.SongPos(p.Y)
	(*track).SetNote(pos, val, m.uniquePatterns)
}

func (v *Notes) FillNibble(value byte, lowNibble bool) {
	defer v.change("FillNibble", MajorChange)()
	rect := Table{v}.Range()
	for y := rect.TopLeft.Y; y <= rect.BottomRight.Y; y++ {
		for x := rect.TopLeft.X; x <= rect.BottomRight.X; x++ {
			val := v.Value(Point{x, y})
			if val == 1 {
				val = 0 // treat hold also as 0
			}
			if lowNibble {
				val = (val & 0xf0) | byte(value&15)
			} else {
				val = (val & 0x0f) | byte((value&15)<<4)
			}
			v.SetValue(Point{x, y}, val)
		}
	}
}
