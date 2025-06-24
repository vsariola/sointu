package gioui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"strconv"
	"strings"

	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/text"
	"gioui.org/unit"
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/tracker"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type (
	InstrumentEditor struct {
		newInstrumentBtn    *Clickable
		enlargeBtn          *Clickable
		deleteInstrumentBtn *Clickable
		linkInstrTrackBtn   *Clickable
		splitInstrumentBtn  *Clickable
		copyInstrumentBtn   *Clickable
		saveInstrumentBtn   *Clickable
		loadInstrumentBtn   *Clickable
		addUnitBtn          *Clickable
		presetMenuBtn       *Clickable
		commentExpandBtn    *Clickable
		soloBtn             *Clickable
		muteBtn             *Clickable
		commentEditor       *Editor
		nameEditor          *Editor
		searchEditor        *Editor
		instrumentDragList  *DragList
		unitDragList        *DragList
		unitEditor          *UnitEditor
		wasFocused          bool
		presetMenuItems     []MenuItem
		presetMenu          Menu

		addUnit tracker.Action

		enlargeHint, shrinkHint string
		addInstrumentHint       string
		octaveHint              string
		expandCommentHint       string
		collapseCommentHint     string
		deleteInstrumentHint    string
		muteHint, unmuteHint    string
		soloHint, unsoloHint    string
		linkDisabledHint        string
		linkEnabledHint         string
		splitInstrumentHint     string
	}

	AddUnitThenFocus InstrumentEditor
)

func NewInstrumentEditor(model *tracker.Model) *InstrumentEditor {
	ret := &InstrumentEditor{
		newInstrumentBtn:    new(Clickable),
		enlargeBtn:          new(Clickable),
		deleteInstrumentBtn: new(Clickable),
		linkInstrTrackBtn:   new(Clickable),
		splitInstrumentBtn:  new(Clickable),
		copyInstrumentBtn:   new(Clickable),
		saveInstrumentBtn:   new(Clickable),
		loadInstrumentBtn:   new(Clickable),
		commentExpandBtn:    new(Clickable),
		presetMenuBtn:       new(Clickable),
		soloBtn:             new(Clickable),
		muteBtn:             new(Clickable),
		addUnitBtn:          new(Clickable),
		commentEditor:       NewEditor(false, false, text.Start),
		nameEditor:          NewEditor(true, true, text.Middle),
		searchEditor:        NewEditor(true, true, text.Start),
		instrumentDragList:  NewDragList(model.Instruments().List(), layout.Horizontal),
		unitDragList:        NewDragList(model.Units().List(), layout.Vertical),
		unitEditor:          NewUnitEditor(model),
		presetMenuItems:     []MenuItem{},
	}
	model.IterateInstrumentPresets(func(index int, name string) bool {
		ret.presetMenuItems = append(ret.presetMenuItems, MenuItem{Text: name, IconBytes: icons.ImageAudiotrack, Doer: model.LoadPreset(index)})
		return true
	})
	ret.addUnit = model.AddUnit(false)
	ret.enlargeHint = makeHint("Enlarge", " (%s)", "InstrEnlargedToggle")
	ret.shrinkHint = makeHint("Shrink", " (%s)", "InstrEnlargedToggle")
	ret.addInstrumentHint = makeHint("Add\ninstrument", "\n(%s)", "AddInstrument")
	ret.octaveHint = makeHint("Octave down", " (%s)", "OctaveNumberInputSubtract") + makeHint(" or up", " (%s)", "OctaveNumberInputAdd")
	ret.expandCommentHint = makeHint("Expand comment", " (%s)", "CommentExpandedToggle")
	ret.collapseCommentHint = makeHint("Collapse comment", " (%s)", "CommentExpandedToggle")
	ret.deleteInstrumentHint = makeHint("Delete\ninstrument", "\n(%s)", "DeleteInstrument")
	ret.muteHint = makeHint("Mute", " (%s)", "MuteToggle")
	ret.unmuteHint = makeHint("Unmute", " (%s)", "MuteToggle")
	ret.soloHint = makeHint("Solo", " (%s)", "SoloToggle")
	ret.unsoloHint = makeHint("Unsolo", " (%s)", "SoloToggle")
	ret.linkDisabledHint = makeHint("Instrument-Track\nlinking disabled", "\n(%s)", "LinkInstrTrackToggle")
	ret.linkEnabledHint = makeHint("Instrument-Track\nlinking enabled", "\n(%s)", "LinkInstrTrackToggle")
	ret.splitInstrumentHint = makeHint("Split instrument", " (%s)", "SplitInstrument")
	return ret
}

