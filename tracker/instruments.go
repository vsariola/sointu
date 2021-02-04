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
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type C = layout.Context
type D = layout.Dimensions

func (t *Tracker) updateInstrumentScroll() {
	if t.CurrentInstrument > 7 {
		t.InstrumentList.Position.First = t.CurrentInstrument - 7
	} else {
		t.InstrumentList.Position.First = 0
	}
}

func (t *Tracker) layoutInstruments() layout.Widget {
	btnStyle := material.IconButton(t.Theme, t.NewInstrumentBtn, widgetForIcon(icons.ContentAdd))
	btnStyle.Background = transparent
	btnStyle.Inset = layout.UniformInset(unit.Dp(6))
	if t.song.Patch.TotalVoices() < 32 {
		btnStyle.Color = primaryColor
	} else {
		btnStyle.Color = disabledTextColor
	}
	return func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Flex{}.Layout(
					gtx,
					layout.Flexed(1, t.layoutInstrumentNames()),
					layout.Rigid(func(gtx C) D {
						return layout.E.Layout(gtx, btnStyle.Layout)
					}),
				)
			}),
			layout.Rigid(t.layoutInstrumentHeader()),
			layout.Flexed(1, t.layoutInstrumentEditor()))
	}
}

func (t *Tracker) layoutInstrumentHeader() layout.Widget {
	headerBg := func(gtx C) D {
		paint.FillShape(gtx.Ops, instrumentSurfaceColor, clip.Rect{
			Max: gtx.Constraints.Min,
		}.Op())
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}
	header := func(gtx C) D {
		deleteInstrumentBtnStyle := material.IconButton(t.Theme, t.DeleteInstrumentBtn, widgetForIcon(icons.ActionDelete))
		deleteInstrumentBtnStyle.Background = transparent
		deleteInstrumentBtnStyle.Inset = layout.UniformInset(unit.Dp(6))
		if len(t.song.Patch.Instruments) > 1 {
			deleteInstrumentBtnStyle.Color = primaryColor
		} else {
			deleteInstrumentBtnStyle.Color = disabledTextColor
		}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(Label("Voices:", white)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				maxRemain := 32 - t.song.Patch.TotalVoices() + t.song.Patch.Instruments[t.CurrentInstrument].NumVoices
				if maxRemain < 0 {
					maxRemain = 0
				}
				t.InstrumentVoices.Value = t.song.Patch.Instruments[t.CurrentInstrument].NumVoices
				numStyle := NumericUpDown(t.Theme, t.InstrumentVoices, 0, maxRemain)
				gtx.Constraints.Min.Y = gtx.Px(unit.Dp(20))
				gtx.Constraints.Min.X = gtx.Px(unit.Dp(70))
				dims := numStyle.Layout(gtx)
				t.SetInstrumentVoices(t.InstrumentVoices.Value)
				return dims
			}),
			layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
			layout.Rigid(deleteInstrumentBtnStyle.Layout))
	}
	for t.DeleteInstrumentBtn.Clicked() {
		t.DeleteInstrument()
	}
	return func(gtx C) D {
		return layout.Stack{Alignment: layout.Center}.Layout(gtx,
			layout.Expanded(headerBg),
			layout.Stacked(header))
	}
}

func (t *Tracker) layoutInstrumentNames() layout.Widget {
	return func(gtx C) D {
		gtx.Constraints.Max.Y = gtx.Px(unit.Dp(36))
		gtx.Constraints.Min.Y = gtx.Px(unit.Dp(36))

		count := len(t.song.Patch.Instruments)
		if len(t.InstrumentBtns) < count {
			tail := make([]*widget.Clickable, count-len(t.InstrumentBtns))
			for t := range tail {
				tail[t] = new(widget.Clickable)
			}
			t.InstrumentBtns = append(t.InstrumentBtns, tail...)
		}

		defer op.Save(gtx.Ops).Load()

		t.InstrumentList.Layout(gtx, count, func(gtx C, index int) D {
			for t.InstrumentBtns[index].Clicked() {
				t.CurrentInstrument = index
			}
			btnStyle := material.Button(t.Theme, t.InstrumentBtns[index], fmt.Sprintf("%v", index))
			btnStyle.CornerRadius = unit.Dp(0)
			btnStyle.Color = t.Theme.Fg
			if t.CurrentInstrument == index {
				btnStyle.Background = instrumentSurfaceColor
			} else {
				btnStyle.Background = transparent
			}
			return btnStyle.Layout(gtx)
		})

		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}
func (t *Tracker) layoutInstrumentEditor() layout.Widget {
	for t.AddUnitBtn.Clicked() {
		t.AddUnit()
	}
	addUnitBtnStyle := material.IconButton(t.Theme, t.AddUnitBtn, widgetForIcon(icons.ContentAdd))
	addUnitBtnStyle.Inset = layout.UniformInset(unit.Dp(4))
	margin := layout.UniformInset(unit.Dp(2))

	return func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Stack{Alignment: layout.SE}.Layout(gtx,
					layout.Expanded(t.layoutUnitList()),
					layout.Stacked(func(gtx C) D {
						return margin.Layout(gtx, addUnitBtnStyle.Layout)
					}))
			}),
			layout.Rigid(t.layoutUnitEditor()))
	}
}

func (t *Tracker) layoutUnitList() layout.Widget {
	return func(gtx C) D {
		paint.FillShape(gtx.Ops, unitListSurfaceColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}.Op())
		defer op.Save(gtx.Ops).Load()

		gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
		units := t.song.Patch.Instruments[t.CurrentInstrument].Units
		count := len(units)
		for len(t.UnitBtns) < count {
			t.UnitBtns = append(t.UnitBtns, new(widget.Clickable))
		}

		listElem := func(gtx C, i int) D {
			for t.UnitBtns[i].Clicked() {
				t.CurrentUnit = i
				op.InvalidateOp{}.Add(gtx.Ops)
			}
			u := t.song.Patch.Instruments[t.CurrentInstrument].Units[i]
			labelStyle := LabelStyle{Text: u.Type, ShadeColor: black, Color: white, Font: labelDefaultFont, FontSize: unit.Sp(12)}
			if labelStyle.Text == "" {
				labelStyle.Text = "---"
				labelStyle.Alignment = layout.Center
			}
			bg := func(gtx C) D {
				gtx.Constraints = layout.Exact(image.Pt(120, 20))
				var color color.NRGBA
				if t.CurrentUnit == i {
					color = unitListSelectedColor
				} else if t.UnitBtns[i].Hovered() {
					color = unitListHighlightColor
				}
				paint.FillShape(gtx.Ops, color, clip.Rect{Max: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}.Op())
				return D{Size: gtx.Constraints.Min}
			}
			return layout.Stack{Alignment: layout.W}.Layout(gtx,
				layout.Stacked(bg),
				layout.Expanded(labelStyle.Layout),
				layout.Expanded(t.UnitBtns[i].Layout))
		}
		return t.UnitList.Layout(gtx, len(t.song.Patch.Instruments[t.CurrentInstrument].Units), listElem)
	}
}
