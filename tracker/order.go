package tracker

import (
	"errors"

	"github.com/vsariola/sointu"
	"gopkg.in/yaml.v3"
)

// Order returns the Order view of the model, containing methods to manipulate
// the pattern order list.
func (m *Model) Order() *OrderModel { return (*OrderModel)(m) }

type OrderModel Model

// PatternUnique returns true if the given pattern in the given track is used
// only once in the pattern order list.
func (m *OrderModel) PatternUnique(track, pat int) bool {
	if track < 0 || track >= len(m.derived.tracks) {
		return false
	}
	if pat < 0 || pat >= len(m.derived.tracks[track].patternUseCounts) {
		return false
	}
	return m.derived.tracks[track].patternUseCounts[pat] <= 1
}

// AddRow returns an Action that adds an order row before or after the current
// cursor row.
func (m *OrderModel) AddRow(before bool) Action {
	return MakeAction(addOrderRow{Before: before, Model: (*Model)(m)})
}

type addOrderRow struct {
	Before bool
	*Model
}

func (a addOrderRow) Do() {
	m := a.Model
	defer m.change("AddOrderRowAction", ScoreChange, MinorChange)()
	if !a.Before {
		m.d.Cursor.OrderRow++
	}
	m.d.Cursor2.OrderRow = m.d.Cursor.OrderRow
	from := m.d.Cursor.OrderRow
	m.d.Song.Score.Length++
	for i := range m.d.Song.Score.Tracks {
		order := &m.d.Song.Score.Tracks[i].Order
		if len(*order) > from {
			*order = append(*order, -1)
			copy((*order)[from+1:], (*order)[from:])
			(*order)[from] = -1
		}
	}
}

// DeleteRow returns an Action to delete the current row of in the pattern order
// list.
func (m *OrderModel) DeleteRow(backwards bool) Action {
	return MakeAction(deleteOrderRow{Backwards: backwards, Model: (*Model)(m)})
}

type deleteOrderRow struct {
	Backwards bool
	*Model
}

func (d deleteOrderRow) Do() {
	m := d.Model
	defer m.change("DeleteOrderRowAction", ScoreChange, MinorChange)()
	from := m.d.Cursor.OrderRow
	m.d.Song.Score.Length--
	for i := range m.d.Song.Score.Tracks {
		order := &m.d.Song.Score.Tracks[i].Order
		if len(*order) > from {
			copy((*order)[from:], (*order)[from+1:])
			*order = (*order)[:len(*order)-1]
		}
	}
	if d.Backwards {
		if m.d.Cursor.OrderRow > 0 {
			m.d.Cursor.OrderRow--
		}
	}
	m.d.Cursor2.OrderRow = m.d.Cursor.OrderRow
}

// Table returns a Table of all the pattern order data.
func (v *OrderModel) Table() Table { return Table{v} }

func (m *OrderModel) Cursor() Point {
	t := max(min(m.d.Cursor.Track, len(m.d.Song.Score.Tracks)-1), 0)
	p := max(min(m.d.Cursor.OrderRow, m.d.Song.Score.Length-1), 0)
	return Point{t, p}
}

func (m *OrderModel) Cursor2() Point {
	t := max(min(m.d.Cursor2.Track, len(m.d.Song.Score.Tracks)-1), 0)
	p := max(min(m.d.Cursor2.OrderRow, m.d.Song.Score.Length-1), 0)
	return Point{t, p}
}

func (m *OrderModel) SetCursor(p Point) {
	m.d.Cursor.Track = max(min(p.X, len(m.d.Song.Score.Tracks)-1), 0)
	y := max(min(p.Y, m.d.Song.Score.Length-1), 0)
	if y != m.d.Cursor.OrderRow {
		m.follow = false
	}
	m.d.Cursor.OrderRow = y
	m.updateCursorRows()
}

func (m *OrderModel) SetCursor2(p Point) {
	m.d.Cursor2.Track = max(min(p.X, len(m.d.Song.Score.Tracks)-1), 0)
	m.d.Cursor2.OrderRow = max(min(p.Y, m.d.Song.Score.Length-1), 0)
	m.updateCursorRows()
}

func (v *OrderModel) updateCursorRows() {
	if v.Cursor() == v.Cursor2() {
		v.d.Cursor.PatternRow = 0
		v.d.Cursor2.PatternRow = 0
		return
	}
	if v.d.Cursor.OrderRow > v.d.Cursor2.OrderRow {
		v.d.Cursor.PatternRow = v.d.Song.Score.RowsPerPattern - 1
		v.d.Cursor2.PatternRow = 0
	} else {
		v.d.Cursor.PatternRow = 0
		v.d.Cursor2.PatternRow = v.d.Song.Score.RowsPerPattern - 1
	}
}