func (ie *InstrumentEditor) AddUnitThenFocus() tracker.Action {
	return tracker.MakeAction((*AddUnitThenFocus)(ie))
}
func (a *AddUnitThenFocus) Enabled() bool { return a.addUnit.Enabled() }
func (a *AddUnitThenFocus) Do() {
	a.addUnit.Do()
	a.searchEditor.Focus()
}

func (ie *InstrumentEditor) Focus() {
	ie.unitDragList.Focus()
}

func (ie *InstrumentEditor) Focused(gtx C) bool {
	return gtx.Focused(ie.unitDragList)
}

func (ie *InstrumentEditor) childFocused(gtx C) bool {
	return ie.unitEditor.sliderList.Focused(gtx) ||
		ie.instrumentDragList.Focused(gtx) || gtx.Source.Focused(ie.commentEditor) || gtx.Source.Focused(ie.nameEditor) || gtx.Source.Focused(ie.searchEditor) ||
		gtx.Source.Focused(ie.addUnitBtn) || gtx.Source.Focused(ie.commentExpandBtn) || gtx.Source.Focused(ie.presetMenuBtn) ||
		gtx.Source.Focused(ie.deleteInstrumentBtn) || gtx.Source.Focused(ie.copyInstrumentBtn)
}

func (ie *InstrumentEditor) Layout(gtx C, t *Tracker) D {
	ie.wasFocused = ie.Focused(gtx) || ie.childFocused(gtx)

	octave := func(gtx C) D {
		in := layout.UniformInset(unit.Dp(1))
		octave := NumUpDown(t.Model.Octave(), t.Theme, t.OctaveNumberInput, "Octave")
		return in.Layout(gtx, octave.Layout)
	}

	ret := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(
				gtx,
				layout.Flexed(1, func(gtx C) D {
					return ie.layoutInstrumentList(gtx, t)
				}),
				layout.Rigid(layout.Spacer{Width: 10}.Layout),
				layout.Rigid(Label(t.Theme, &t.Theme.InstrumentEditor.Octave, "Octave").Layout),
				layout.Rigid(layout.Spacer{Width: 4}.Layout),
				layout.Rigid(octave),
				layout.Rigid(func(gtx C) D {
					linkInstrTrackBtn := ToggleIconBtn(t.Model.LinkInstrTrack(), t.Theme, ie.linkInstrTrackBtn, icons.NotificationSyncDisabled, icons.NotificationSync, ie.linkDisabledHint, ie.linkEnabledHint)
					return layout.E.Layout(gtx, linkInstrTrackBtn.Layout)
				}),
				layout.Rigid(func(gtx C) D {
					instrEnlargedBtn := ToggleIconBtn(t.Model.InstrEnlarged(), t.Theme, ie.enlargeBtn, icons.NavigationFullscreen, icons.NavigationFullscreenExit, ie.enlargeHint, ie.shrinkHint)
					return layout.E.Layout(gtx, instrEnlargedBtn.Layout)
				}),
				layout.Rigid(func(gtx C) D {
					addInstrumentBtn := ActionIconBtn(t.Model.AddInstrument(), t.Theme, ie.newInstrumentBtn, icons.ContentAdd, ie.addInstrumentHint)
					return layout.E.Layout(gtx, addInstrumentBtn.Layout)
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return ie.layoutInstrumentHeader(gtx, t)
		}),
		layout.Flexed(1, func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return ie.layoutUnitList(gtx, t)
				}),
				layout.Flexed(1, func(gtx C) D {
					return ie.unitEditor.Layout(gtx, t)
				}),
			)
		}))
	return ret
}

