package gioui

import (
	"bytes"
	"fmt"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/tracker"
	"golang.org/x/exp/shiny/materialdesign/icons"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"image"
	"io"
	"math"
)

type UnitEditor struct {
	sliderRows     []*DragList
	sliderColumns  *DragList
	searchList     *DragList
	Parameters     [][]*ParameterWidget
	DeleteUnitBtn  *ActionClickable
	CopyUnitBtn    *TipClickable
	ClearUnitBtn   *ActionClickable
	DisableUnitBtn *BoolClickable
	SelectTypeBtn  *widget.Clickable
	MultiUnitsBtn  *BoolClickable
	commentEditor  *Editor
	caser          cases.Caser

	copyHint        string
	disableUnitHint string
	enableUnitHint  string
	multiUnitsHint  string

	totalWidthForUnit map[int]int
	paramWidthForUnit map[int]int
}

func NewUnitEditor(m *tracker.Model) *UnitEditor {
	ret := &UnitEditor{
		DeleteUnitBtn:     NewActionClickable(m.DeleteUnit()),
		ClearUnitBtn:      NewActionClickable(m.ClearUnit()),
		DisableUnitBtn:    NewBoolClickable(m.UnitDisabled().Bool()),
		MultiUnitsBtn:     NewBoolClickable(m.EnableMultiUnits().Bool()),
		CopyUnitBtn:       new(TipClickable),
		SelectTypeBtn:     new(widget.Clickable),
		commentEditor:     NewEditor(widget.Editor{SingleLine: true, Submit: true}),
		sliderColumns:     NewDragList(m.Units().List(), layout.Horizontal),
		searchList:        NewDragList(m.SearchResults().List(), layout.Vertical),
		totalWidthForUnit: make(map[int]int),
		paramWidthForUnit: make(map[int]int),
	}
	ret.caser = cases.Title(language.English)
	ret.copyHint = makeHint("Copy unit", " (%s)", "Copy")
	ret.disableUnitHint = makeHint("Disable unit", " (%s)", "UnitDisabledToggle")
	ret.enableUnitHint = makeHint("Enable unit", " (%s)", "UnitDisabledToggle")
	ret.multiUnitsHint = "Toggle Multi-Unit View"

	ret.MultiUnitsBtn.Clickable.OnClick = func() {
		ret.ScrollToUnit(m.Units().Selected())
	}

	return ret
}

func (pe *UnitEditor) Layout(gtx C, t *Tracker) D {
	for {
		e, ok := gtx.Event(
			key.Filter{Focus: pe.sliderColumns, Name: key.NameLeftArrow, Optional: key.ModShift},
			key.Filter{Focus: pe.sliderColumns, Name: key.NameRightArrow, Optional: key.ModShift},
			key.Filter{Focus: pe.sliderColumns, Name: key.NameEscape},
		)
		if !ok {
			break
		}
		switch e := e.(type) {
		case key.Event:
			if e.State == key.Press {
				pe.command(e, t)
			}
		}
	}
	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
	editorFunc := pe.layoutColumns

	if t.UnitSearching().Value() || pe.sliderColumns.TrackerList.Count() == 0 {
		editorFunc = pe.layoutUnitTypeChooser
	}
	return Surface{Gray: 24, Focus: t.InstrumentEditor.wasFocused}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Flexed(1, func(gtx C) D {
				return editorFunc(gtx, t)
			}),
			layout.Rigid(func(gtx C) D {
				return pe.layoutFooter(gtx, t)
			}),
		)
	})
}

