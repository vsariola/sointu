package tracker

import (
	"fmt"
	"sort"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
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
	return func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(t.layoutInstrumentNames()),
			layout.Flexed(1, t.layoutInstrumentEditor()))
	}
}

func (t *Tracker) layoutInstrumentNames() layout.Widget {
	return func(gtx C) D {
		count := len(t.song.Patch.Instruments)
		if len(t.InstrumentBtns) < count {
			tail := make([]*widget.Clickable, count-len(t.InstrumentBtns))
			for t := range tail {
				tail[t] = new(widget.Clickable)
			}
			t.InstrumentBtns = append(t.InstrumentBtns, tail...)
		}
		return t.InstrumentList.Layout(gtx, count, func(gtx C, index int) D {
			for t.InstrumentBtns[index].Clicked() {
				t.CurrentInstrument = index
			}
			btnStyle := material.Button(t.Theme, t.InstrumentBtns[index], fmt.Sprintf("%v", index))
			btnStyle.CornerRadius = unit.Dp(0)
			if t.CurrentInstrument == index {
				btnStyle.Background = instrumentSurfaceColor
			} else {
				btnStyle.Background = transparent
			}
			return btnStyle.Layout(gtx)
		})
	}
}
func (t *Tracker) layoutInstrumentEditor() layout.Widget {
	return func(gtx C) D {
		paint.FillShape(gtx.Ops, instrumentSurfaceColor, clip.Rect{
			Max: gtx.Constraints.Max,
		}.Op())
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(t.layoutUnitList()),
			layout.Rigid(t.layoutUnitControls()))
	}
}

func (t *Tracker) layoutUnitList() layout.Widget {
	return func(gtx C) D {
		units := t.song.Patch.Instruments[t.CurrentInstrument].Units
		count := len(units)
		if len(t.UnitBtns) < count {
			tail := make([]*widget.Clickable, count-len(t.UnitBtns))
			for t := range tail {
				tail[t] = new(widget.Clickable)
			}
			t.UnitBtns = append(t.UnitBtns, tail...)
		}
		children := make([]layout.FlexChild, len(t.song.Patch.Instruments[t.CurrentInstrument].Units))
		for i, u := range t.song.Patch.Instruments[t.CurrentInstrument].Units {
			for t.UnitBtns[i].Clicked() {
				t.CurrentUnit = i
			}
			btnStyle := material.Button(t.Theme, t.UnitBtns[i], u.Type)
			btnStyle.Background = transparent
			children[i] = layout.Rigid(btnStyle.Layout)
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

func (t *Tracker) layoutUnitControls() layout.Widget {
	return func(gtx C) D {
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
}