func (ie *InstrumentEditor) layoutInstrumentHeader(gtx C, t *Tracker) D {
	header := func(gtx C) D {
		m := PopupMenu(t.Theme, &t.Theme.Menu.Text, &ie.presetMenu)

		for ie.copyInstrumentBtn.Clicked(gtx) {
			if contents, ok := t.Instruments().List().CopyElements(); ok {
				gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(bytes.NewReader(contents))})
				t.Alerts().Add("Instrument copied to clipboard", tracker.Info)
			}
		}

		for ie.saveInstrumentBtn.Clicked(gtx) {
			writer, err := t.Explorer.CreateFile(t.InstrumentName().Value() + ".yml")
			if err != nil {
				continue
			}
			t.SaveInstrument(writer)
		}

		for ie.loadInstrumentBtn.Clicked(gtx) {
			reader, err := t.Explorer.ChooseFile(".yml", ".json", ".4ki", ".4kp")
			if err != nil {
				continue
			}
			t.LoadInstrument(reader)
		}

		splitInstrumentBtn := ActionIconBtn(t.SplitInstrument(), t.Theme, ie.splitInstrumentBtn, icons.CommunicationCallSplit, ie.splitInstrumentHint)
		commentExpandedBtn := ToggleIconBtn(t.CommentExpanded(), t.Theme, ie.commentExpandBtn, icons.NavigationExpandMore, icons.NavigationExpandLess, ie.expandCommentHint, ie.collapseCommentHint)
		soloBtn := ToggleIconBtn(t.Solo(), t.Theme, ie.soloBtn, icons.SocialGroup, icons.SocialPerson, ie.soloHint, ie.unsoloHint)
		muteBtn := ToggleIconBtn(t.Mute(), t.Theme, ie.muteBtn, icons.AVVolumeUp, icons.AVVolumeOff, ie.muteHint, ie.unmuteHint)
		saveInstrumentBtn := IconBtn(t.Theme, &t.Theme.IconButton.Enabled, ie.saveInstrumentBtn, icons.ContentSave, "Save instrument")
		loadInstrumentBtn := IconBtn(t.Theme, &t.Theme.IconButton.Enabled, ie.loadInstrumentBtn, icons.FileFolderOpen, "Load instrument")
		copyInstrumentBtn := IconBtn(t.Theme, &t.Theme.IconButton.Enabled, ie.copyInstrumentBtn, icons.ContentContentCopy, "Copy instrument")
		deleteInstrumentBtn := ActionIconBtn(t.DeleteInstrument(), t.Theme, ie.deleteInstrumentBtn, icons.ActionDelete, ie.deleteInstrumentHint)
		instrumentVoices := NumUpDown(t.Model.InstrumentVoices(), t.Theme, t.InstrumentVoices, "Number of voices for this instrument")

		header := func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(layout.Spacer{Width: 6}.Layout),
				layout.Rigid(Label(t.Theme, &t.Theme.InstrumentEditor.Voices, "Voices").Layout),
				layout.Rigid(layout.Spacer{Width: 4}.Layout),
				layout.Rigid(instrumentVoices.Layout),
				layout.Rigid(splitInstrumentBtn.Layout),
				layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
				layout.Rigid(commentExpandedBtn.Layout),
				layout.Rigid(soloBtn.Layout),
				layout.Rigid(muteBtn.Layout),
				layout.Rigid(func(gtx C) D {
					presetBtn := IconBtn(t.Theme, &t.Theme.IconButton.Enabled, ie.presetMenuBtn, icons.NavigationMenu, "Load preset")
					dims := presetBtn.Layout(gtx)
					op.Offset(image.Pt(0, dims.Size.Y)).Add(gtx.Ops)
					gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(500))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(180))
					m.Layout(gtx, ie.presetMenuItems...)
					return dims
				}),
				layout.Rigid(saveInstrumentBtn.Layout),
				layout.Rigid(loadInstrumentBtn.Layout),
				layout.Rigid(copyInstrumentBtn.Layout),
				layout.Rigid(deleteInstrumentBtn.Layout),
			)
		}

		for ie.presetMenuBtn.Clicked(gtx) {
			ie.presetMenu.Visible = true
		}

		if t.CommentExpanded().Value() || gtx.Source.Focused(ie.commentEditor) { // we draw once the widget after it manages to lose focus
			ret := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(header),
				layout.Rigid(func(gtx C) D {
					defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
					for ie.commentEditor.Update(gtx, t.InstrumentComment()) != EditorEventNone {
						ie.instrumentDragList.Focus()
					}
					ret := layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
						return ie.commentEditor.Layout(gtx, t.InstrumentComment(), t.Theme, &t.Theme.InstrumentEditor.InstrumentComment, "Comment")
					})
					return ret
				}),
			)
			return ret
		}
		return header(gtx)
	}

	return Surface{Gray: 37, Focus: ie.wasFocused}.Layout(gtx, header)
}

