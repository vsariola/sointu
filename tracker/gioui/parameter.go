package gioui

import (
	"fmt"
	"image"
	"math"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/vsariola/sointu/tracker"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type ParameterWidget struct {
	floatWidget widget.Float
	boolWidget  widget.Bool
	labelBtn    widget.Clickable
	instrBtn    widget.Clickable
	instrMenu   Menu
	unitBtn     widget.Clickable
	unitMenu    Menu
}

type ParameterStyle struct {
	tracker         *Tracker
	Parameter       *tracker.Parameter
	ParameterWidget *ParameterWidget
	Theme           *material.Theme
	Focus           bool
}

func (t *Tracker) ParamStyle(th *material.Theme, param *tracker.Parameter, paramWidget *ParameterWidget) ParameterStyle {
	return ParameterStyle{
		tracker:         t, // TODO: we need this to pull the instrument names for ID style parameters, find out another way
		Parameter:       param,
		Theme:           th,
		ParameterWidget: paramWidget,
	}
}

func (p *ParameterWidget) Clicked() bool {
	return p.labelBtn.Clicked()
}

func (p ParameterStyle) Layout(gtx C) D {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return p.ParameterWidget.labelBtn.Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(110))
				return layout.E.Layout(gtx, Label(p.Parameter.Name, white))
			})
		}),
		layout.Rigid(func(gtx C) D {
			switch p.Parameter.Type {
			case tracker.IntegerParameter:
				for _, e := range gtx.Events(&p.ParameterWidget.floatWidget) {
					switch ev := e.(type) {
					case pointer.Event:
						if ev.Type == pointer.Scroll {
							delta := math.Min(math.Max(float64(ev.Scroll.Y), -1), 1)
							p.Parameter.Value += int(math.Round(delta))
						}
					}
				}
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
				gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(40))
				if p.Focus {
					paint.FillShape(gtx.Ops, cursorColor, clip.Rect{
						Max: gtx.Constraints.Min,
					}.Op())
				}
				if !p.ParameterWidget.floatWidget.Dragging() {
					p.ParameterWidget.floatWidget.Value = float32(p.Parameter.Value)
				}
				sliderStyle := material.Slider(p.Theme, &p.ParameterWidget.floatWidget, float32(p.Parameter.Min), float32(p.Parameter.Max))
				sliderStyle.Color = p.Theme.Fg
				r := image.Rectangle{Max: gtx.Constraints.Min}
				area := clip.Rect(r).Push(gtx.Ops)
				pointer.InputOp{Tag: &p.ParameterWidget.floatWidget, Types: pointer.Scroll, ScrollBounds: image.Rectangle{Min: image.Pt(0, -1e6), Max: image.Pt(0, 1e6)}}.Add(gtx.Ops)
				dims := sliderStyle.Layout(gtx)
				area.Pop()
				p.Parameter.Value = int(p.ParameterWidget.floatWidget.Value + 0.5)
				return dims
			case tracker.BoolParameter:
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(60))
				gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(40))
				if p.Focus {
					paint.FillShape(gtx.Ops, cursorColor, clip.Rect{
						Max: gtx.Constraints.Min,
					}.Op())
				}
				p.ParameterWidget.boolWidget.Value = p.Parameter.Value > p.Parameter.Min
				boolStyle := material.Switch(p.Theme, &p.ParameterWidget.boolWidget, "Toggle boolean parameter")
				boolStyle.Color.Disabled = p.Theme.Fg
				boolStyle.Color.Enabled = white
				dims := layout.Center.Layout(gtx, boolStyle.Layout)
				if p.ParameterWidget.boolWidget.Value {
					p.Parameter.Value = p.Parameter.Max
				} else {
					p.Parameter.Value = p.Parameter.Min
				}
				return dims
			case tracker.IDParameter:
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
				gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(40))
				if p.Focus {
					paint.FillShape(gtx.Ops, cursorColor, clip.Rect{
						Max: gtx.Constraints.Min,
					}.Op())
				}
				for clickedItem, hasClicked := p.ParameterWidget.instrMenu.Clicked(); hasClicked; {
					p.Parameter.Value = p.tracker.Song().Patch[clickedItem].Units[0].ID
					clickedItem, hasClicked = p.ParameterWidget.instrMenu.Clicked()
				}
				instrItems := make([]MenuItem, len(p.tracker.Song().Patch))
				for i, instr := range p.tracker.Song().Patch {
					instrItems[i].Text = instr.Name
					instrItems[i].IconBytes = icons.NavigationChevronRight
				}
				var unitItems []MenuItem
				instrName := "<instr>"
				unitName := "<unit>"
				targetI, targetU, err := p.tracker.Song().Patch.FindUnit(p.Parameter.Value)
				if err == nil {
					targetInstrument := p.tracker.Song().Patch[targetI]
					instrName = targetInstrument.Name
					units := targetInstrument.Units
					unitName = fmt.Sprintf("%v: %v", targetU, units[targetU].Type)
					unitItems = make([]MenuItem, len(units))
					for clickedItem, hasClicked := p.ParameterWidget.unitMenu.Clicked(); hasClicked; {
						p.Parameter.Value = units[clickedItem].ID
						clickedItem, hasClicked = p.ParameterWidget.unitMenu.Clicked()
					}
					for j, unit := range units {
						unitItems[j].Text = fmt.Sprintf("%v: %v", j, unit.Type)
						unitItems[j].IconBytes = icons.NavigationChevronRight
					}
				}
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(p.tracker.layoutMenu(instrName, &p.ParameterWidget.instrBtn, &p.ParameterWidget.instrMenu, unit.Dp(200),
						instrItems...,
					)),
					layout.Rigid(p.tracker.layoutMenu(unitName, &p.ParameterWidget.unitBtn, &p.ParameterWidget.unitMenu, unit.Dp(200),
						unitItems...,
					)),
				)
			}
			return D{}
		}),
		layout.Rigid(func(gtx C) D {
			if p.Parameter.Type != tracker.IDParameter {
				return Label(p.Parameter.Hint, white)(gtx)
			}
			return D{}
		}),
	)
}