func (v *OrderModel) Width() int  { return len((*Model)(v).d.Song.Score.Tracks) }
func (v *OrderModel) Height() int { return (*Model)(v).d.Song.Score.Length }

func (v *OrderModel) MoveCursor(dx, dy int) (ok bool) {
	p := v.Cursor()
	p.X += dx
	p.Y += dy
	v.SetCursor(p)
	return p == v.Cursor()
}

func (m *OrderModel) clear(p Point) {
	m.d.Song.Score.Tracks[p.X].Order.Set(p.Y, -1)
}

func (m *OrderModel) set(p Point, value int) {
	m.d.Song.Score.Tracks[p.X].Order.Set(p.Y, value)
}

func (v *OrderModel) add(rect Rect, delta int, largeStep bool) (ok bool) {
	if largeStep {
		delta *= 8
	}
	for x := rect.TopLeft.X; x <= rect.BottomRight.X; x++ {
		for y := rect.TopLeft.Y; y <= rect.BottomRight.Y; y++ {
			if !v.add1(Point{x, y}, delta) {
				return false
			}
		}
	}
	return true
}

func (v *OrderModel) add1(p Point, delta int) (ok bool) {
	if p.X < 0 || p.X >= len(v.d.Song.Score.Tracks) {
		return true
	}
	val := v.d.Song.Score.Tracks[p.X].Order.Get(p.Y)
	if val < 0 {
		return true
	}
	val += delta
	if val < 0 || val > 36 {
		return false
	}
	v.d.Song.Score.Tracks[p.X].Order.Set(p.Y, val)
	return true
}

type marshalOrder struct {
	Order []int `yaml:",flow"`
}

type marshalTracks struct {
	Tracks []marshalOrder
}

func (m *OrderModel) marshal(rect Rect) (data []byte, ok bool) {
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

func (m *OrderModel) unmarshal(data []byte) (marshalTracks, bool) {
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

func (v *OrderModel) unmarshalAtCursor(data []byte) bool {
	table, ok := v.unmarshal(data)
	if !ok {
		return false
	}
	for i := 0; i < len(table.Tracks); i++ {
		for j, q := range table.Tracks[i].Order {
			if table.Tracks[i].Order[j] < -1 || table.Tracks[i].Order[j] > 36 {
				continue
			}
			x := i + v.Cursor().X
			y := j + v.Cursor().Y
			if x < 0 || x >= len(v.d.Song.Score.Tracks) || y < 0 || y >= v.d.Song.Score.Length {
				continue
			}
			v.d.Song.Score.Tracks[x].Order.Set(y, q)
		}
	}
	return true
}

func (v *OrderModel) unmarshalRange(rect Rect, data []byte) bool {
	table, ok := v.unmarshal(data)
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
			if x < 0 || x >= len(v.d.Song.Score.Tracks) || y < 0 || y >= v.d.Song.Score.Length {
				continue
			}
			v.d.Song.Score.Tracks[x].Order.Set(y, a)
		}
	}
	return true
}

func (v *OrderModel) change(kind string, severity ChangeSeverity) func() {
	return (*Model)(v).change("OrderTableView."+kind, ScoreChange, severity)
}

func (v *OrderModel) cancel() {
	v.changeCancel = true
}

func (m *OrderModel) Value(p Point) int {
	if p.X < 0 || p.X >= len(m.d.Song.Score.Tracks) {
		return -1
	}
	return m.d.Song.Score.Tracks[p.X].Order.Get(p.Y)
}

func (m *OrderModel) SetValue(p Point, val int) {
	defer (*Model)(m).change("OrderElement.SetValue", ScoreChange, MinorChange)()
	m.d.Song.Score.Tracks[p.X].Order.Set(p.Y, val)
}

// RowList returns a List of all the rows of the pattern order table.
func (m *OrderModel) RowList() List { return List{(*orderRows)(m)} }

type orderRows OrderModel

func (v *orderRows) Count() int             { return v.d.Song.Score.Length }
func (v *orderRows) Selected() int          { return v.d.Cursor.OrderRow }
func (v *orderRows) Selected2() int         { return v.d.Cursor2.OrderRow }
func (v *orderRows) SetSelected2(value int) { v.d.Cursor2.OrderRow = value }
func (v *orderRows) SetSelected(value int) {
	if value != v.d.Cursor.OrderRow {
		v.follow = false
	}
	v.d.Cursor.OrderRow = value
}

