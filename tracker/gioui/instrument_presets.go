package gioui

import (
	"image"

	"gioui.org/io/key"
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
		saveUserPreset    *Clickable
		deleteUserPreset  *Clickable
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
		saveUserPreset:    new(Clickable),
		deleteUserPreset:  new(Clickable),
		dirList:           NewDragList(m.PresetDirList().List(), layout.Vertical),
		resultList:        NewDragList(m.PresetResultList().List(), layout.Vertical),
	}
}

func (ip *InstrumentPresets) Tags(level int, yield TagYieldFunc) bool {
	return yield(level, &ip.searchEditor.widgetEditor) &&
		yield(level+1, ip.clearSearchBtn) &&
		yield(level+1, ip.builtinPresetsBtn) &&
		yield(level+1, ip.userPresetsBtn) &&
		yield(level+1, ip.gmDlsBtn) &&
		yield(level, ip.dirList) &&
		yield(level, ip.resultList) &&
		yield(level+1, ip.saveUserPreset) &&
		yield(level+1, ip.deleteUserPreset)
}

func (ip *InstrumentPresets) update(gtx C) {
	for {
		event, ok := gtx.Event(
			key.Filter{Focus: ip.resultList, Name: key.NameLeftArrow},
		)
		if !ok {
			break
		}
		if e, ok := event.(key.Event); ok && e.State == key.Press {
			switch e.Name {
			case key.NameLeftArrow:
				ip.dirList.Focus()
			}
		}
	}
	for {
		event, ok := gtx.Event(
			key.Filter{Focus: ip.dirList, Name: key.NameRightArrow},
		)
		if !ok {
			break
		}
		if e, ok := event.(key.Event); ok && e.State == key.Press {
			switch e.Name {
			case key.NameRightArrow:
				ip.resultList.Focus()
			}
		}
	}
}

func (ip *InstrumentPresets) layout(gtx C) D {
	ip.update(gtx)
	// get tracker from values
	tr := TrackerFromContext(gtx)
	gmDlsBtn := ToggleBtn(tr.NoGmDls(), tr.Theme, ip.gmDlsBtn, "No gm.dls", "Exclude presets using gm.dls")
	userPresetsFilterBtn := ToggleBtn(tr.UserPresetFilter(), tr.Theme, ip.userPresetsBtn, "User", "Show only user presets")
	builtinPresetsFilterBtn := ToggleBtn(tr.BuiltinPresetsFilter(), tr.Theme, ip.builtinPresetsBtn, "Builtin", "Show only builtin presets")
	saveUserPresetBtn := ActionIconBtn(tr.SaveAsUserPreset(), tr.Theme, ip.saveUserPreset, icons.ContentSave, "Save instrument as user preset")
	deleteUserPresetBtn := ActionIconBtn(tr.TryDeleteUserPreset(), tr.Theme, ip.deleteUserPreset, icons.ActionDelete, "Delete user preset")
	dirElem := func(gtx C, i int) D {
		return Label(tr.Theme, &tr.Theme.InstrumentEditor.Presets.Directory, tr.Model.PresetDirList().Value(i)).Layout(gtx)
	}
	dirs := func(gtx C) D {
		gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(140), gtx.Constraints.Max.Y))
		fdl := FilledDragList(tr.Theme, ip.dirList)
		dims := fdl.Layout(gtx, dirElem, nil)
		fdl.LayoutScrollBar(gtx)
		return dims
	}
	dirSurface := func(gtx C) D {
		return Surface{Gray: 36, Focus: tr.PatchPanel.TreeFocused(gtx)}.Layout(gtx, dirs)
	}
	resultElem := func(gtx C, i int) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		n, d, u := tr.Model.PresetResultList().Value(i)
		if u {
			ln := Label(tr.Theme, &tr.Theme.InstrumentEditor.Presets.Results.User, n)
			ld := Label(tr.Theme, &tr.Theme.InstrumentEditor.Presets.Results.UserDir, d)
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(ln.Layout),
				layout.Rigid(layout.Spacer{Width: 6}.Layout),
				layout.Rigid(ld.Layout),
			)
		}
		return Label(tr.Theme, &tr.Theme.InstrumentEditor.Presets.Results.Builtin, n).Layout(gtx)
	}
	floatButtons := func(gtx C) D {
		if tr.Model.DeleteUserPreset().Enabled() {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(deleteUserPresetBtn.Layout),
				layout.Rigid(saveUserPresetBtn.Layout),
				layout.Rigid(layout.Spacer{Width: 10}.Layout),
			)
		}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(saveUserPresetBtn.Layout),
			layout.Rigid(layout.Spacer{Width: 10}.Layout),
		)
	}
	results := func(gtx C) D {
		gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
		fdl := FilledDragList(tr.Theme, ip.resultList)
		dims := fdl.Layout(gtx, resultElem, nil)
		layout.SE.Layout(gtx, floatButtons)
		fdl.LayoutScrollBar(gtx)
		return dims
	}
	resultSurface := func(gtx C) D {
		return Surface{Gray: 30, Focus: tr.PatchPanel.TreeFocused(gtx)}.Layout(gtx, results)
	}
	bottom := func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(dirSurface),
			layout.Flexed(1, resultSurface),
		)
	}
	// layout
	f := func(gtx C) D {
		m := gtx.Constraints.Max
		gtx.Constraints.Max.X = min(gtx.Dp(360), gtx.Constraints.Max.X)
		layout.Flex{Axis: layout.Vertical, Alignment: layout.Start}.Layout(gtx,
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
		return D{Size: m}
	}
	return Surface{Gray: 24, Focus: tr.PatchPanel.TreeFocused(gtx)}.Layout(gtx, f)
}

func (ip *InstrumentPresets) layoutSearch(gtx C) D {
	// draw search icon on left  and clear button on right
	// return ip.searchEditor.Layout(gtx, tr.Model.PresetSearchString(), tr.Theme, &tr.Theme.InstrumentEditor.InstrumentComment, "Search presets")
	tr := TrackerFromContext(gtx)
	bg := func(gtx C) D {
		rr := gtx.Dp(18)
		defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, rr).Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, tr.Theme.InstrumentEditor.Presets.SearchBg)
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
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(icon),
			layout.Flexed(1, ed),
			layout.Rigid(clr),
		)
	}
	return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(bg),
			layout.Stacked(w),
		)
	})
}
