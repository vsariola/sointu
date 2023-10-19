package gioui

import (
	"image"
	"image/color"
	"strings"
	"time"

	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/vsariola/sointu"
	"golang.org/x/exp/shiny/materialdesign/icons"
	"gopkg.in/yaml.v3"
)

type ParamEditor struct {
	list               *layout.List
	scrollBar          *ScrollBar
	Parameters         []*ParameterWidget
	DeleteUnitBtn      *TipClickable
	CopyUnitBtn        *TipClickable
	ClearUnitBtn       *TipClickable
	ChooseUnitTypeBtns []*widget.Clickable
	tag                bool
	focused            bool
	requestFocus       bool
}

func (pe *ParamEditor) Focus() {
	pe.requestFocus = true
}

func (pe *ParamEditor) Focused() bool {
	return pe.focused
}

func NewParamEditor() *ParamEditor {
	ret := &ParamEditor{
		DeleteUnitBtn: new(TipClickable),
		ClearUnitBtn:  new(TipClickable),
		CopyUnitBtn:   new(TipClickable),
		list:          &layout.List{Axis: layout.Vertical},
		scrollBar:     &ScrollBar{Axis: layout.Vertical},
	}
	for range sointu.UnitNames {
		ret.ChooseUnitTypeBtns = append(ret.ChooseUnitTypeBtns, new(widget.Clickable))
	}
	return ret
}

func (pe *ParamEditor) Bind(t *Tracker) layout.Widget {
	return func(gtx C) D {
		for _, e := range gtx.Events(&pe.tag) {
			switch e := e.(type) {
			case key.FocusEvent:
				pe.focused = e.Focus
			case pointer.Event:
				if e.Type == pointer.Press {
					key.FocusOp{Tag: &pe.tag}.Add(gtx.Ops)
				}
			case key.Event:
				if e.Modifiers.Contain(key.ModShortcut) {
					continue
				}
				switch e.State {
				case key.Press:
					switch e.Name {
					case key.NameUpArrow:
						t.SetParamIndex(t.ParamIndex() - 1)
					case key.NameDownArrow:
						t.SetParamIndex(t.ParamIndex() + 1)
					case key.NameLeftArrow:
						p, err := t.Param(t.ParamIndex())
						if err != nil {
							break
						}
						if e.Modifiers.Contain(key.ModShift) {
							t.SetParam(p.Value - p.LargeStep)
						} else {
							t.SetParam(p.Value - 1)
						}
					case key.NameRightArrow:
						p, err := t.Param(t.ParamIndex())
						if err != nil {
							break
						}
						if e.Modifiers.Contain(key.ModShift) {
							t.SetParam(p.Value + p.LargeStep)
						} else {
							t.SetParam(p.Value + 1)
						}
					case key.NameEscape:
						t.InstrumentEditor.unitDragList.Focus()
					}
				}
			}
		}
		if pe.requestFocus {
			pe.requestFocus = false
			key.FocusOp{Tag: &pe.tag}.Add(gtx.Ops)
		}
		editorFunc := pe.layoutUnitSliders
		if y := t.Unit().Type; y == "" || y != t.InstrumentEditor.unitTypeEditor.Text() {
			editorFunc = pe.layoutUnitTypeChooser
		}
		return Surface{Gray: 24, Focus: t.InstrumentEditor.wasFocused}.Layout(gtx, func(gtx C) D {
			ret := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Flexed(1, func(gtx C) D {
					return editorFunc(gtx, t)
				}),
				layout.Rigid(pe.layoutUnitFooter(t)))
			rect := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
			area := clip.Rect(rect).Push(gtx.Ops)
			defer pointer.PassOp{}.Push(gtx.Ops).Pop()
			pointer.InputOp{Tag: &pe.tag,
				Types: pointer.Press,
			}.Add(gtx.Ops)
			key.InputOp{Tag: &pe.tag, Keys: "←|Shift-←|→|Shift-→|↑|↓|⎋"}.Add(gtx.Ops)
			area.Pop()
			return ret
		})
	}
}

