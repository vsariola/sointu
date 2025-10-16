package gioui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"github.com/vsariola/sointu/tracker"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type (
	InstrumentPresets struct {
		searchEditor      *Editor
		gmDlsBtn          *Clickable
		userPresetsBtn    *Clickable
		builtinPresetsBtn *Clickable
		clearSearchBtn    *Clickable
		dirBtn            *Clickable
		dirList           *DragList
		resultList        *DragList
	}
)

func NewInstrumentPresets(m *tracker.Model) *InstrumentPresets {
	return &InstrumentPresets{
		searchEditor:      NewEditor(true, true, text.Start),
		gmDlsBtn:          new(Clickable),
		clearSearchBtn:    new(Clickable),
		userPresetsBtn:    new(Clickable),
		builtinPresetsBtn: new(Clickable),
		dirBtn:            new(Clickable),
		dirList:           NewDragList(m.PresetDirList().List(), layout.Vertical),
		resultList:        NewDragList(m.PresetResultList().List(), layout.Vertical),
	}
}

func (ip *InstrumentPresets) layout(gtx C) D {
	// get tracker from values
	tr := TrackerFromContext(gtx)
	gmDlsBtn := ToggleBtn(tr.NoGmDls(), tr.Theme, ip.gmDlsBtn, "No gm.dls", "Exclude presets using gm.dls")
	userPresetsFilterBtn := ToggleBtn(tr.UserPresetFilter(), tr.Theme, ip.userPresetsBtn, "User", "Show only user presets")
	builtinPresetsFilterBtn := ToggleBtn(tr.BuiltinPresetsFilter(), tr.Theme, ip.builtinPresetsBtn, "Builtin", "Show only builtin presets")
	dirElem := func(gtx C, i int) D {
		return Label(tr.Theme, &tr.Theme.Dialog.Text, tr.Model.PresetDirList().Value(i)).Layout(gtx)
	}
	dirs := func(gtx C) D {
		gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(140), gtx.Constraints.Max.Y))
		style := FilledDragList(tr.Theme, ip.dirList)
		dims := style.Layout(gtx, dirElem, nil)
		style.LayoutScrollBar(gtx)
		return dims
	}
	dirSurface := func(gtx C) D {
		return Surface{Gray: 30, Focus: tr.PatchPanel.TreeFocused(gtx)}.Layout(gtx, dirs)
	}
	resultElem := func(gtx C, i int) D {
		return Label(tr.Theme, &tr.Theme.Dialog.Text, tr.Model.PresetResultList().Value(i)).Layout(gtx)
	}
	results := func(gtx C) D {
		return FilledDragList(tr.Theme, ip.resultList).Layout(gtx, resultElem, nil)
	}
	bottom := func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(dirSurface),
			layout.Flexed(1, results),
		)
	}
	// layout
	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Start, Spacing: 6}.Layout(gtx,
		layout.Rigid(ip.layoutSearch),
		layout.Rigid(func(gtx C) D {
			return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(userPresetsFilterBtn.Layout),
					layout.Rigid(builtinPresetsFilterBtn.Layout),
					layout.Rigid(gmDlsBtn.Layout),
				)
			})
		}),
		layout.Rigid(bottom),
	)
}

func (ip *InstrumentPresets) layoutSearch(gtx C) D {
	// draw search icon on left  and clear button on right
	// return ip.searchEditor.Layout(gtx, tr.Model.PresetSearchString(), tr.Theme, &tr.Theme.InstrumentEditor.InstrumentComment, "Search presets")
	tr := TrackerFromContext(gtx)
	bg := func(gtx C) D {
		rr := gtx.Dp(18)
		defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, rr).Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, tr.Theme.Material.ContrastFg)
		return D{Size: gtx.Constraints.Min}
	}
	// icon, search editor, clear button
	icon := func(gtx C) D {
		return tr.Theme.IconButton.Enabled.Inset.Layout(gtx, func(gtx C) D {
			return tr.Theme.Icon(icons.ActionSearch).Layout(gtx, tr.Theme.Material.Fg)
		})
	}
	ed := func(gtx C) D {
		return ip.searchEditor.Layout(gtx, tr.Model.PresetSearchString(), tr.Theme, &tr.Theme.InstrumentEditor.UnitComment, "Search presets")
	}
	clr := func(gtx C) D {
		btn := ActionIconBtn(tr.ClearPresetSearch(), tr.Theme, ip.clearSearchBtn, icons.ContentClear, "Clear search")
		return btn.Layout(gtx)
	}
	w := func(gtx C) D {
		gtx.Constraints.Max.X = min(gtx.Dp(360), gtx.Constraints.Max.X)
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(icon),
			layout.Flexed(1, ed),
			layout.Rigid(clr),
		)
	}
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(bg),
		layout.Stacked(w),
	)
}