/*

func (t *Tracker) layoutParameter(gtx C, index int) D {
	u := t.Unit()
	ut, _ := sointu.UnitTypes[u.Type]

	params := u.Parameters
	var name string
	var value, min, max int
	var valueText string
	if u.Type == "oscillator" && index == len(ut) {
		name = "sample"
		key := compiler.SampleOffset{Start: uint32(params["samplestart"]), LoopStart: uint16(params["loopstart"]), LoopLength: uint16(params["looplength"])}
		if v, ok := tracker.GmDlsEntryMap[key]; ok {
			value = v + 1
			valueText = fmt.Sprintf("%v / %v", value, tracker.GmDlsEntries[v].Name)
		} else {
			value = 0
			valueText = "0 / custom"
		}
		min, max = 0, len(tracker.GmDlsEntries)
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
			max = t.Song().Patch.NumVoices()
		} else if u.Type == "send" && name == "unit" { // set the maximum values depending on the send target
			instrIndex, _, _ := t.Song().Patch.FindSendTarget(t.Unit().Parameters["target"])
			if instrIndex != -1 {
				max = len(t.Song().Patch[instrIndex].Units) - 1
			}
		} else if u.Type == "send" && name == "port" { // set the maximum values depending on the send target
			instrIndex, unitIndex, _ := t.Song().Patch.FindSendTarget(t.Unit().Parameters["target"])
			if instrIndex != -1 && unitIndex != -1 {
				max = len(sointu.Ports[t.Song().Patch[instrIndex].Units[unitIndex].Type]) - 1
			}
		}
		hint := t.Song().Patch.ParamHintString(t.InstrIndex(), t.UnitIndex(), name)
		if hint != "" {
			valueText = fmt.Sprintf("%v / %v", value, hint)
		} else {
			valueText = fmt.Sprintf("%v", value)
		}
	}

}*/
