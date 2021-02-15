package tracker

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/compiler"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

func (t *Tracker) layoutUnitEditor(gtx C) D {
	editorFunc := t.layoutUnitSliders
	if t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit].Type == "" {
		editorFunc = t.layoutUnitTypeChooser
	}
	return Surface{Gray: 24, Focus: t.EditMode == EditUnits || t.EditMode == EditParameters}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Flexed(1, editorFunc),
			layout.Rigid(t.layoutUnitFooter()))
	})

}

func (t *Tracker) layoutUnitSliders(gtx C) D {
	u := t.song.Patch.Instruments[t.CurrentInstrument].Units[t.CurrentUnit]
	ut, ok := sointu.UnitTypes[u.Type]
	if !ok {
		return layout.Dimensions{}
	}
	listElements := func(gtx C, index int) D {
		for len(t.ParameterSliders) <= index {
			t.ParameterSliders = append(t.ParameterSliders, new(widget.Float))
		}
		params := u.Parameters
		var name string
		var value, min, max int
		var valueText string
		if u.Type == "oscillator" && index == len(ut) {
			name = "sample"
			key := compiler.SampleOffset{Start: uint32(params["samplestart"]), LoopStart: uint16(params["loopstart"]), LoopLength: uint16(params["looplength"])}
			if v, ok := gmDlsEntryMap[key]; ok {
				value = v + 1
				valueText = fmt.Sprintf("%v / %v", value, gmDlsEntries[v].Name)
			} else {
				value = 0
				valueText = "0 / custom"
			}
			min, max = 0, len(gmDlsEntries)
		} else {
			if ut[index].MaxValue < ut[index].MinValue {
				return layout.Dimensions{}
			}
			name = ut[index].Name
			if u.Type == "oscillator" && (name == "samplestart" || name == "loopstart" || name == "looplength") {
				if params["type"] != sointu.Sample {
					return layout.Dimensions{}
				}
			}
			value = params[name]
			min, max = ut[index].MinValue, ut[index].MaxValue
			if u.Type == "send" && name == "voice" {
				max = t.song.Patch.TotalVoices()
			} else if u.Type == "send" && name == "unit" { // set the maximum values depending on the send target
				instrIndex, _, _, _ := t.song.Patch.FindSendTarget(t.CurrentInstrument, t.CurrentUnit)
				if instrIndex != -1 {
					max = len(t.song.Patch.Instruments[instrIndex].Units) - 1
				}
			} else if u.Type == "send" && name == "port" { // set the maximum values depending on the send target
				instrIndex, unitIndex, _, _ := t.song.Patch.FindSendTarget(t.CurrentInstrument, t.CurrentUnit)
				if instrIndex != -1 && unitIndex != -1 {
					max = len(sointu.Ports[t.song.Patch.Instruments[instrIndex].Units[unitIndex].Type]) - 1
				}
			}
			hint := t.song.ParamHintString(t.CurrentInstrument, t.CurrentUnit, name)
			if hint != "" {
				valueText = fmt.Sprintf("%v / %v", value, hint)
			} else {
				valueText = fmt.Sprintf("%v", value)
			}
		}
		if !t.ParameterSliders[index].Dragging() {
			t.ParameterSliders[index].Value = float32(value)
		}
		if max < min {
			max = min
		}
		sliderStyle := material.Slider(t.Theme, t.ParameterSliders[index], float32(min), float32(max))
		sliderStyle.Color = t.Theme.Fg
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Px(unit.Dp(110))
				return layout.E.Layout(gtx, Label(name, white))
			}),
			layout.Rigid(func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Px(unit.Dp(200))
				gtx.Constraints.Min.Y = gtx.Px(unit.Dp(40))
				if t.EditMode == EditParameters && t.CurrentParam == index {
					paint.FillShape(gtx.Ops, cursorColor, clip.Rect{
						Max: gtx.Constraints.Min,
					}.Op())
				}
				dims := sliderStyle.Layout(gtx)
				for sliderStyle.Float.Changed() {
					t.EditMode = EditParameters
					t.CurrentParam = index
					if u.Type == "oscillator" && name == "sample" {
						v := int(t.ParameterSliders[index].Value+0.5) - 1
						if v >= 0 {
							t.SetGmDlsEntry(v)
						}
					} else {
						t.SetUnitParam(int(t.ParameterSliders[index].Value + 0.5))
					}
				}
				return dims
			}),
			layout.Rigid(Label(valueText, white)),
		)
	}
	l := len(ut)
	if u.Type == "oscillator" && u.Parameters["type"] == sointu.Sample {
		l++
	}
	return t.ParameterList.Layout(gtx, l, listElements)
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
	listElem := func(gtx C, i int) D {
		for t.ChooseUnitTypeBtns[i].Clicked() {
			t.SetUnit(allUnits[i])
		}
		labelStyle := LabelStyle{Text: allUnits[i], ShadeColor: black, Color: white, Font: labelDefaultFont, FontSize: unit.Sp(12)}
		bg := func(gtx C) D {
			gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, 20))
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
	hintText := Label("Choose unit type:", white)
	inset := layout.Inset{Left: unit.Dp(6), Top: unit.Dp(6)}
	return inset.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(hintText),
			layout.Flexed(1, func(gtx C) D {
				return t.ChooseUnitTypeList.Layout(gtx, len(allUnits), listElem)
			}))
	})
}