func (pe *UnitEditor) layoutColumns(gtx C, t *Tracker) D {
	numUnits := pe.sliderColumns.TrackerList.Count()
	for len(pe.Parameters) < numUnits {
		pe.Parameters = append(pe.Parameters, []*ParameterWidget{})
	}
	for u := len(pe.sliderRows); u < numUnits; u++ {
		paramList := NewDragList(t.ParamsForUnit(u).List(), layout.Vertical)
		pe.sliderRows = append(pe.sliderRows, paramList)
	}

	if !t.Model.EnableMultiUnits().Value() {
		return pe.layoutSliderColumn(gtx, t, t.Model.Units().Selected(), false)
	}

	column := func(gtx C, index int) D {
		if index < 0 || index > numUnits {
			return D{}
		}
		dims := pe.layoutSliderColumn(gtx, t, index, true)
		return D{Size: image.Pt(dims.Size.X, gtx.Constraints.Max.Y)}
	}
	fdl := FilledDragList(t.Theme, pe.sliderColumns, column, nil)
	dims := fdl.Layout(gtx)
	gtx.Constraints = layout.Exact(dims.Size)
	fdl.LayoutScrollBar(gtx)
	return dims
}

func (pe *UnitEditor) layoutSliderColumn(gtx C, t *Tracker, u int, multiUnits bool) D {
	numParams := 0
	for param := range t.Model.ParamsForUnit(u).Iterate {
		for len(pe.Parameters[u]) < numParams+1 {
			pe.Parameters[u] = append(pe.Parameters[u], new(ParameterWidget))
		}

		pe.Parameters[u][numParams].Parameter = param
		numParams++
	}

	unitId := t.Model.Units().CurrentInstrumentUnitAt(u).ID
	columnWidth := gtx.Constraints.Max.X
	if multiUnits {
		columnWidth = pe.totalWidthForUnit[unitId]
	}

	element := func(gtx C, index int) D {
		if index < 0 || index >= numParams {
			return D{}
		}
		paramStyle := t.ParamStyle(t.Theme, pe.Parameters[u][index])
		paramStyle.Focus = pe.sliderRows[u].TrackerList.Selected() == index
		dims := paramStyle.Layout(gtx, pe.paramWidthForUnit, unitId)
		if multiUnits && pe.totalWidthForUnit[unitId] < dims.Size.X {
			pe.totalWidthForUnit[unitId] = dims.Size.X
		}
		return D{Size: image.Pt(columnWidth, dims.Size.Y)}
	}

	fdl := FilledDragList(t.Theme, pe.sliderRows[u], element, nil)
	var dims D
	if multiUnits {
		name := buildUnitName(t.Model.Units().CurrentInstrumentUnitAt(u))
		dims = layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Flexed(1, fdl.Layout),
			layout.Rigid(func(gtx C) D {
				gtx.Constraints.Min.X = columnWidth
				gtx.Constraints.Min.Y = gtx.Sp(t.Theme.TextSize * 3)
				return layout.Center.Layout(gtx, Label(name, primaryColor, t.Theme.Shaper))
			}),
		)
		dims.Size.Y -= gtx.Dp(fdl.ScrollBarWidth)
	} else {
		dims = fdl.Layout(gtx)
	}
	gtx.Constraints = layout.Exact(dims.Size)
	fdl.LayoutScrollBar(gtx)
	return D{Size: image.Pt(columnWidth, dims.Size.Y)}
}