func (ie *InstrumentEditor) layoutInstrumentList(gtx C, t *Tracker) D {
	gtx.Constraints.Max.Y = gtx.Dp(36)
	gtx.Constraints.Min.Y = gtx.Dp(36)
	element := func(gtx C, i int) D {
		grabhandle := Label(t.Theme, &t.Theme.InstrumentEditor.InstrumentList.Number, strconv.Itoa(i+1))
		label := func(gtx C) D {
			name, level, mute, ok := (*tracker.Instruments)(t.Model).Item(i)
			if !ok {
				labelStyle := Label(t.Theme, &t.Theme.InstrumentEditor.InstrumentList.Number, "")
				return layout.Center.Layout(gtx, labelStyle.Layout)
			}
			s := t.Theme.InstrumentEditor.InstrumentList.NameMuted
			if !mute {
				s = t.Theme.InstrumentEditor.InstrumentList.Name
				k := byte(255 - level*127)
				s.Color = color.NRGBA{R: 255, G: k, B: 255, A: 255}
			}
			if i == ie.instrumentDragList.TrackerList.Selected() {
				for ie.nameEditor.Update(gtx, t.InstrumentName()) != EditorEventNone {
					ie.instrumentDragList.Focus()
				}
				return layout.Center.Layout(gtx, func(gtx C) D {
					defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
					return ie.nameEditor.Layout(gtx, t.InstrumentName(), t.Theme, &s, "Instr")
				})
			}
			if name == "" {
				name = "Instr"
			}
			l := s.AsLabelStyle()
			return layout.Center.Layout(gtx, Label(t.Theme, &l, name).Layout)
		}
		return layout.Center.Layout(gtx, func(gtx C) D {
			return layout.Inset{Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(grabhandle.Layout),
					layout.Rigid(label),
				)
			})
		})
	}

	instrumentList := FilledDragList(t.Theme, ie.instrumentDragList)
	instrumentList.ScrollBar = t.Theme.InstrumentEditor.InstrumentList.ScrollBar

	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
	for {
		event, ok := gtx.Event(
			key.Filter{Focus: ie.instrumentDragList, Name: key.NameDownArrow},
			key.Filter{Focus: ie.instrumentDragList, Name: key.NameReturn},
			key.Filter{Focus: ie.instrumentDragList, Name: key.NameEnter},
		)
		if !ok {
			break
		}
		switch e := event.(type) {
		case key.Event:
			switch e.State {
			case key.Press:
				switch e.Name {
				case key.NameDownArrow:
					ie.unitDragList.Focus()
				case key.NameReturn, key.NameEnter:
					ie.nameEditor.Focus()
				}
			}
		}
	}

	dims := instrumentList.Layout(gtx, element, nil)
	gtx.Constraints = layout.Exact(dims.Size)
	instrumentList.LayoutScrollBar(gtx)
	return dims
}