func (v *orderRows) Move(r Range, delta int) (ok bool) {
	swaps := r.Swaps(delta)
	for i, t := range v.d.Song.Score.Tracks {
		for a, b := range swaps {
			ea, eb := t.Order.Get(a), t.Order.Get(b)
			v.d.Song.Score.Tracks[i].Order.Set(a, eb)
			v.d.Song.Score.Tracks[i].Order.Set(b, ea)
		}
	}
	return true
}

func (v *orderRows) Delete(r Range) (ok bool) {
	for i, t := range v.d.Song.Score.Tracks {
		r2 := r.Intersect(Range{0, len(t.Order)})
		v.d.Song.Score.Tracks[i].Order = append(t.Order[:r2.Start], t.Order[r2.End:]...)
	}
	return true
}

func (v *orderRows) Change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("OrderRowList."+n, ScoreChange, severity)
}

func (v *orderRows) Cancel() {
	v.changeCancel = true
}

type marshalOrderRows struct {
	Columns [][]int `yaml:",flow"`
}

func (v *orderRows) Marshal(r Range) ([]byte, error) {
	var table marshalOrderRows
	for i := range v.d.Song.Score.Tracks {
		table.Columns = append(table.Columns, make([]int, r.Len()))
		for j := 0; j < r.Len(); j++ {
			table.Columns[i][j] = v.d.Song.Score.Tracks[i].Order.Get(r.Start + j)
		}
	}
	return yaml.Marshal(table)
}

func (v *orderRows) Unmarshal(data []byte) (r Range, err error) {
	var table marshalOrderRows
	err = yaml.Unmarshal(data, &table)
	if err != nil {
		return
	}
	if len(table.Columns) == 0 {
		err = errors.New("OrderRowList.unmarshal: no rows")
		return
	}
	r.Start = v.d.Cursor.OrderRow
	r.End = v.d.Cursor.OrderRow + len(table.Columns[0])
	for i := range v.d.Song.Score.Tracks {
		if i >= len(table.Columns) {
			break
		}
		order := &v.d.Song.Score.Tracks[i].Order
		for j := 0; j < r.Start-len(*order); j++ {
			*order = append(*order, -1)
		}
		if len(*order) > r.Start {
			table.Columns[i] = append(table.Columns[i], (*order)[r.Start:]...)
			*order = (*order)[:r.Start]
		}
		*order = append(*order, table.Columns[i]...)
	}
	return
}

// RemoveUnused returns an Action that removes all unused patterns from all
// tracks in the song, and updates the pattern orders accordingly.
func (m *OrderModel) RemoveUnusedPatterns() Action { return MakeAction((*removeUnused)(m)) }

type removeUnused OrderModel

func (m *removeUnused) Do() {
	defer (*Model)(m).change("RemoveUnusedAction", ScoreChange, MajorChange)()
	for trkIndex, trk := range m.d.Song.Score.Tracks {
		// assign new indices to patterns
		newIndex := map[int]int{}
		runningIndex := 0
		length := 0
		if len(trk.Order) > m.d.Song.Score.Length {
			trk.Order = trk.Order[:m.d.Song.Score.Length]
		}
		for i, p := range trk.Order {
			// if the pattern hasn't been considered and is within limits
			if _, ok := newIndex[p]; !ok && p >= 0 && p < len(trk.Patterns) {
				pat := trk.Patterns[p]
				useful := false
				for _, n := range pat { // patterns that have anything else than all holds are useful and to be kept
					if n != 1 {
						useful = true
						break
					}
				}
				if useful {
					newIndex[p] = runningIndex
					runningIndex++
				} else {
					newIndex[p] = -1
				}
			}
			if ind, ok := newIndex[p]; ok && ind > -1 {
				length = i + 1
				trk.Order[i] = ind
			} else {
				trk.Order[i] = -1
			}
		}
		trk.Order = trk.Order[:length]
		newPatterns := make([]sointu.Pattern, runningIndex)
		for i, pat := range trk.Patterns {
			if ind, ok := newIndex[i]; ok && ind > -1 {
				patLength := 0
				for j, note := range pat { // find last note that is something else that hold
					if note != 1 {
						patLength = j + 1
					}
				}
				if patLength > m.d.Song.Score.RowsPerPattern {
					patLength = m.d.Song.Score.RowsPerPattern
				}
				newPatterns[ind] = pat[:patLength] // crop to either RowsPerPattern or last row having something else than hold
			}
		}
		trk.Patterns = newPatterns
		m.d.Song.Score.Tracks[trkIndex] = trk
	}
}
