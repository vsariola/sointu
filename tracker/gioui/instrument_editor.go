package gioui

import (
	"bytes"
	"image"
	"image/color"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"gioui.org/f32"
	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/tracker"
	"golang.org/x/exp/shiny/materialdesign/icons"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type (
	InstrumentEditor struct {
		unitList   UnitList
		unitEditor UnitEditor
	}

	UnitList struct {
		dragList      *DragList
		searchEditor  *Editor
		addUnitBtn    *Clickable
		addUnitAction tracker.Action
	}

	UnitEditor struct {
		paramTable     *ScrollTable
		searchList     *DragList
		Parameters     [][]*ParamState
		DeleteUnitBtn  *Clickable
		CopyUnitBtn    *Clickable
		ClearUnitBtn   *Clickable
		DisableUnitBtn *Clickable
		SelectTypeBtn  *Clickable
		commentEditor  *Editor
		caser          cases.Caser

		copyHint        string
		disableUnitHint string
		enableUnitHint  string

		searching tracker.Bool
	}
)

func MakeInstrumentEditor(model *tracker.Model) InstrumentEditor {
	return InstrumentEditor{
		unitList:   MakeUnitList(model),
		unitEditor: *NewUnitEditor(model),
	}
}

func (ie *InstrumentEditor) layout(gtx C) D {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(ie.unitList.Layout),
		layout.Flexed(1, ie.unitEditor.Layout),
	)
}

func (ie *InstrumentEditor) Tags(level int, yield TagYieldFunc) bool {
	return ie.unitList.Tags(level, yield) &&
		ie.unitEditor.Tags(level, yield)
}

// UnitList methods

func MakeUnitList(m *tracker.Model) UnitList {
	ret := UnitList{
		dragList:     NewDragList(m.Units().List(), layout.Vertical),
		addUnitBtn:   new(Clickable),
		searchEditor: NewEditor(true, true, text.Start),
	}
	ret.addUnitAction = tracker.MakeEnabledAction(tracker.DoFunc(func() {
		m.AddUnit(false).Do()
		ret.searchEditor.Focus()
	}))
	return ret
}

func (ul *UnitList) Layout(gtx C) D {
	t := TrackerFromContext(gtx)
	ul.update(gtx, t)
	element := func(gtx C, i int) D {
		gtx.Constraints.Max.Y = gtx.Dp(20)
		gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
		u := t.Units().Item(i)
		editorStyle := t.Theme.InstrumentEditor.UnitList.Name
		signalError := t.RailError()
		switch {
		case u.Disabled:
			editorStyle = t.Theme.InstrumentEditor.UnitList.NameDisabled
		case signalError.Err != nil && signalError.UnitIndex == i:
			editorStyle.Color = t.Theme.InstrumentEditor.UnitList.Error
		}
		unitName := func(gtx C) D {
			if i == ul.dragList.TrackerList.Selected() {
				defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
				return ul.searchEditor.Layout(gtx, t.Model.UnitSearch(), t.Theme, &editorStyle, "---")
			} else {
				text := u.Type
				if text == "" {
					text = "---"
				}
				l := editorStyle.AsLabelStyle()
				return Label(t.Theme, &l, text).Layout(gtx)
			}
		}
		stackText := strconv.FormatInt(int64(u.Signals.StackAfter()), 10)
		commentLabel := Label(t.Theme, &t.Theme.InstrumentEditor.UnitList.Comment, u.Comment)
		stackLabel := Label(t.Theme, &t.Theme.InstrumentEditor.UnitList.Stack, stackText)
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(unitName),
			layout.Rigid(layout.Spacer{Width: 5}.Layout),
			layout.Flexed(1, commentLabel.Layout),
			layout.Rigid(stackLabel.Layout),
			layout.Rigid(layout.Spacer{Width: 10}.Layout),
		)
	}
	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	unitList := FilledDragList(t.Theme, ul.dragList)
	surface := func(gtx C) D {
		return layout.Stack{Alignment: layout.SE}.Layout(gtx,
			layout.Expanded(func(gtx C) D {
				defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
				gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(140), gtx.Constraints.Max.Y))
				dims := unitList.Layout(gtx, element, nil)
				unitList.LayoutScrollBar(gtx)
				return dims
			}),
			layout.Stacked(func(gtx C) D {
				margin := layout.Inset{Right: unit.Dp(20), Bottom: unit.Dp(1)}
				addUnitBtn := IconBtn(t.Theme, &t.Theme.IconButton.Emphasis, ul.addUnitBtn, icons.ContentAdd, "Add unit (Enter)")
				return margin.Layout(gtx, addUnitBtn.Layout)
			}),
		)
	}
	return Surface{Gray: 30, Focus: t.PatchPanel.TreeFocused(gtx)}.Layout(gtx, surface)
}