func (pe *UnitEditor) layoutFooter(gtx C, t *Tracker) D {
	for pe.CopyUnitBtn.Clickable.Clicked(gtx) {
		if contents, ok := t.Units().List().CopyElements(); ok {
			gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(bytes.NewReader(contents))})
			t.Alerts().Add("Unit copied to clipboard", tracker.Info)
		}
	}
	copyUnitBtnStyle := TipIcon(t.Theme, pe.CopyUnitBtn, icons.ContentContentCopy, pe.copyHint)
	deleteUnitBtnStyle := ActionIcon(gtx, t.Theme, pe.DeleteUnitBtn, icons.ActionDelete, "Delete unit (Ctrl+Backspace)")
	disableUnitBtnStyle := ToggleIcon(gtx, t.Theme, pe.DisableUnitBtn, icons.AVVolumeUp, icons.AVVolumeOff, pe.disableUnitHint, pe.enableUnitHint)
	multiUnitsBtnStyle := ToggleIcon(gtx, t.Theme, pe.MultiUnitsBtn, icons.ActionViewWeek, icons.ActionViewWeek, pe.multiUnitsHint, pe.multiUnitsHint)
	text := t.Units().SelectedType()
	if text == "" {
		text = "Choose unit type"
	} else {
		text = pe.caser.String(text)
	}
	hintText := Label(text, white, t.Theme.Shaper)
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(deleteUnitBtnStyle.Layout),
		layout.Rigid(copyUnitBtnStyle.Layout),
		layout.Rigid(disableUnitBtnStyle.Layout),
		layout.Rigid(func(gtx C) D {
			var dims D
			if t.Units().SelectedType() != "" {
				clearUnitBtnStyle := ActionIcon(gtx, t.Theme, pe.ClearUnitBtn, icons.ContentClear, "Clear unit")
				dims = clearUnitBtnStyle.Layout(gtx)
			}
			return D{Size: image.Pt(gtx.Dp(unit.Dp(48)), dims.Size.Y)}
		}),
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Dp(120)
			return hintText(gtx)
		}),
		layout.Flexed(1, func(gtx C) D {
			s := t.UnitComment().String()
			pe.commentEditor.SetText(s.Value())
			for pe.commentEditor.Submitted(gtx) || pe.commentEditor.Cancelled(gtx) {
				t.InstrumentEditor.Focus()
			}
			commentStyle := MaterialEditor(t.Theme, pe.commentEditor, "---")
			commentStyle.Font = labelDefaultFont
			commentStyle.TextSize = labelDefaultFontSize
			commentStyle.Color = mediumEmphasisTextColor
			commentStyle.HintColor = mediumEmphasisTextColor
			ret := commentStyle.Layout(gtx)
			s.Set(pe.commentEditor.Text())
			return ret
		}),
		layout.Rigid(multiUnitsBtnStyle.Layout),
	)
}

func (pe *UnitEditor) layoutUnitTypeChooser(gtx C, t *Tracker) D {
	var names [256]string
	for i, item := range t.Model.SearchResults().Iterate {
		if i >= 256 {
			break
		}
		names[i] = item
	}
	element := func(gtx C, i int) D {
		w := LabelStyle{Text: names[i], ShadeColor: black, Color: white, Font: labelDefaultFont, FontSize: unit.Sp(12), Shaper: t.Theme.Shaper}
		if i == pe.searchList.TrackerList.Selected() {
			for pe.SelectTypeBtn.Clicked(gtx) {
				t.Units().SetSelectedType(names[i])
			}
			return pe.SelectTypeBtn.Layout(gtx, w.Layout)
		}
		return w.Layout(gtx)
	}
	fdl := FilledDragList(t.Theme, pe.searchList, element, nil)
	dims := fdl.Layout(gtx)
	gtx.Constraints = layout.Exact(dims.Size)
	fdl.LayoutScrollBar(gtx)
	return dims
}

func (pe *UnitEditor) command(e key.Event, t *Tracker) {
	params := (*tracker.Params)(t.Model)
	switch e.State {
	case key.Press:
		switch e.Name {
		case key.NameLeftArrow:
			sel := params.SelectedItem()
			if sel == nil {
				return
			}
			i := &tracker.Int{IntData: sel}
			if e.Modifiers.Contain(key.ModShift) {
				i.Set(i.Value() - sel.LargeStep())
			} else {
				i.Set(i.Value() - 1)
			}
		case key.NameRightArrow:
			sel := params.SelectedItem()
			if sel == nil {
				return
			}
			i := &tracker.Int{IntData: sel}
			if e.Modifiers.Contain(key.ModShift) {
				i.Set(i.Value() + sel.LargeStep())
			} else {
				i.Set(i.Value() + 1)
			}
		case key.NameEscape:
			t.InstrumentEditor.unitDragList.Focus()
		}
	}
}

