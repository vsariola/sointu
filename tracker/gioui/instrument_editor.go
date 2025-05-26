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
	"gioui.org/widget"
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/tracker"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type (
	InstrumentEditor struct {
		newInstrumentBtn    *ActionClickable
		enlargeBtn          *BoolClickable
		deleteInstrumentBtn *ActionClickable
		linkInstrTrackBtn   *BoolClickable
		splitInstrumentBtn  *ActionClickable
		copyInstrumentBtn   *TipClickable
		saveInstrumentBtn   *TipClickable
		loadInstrumentBtn   *TipClickable
		addUnitBtn          *ActionClickable
		presetMenuBtn       *TipClickable
		commentExpandBtn    *BoolClickable
		soloBtn             *BoolClickable
		muteBtn             *BoolClickable
		commentEditor       *Editor
		commentString       tracker.String
		nameEditor          *Editor
		nameString          tracker.String
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
		newInstrumentBtn:    NewActionClickable(model.AddInstrument()),
		enlargeBtn:          NewBoolClickable(model.InstrEnlarged()),
		deleteInstrumentBtn: NewActionClickable(model.DeleteInstrument()),
		linkInstrTrackBtn:   NewBoolClickable(model.LinkInstrTrack()),
		splitInstrumentBtn:  NewActionClickable(model.SplitInstrument()),
		copyInstrumentBtn:   new(TipClickable),
		saveInstrumentBtn:   new(TipClickable),
		loadInstrumentBtn:   new(TipClickable),
		commentExpandBtn:    NewBoolClickable(model.CommentExpanded()),
		presetMenuBtn:       new(TipClickable),
		soloBtn:             NewBoolClickable(model.Solo()),
		muteBtn:             NewBoolClickable(model.Mute()),
		commentEditor:       NewEditor(widget.Editor{}),
		nameEditor:          NewEditor(widget.Editor{SingleLine: true, Submit: true, Alignment: text.Middle}),
		searchEditor:        NewEditor(widget.Editor{SingleLine: true, Submit: true, Alignment: text.Start}),
		commentString:       model.InstrumentComment(),
		nameString:          model.InstrumentName(),
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
	ret.addUnitBtn = NewActionClickable(tracker.MakeEnabledAction(ret.AddUnitThenFocus()))
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

func (ie *InstrumentEditor) Focused() bool {
	return ie.unitDragList.focused
}

func (ie *InstrumentEditor) childFocused(gtx C) bool {
	return ie.unitEditor.sliderList.Focused() ||
		ie.instrumentDragList.Focused() || gtx.Source.Focused(ie.commentEditor) || gtx.Source.Focused(ie.nameEditor) || gtx.Source.Focused(ie.searchEditor) ||
		gtx.Source.Focused(ie.addUnitBtn.Clickable) || gtx.Source.Focused(ie.commentExpandBtn.Clickable) || gtx.Source.Focused(ie.presetMenuBtn.Clickable) ||
		gtx.Source.Focused(ie.deleteInstrumentBtn.Clickable) || gtx.Source.Focused(ie.copyInstrumentBtn.Clickable)
}

func (ie *InstrumentEditor) Layout(gtx C, t *Tracker) D {
	ie.wasFocused = ie.Focused() || ie.childFocused(gtx)
	fullscreenBtnStyle := ToggleIcon(gtx, t.Theme, ie.enlargeBtn, icons.NavigationFullscreen, icons.NavigationFullscreenExit, ie.enlargeHint, ie.shrinkHint)
	linkBtnStyle := ToggleIcon(gtx, t.Theme, ie.linkInstrTrackBtn, icons.NotificationSyncDisabled, icons.NotificationSync, ie.linkDisabledHint, ie.linkEnabledHint)

	octave := func(gtx C) D {
		in := layout.UniformInset(unit.Dp(1))
		numStyle := NumUpDown(t.Theme, t.OctaveNumberInput, ie.octaveHint)
		dims := in.Layout(gtx, numStyle.Layout)
		return dims
	}

	newBtnStyle := ActionIcon(gtx, t.Theme, ie.newInstrumentBtn, icons.ContentAdd, ie.addInstrumentHint)
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
					return layout.E.Layout(gtx, linkBtnStyle.Layout)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.E.Layout(gtx, fullscreenBtnStyle.Layout)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.E.Layout(gtx, newBtnStyle.Layout)
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
		commentExpandBtnStyle := ToggleIcon(gtx, t.Theme, ie.commentExpandBtn, icons.NavigationExpandMore, icons.NavigationExpandLess, ie.expandCommentHint, ie.collapseCommentHint)
		presetMenuBtnStyle := TipIcon(t.Theme, ie.presetMenuBtn, icons.NavigationMenu, "Load preset")
		copyInstrumentBtnStyle := TipIcon(t.Theme, ie.copyInstrumentBtn, icons.ContentContentCopy, "Copy instrument")
		saveInstrumentBtnStyle := TipIcon(t.Theme, ie.saveInstrumentBtn, icons.ContentSave, "Save instrument")
		loadInstrumentBtnStyle := TipIcon(t.Theme, ie.loadInstrumentBtn, icons.FileFolderOpen, "Load instrument")
		deleteInstrumentBtnStyle := ActionIcon(gtx, t.Theme, ie.deleteInstrumentBtn, icons.ActionDelete, ie.deleteInstrumentHint)
		splitInstrumentBtnStyle := ActionIcon(gtx, t.Theme, ie.splitInstrumentBtn, icons.CommunicationCallSplit, ie.splitInstrumentHint)
		soloBtnStyle := ToggleIcon(gtx, t.Theme, ie.soloBtn, icons.SocialGroup, icons.SocialPerson, ie.soloHint, ie.unsoloHint)
		muteBtnStyle := ToggleIcon(gtx, t.Theme, ie.muteBtn, icons.AVVolumeUp, icons.AVVolumeOff, ie.muteHint, ie.unmuteHint)

		m := PopupMenu(t.Theme, &t.Theme.Menu.Text, &ie.presetMenu)

		for ie.copyInstrumentBtn.Clickable.Clicked(gtx) {
			if contents, ok := t.Instruments().List().CopyElements(); ok {
				gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(bytes.NewReader(contents))})
				t.Alerts().Add("Instrument copied to clipboard", tracker.Info)
			}
		}

		for ie.saveInstrumentBtn.Clickable.Clicked(gtx) {
			writer, err := t.Explorer.CreateFile(t.InstrumentName().Value() + ".yml")
			if err != nil {
				continue
			}
			t.SaveInstrument(writer)
		}

		for ie.loadInstrumentBtn.Clickable.Clicked(gtx) {
			reader, err := t.Explorer.ChooseFile(".yml", ".json", ".4ki", ".4kp")
			if err != nil {
				continue
			}
			t.LoadInstrument(reader)
		}

		header := func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(layout.Spacer{Width: 6}.Layout),
				layout.Rigid(Label(t.Theme, &t.Theme.InstrumentEditor.Voices, "Voices").Layout),
				layout.Rigid(layout.Spacer{Width: 4}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					numStyle := NumUpDown(t.Theme, t.InstrumentVoices, "Number of voices for this instrument")
					dims := numStyle.Layout(gtx)
					return dims
				}),
				layout.Rigid(splitInstrumentBtnStyle.Layout),
				layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
				layout.Rigid(commentExpandBtnStyle.Layout),
				layout.Rigid(soloBtnStyle.Layout),
				layout.Rigid(muteBtnStyle.Layout),
				layout.Rigid(func(gtx C) D {
					//defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
					dims := presetMenuBtnStyle.Layout(gtx)
					op.Offset(image.Pt(0, dims.Size.Y)).Add(gtx.Ops)
					gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(500))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(180))
					m.Layout(gtx, ie.presetMenuItems...)
					return dims
				}),
				layout.Rigid(saveInstrumentBtnStyle.Layout),
				layout.Rigid(loadInstrumentBtnStyle.Layout),
				layout.Rigid(copyInstrumentBtnStyle.Layout),
				layout.Rigid(deleteInstrumentBtnStyle.Layout))
		}

		for ie.presetMenuBtn.Clickable.Clicked(gtx) {
			ie.presetMenu.Visible = true
		}

		if ie.commentExpandBtn.Bool.Value() || gtx.Source.Focused(ie.commentEditor) { // we draw once the widget after it manages to lose focus
			ret := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(header),
				layout.Rigid(func(gtx C) D {
					defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
					ie.commentEditor.SetText(ie.commentString.Value())
					for ie.commentEditor.Submitted(gtx) || ie.commentEditor.Cancelled(gtx) {
						ie.instrumentDragList.Focus()
					}
					style := MaterialEditor(t.Theme, &t.Theme.InstrumentEditor.InstrumentComment, ie.commentEditor, "Comment")
					ret := layout.UniformInset(unit.Dp(6)).Layout(gtx, style.Layout)
					ie.commentString.SetValue(ie.commentEditor.Text())
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
			k := byte(255 - level*127)
			color := color.NRGBA{R: 255, G: k, B: 255, A: 255}
			s := t.Theme.InstrumentEditor.InstrumentList.Name
			if mute {
				s = t.Theme.InstrumentEditor.InstrumentList.NameMuted
			}
			s.Color = color
			if i == ie.instrumentDragList.TrackerList.Selected() {
				ie.nameEditor.SetText(name)
				for ie.nameEditor.Submitted(gtx) || ie.nameEditor.Cancelled(gtx) {
					ie.instrumentDragList.Focus()
				}
				style := MaterialEditor(t.Theme, &s, ie.nameEditor, "Instr")
				dims := layout.Center.Layout(gtx, func(gtx C) D {
					defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
					return style.Layout(gtx)
				})
				ie.nameString.SetValue(ie.nameEditor.Text())
				return dims
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

	instrumentList := FilledDragList(t.Theme, ie.instrumentDragList, element, nil)
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
					gtx.Execute(key.FocusCmd{Tag: &ie.nameEditor.Editor})
					l := len(ie.nameEditor.Editor.Text())
					ie.nameEditor.Editor.SetCaret(l, l)
				}
			}
		}
	}

	dims := instrumentList.Layout(gtx)
	gtx.Constraints = layout.Exact(dims.Size)
	instrumentList.LayoutScrollBar(gtx)
	return dims
}