func (ul *UnitList) update(gtx C, t *Tracker) {
	for ul.addUnitBtn.Clicked(gtx) {
		ul.addUnitAction.Do()
		t.UnitSearching().SetValue(true)
		ul.searchEditor.Focus()
	}
	for {
		event, ok := gtx.Event(
			key.Filter{Focus: ul.dragList, Name: key.NameRightArrow},
			key.Filter{Focus: ul.dragList, Name: key.NameEnter, Optional: key.ModCtrl},
			key.Filter{Focus: ul.dragList, Name: key.NameReturn, Optional: key.ModCtrl},
			key.Filter{Focus: ul.dragList, Name: key.NameDeleteBackward},
		)
		if !ok {
			break
		}
		if e, ok := event.(key.Event); ok && e.State == key.Press {
			switch e.Name {
			case key.NameRightArrow:
				t.PatchPanel.instrEditor.unitEditor.paramTable.RowTitleList.Focus()
			case key.NameDeleteBackward:
				t.Units().SetSelectedType("")
				t.UnitSearching().SetValue(true)
				ul.searchEditor.Focus()
			case key.NameEnter, key.NameReturn:
				t.Model.AddUnit(e.Modifiers.Contain(key.ModCtrl)).Do()
				t.UnitSearching().SetValue(true)
				ul.searchEditor.Focus()
			}
		}
	}
	str := t.Model.UnitSearch()
	for ev := ul.searchEditor.Update(gtx, str); ev != EditorEventNone; ev = ul.searchEditor.Update(gtx, str) {
		if ev == EditorEventSubmit {
			if str.Value() != "" {
				for _, n := range sointu.UnitNames {
					if strings.HasPrefix(n, str.Value()) {
						t.Units().SetSelectedType(n)
						break
					}
				}
			} else {
				t.Units().SetSelectedType("")
			}
		}
		ul.dragList.Focus()
		t.UnitSearching().SetValue(false)
	}
}

func (ul *UnitList) Tags(curLevel int, yield TagYieldFunc) bool {
	return yield(curLevel, ul.dragList) && yield(curLevel+1, &ul.searchEditor.widgetEditor)
}

func NewUnitEditor(m *tracker.Model) *UnitEditor {
	ret := &UnitEditor{
		DeleteUnitBtn:  new(Clickable),
		ClearUnitBtn:   new(Clickable),
		DisableUnitBtn: new(Clickable),
		CopyUnitBtn:    new(Clickable),
		SelectTypeBtn:  new(Clickable),
		commentEditor:  NewEditor(true, true, text.Start),
		paramTable:     NewScrollTable(m.Params().Table(), m.ParamVertList().List(), m.Units().List()),
		searchList:     NewDragList(m.SearchResults().List(), layout.Vertical),
		searching:      m.UnitSearching(),
	}
	ret.caser = cases.Title(language.English)
	ret.copyHint = makeHint("Copy unit", " (%s)", "Copy")
	ret.disableUnitHint = makeHint("Disable unit", " (%s)", "UnitDisabledToggle")
	ret.enableUnitHint = makeHint("Enable unit", " (%s)", "UnitDisabledToggle")
	return ret
}

func (pe *UnitEditor) Layout(gtx C) D {
	t := TrackerFromContext(gtx)
	pe.update(gtx, t)
	editorFunc := pe.layoutRack
	if pe.showingChooser() {
		editorFunc = pe.layoutUnitTypeChooser
	}
	return Surface{Gray: 24, Focus: t.PatchPanel.TreeFocused(gtx)}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Flexed(1, editorFunc),
			layout.Rigid(pe.layoutFooter),
		)
	})
}

func (pe *UnitEditor) showingChooser() bool {
	return pe.searching.Value()
}

