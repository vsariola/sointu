package gioui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"

	"gioui.org/f32"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/tracker"
	"golang.org/x/exp/shiny/materialdesign/icons"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type (
	UnitEditor struct {
		paramTable     *ScrollTable
		searchList     *DragList
		Parameters     [][]*ParameterWidget
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
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
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
	for {
		e, ok := gtx.Event(
			key.Filter{Focus: pe.searchList, Name: key.NameEnter},
			key.Filter{Focus: pe.searchList, Name: key.NameReturn},
		)
		if !ok {
			break
		}
		if e, ok := e.(key.Event); ok && e.State == key.Press {
			pe.ChooseUnitType(t)
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
			params := t.Model.Params()
			item := params.Item(params.Cursor())
			switch e.Name {
			case key.NameLeftArrow:
				if e.Modifiers.Contain(key.ModShortcut) {
					item.SetValue(item.Value() - item.LargeStep())
				} else {
					item.SetValue(item.Value() - 1)
				}
			case key.NameRightArrow:
				if e.Modifiers.Contain(key.ModShortcut) {
					item.SetValue(item.Value() + item.LargeStep())
				} else {
					item.SetValue(item.Value() + 1)
				}
			case key.NameDeleteBackward, key.NameDeleteForward:
				item.Reset()
			}
		}
	}
}

func (pe *UnitEditor) ChooseUnitType(t *Tracker) {
	if ut, ok := t.SearchResults().Item(pe.searchList.TrackerList.Selected()); ok {
		t.Units().SetSelectedType(ut)
		t.PatchPanel.unitList.dragList.Focus()
	}
}

func (pe *UnitEditor) layoutRack(gtx C) D {
	t := TrackerFromContext(gtx)
	// create enough parameter widget to match the number of parameters
	width := pe.paramTable.Table.Width()
	for len(pe.Parameters) < pe.paramTable.Table.Height() {
		pe.Parameters = append(pe.Parameters, make([]*ParameterWidget, 0))
	}
	cellWidth := gtx.Dp(t.Theme.UnitEditor.Width)
	cellHeight := gtx.Dp(t.Theme.UnitEditor.Height)
	rowTitleLabelWidth := gtx.Dp(t.Theme.UnitEditor.RowTitleWidth)
	rowTitleSignalWidth := gtx.Dp(t.Theme.SignalRail.SignalWidth) * t.SignalRail().MaxWidth()
	rowTitleWidth := rowTitleLabelWidth + rowTitleSignalWidth
	signalError := t.SignalRail().Error()
	columnTitleHeight := gtx.Dp(0)
	for i := range pe.Parameters {
		for len(pe.Parameters[i]) < width {
			pe.Parameters[i] = append(pe.Parameters[i], &ParameterWidget{})
		}
	}
	coltitle := func(gtx C, x int) D {
		return D{Size: image.Pt(cellWidth, columnTitleHeight)}
	}
	rowTitleBg := func(gtx C, j int) D {
		paint.FillShape(gtx.Ops, t.Theme.NoteEditor.Play, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, 1)}.Op())
		return D{}
	}
	rowtitle := func(gtx C, y int) D {
		if y < 0 || y >= len(pe.Parameters) {
			return D{}
		}

		sr := SignalRail(t.Theme, t.SignalRail().Item(y))
		label := Label(t.Theme, &t.Theme.UnitEditor.RowTitle, t.Units().Item(y).Type)
		if signalError.Err != nil && signalError.UnitIndex == y {
			label.Color = t.Theme.UnitEditor.Error
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
		if point == cursor {
			c := t.Theme.Cursor.Inactive
			if gtx.Focused(pe.paramTable) {
				c = t.Theme.Cursor.Active
			}
			paint.FillShape(gtx.Ops, c, clip.Rect{Min: image.Pt(0, 0), Max: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}.Op())
		}

		param := t.Model.Params().Item(tracker.Point{X: x, Y: y})
		pe.Parameters[y][x].Parameter = param
		paramStyle := t.ParamStyle(t.Theme, pe.Parameters[y][x])
		paramStyle.Focus = pe.paramTable.Table.Cursor() == tracker.Point{X: x, Y: y}
		paramStyle.Layout(gtx)
		return D{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}
	}
	table := FilledScrollTable(t.Theme, pe.paramTable)
	table.RowTitleWidth = gtx.Metric.PxToDp(rowTitleWidth)
	table.ColumnTitleHeight = 0
	table.CellWidth = t.Theme.UnitEditor.Width
	table.CellHeight = t.Theme.UnitEditor.Height
	pe.drawSignals(gtx, rowTitleWidth)
	dims := table.Layout(gtx, cell, coltitle, rowtitle, nil, rowTitleBg)
	return dims
}