func (ie *InstrumentEditor) layoutUnitList(gtx C, t *Tracker) D {
	// TODO: how to ie.unitDragList.Focus()
	addUnitBtnStyle := ActionIcon(gtx, t.Theme, ie.addUnitBtn, icons.ContentAdd, "Add unit (Enter)")
	addUnitBtnStyle.IconButtonStyle.Color = t.Theme.Material.ContrastFg
	addUnitBtnStyle.IconButtonStyle.Background = t.Theme.Material.ContrastBg
	addUnitBtnStyle.IconButtonStyle.Inset = layout.UniformInset(unit.Dp(4))

	var units [256]tracker.UnitListItem
	for i, item := range (*tracker.Units)(t.Model).Iterate {
		if i >= 256 {
			break
		}
		units[i] = item
	}
	count := min(ie.unitDragList.TrackerList.Count(), 256)

	if ie.searchEditor.requestFocus {
		// for now, only the searchEditor has its requestFocus flag
		ie.searchEditor.requestFocus = false
		gtx.Execute(key.FocusCmd{Tag: &ie.searchEditor.Editor})
	}

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
					ie.searchEditor.SetText(str.Value())
					for ie.searchEditor.Submitted(gtx) {
						ie.unitDragList.Focus()
						if text := ie.searchEditor.Text(); text != "" {
							for _, n := range sointu.UnitNames {
								if strings.HasPrefix(n, ie.searchEditor.Text()) {
									t.Units().SetSelectedType(n)
									break
								}
							}
						}
						t.UnitSearching().SetValue(false)
						ie.searchEditor.SetText(str.Value())
					}
					for ie.searchEditor.Cancelled(gtx) {
						t.UnitSearching().SetValue(false)
						ie.searchEditor.SetText(str.Value())
						ie.unitDragList.Focus()
					}
					style := MaterialEditor(t.Theme, &editorStyle, ie.searchEditor, "---")
					ret := style.Layout(gtx)
					str.SetValue(ie.searchEditor.Text())
					return ret
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
	unitList := FilledDragList(t.Theme, ie.unitDragList, element, nil)
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
					gtx.Execute(key.FocusCmd{Tag: &ie.searchEditor.Editor})
				case key.NameEnter, key.NameReturn:
					t.Model.AddUnit(e.Modifiers.Contain(key.ModCtrl)).Do()
					t.UnitSearching().SetValue(true)
					gtx.Execute(key.FocusCmd{Tag: &ie.searchEditor.Editor})
				}
			}
		}
	}
	return Surface{Gray: 30, Focus: ie.wasFocused}.Layout(gtx, func(gtx C) D {
		return layout.Stack{Alignment: layout.SE}.Layout(gtx,
			layout.Expanded(func(gtx C) D {
				defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
				gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(140), gtx.Constraints.Max.Y))
				dims := unitList.Layout(gtx)
				unitList.LayoutScrollBar(gtx)
				return dims
			}),
			layout.Stacked(func(gtx C) D {
				margin := layout.Inset{Right: unit.Dp(20), Bottom: unit.Dp(1)}
				return margin.Layout(gtx, addUnitBtnStyle.Layout)
			}),
		)
	})
}
