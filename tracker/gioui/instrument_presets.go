package gioui

import (
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"github.com/vsariola/sointu/tracker"
)

type (
	InstrumentPresets struct {
		searchEditor      *Editor
		gmDlsBtn          *Clickable
		userPresetsBtn    *Clickable
		builtinPresetsBtn *Clickable
		clearSearchBtn    *Clickable
		dirList           *DragList
		resultList        *DragList
	}
)

func NewInstrumentPresets(m *tracker.Model) *InstrumentPresets {
	return &InstrumentPresets{
		searchEditor:      NewEditor(false, false, text.Start),
		gmDlsBtn:          new(Clickable),
		clearSearchBtn:    new(Clickable),
		userPresetsBtn:    new(Clickable),
		builtinPresetsBtn: new(Clickable),
		dirList:           NewDragList(m.Instruments().List(), layout.Vertical),
		resultList:        NewDragList(m.Instruments().List(), layout.Vertical),
	}
}

func (ip *InstrumentPresets) layout(gtx C) D {
	// get tracker from values
	tr := TrackerFromContext(gtx)
	gmDlsBtn := ToggleBtn(tr.NoGmDls(), tr.Theme, ip.gmDlsBtn, "No gm.dls", "Exclude presets using gm.dls")
	userPresetsFilterBtn := ToggleBtn(tr.UserPresetFilter(), tr.Theme, ip.userPresetsBtn, "User", "Show only user presets")
	builtinPresetsFilterBtn := ToggleBtn(tr.BuiltinPresetsFilter(), tr.Theme, ip.builtinPresetsBtn, "Builtin", "Show only builtin presets")
	// layout
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx C) D {
						return ip.searchEditor.Layout(gtx, tr.Model.PresetSearchString(), tr.Theme, &tr.Theme.InstrumentEditor.InstrumentComment, "Search presets")
					}),
					layout.Rigid(userPresetsFilterBtn.Layout),
					layout.Rigid(builtinPresetsFilterBtn.Layout),
					layout.Rigid(gmDlsBtn.Layout),
				)
			})
		}),
	)
}