func (ie *InstrumentEditor) layoutUnitList(gtx C, t *Tracker) D {
	var units [256]tracker.UnitListItem
	for i, item := range (*tracker.Units)(t.Model).Iterate {
		if i >= 256 {
			break
		}
		units[i] = item
	}
	count := min(ie.unitDragList.TrackerList.Count(), 256)

	element := func(gtx C, i int) D {
		gtx.Constraints.Max.Y = gtx.Dp(20)
		gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
		if i < 0 || i > 255 {
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}
		u := units[i]

		editorStyle := t.Theme.InstrumentEditor.UnitList.Name
		if u.Disabled {
			editorStyle = t.Theme.InstrumentEditor.UnitList.NameDisabled
		}

		stackText := strconv.FormatInt(int64(u.StackAfter), 10)
		if u.StackNeed > u.StackBefore {
			editorStyle.Color = t.Theme.InstrumentEditor.UnitList.Error
			(*tracker.Alerts)(t.Model).AddNamed("UnitNeedsInputs", fmt.Sprintf("%v needs at least %v input signals, got %v", u.Type, u.StackNeed, u.StackBefore), tracker.Error)
		} else if i == count-1 && u.StackAfter != 0 {
			editorStyle.Color = t.Theme.InstrumentEditor.UnitList.Warning
			(*tracker.Alerts)(t.Model).AddNamed("InstrumentLeavesSignals", fmt.Sprintf("Instrument leaves %v signal(s) on the stack", u.StackAfter), tracker.Warning)
		}

		stackLabel := Label(t.Theme, &t.Theme.InstrumentEditor.UnitList.Stack, stackText)

		rightMargin := layout.Inset{Right: unit.Dp(10)}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				if i == ie.unitDragList.TrackerList.Selected() {
					defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
					str := t.Model.UnitSearch()
					for ev := ie.searchEditor.Update(gtx, str); ev != EditorEventNone; ev = ie.searchEditor.Update(gtx, str) {
						if ev == EditorEventSubmit {
							if str.Value() != "" {
								for _, n := range sointu.UnitNames {
									if strings.HasPrefix(n, str.Value()) {
										t.Units().SetSelectedType(n)
										break
									}
								}
							} else {
								t.Units().SetSelectedType("")
							}
						}
						ie.unitDragList.Focus()
						t.UnitSearching().SetValue(false)
					}
					return ie.searchEditor.Layout(gtx, str, t.Theme, &editorStyle, "---")
				} else {
					text := u.Type
					if text == "" {
						text = "---"
					}
					l := editorStyle.AsLabelStyle()
					return Label(t.Theme, &l, text).Layout(gtx)
				}
			}),
			layout.Flexed(1, func(gtx C) D {
				unitNameLabel := Label(t.Theme, &t.Theme.InstrumentEditor.UnitList.Comment, u.Comment)
				inset := layout.Inset{Left: unit.Dp(5)}
				return inset.Layout(gtx, unitNameLabel.Layout)
			}),
			layout.Rigid(func(gtx C) D {
				return rightMargin.Layout(gtx, stackLabel.Layout)
			}),
		)
	}

	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	unitList := FilledDragList(t.Theme, ie.unitDragList)
	for {
		event, ok := gtx.Event(
			key.Filter{Focus: ie.unitDragList, Name: key.NameRightArrow},
			key.Filter{Focus: ie.unitDragList, Name: key.NameEnter, Optional: key.ModCtrl},
			key.Filter{Focus: ie.unitDragList, Name: key.NameReturn, Optional: key.ModCtrl},
			key.Filter{Focus: ie.unitDragList, Name: key.NameDeleteBackward},
			key.Filter{Focus: ie.unitDragList, Name: key.NameEscape},
		)
		if !ok {
			break
		}
		switch e := event.(type) {
		case key.Event:
			switch e.State {
			case key.Press:
				switch e.Name {
				case key.NameEscape:
					ie.instrumentDragList.Focus()
				case key.NameRightArrow:
					ie.unitEditor.sliderList.Focus()
				case key.NameDeleteBackward:
					t.Units().SetSelectedType("")
					t.UnitSearching().SetValue(true)
					ie.searchEditor.Focus()
				case key.NameEnter, key.NameReturn:
					t.Model.AddUnit(e.Modifiers.Contain(key.ModCtrl)).Do()
					t.UnitSearching().SetValue(true)
					ie.searchEditor.Focus()
				}
			}
		}
	}
	return Surface{Gray: 30, Focus: ie.wasFocused}.Layout(gtx, func(gtx C) D {
		return layout.Stack{Alignment: layout.SE}.Layout(gtx,
			layout.Expanded(func(gtx C) D {
				defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
				gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(140), gtx.Constraints.Max.Y))
				dims := unitList.Layout(gtx, element, nil)
				unitList.LayoutScrollBar(gtx)
				return dims
			}),
			layout.Stacked(func(gtx C) D {
				for ie.addUnitBtn.Clicked(gtx) {
					t.AddUnit(false).Do()
				}
				margin := layout.Inset{Right: unit.Dp(20), Bottom: unit.Dp(1)}
				addUnitBtn := IconBtn(t.Theme, &t.Theme.IconButton.Emphasis, ie.addUnitBtn, icons.ContentAdd, "Add unit (Enter)")
				return margin.Layout(gtx, addUnitBtn.Layout)
			}),
		)
	})
}