type ParameterWidget struct {
	floatWidget widget.Float
	boolWidget  widget.Bool
	instrBtn    widget.Clickable
	instrMenu   Menu
	unitBtn     widget.Clickable
	unitMenu    Menu
	Parameter   tracker.Parameter
	tipArea     component.TipArea
}

type ParameterStyle struct {
	tracker         *Tracker
	w               *ParameterWidget
	Theme           *material.Theme
	SendTargetTheme *material.Theme
	Focus           bool
}

func (t *Tracker) ParamStyle(th *material.Theme, paramWidget *ParameterWidget) ParameterStyle {
	sendTargetTheme := th.WithPalette(material.Palette{
		Bg:         th.Bg,
		Fg:         paramIsSendTargetColor,
		ContrastBg: th.ContrastBg,
		ContrastFg: th.ContrastFg,
	})
	return ParameterStyle{
		tracker:         t, // TODO: we need this to pull the instrument names for ID style parameters, find out another way
		Theme:           th,
		SendTargetTheme: &sendTargetTheme,
		w:               paramWidget,
	}
}

func spacer(px int) layout.FlexChild {
	return layout.Rigid(func(gtx C) D {
		return D{Size: image.Pt(px, px)}
	})
}

func (p ParameterStyle) Layout(gtx C, paramWidthMap map[int]int, unitId int) D {
	isSendTarget, info := p.tryDerivedParameterInfo(unitId)
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		spacer(24),
		layout.Rigid(func(gtx C) D {
			dims := layout.E.Layout(gtx, Label(p.w.Parameter.Name(), white, p.tracker.Theme.Shaper))
			if paramWidthMap[unitId] < dims.Size.X {
				paramWidthMap[unitId] = dims.Size.X
			}
			dims.Size.X = paramWidthMap[unitId]
			return dims
		}),
		spacer(8),
		layout.Rigid(func(gtx C) D {
			switch p.w.Parameter.Type() {
			case tracker.IntegerParameter:
				for p.Focus {
					e, ok := gtx.Event(pointer.Filter{
						Target:  &p.w.floatWidget,
						Kinds:   pointer.Scroll,
						ScrollY: pointer.ScrollRange{Min: -1e6, Max: 1e6},
					})
					if !ok {
						break
					}
					if ev, ok := e.(pointer.Event); ok && ev.Kind == pointer.Scroll {
						delta := math.Min(math.Max(float64(ev.Scroll.Y), -1), 1)
						tracker.Int{IntData: p.w.Parameter}.Add(-int(delta))
					}
				}
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
				gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(40))
				ra := p.w.Parameter.Range()
				if !p.w.floatWidget.Dragging() {
					p.w.floatWidget.Value = (float32(p.w.Parameter.Value()) - float32(ra.Min)) / float32(ra.Max-ra.Min)
				}
				sliderStyle := material.Slider(p.Theme, &p.w.floatWidget)
				sliderStyle.Color = p.Theme.Fg
				if isSendTarget {
					sliderStyle.Color = paramIsSendTargetColor
				}
				r := image.Rectangle{Max: gtx.Constraints.Min}
				defer clip.Rect(r).Push(gtx.Ops).Pop()
				defer pointer.PassOp{}.Push(gtx.Ops).Pop()
				if p.Focus {
					event.Op(gtx.Ops, &p.w.floatWidget)
				}
				dims := sliderStyle.Layout(gtx)
				tracker.Int{IntData: p.w.Parameter}.Set(int(p.w.floatWidget.Value*float32(ra.Max-ra.Min) + float32(ra.Min) + 0.5))
				return dims
			case tracker.BoolParameter:
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(60))
				gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(40))
				ra := p.w.Parameter.Range()
				p.w.boolWidget.Value = p.w.Parameter.Value() > ra.Min
				boolStyle := material.Switch(p.Theme, &p.w.boolWidget, "Toggle boolean parameter")
				boolStyle.Color.Disabled = p.Theme.Fg
				boolStyle.Color.Enabled = white
				defer pointer.PassOp{}.Push(gtx.Ops).Pop()
				dims := layout.Center.Layout(gtx, boolStyle.Layout)
				if p.w.boolWidget.Value {
					tracker.Int{IntData: p.w.Parameter}.Set(ra.Max)
				} else {
					tracker.Int{IntData: p.w.Parameter}.Set(ra.Min)
				}
				return dims
			case tracker.IDParameter:
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
				gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(40))
				instrItems := make([]MenuItem, p.tracker.Instruments().Count())
				for i := range instrItems {
					i := i
					name, _, _, _ := p.tracker.Instruments().Item(i)
					instrItems[i].Text = name
					instrItems[i].IconBytes = icons.NavigationChevronRight
					instrItems[i].Doer = tracker.Allow(func() {
						if id, ok := p.tracker.Instruments().FirstID(i); ok {
							tracker.Int{IntData: p.w.Parameter}.Set(id)
						}
					})
				}
				var unitItems []MenuItem
				instrName := "<instr>"
				unitName := "<unit>"
				targetInstrName, units, targetUnitIndex, ok := p.tracker.UnitInfo(p.w.Parameter.Value())
				if ok {
					instrName = targetInstrName
					unitName = buildUnitLabel(targetUnitIndex, units[targetUnitIndex])
					unitItems = make([]MenuItem, len(units))
					for j, unit := range units {
						id := unit.ID
						unitItems[j].Text = buildUnitLabel(j, unit)
						unitItems[j].IconBytes = icons.NavigationChevronRight
						unitItems[j].Doer = tracker.Allow(func() {
							tracker.Int{IntData: p.w.Parameter}.Set(id)
						})
					}
				}
				defer pointer.PassOp{}.Push(gtx.Ops).Pop()
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(p.tracker.layoutMenu(gtx, instrName, &p.w.instrBtn, &p.w.instrMenu, unit.Dp(200),
						instrItems...,
					)),
					layout.Rigid(p.tracker.layoutMenu(gtx, unitName, &p.w.unitBtn, &p.w.unitMenu, unit.Dp(240),
						unitItems...,
					)),
				)
			}
			return D{}
		}),
		spacer(8),
		layout.Rigid(func(gtx C) D {
			if p.w.Parameter.Type() != tracker.IDParameter {
				color := white
				hint := p.w.Parameter.Hint()
				if !hint.Valid {
					color = paramValueInvalidColor
				}
				label := Label(hint.Label, color, p.tracker.Theme.Shaper)
				if info == "" {
					return label(gtx)
				}
				tooltip := component.PlatformTooltip(p.SendTargetTheme, info)
				return p.w.tipArea.Layout(gtx, tooltip, label)
			}
			return D{}
		}),
		spacer(24),
	)
}

func buildUnitLabel(index int, u sointu.Unit) string {
	return fmt.Sprintf("%d: %s", index, buildUnitName(u))
}

func buildUnitName(u sointu.Unit) string {
	if u.Comment != "" {
		return fmt.Sprintf("%s \"%s\"", u.Type, u.Comment)
	}
	return u.Type
}

func (p ParameterStyle) tryDerivedParameterInfo(unitId int) (isSendTarget bool, sendInfo string) {
	param, ok := (p.w.Parameter).(tracker.NamedParameter)
	if !ok {
		return false, ""
	}
	isSendTarget, sendInfo, _ = p.tracker.ParameterInfo(unitId, param.Name())
	return isSendTarget, sendInfo
}

func (pe *UnitEditor) ScrollToUnit(index int) {
	pe.sliderColumns.List.Position.First = index
	pe.sliderColumns.List.Position.Offset = 0
}