func (pe *UnitEditor) update(gtx C, t *Tracker) {
	for pe.CopyUnitBtn.Clicked(gtx) {
		if contents, ok := t.Units().List().CopyElements(); ok {
			gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(bytes.NewReader(contents))})
			t.Alerts().Add("Unit(s) copied to clipboard", tracker.Info)
		}
	}
	for pe.SelectTypeBtn.Clicked(gtx) {
		pe.ChooseUnitType(t)
	}
	for pe.commentEditor.Update(gtx, t.UnitComment()) != EditorEventNone {
		t.FocusPrev(gtx, false)
	}
	for pe.ClearUnitBtn.Clicked(gtx) {
		t.ClearUnit().Do()
		t.UnitSearch().SetValue("")
		t.UnitSearching().SetValue(true)
		pe.searchList.Focus()
	}
	for {
		e, ok := gtx.Event(
			key.Filter{Focus: pe.searchList, Name: key.NameEnter},
			key.Filter{Focus: pe.searchList, Name: key.NameReturn},
			key.Filter{Focus: pe.searchList, Name: key.NameEscape},
		)
		if !ok {
			break
		}
		if e, ok := e.(key.Event); ok && e.State == key.Press {
			switch e.Name {
			case key.NameEscape:
				t.UnitSearching().SetValue(false)
				pe.paramTable.RowTitleList.Focus()
			case key.NameEnter, key.NameReturn:
				pe.ChooseUnitType(t)
			}
		}
	}
	for {
		e, ok := gtx.Event(
			key.Filter{Focus: pe.paramTable, Name: key.NameLeftArrow, Required: key.ModShift, Optional: key.ModShortcut},
			key.Filter{Focus: pe.paramTable, Name: key.NameRightArrow, Required: key.ModShift, Optional: key.ModShortcut},
			key.Filter{Focus: pe.paramTable, Name: key.NameDeleteBackward},
			key.Filter{Focus: pe.paramTable, Name: key.NameDeleteForward},
		)
		if !ok {
			break
		}
		if e, ok := e.(key.Event); ok && e.State == key.Press {
			switch e.Name {
			case key.NameLeftArrow:
				t.Model.Params().Table().Add(-1, e.Modifiers.Contain(key.ModShortcut))
			case key.NameRightArrow:
				t.Model.Params().Table().Add(1, e.Modifiers.Contain(key.ModShortcut))
			case key.NameDeleteBackward, key.NameDeleteForward:
				t.Model.Params().Table().Clear()
			}
			c := t.Model.Params().Cursor()
			if c.X >= 0 && c.Y >= 0 && c.Y < len(pe.Parameters) && c.X < len(pe.Parameters[c.Y]) {
				ta := &pe.Parameters[c.Y][c.X].tipArea
				ta.Appear(gtx.Now)
				ta.Exit.SetTarget(gtx.Now.Add(ta.ExitDuration))
			}
		}
	}
	for {
		e, ok := gtx.Event(
			key.Filter{Focus: pe.paramTable.RowTitleList, Name: key.NameEnter},
			key.Filter{Focus: pe.paramTable.RowTitleList, Name: key.NameReturn},
			key.Filter{Focus: pe.paramTable.RowTitleList, Name: key.NameLeftArrow},
			key.Filter{Focus: pe.paramTable.RowTitleList, Name: key.NameDeleteBackward},
		)
		if !ok {
			break
		}
		if e, ok := e.(key.Event); ok && e.State == key.Press {
			switch e.Name {
			case key.NameLeftArrow:
				t.PatchPanel.instrEditor.unitList.dragList.Focus()
			case key.NameDeleteBackward:
				t.ClearUnit().Do()
				t.UnitSearch().SetValue("")
				t.UnitSearching().SetValue(true)
				pe.searchList.Focus()
			case key.NameEnter, key.NameReturn:
				t.Model.AddUnit(e.Modifiers.Contain(key.ModCtrl)).Do()
				t.UnitSearch().SetValue("")
				t.UnitSearching().SetValue(true)
				pe.searchList.Focus()
			}
		}
	}
}

func (pe *UnitEditor) ChooseUnitType(t *Tracker) {
	if ut, ok := t.SearchResults().Item(pe.searchList.TrackerList.Selected()); ok {
		t.Units().SetSelectedType(ut)
		pe.paramTable.RowTitleList.Focus()
	}
}

