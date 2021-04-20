package gioui

import (
	"image"
	"image/color"
	"strings"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/vsariola/sointu/tracker"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

func (t *Tracker) layoutUnitEditor(gtx C) D {
	editorFunc := t.layoutUnitSliders
	if t.Unit().Type == "" {
		editorFunc = t.layoutUnitTypeChooser
	}
	return Surface{Gray: 24, Focus: t.EditMode() == tracker.EditUnits || t.EditMode() == tracker.EditParameters}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Flexed(1, editorFunc),
			layout.Rigid(t.layoutUnitFooter()))
	})
}

func (t *Tracker) layoutUnitSliders(gtx C) D {
	numItems := t.NumParams()

	for len(t.Parameters) <= numItems {
		t.Parameters = append(t.Parameters, new(ParameterWidget))
	}

	listItem := func(gtx C, index int) D {
		for t.Parameters[index].Clicked() {
			if t.EditMode() != tracker.EditParameters || t.ParamIndex() != index {
				t.SetEditMode(tracker.EditParameters)
				t.SetParamIndex(index)
			} else {
				t.ResetParam()
			}
		}
		param, err := t.Param(index)
		if err != nil {
			return D{}
		}
		oldVal := param.Value
		paramStyle := t.ParamStyle(t.Theme, &param, t.Parameters[index])
		paramStyle.Focus = t.EditMode() == tracker.EditParameters && t.ParamIndex() == index
		dims := paramStyle.Layout(gtx)
		if oldVal != param.Value {
			t.SetEditMode(tracker.EditParameters)
			t.SetParamIndex(index)
			t.SetParam(param.Value)
		}
		return dims
	}

	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx C) D {
			return t.ParameterList.Layout(gtx, numItems, listItem)
		}),
		layout.Stacked(func(gtx C) D {
			gtx.Constraints.Min = gtx.Constraints.Max
			return t.ParameterScrollBar.Layout(gtx, unit.Dp(10), numItems, &t.ParameterList.Position)
		}))
}

func (t *Tracker) layoutUnitFooter() layout.Widget {
	return func(gtx C) D {
		for t.ClearUnitBtn.Clicked() {
			t.SetUnitType("")
			op.InvalidateOp{}.Add(gtx.Ops)
		}
		for t.DeleteUnitBtn.Clicked() {
			t.DeleteUnit(false)
			op.InvalidateOp{}.Add(gtx.Ops)
		}
		deleteUnitBtnStyle := IconButton(t.Theme, t.DeleteUnitBtn, icons.ActionDelete, t.CanDeleteUnit())
		text := t.Unit().Type
		if text == "" {
			text = "Choose unit type"
		} else {
			text = strings.Title(text)
		}
		hintText := Label(text, white)
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(deleteUnitBtnStyle.Layout),
			layout.Rigid(func(gtx C) D {
				var dims D
				if t.Unit().Type != "" {
					clearUnitBtnStyle := IconButton(t.Theme, t.ClearUnitBtn, icons.ContentClear, true)
					dims = clearUnitBtnStyle.Layout(gtx)
				}
				return D{Size: image.Pt(gtx.Px(unit.Dp(48)), dims.Size.Y)}
			}),
			layout.Flexed(1, hintText),
		)
	}
}

func (t *Tracker) layoutUnitTypeChooser(gtx C) D {
	listElem := func(gtx C, i int) D {
		for t.ChooseUnitTypeBtns[i].Clicked() {
			t.SetUnitType(tracker.UnitTypeNames[i])
		}
		labelStyle := LabelStyle{Text: tracker.UnitTypeNames[i], ShadeColor: black, Color: white, Font: labelDefaultFont, FontSize: unit.Sp(12)}
		bg := func(gtx C) D {
			gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, 20))
			var color color.NRGBA
			if t.ChooseUnitTypeBtns[i].Hovered() {
				color = unitTypeListHighlightColor
			}
			paint.FillShape(gtx.Ops, color, clip.Rect{Max: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}.Op())
			return D{Size: gtx.Constraints.Min}
		}
		leftMargin := layout.Inset{Left: unit.Dp(10)}
		return layout.Stack{Alignment: layout.W}.Layout(gtx,
			layout.Stacked(bg),
			layout.Expanded(func(gtx C) D {
				return leftMargin.Layout(gtx, labelStyle.Layout)
			}),
			layout.Expanded(t.ChooseUnitTypeBtns[i].Layout))
	}
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx C) D {
			return t.ChooseUnitTypeList.Layout(gtx, len(tracker.UnitTypeNames), listElem)
		}),
		layout.Expanded(func(gtx C) D {
			return t.ChooseUnitScrollBar.Layout(gtx, unit.Dp(10), len(tracker.UnitTypeNames), &t.ChooseUnitTypeList.Position)
		}),
	)
}
