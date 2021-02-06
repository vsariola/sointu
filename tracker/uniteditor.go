package tracker

import (
	"image"
	"image/color"
	"sort"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

func (t *Tracker) layoutUnitEditor(gtx C) D {
	editorFunc := t.layoutUnitSliders
	if t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit].Type == "" {
		editorFunc = t.layoutUnitTypeChooser
	}
	paint.FillShape(gtx.Ops, unitSurfaceColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}.Op())
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, editorFunc),
		layout.Rigid(t.layoutUnitFooter()),
	)
}

func (t *Tracker) layoutUnitSliders(gtx C) D {
	params := t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit].Parameters
	count := len(params)
	children := make([]layout.FlexChild, 0, count)
	if len(t.ParameterSliders) < count {
		tail := make([]*widget.Float, count-len(t.ParameterSliders))
		for t := range tail {
			tail[t] = new(widget.Float)
		}
		t.ParameterSliders = append(t.ParameterSliders, tail...)
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		for t.ParameterSliders[i].Changed() {
			params[k] = int(t.ParameterSliders[i].Value)
			// TODO: tracker should have functions to update parameters and
			// to do this efficiently i.e. not compile the whole patch again
			t.LoadSong(t.song)
		}
		t.ParameterSliders[i].Value = float32(params[k])
		sliderStyle := material.Slider(t.Theme, t.ParameterSliders[i], 0, 128)
		sliderStyle.Color = t.Theme.Fg
		k2 := k // avoid k changing in the closure
		children = append(children, layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label(k2, white)),
				layout.Rigid(func(gtx C) D {
					gtx.Constraints.Min.X = 200
					return sliderStyle.Layout(gtx)
				}))
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (t *Tracker) layoutUnitFooter() layout.Widget {
	return func(gtx C) D {
		for t.ClearUnitBtn.Clicked() {
			t.ClearUnit()
			op.InvalidateOp{}.Add(gtx.Ops)
		}
		for t.DeleteUnitBtn.Clicked() {
			t.DeleteUnit()
			op.InvalidateOp{}.Add(gtx.Ops)
		}
		deleteUnitBtnStyle := material.IconButton(t.Theme, t.DeleteUnitBtn, widgetForIcon(icons.ActionDelete))
		deleteUnitBtnStyle.Background = transparent
		deleteUnitBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
		if len(t.song.Patch.Instruments[t.CurrentInstrument].Units) > 1 {
			deleteUnitBtnStyle.Color = primaryColor
		} else {
			deleteUnitBtnStyle.Color = disabledTextColor
		}
		if t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit].Type == "" {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
				layout.Rigid(deleteUnitBtnStyle.Layout))
		}
		clearUnitBtnStyle := material.IconButton(t.Theme, t.ClearUnitBtn, widgetForIcon(icons.ContentClear))
		clearUnitBtnStyle.Color = primaryColor
		clearUnitBtnStyle.Background = transparent
		clearUnitBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
			layout.Rigid(clearUnitBtnStyle.Layout),
			layout.Rigid(deleteUnitBtnStyle.Layout))
	}
}

func (t *Tracker) layoutUnitTypeChooser(gtx C) D {
	paint.FillShape(gtx.Ops, unitSurfaceColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}.Op())
	listElem := func(gtx C, i int) D {
		for t.ChooseUnitTypeBtns[i].Clicked() {
			t.SetUnit(allUnits[i])
		}
		labelStyle := LabelStyle{Text: allUnits[i], ShadeColor: black, Color: white, Font: labelDefaultFont, FontSize: unit.Sp(12)}
		bg := func(gtx C) D {
			gtx.Constraints = layout.Exact(image.Pt(120, 20))
			var color color.NRGBA
			if t.ChooseUnitTypeBtns[i].Hovered() {
				color = unitTypeListHighlightColor
			}
			paint.FillShape(gtx.Ops, color, clip.Rect{Max: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}.Op())
			return D{Size: gtx.Constraints.Min}
		}
		return layout.Stack{Alignment: layout.W}.Layout(gtx,
			layout.Stacked(bg),
			layout.Expanded(labelStyle.Layout),
			layout.Expanded(t.ChooseUnitTypeBtns[i].Layout))
	}
	return t.ChooseUnitTypeList.Layout(gtx, len(allUnits), listElem)
}