func (pe *UnitEditor) layoutRack(gtx C) D {
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
	t := TrackerFromContext(gtx)
	// create enough parameter widget to match the number of parameters
	width := pe.paramTable.Table.Width()
	for len(pe.Parameters) < pe.paramTable.Table.Height() {
		pe.Parameters = append(pe.Parameters, make([]*ParamState, 0))
	}
	cellWidth := gtx.Dp(t.Theme.UnitEditor.Width)
	cellHeight := gtx.Dp(t.Theme.UnitEditor.Height)
	rowTitleLabelWidth := gtx.Dp(t.Theme.UnitEditor.UnitList.LabelWidth)
	rowTitleSignalWidth := gtx.Dp(t.Theme.SignalRail.SignalWidth) * t.RailWidth()
	rowTitleWidth := rowTitleLabelWidth + rowTitleSignalWidth
	signalError := t.RailError()
	columnTitleHeight := gtx.Dp(0)
	for i := range pe.Parameters {
		for len(pe.Parameters[i]) < width {
			pe.Parameters[i] = append(pe.Parameters[i], &ParamState{tipArea: TipArea{ExitDuration: time.Second * 2}})
		}
	}
	coltitle := func(gtx C, x int) D {
		return D{Size: image.Pt(cellWidth, columnTitleHeight)}
	}
	rowtitle := func(gtx C, y int) D {
		if y < 0 || y >= len(pe.Parameters) {
			return D{}
		}
		item := t.Units().Item(y)
		sr := Rail(t.Theme, item.Signals)
		label := Label(t.Theme, &t.Theme.UnitEditor.UnitList.Name, item.Type)
		switch {
		case item.Disabled:
			label.LabelStyle = t.Theme.UnitEditor.UnitList.Disabled
		case signalError.Err != nil && signalError.UnitIndex == y:
			label.Color = t.Theme.UnitEditor.UnitList.Error
		}
		gtx.Constraints = layout.Exact(image.Pt(rowTitleWidth, cellHeight))
		sr.Layout(gtx)
		defer op.Affine(f32.Affine2D{}.Rotate(f32.Pt(0, 0), -90*math.Pi/180).Offset(f32.Point{X: float32(rowTitleSignalWidth), Y: float32(cellHeight)})).Push(gtx.Ops).Pop()
		gtx.Constraints = layout.Exact(image.Pt(cellHeight, rowTitleLabelWidth))
		label.Layout(gtx)
		return D{Size: image.Pt(rowTitleWidth, cellHeight)}
	}
	cursor := t.Model.Params().Cursor()
	cell := func(gtx C, x, y int) D {
		gtx.Constraints = layout.Exact(image.Pt(cellWidth, cellHeight))
		point := tracker.Point{X: x, Y: y}
		if y < 0 || y >= len(pe.Parameters) || x < 0 || x >= len(pe.Parameters[y]) {
			return D{}
		}
		selection := pe.paramTable.Table.Range()
		if selection.Contains(point) {
			color := t.Theme.Selection.Inactive
			if gtx.Focused(pe.paramTable) {
				color = t.Theme.Selection.Active
			}
			if point == cursor {
				color = t.Theme.Cursor.Inactive
				if gtx.Focused(pe.paramTable) {
					color = t.Theme.Cursor.Active
				}
			}
			paint.FillShape(gtx.Ops, color, clip.Rect{Min: image.Pt(0, 0), Max: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}.Op())
		}

		param := t.Model.Params().Item(point)
		paramStyle := Param(param, t.Theme, pe.Parameters[y][x], pe.paramTable.Table.Cursor() == point, t.Units().Item(y).Disabled)
		paramStyle.Layout(gtx)
		comment := t.Units().Item(y).Comment
		if comment != "" && x == t.Model.Params().RowWidth(y) {
			label := Label(t.Theme, &t.Theme.UnitEditor.RackComment, comment)
			return layout.W.Layout(gtx, func(gtx C) D {
				gtx.Constraints.Max.X = 1e6
				gtx.Constraints.Min.Y = 0
				return label.Layout(gtx)
			})
		}
		return D{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}
	}
	table := FilledScrollTable(t.Theme, pe.paramTable)
	table.RowTitleWidth = gtx.Metric.PxToDp(rowTitleWidth)
	table.ColumnTitleHeight = 0
	table.CellWidth = t.Theme.UnitEditor.Width
	table.CellHeight = t.Theme.UnitEditor.Height
	pe.drawBackGround(gtx)
	pe.drawSignals(gtx, rowTitleWidth)
	dims := table.Layout(gtx, cell, coltitle, rowtitle, nil, nil)
	return dims
}

