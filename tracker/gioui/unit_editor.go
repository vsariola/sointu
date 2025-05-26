package gioui

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"math"

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
)

type UnitEditor struct {
	sliderList     *DragList
	searchList     *DragList
	Parameters     []*ParameterWidget
	DeleteUnitBtn  *ActionClickable
	CopyUnitBtn    *TipClickable
	ClearUnitBtn   *ActionClickable
	DisableUnitBtn *BoolClickable
	SelectTypeBtn  *Clickable
	commentEditor  *Editor
	caser          cases.Caser

	copyHint        string
	disableUnitHint string
	enableUnitHint  string
}

func NewUnitEditor(m *tracker.Model) *UnitEditor {
	ret := &UnitEditor{
		DeleteUnitBtn:  NewActionClickable(m.DeleteUnit()),
		ClearUnitBtn:   NewActionClickable(m.ClearUnit()),
		DisableUnitBtn: NewBoolClickable(m.UnitDisabled()),
		CopyUnitBtn:    new(TipClickable),
		SelectTypeBtn:  new(Clickable),
		commentEditor:  NewEditor(widget.Editor{SingleLine: true, Submit: true}),
		sliderList:     NewDragList(m.Params().List(), layout.Vertical),
		searchList:     NewDragList(m.SearchResults().List(), layout.Vertical),
	}
	ret.caser = cases.Title(language.English)
	ret.copyHint = makeHint("Copy unit", " (%s)", "Copy")
	ret.disableUnitHint = makeHint("Disable unit", " (%s)", "UnitDisabledToggle")
	ret.enableUnitHint = makeHint("Enable unit", " (%s)", "UnitDisabledToggle")
	return ret
}

func (pe *UnitEditor) Layout(gtx C, t *Tracker) D {
	for {
		e, ok := gtx.Event(
			key.Filter{Focus: pe.sliderList, Name: key.NameLeftArrow, Optional: key.ModShift},
			key.Filter{Focus: pe.sliderList, Name: key.NameRightArrow, Optional: key.ModShift},
			key.Filter{Focus: pe.sliderList, Name: key.NameEscape},
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
	editorFunc := pe.layoutSliders

	if t.UnitSearching().Value() || pe.sliderList.TrackerList.Count() == 0 {
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

func (pe *UnitEditor) layoutSliders(gtx C, t *Tracker) D {
	numItems := pe.sliderList.TrackerList.Count()

	for len(pe.Parameters) < numItems {
		pe.Parameters = append(pe.Parameters, new(ParameterWidget))
	}

	index := 0
	for param := range t.Model.Params().Iterate {
		pe.Parameters[index].Parameter = param
		index++
	}
	element := func(gtx C, index int) D {
		if index < 0 || index >= numItems {
			return D{}
		}
		paramStyle := t.ParamStyle(t.Theme, pe.Parameters[index])
		paramStyle.Focus = pe.sliderList.TrackerList.Selected() == index
		dims := paramStyle.Layout(gtx)
		return D{Size: image.Pt(gtx.Constraints.Max.X, dims.Size.Y)}
	}

	fdl := FilledDragList(t.Theme, pe.sliderList, element, nil)
	dims := fdl.Layout(gtx)
	gtx.Constraints = layout.Exact(dims.Size)
	fdl.LayoutScrollBar(gtx)
	return dims
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
	text := t.Units().SelectedType()
	if text == "" {
		text = "Choose unit type"
	} else {
		text = pe.caser.String(text)
	}
	hintText := Label(t.Theme, &t.Theme.UnitEditor.Hint, text)
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
			return hintText.Layout(gtx)
		}),
		layout.Flexed(1, func(gtx C) D {
			s := t.UnitComment().String()
			pe.commentEditor.SetText(s.Value())
			for pe.commentEditor.Submitted(gtx) || pe.commentEditor.Cancelled(gtx) {
				t.InstrumentEditor.Focus()
			}
			commentStyle := MaterialEditor(t.Theme, &t.Theme.InstrumentEditor.UnitComment, pe.commentEditor, "---")
			ret := commentStyle.Layout(gtx)
			s.Set(pe.commentEditor.Text())
			return ret
		}),
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
		w := Label(t.Theme, &t.Theme.UnitEditor.Chooser, names[i])

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
			i := (&tracker.Int{IntData: sel})
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
			i := (&tracker.Int{IntData: sel})
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
	instrBtn    Clickable
	instrMenu   Menu
	unitBtn     Clickable
	unitMenu    Menu
	Parameter   tracker.Parameter
	tipArea     component.TipArea
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
	isSendTarget, info := p.tryDerivedParameterInfo()
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(110))
			return layout.E.Layout(gtx, Label(p.Theme, &p.Theme.UnitEditor.ParameterName, p.w.Parameter.Name()).Layout)
		}),
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
				sliderStyle := material.Slider(&p.Theme.Material, &p.w.floatWidget)
				if isSendTarget {
					sliderStyle.Color = p.Theme.UnitEditor.SendTarget
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
				boolStyle := material.Switch(&p.Theme.Material, &p.w.boolWidget, "Toggle boolean parameter")
				boolStyle.Color.Disabled = p.Theme.Material.Fg
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
					instrItems[i].Doer = tracker.MakeEnabledAction((tracker.DoFunc)(func() {
						if id, ok := p.tracker.Instruments().FirstID(i); ok {
							tracker.Int{IntData: p.w.Parameter}.Set(id)
						}
					}))
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
						unitItems[j].Doer = tracker.MakeEnabledAction((tracker.DoFunc)(func() {
							tracker.Int{IntData: p.w.Parameter}.Set(id)
						}))
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
		layout.Rigid(func(gtx C) D {
			if p.w.Parameter.Type() != tracker.IDParameter {
				hint := p.w.Parameter.Hint()
				label := Label(p.tracker.Theme, &p.tracker.Theme.UnitEditor.Hint, hint.Label)
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
		}),
	)
}

func buildUnitLabel(index int, u sointu.Unit) string {
	text := u.Type
	if u.Comment != "" {
		text = fmt.Sprintf("%s \"%s\"", text, u.Comment)
	}
	return fmt.Sprintf("%d: %s", index, text)
}

func (p ParameterStyle) tryDerivedParameterInfo() (isSendTarget bool, sendInfo string) {
	param, ok := (p.w.Parameter).(tracker.NamedParameter)
	if !ok {
		return false, ""
	}
	isSendTarget, sendInfo, _ = p.tracker.ParameterInfo(param.Unit().ID, param.Name())
	return isSendTarget, sendInfo
}