func (pe *ParamEditor) layoutUnitSliders(gtx C, t *Tracker) D {
	numItems := t.NumParams()

	for len(pe.Parameters) <= numItems {
		pe.Parameters = append(pe.Parameters, new(ParameterWidget))
	}

	listItem := func(gtx C, index int) D {
		for pe.Parameters[index].Clicked() {
			if t.ParamIndex() != index {
				t.SetParamIndex(index)
			} else {
				t.ResetParam()
			}
			pe.Focus()
		}
		param, err := t.Param(index)
		if err != nil {
			return D{}
		}
		oldVal := param.Value
		paramStyle := t.ParamStyle(t.Theme, &param, pe.Parameters[index])
		paramStyle.Focus = pe.focused && t.ParamIndex() == index
		dims := paramStyle.Layout(gtx)
		if oldVal != param.Value {
			pe.Focus()
			t.SetParamIndex(index)
			t.SetParam(param.Value)
		}
		return dims
	}

	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx C) D {
			return pe.list.Layout(gtx, numItems, listItem)
		}),
		layout.Stacked(func(gtx C) D {
			gtx.Constraints.Min = gtx.Constraints.Max
			return pe.scrollBar.Layout(gtx, unit.Dp(10), numItems, &pe.list.Position)
		}))
}

func (pe *ParamEditor) layoutUnitFooter(t *Tracker) layout.Widget {
	return func(gtx C) D {
		for pe.ClearUnitBtn.Clickable.Clicked() {
			t.SetUnitType("")
			op.InvalidateOp{}.Add(gtx.Ops)
			t.InstrumentEditor.unitDragList.Focus()
		}
		for pe.DeleteUnitBtn.Clickable.Clicked() {
			t.DeleteUnits(false, t.UnitIndex(), t.UnitIndex())
			op.InvalidateOp{}.Add(gtx.Ops)
			t.InstrumentEditor.unitDragList.Focus()
		}
		for pe.CopyUnitBtn.Clickable.Clicked() {
			op.InvalidateOp{}.Add(gtx.Ops)
			contents, err := yaml.Marshal([]sointu.Unit{t.Unit()})
			if err == nil {
				clipboard.WriteOp{Text: string(contents)}.Add(gtx.Ops)
				t.Alert.Update("Unit copied to clipboard", Notify, time.Second*3)
			}
		}
		copyUnitBtnStyle := IconButton(t.Theme, pe.CopyUnitBtn, icons.ContentContentCopy, true, "Copy unit (Ctrl+C)")
		deleteUnitBtnStyle := IconButton(t.Theme, pe.DeleteUnitBtn, icons.ActionDelete, t.CanDeleteUnit(), "Delete unit (Del)")
		text := t.Unit().Type
		if text == "" {
			text = "Choose unit type"
		} else {
			text = strings.Title(text)
		}
		hintText := Label(text, white)
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(deleteUnitBtnStyle.Layout),
			layout.Rigid(copyUnitBtnStyle.Layout),
			layout.Rigid(func(gtx C) D {
				var dims D
				if t.Unit().Type != "" {
					clearUnitBtnStyle := IconButton(t.Theme, pe.ClearUnitBtn, icons.ContentClear, true, "Clear unit")
					dims = clearUnitBtnStyle.Layout(gtx)
				}
				return D{Size: image.Pt(gtx.Dp(unit.Dp(48)), dims.Size.Y)}
			}),
			layout.Flexed(1, hintText),
		)
	}
}

func (pe *ParamEditor) layoutUnitTypeChooser(gtx C, t *Tracker) D {
	listElem := func(gtx C, i int) D {
		for pe.ChooseUnitTypeBtns[i].Clicked() {
			t.SetUnitType(sointu.UnitNames[i])
			t.InstrumentEditor.unitTypeEditor.SetText(sointu.UnitNames[i])
		}
		text := sointu.UnitNames[i]
		if t.InstrumentEditor.unitTypeEditor.Focused() && !strings.HasPrefix(text, t.InstrumentEditor.unitTypeEditor.Text()) {
			return D{}
		}
		labelStyle := LabelStyle{Text: text, ShadeColor: black, Color: white, Font: labelDefaultFont, FontSize: unit.Sp(12)}
		bg := func(gtx C) D {
			gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, 20))
			var color color.NRGBA
			if pe.ChooseUnitTypeBtns[i].Hovered() {
				color = unitTypeListHighlightColor
			}
			paint.FillShape(gtx.Ops, color, clip.Rect{Max: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}.Op())
			return D{Size: gtx.Constraints.Min}
		}
		leftMargin := layout.Inset{Left: unit.Dp(10)}
		return layout.Stack{Alignment: layout.W}.Layout(gtx,
			layout.Stacked(bg),
			layout.Expanded(func(gtx C) D {
				return pe.ChooseUnitTypeBtns[i].Layout(gtx, func(gtx C) D {
					return leftMargin.Layout(gtx, labelStyle.Layout)
				})
			}))
	}
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx C) D {
			return pe.list.Layout(gtx, len(sointu.UnitNames), listElem)
		}),
		layout.Expanded(func(gtx C) D {
			return pe.scrollBar.Layout(gtx, unit.Dp(10), len(sointu.UnitNames), &pe.list.Position)
		}),
	)
}