func (pe *UnitEditor) drawSignals(gtx C, rowTitleWidth int) {
	t := TrackerFromContext(gtx)
	colP := pe.paramTable.ColTitleList.List.Position
	rowP := pe.paramTable.RowTitleList.List.Position
	p := image.Pt(rowTitleWidth, 0)
	defer op.Offset(p).Push(gtx.Ops).Pop()
	gtx.Constraints.Max = gtx.Constraints.Max.Sub(p)
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	defer op.Offset(image.Pt(-colP.Offset, -rowP.Offset)).Push(gtx.Ops).Pop()
	for wire := range t.Wires {
		clr := t.Theme.UnitEditor.WireColor
		if wire.Highlight {
			clr = t.Theme.UnitEditor.WireHighlight
		}
		switch {
		case wire.FromSet && !wire.ToSet:
			pe.drawRemoteSendSignal(gtx, wire, colP.First, rowP.First)
		case !wire.FromSet && wire.ToSet:
			pe.drawRemoteReceiveSignal(gtx, wire, colP.First, rowP.First, clr)
		case wire.FromSet && wire.ToSet:
			pe.drawSignal(gtx, wire, colP.First, rowP.First, clr)
		}
	}
}

func (pe *UnitEditor) drawBackGround(gtx C) {
	t := TrackerFromContext(gtx)
	rowP := pe.paramTable.RowTitleList.List.Position
	defer op.Offset(image.Pt(0, -rowP.Offset)).Push(gtx.Ops).Pop()
	for range pe.paramTable.RowTitleList.List.Position.Count + 1 {
		paint.FillShape(gtx.Ops, t.Theme.UnitEditor.Divider, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, 1)}.Op())
		op.Offset(image.Pt(0, gtx.Dp(t.Theme.UnitEditor.Height))).Add(gtx.Ops)
	}
}

func (pe *UnitEditor) drawRemoteSendSignal(gtx C, wire tracker.Wire, col, row int) {
	sy := wire.From - row
	t := TrackerFromContext(gtx)
	defer op.Offset(image.Pt(gtx.Dp(5), (sy+1)*gtx.Dp(t.Theme.UnitEditor.Height)-gtx.Dp(16))).Push(gtx.Ops).Pop()
	Label(t.Theme, &t.Theme.UnitEditor.WireHint, wire.Hint).Layout(gtx)
}

func (pe *UnitEditor) drawRemoteReceiveSignal(gtx C, wire tracker.Wire, col, row int, clr color.NRGBA) {
	ex := wire.To.X - col
	ey := wire.To.Y - row
	t := TrackerFromContext(gtx)
	width := float32(gtx.Dp(t.Theme.UnitEditor.Width))
	height := float32(gtx.Dp(t.Theme.UnitEditor.Height))
	topLeft := f32.Pt(float32(ex)*width, float32(ey)*height)
	center := topLeft.Add(f32.Pt(width/2, height/2))
	c := float32(gtx.Dp(t.Theme.Knob.Diameter)) / 2 / float32(math.Sqrt2)
	from := f32.Pt(c, c).Add(center)
	q := c
	c1 := f32.Pt(c+q, c+q).Add(center)
	o := float32(gtx.Dp(8))
	c2 := f32.Pt(width-q, height-o).Add(topLeft)
	to := f32.Pt(width, height-o).Add(topLeft)
	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(from)
	path.CubeTo(c1, c2, to)
	paint.FillShape(gtx.Ops, clr,
		clip.Stroke{
			Path:  path.End(),
			Width: float32(gtx.Dp(t.Theme.SignalRail.LineWidth)),
		}.Op())
	defer op.Offset(image.Pt((ex+1)*gtx.Dp(t.Theme.UnitEditor.Width)+gtx.Dp(5), (ey+1)*gtx.Dp(t.Theme.UnitEditor.Height)-gtx.Dp(16))).Push(gtx.Ops).Pop()
	Label(t.Theme, &t.Theme.UnitEditor.WireHint, wire.Hint).Layout(gtx)
}