func (pe *UnitEditor) drawSignals(gtx C, rowTitleWidth int) {
	t := TrackerFromContext(gtx)
	units := t.Units()
	colP := pe.paramTable.ColTitleList.List.Position
	rowP := pe.paramTable.RowTitleList.List.Position
	p := image.Pt(rowTitleWidth, 0)
	defer op.Offset(p).Push(gtx.Ops).Pop()
	gtx.Constraints.Max = gtx.Constraints.Max.Sub(p)
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	defer op.Offset(image.Pt(-colP.Offset, -rowP.Offset)).Push(gtx.Ops).Pop()
	for i := 0; i < units.Count(); i++ {
		item := units.Item(i)
		if item.TargetOk {
			pe.drawSignal(gtx, i-rowP.First, item.TargetX-colP.First, item.TargetY-rowP.First)
		}
	}
}

func (pe *UnitEditor) drawSignal(gtx C, sy, ex, ey int) {
	t := TrackerFromContext(gtx)
	width := float32(gtx.Dp(t.Theme.UnitEditor.Width))
	height := float32(gtx.Dp(t.Theme.UnitEditor.Height))
	diam := gtx.Dp(t.Theme.Knob.Diameter)
	from := f32.Pt(0, float32((sy+1)*gtx.Dp(t.Theme.UnitEditor.Height))-float32(gtx.Dp(t.Theme.SignalRail.SignalWidth)/2))
	corner := f32.Pt(1, 1)
	if ex > 0 {
		corner.X = -corner.X
	}
	if sy < ey {
		corner.Y = -corner.Y
	}
	c := float32(diam) / 2 / float32(math.Sqrt2)
	topLeft := f32.Pt(float32(ex)*width, float32(ey)*height)
	center := topLeft.Add(f32.Pt(width/2, height/2))
	to := mulVec(corner, f32.Pt(c, c)).Add(center)
	p2 := mulVec(corner, f32.Pt(width/2, height/2)).Add(center)
	p1 := f32.Pt(p2.X, float32((sy+1)*gtx.Dp(t.Theme.UnitEditor.Height)))
	if sy > ey {
		p1 = f32.Pt(p2.X, (float32(sy)+0.5)*float32(gtx.Dp(t.Theme.UnitEditor.Height))+float32(diam)/2)
	}
	k := float32(width) / 4
	//toTan := mulVec(corner, f32.Pt(-k, -k))
	p2Tan := mulVec(corner, f32.Pt(-k, -k))
	p1Tan := f32.Pt(k, p2Tan.Y)
	fromTan := f32.Pt(k, 0)
	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(from)
	path.CubeTo(from.Add(fromTan), p1.Sub(p1Tan), p1)
	path.CubeTo(p1.Add(p1Tan), p2, to)
	paint.FillShape(gtx.Ops, t.Theme.UnitEditor.SendTarget,
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
	st := t.Units().SelectedType()
	text := "Choose unit type"
	if st != "" {
		text = pe.caser.String(st)
	}
	hintText := Label(t.Theme, &t.Theme.UnitEditor.Hint, text)
	deleteUnitBtn := ActionIconBtn(t.DeleteUnit(), t.Theme, pe.DeleteUnitBtn, icons.ActionDelete, "Delete unit (Ctrl+Backspace)")
	copyUnitBtn := IconBtn(t.Theme, &t.Theme.IconButton.Enabled, pe.CopyUnitBtn, icons.ContentContentCopy, pe.copyHint)
	disableUnitBtn := ToggleIconBtn(t.UnitDisabled(), t.Theme, pe.DisableUnitBtn, icons.AVVolumeUp, icons.AVVolumeOff, pe.disableUnitHint, pe.enableUnitHint)
	w := layout.Spacer{Width: t.Theme.IconButton.Enabled.Size}.Layout
	if st != "" {
		clearUnitBtn := ActionIconBtn(t.ClearUnit(), t.Theme, pe.ClearUnitBtn, icons.ContentClear, "Clear unit")
		w = clearUnitBtn.Layout
	}
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(deleteUnitBtn.Layout),
		layout.Rigid(copyUnitBtn.Layout),
		layout.Rigid(disableUnitBtn.Layout),
		layout.Rigid(w),
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Dp(120)
			return hintText.Layout(gtx)
		}),
		layout.Flexed(1, func(gtx C) D {
			return pe.commentEditor.Layout(gtx, t.UnitComment(), t.Theme, &t.Theme.InstrumentEditor.UnitComment, "---")
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
	widget := event.Tag(t.paramTable)
	if t.showingChooser() {
		widget = event.Tag(t.searchList)
	}
	return yield(level, widget) && yield(level+1, &t.commentEditor.widgetEditor)
}

type ParameterWidget struct {
	knobState  KnobState
	boolWidget widget.Bool
	instrBtn   Clickable
	instrMenu  MenuState
	unitBtn    Clickable
	unitMenu   MenuState
	Parameter  tracker.Parameter
	tipArea    TipArea
}

type ParameterStyle struct {
	tracker         *Tracker
	w               *ParameterWidget
	Theme           *Theme
	SendTargetTheme *material.Theme
	Focus           bool
}

func (t *Tracker) ParamStyle(th *Theme, paramWidget *ParameterWidget) ParameterStyle {
	sendTargetTheme := th.Material.WithPalette(material.Palette{
		Bg:         th.Material.Bg,
		Fg:         th.UnitEditor.SendTarget,
		ContrastBg: th.Material.ContrastBg,
		ContrastFg: th.Material.ContrastFg,
	})
	return ParameterStyle{
		tracker:         t, // TODO: we need this to pull the instrument names for ID style parameters, find out another way
		Theme:           th,
		SendTargetTheme: &sendTargetTheme,
		w:               paramWidget,
	}
}

func (p ParameterStyle) Layout(gtx C) D {
	//_, _ := p.w.Parameter.Info()
	title := Label(p.Theme, &p.Theme.UnitEditor.Name, p.w.Parameter.Name())
	widget := func(gtx C) D {
		switch p.w.Parameter.Type() {
		case tracker.IntegerParameter:
			k := Knob(p.w.Parameter, p.Theme, &p.w.knobState, p.w.Parameter.Hint().Label, p.Focus)
			return k.Layout(gtx)
		case tracker.BoolParameter:
			ra := p.w.Parameter.Range()
			p.w.boolWidget.Value = p.w.Parameter.Value() > ra.Min
			boolStyle := material.Switch(&p.Theme.Material, &p.w.boolWidget, "Toggle boolean parameter")
			boolStyle.Color.Disabled = p.Theme.Material.Fg
			defer pointer.PassOp{}.Push(gtx.Ops).Pop()
			dims := layout.Center.Layout(gtx, boolStyle.Layout)
			if p.w.boolWidget.Value {
				p.w.Parameter.SetValue(ra.Max)
			} else {
				p.w.Parameter.SetValue(ra.Min)
			}
			return dims
		case tracker.IDParameter:
			return drawCircle(gtx, gtx.Dp(p.Theme.Knob.Diameter), p.Theme.Knob.Pos.Bg)
			/*instrItems := make([]ActionMenuItem, p.tracker.Instruments().Count())
			for i := range instrItems {
				i := i
				name, _, _, _ := p.tracker.Instruments().Item(i)
				instrItems[i].Text = name
				instrItems[i].Icon = icons.NavigationChevronRight
				instrItems[i].Action = tracker.MakeEnabledAction((tracker.DoFunc)(func() {
					if id, ok := p.tracker.Instruments().FirstID(i); ok {
						p.w.Parameter.SetValue(id)
					}
				}))
			}
			var unitItems []ActionMenuItem
			instrName := "<instr>"
			unitName := "<unit>"
			targetInstrName, units, targetUnitIndex, ok := p.tracker.UnitInfo(p.w.Parameter.Value())
			if ok {
				instrName = targetInstrName
				unitName = buildUnitLabel(targetUnitIndex, units[targetUnitIndex])
				unitItems = make([]ActionMenuItem, len(units))
				for j, unit := range units {
					id := unit.ID
					unitItems[j].Text = buildUnitLabel(j, unit)
					unitItems[j].Icon = icons.NavigationChevronRight
					unitItems[j].Action = tracker.MakeEnabledAction((tracker.DoFunc)(func() {
						p.w.Parameter.SetValue(id)
					}))
				}
			}
			defer pointer.PassOp{}.Push(gtx.Ops).Pop()
			instrBtn := MenuBtn(p.tracker.Theme, &p.w.instrMenu, &p.w.instrBtn, instrName)
			unitBtn := MenuBtn(p.tracker.Theme, &p.w.unitMenu, &p.w.unitBtn, unitName)
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return instrBtn.Layout(gtx, instrItems...)
				}),
				layout.Rigid(func(gtx C) D {
					return unitBtn.Layout(gtx, unitItems...)
				}),
			)*/
		}
		return D{}
	}
	title.Layout(gtx)
	layout.Center.Layout(gtx, widget)
	return D{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}
	/*	layout.Rigid(func(gtx C) D {
		if p.w.Parameter.Type() != tracker.IDParameter {
			hint := p.w.Parameter.Hint()
			label := Label(p.tracker.Theme, &p.tracker.Theme.UnitEditor.Hint, hint.Label)
			label.Alignment = text.Middle
			if !hint.Valid {
				label.Color = p.tracker.Theme.UnitEditor.InvalidParam
			}
			if info == "" {
				return label.Layout(gtx)
			}
			tooltip := component.PlatformTooltip(p.SendTargetTheme, info)
			return p.w.tipArea.Layout(gtx, tooltip, label.Layout)
		}
		return D{}
	}),*/
}

func drawCircle(gtx C, i int, nRGBA color.NRGBA) D {
	defer clip.Ellipse(image.Rectangle{Max: image.Pt(i, i)}).Push(gtx.Ops).Pop()
	paint.FillShape(gtx.Ops, nRGBA, clip.Ellipse{Max: image.Pt(i, i)}.Op(gtx.Ops))
	return D{Size: image.Pt(i, i)}
}

func buildUnitLabel(index int, u sointu.Unit) string {
	text := u.Type
	if u.Comment != "" {
		text = fmt.Sprintf("%s \"%s\"", text, u.Comment)
	}
	return fmt.Sprintf("%d: %s", index, text)
}