func (pe *UnitEditor) drawSignal(gtx C, wire tracker.Wire, col, row int, clr color.NRGBA) {
	sy := wire.From - row
	ex := wire.To.X - col
	ey := wire.To.Y - row
	t := TrackerFromContext(gtx)
	diam := gtx.Dp(t.Theme.Knob.Diameter)
	c := float32(diam) / 2 / float32(math.Sqrt2)
	width := float32(gtx.Dp(t.Theme.UnitEditor.Width))
	height := float32(gtx.Dp(t.Theme.UnitEditor.Height))
	from := f32.Pt(0, float32((sy+1)*gtx.Dp(t.Theme.UnitEditor.Height))-float32(gtx.Dp(8)))
	corner := f32.Pt(1, 1)
	if ex > 0 {
		corner.X = -corner.X
	}
	if sy < ey {
		corner.Y = -corner.Y
	}
	topLeft := f32.Pt(float32(ex)*width, float32(ey)*height)
	center := topLeft.Add(f32.Pt(width/2, height/2))
	to := mulVec(corner, f32.Pt(c, c)).Add(center)
	p2 := mulVec(corner, f32.Pt(width/2, height/2)).Add(center)
	p1 := f32.Pt(p2.X, float32((sy+1)*gtx.Dp(t.Theme.UnitEditor.Height)))
	if sy > ey {
		p1 = f32.Pt(p2.X, (float32(sy)+0.5)*float32(gtx.Dp(t.Theme.UnitEditor.Height))+float32(diam)/2)
	}
	k := float32(width) / 4
	p2Tan := mulVec(corner, f32.Pt(-k, -k))
	p1Tan := f32.Pt(k, p2Tan.Y)
	fromTan := f32.Pt(k, 0)
	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(from)
	path.CubeTo(from.Add(fromTan), p1.Sub(p1Tan), p1)
	path.CubeTo(p1.Add(p1Tan), p2, to)
	paint.FillShape(gtx.Ops, clr,
		clip.Stroke{
			Path:  path.End(),
			Width: float32(gtx.Dp(t.Theme.SignalRail.LineWidth)),
		}.Op())
}

func mulVec(a, b f32.Point) f32.Point {
	return f32.Pt(a.X*b.X, a.Y*b.Y)
}

func (pe *UnitEditor) layoutFooter(gtx C) D {
	t := TrackerFromContext(gtx)
	text := "Choose unit type"
	if !t.UnitSearching().Value() {
		text = pe.caser.String(t.Units().SelectedType())
	}
	hintText := Label(t.Theme, &t.Theme.UnitEditor.Hint, text)
	deleteUnitBtn := ActionIconBtn(t.DeleteUnit(), t.Theme, pe.DeleteUnitBtn, icons.ActionDelete, "Delete unit (Ctrl+Backspace)")
	copyUnitBtn := IconBtn(t.Theme, &t.Theme.IconButton.Enabled, pe.CopyUnitBtn, icons.ContentContentCopy, pe.copyHint)
	disableUnitBtn := ToggleIconBtn(t.UnitDisabled(), t.Theme, pe.DisableUnitBtn, icons.AVVolumeUp, icons.AVVolumeOff, pe.disableUnitHint, pe.enableUnitHint)
	clearUnitBtn := IconBtn(t.Theme, &t.Theme.IconButton.Enabled, pe.ClearUnitBtn, icons.ContentClear, "Clear unit")
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(deleteUnitBtn.Layout),
		layout.Rigid(copyUnitBtn.Layout),
		layout.Rigid(disableUnitBtn.Layout),
		layout.Rigid(clearUnitBtn.Layout),
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Dp(130)
			return hintText.Layout(gtx)
		}),
		layout.Flexed(1, func(gtx C) D {
			return pe.commentEditor.Layout(gtx, t.UnitComment(), t.Theme, &t.Theme.InstrumentEditor.UnitComment, "Comment")
		}),
	)
}

func (pe *UnitEditor) layoutUnitTypeChooser(gtx C) D {
	t := TrackerFromContext(gtx)
	var namesArray [256]string
	names := namesArray[:0]
	for _, item := range t.Model.SearchResults().Iterate {
		names = append(names, item)
	}
	element := func(gtx C, i int) D {
		if i < 0 || i >= len(names) {
			return D{}
		}
		w := Label(t.Theme, &t.Theme.UnitEditor.Chooser, names[i])
		if i == pe.searchList.TrackerList.Selected() {
			return pe.SelectTypeBtn.Layout(gtx, w.Layout)
		}
		return w.Layout(gtx)
	}
	fdl := FilledDragList(t.Theme, pe.searchList)
	dims := fdl.Layout(gtx, element, nil)
	gtx.Constraints = layout.Exact(dims.Size)
	fdl.LayoutScrollBar(gtx)
	return dims
}

func (t *UnitEditor) Tags(level int, yield TagYieldFunc) bool {
	if t.showingChooser() {
		return yield(level, t.searchList) && yield(level+1, &t.commentEditor.widgetEditor)
	}
	return yield(level+1, t.paramTable.RowTitleList) && yield(level, t.paramTable) && yield(level+1, &t.commentEditor.widgetEditor)
}
