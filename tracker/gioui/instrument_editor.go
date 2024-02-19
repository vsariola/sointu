package gioui

import (
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"

	"gioui.org/font"
	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/tracker"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type InstrumentEditor struct {
	newInstrumentBtn    *ActionClickable
	enlargeBtn          *BoolClickable
	deleteInstrumentBtn *ActionClickable
	copyInstrumentBtn   *TipClickable
	saveInstrumentBtn   *TipClickable
	loadInstrumentBtn   *TipClickable
	addUnitBtn          *ActionClickable
	presetMenuBtn       *TipClickable
	commentExpandBtn    *BoolClickable
	commentEditor       *widget.Editor
	commentString       tracker.String
	nameEditor          *widget.Editor
	nameString          tracker.String
	searchEditor        *widget.Editor
	instrumentDragList  *DragList
	unitDragList        *DragList
	presetDragList      *DragList
	unitEditor          *UnitEditor
	tag                 bool
	wasFocused          bool
	presetMenuItems     []MenuItem
	presetMenu          Menu
}

func NewInstrumentEditor(model *tracker.Model) *InstrumentEditor {
	ret := &InstrumentEditor{
		newInstrumentBtn:    NewActionClickable(model.AddInstrument()),
		enlargeBtn:          NewBoolClickable(model.InstrEnlarged().Bool()),
		deleteInstrumentBtn: NewActionClickable(model.DeleteInstrument()),
		copyInstrumentBtn:   new(TipClickable),
		saveInstrumentBtn:   new(TipClickable),
		loadInstrumentBtn:   new(TipClickable),
		addUnitBtn:          NewActionClickable(model.AddUnit(false)),
		commentExpandBtn:    NewBoolClickable(model.CommentExpanded().Bool()),
		presetMenuBtn:       new(TipClickable),
		commentEditor:       new(widget.Editor),
		nameEditor:          &widget.Editor{SingleLine: true, Submit: true, Alignment: text.Middle},
		searchEditor:        &widget.Editor{SingleLine: true, Submit: true, Alignment: text.Start},
		commentString:       model.InstrumentComment().String(),
		nameString:          model.InstrumentName().String(),
		instrumentDragList:  NewDragList(model.Instruments().List(), layout.Horizontal),
		unitDragList:        NewDragList(model.Units().List(), layout.Vertical),
		unitEditor:          NewUnitEditor(model),
		presetMenuItems:     []MenuItem{},
	}
	model.IterateInstrumentPresets(func(index int, name string) bool {
		ret.presetMenuItems = append(ret.presetMenuItems, MenuItem{Text: name, IconBytes: icons.ImageAudiotrack, Doer: model.LoadPreset(index)})
		return true
	})
	return ret
}

func (ie *InstrumentEditor) Focus() {
	ie.unitDragList.Focus()
}

func (ie *InstrumentEditor) Focused() bool {
	return ie.unitDragList.focused
}

func (ie *InstrumentEditor) ChildFocused() bool {
	return ie.unitEditor.sliderList.Focused() || ie.instrumentDragList.Focused() || ie.commentEditor.Focused() || ie.nameEditor.Focused() || ie.searchEditor.Focused() ||
		ie.addUnitBtn.Clickable.Focused() || ie.commentExpandBtn.Clickable.Focused() || ie.presetMenuBtn.Clickable.Focused() || ie.deleteInstrumentBtn.Clickable.Focused() || ie.copyInstrumentBtn.Clickable.Focused()
}

func (ie *InstrumentEditor) Layout(gtx C, t *Tracker) D {
	ie.wasFocused = ie.Focused() || ie.ChildFocused()
	fullscreenBtnStyle := ToggleIcon(t.Theme, ie.enlargeBtn, icons.NavigationFullscreen, icons.NavigationFullscreenExit, "Enlarge (Ctrl+E)", "Shrink (Ctrl+E)")

	octave := func(gtx C) D {
		in := layout.UniformInset(unit.Dp(1))
		numStyle := NumericUpDown(t.Theme, t.OctaveNumberInput, "Octave down (<) or up (>)")
		dims := in.Layout(gtx, numStyle.Layout)
		return dims
	}

	newBtnStyle := ActionIcon(t.Theme, ie.newInstrumentBtn, icons.ContentAdd, "Add\ninstrument\n(Ctrl+I)")
	ret := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{}.Layout(
				gtx,
				layout.Flexed(1, func(gtx C) D {
					return ie.layoutInstrumentList(gtx, t)
				}),
				layout.Rigid(func(gtx C) D {
					inset := layout.UniformInset(unit.Dp(6))
					return inset.Layout(gtx, func(gtx C) D {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(Label("OCT:", white, t.Theme.Shaper)),
							layout.Rigid(octave),
						)
					})
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
		commentExpandBtnStyle := ToggleIcon(t.Theme, ie.commentExpandBtn, icons.NavigationExpandMore, icons.NavigationExpandLess, "Expand comment", "Collapse comment")
		presetMenuBtnStyle := TipIcon(t.Theme, ie.presetMenuBtn, icons.NavigationMenu, "Load preset")
		copyInstrumentBtnStyle := TipIcon(t.Theme, ie.copyInstrumentBtn, icons.ContentContentCopy, "Copy instrument")
		saveInstrumentBtnStyle := TipIcon(t.Theme, ie.saveInstrumentBtn, icons.ContentSave, "Save instrument")
		loadInstrumentBtnStyle := TipIcon(t.Theme, ie.loadInstrumentBtn, icons.FileFolderOpen, "Load instrument")
		deleteInstrumentBtnStyle := ActionIcon(t.Theme, ie.deleteInstrumentBtn, icons.ActionDelete, "Delete\ninstrument")

		m := PopupMenu(&ie.presetMenu, t.Theme.Shaper)

		for ie.copyInstrumentBtn.Clickable.Clicked() {
			if contents, ok := t.Instruments().List().CopyElements(); ok {
				clipboard.WriteOp{Text: string(contents)}.Add(gtx.Ops)
				t.Alerts().Add("Instrument copied to clipboard", tracker.Info)
			}
		}

		for ie.saveInstrumentBtn.Clickable.Clicked() {
			writer, err := t.Explorer.CreateFile(t.InstrumentName().Value() + ".yml")
			if err != nil {
				continue
			}
			t.SaveInstrument(writer)
		}

		for ie.loadInstrumentBtn.Clickable.Clicked() {
			reader, err := t.Explorer.ChooseFile(".yml", ".json", ".4ki", ".4kp")
			if err != nil {
				continue
			}
			t.LoadInstrument(reader)
		}

		header := func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(Label("Voices: ", white, t.Theme.Shaper)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					numStyle := NumericUpDown(t.Theme, t.InstrumentVoices, "Number of voices for this instrument")
					dims := numStyle.Layout(gtx)
					return dims
				}),
				layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
				layout.Rigid(commentExpandBtnStyle.Layout),
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

		for ie.presetMenuBtn.Clickable.Clicked() {
			ie.presetMenu.Visible = true
		}

		if ie.commentExpandBtn.Bool.Value() || ie.commentEditor.Focused() { // we draw once the widget after it manages to lose focus
			if ie.commentEditor.Text() != ie.commentString.Value() {
				ie.commentEditor.SetText(ie.commentString.Value())
			}
			ret := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(header),
				layout.Rigid(func(gtx C) D {
					defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
					key.InputOp{Tag: &ie.unitDragList, Keys: globalKeys + "|⎋"}.Add(gtx.Ops)
					for _, event := range gtx.Events(&ie.unitDragList) {
						if e, ok := event.(key.Event); ok && e.State == key.Press && e.Name == key.NameEscape {
							ie.instrumentDragList.Focus()
						}
					}
					editorStyle := material.Editor(t.Theme, ie.commentEditor, "Comment")
					editorStyle.Color = highEmphasisTextColor
					return layout.UniformInset(unit.Dp(6)).Layout(gtx, editorStyle.Layout)
				}),
			)
			ie.commentString.Set(ie.commentEditor.Text())
			return ret
		}
		return header(gtx)
	}

	return Surface{Gray: 37, Focus: ie.wasFocused}.Layout(gtx, header)
}

func (ie *InstrumentEditor) layoutInstrumentList(gtx C, t *Tracker) D {
	gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(36))
	element := func(gtx C, i int) D {
		gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(36))
		gtx.Constraints.Min.X = gtx.Dp(unit.Dp(30))
		grabhandle := LabelStyle{Text: "", ShadeColor: black, Color: white, FontSize: unit.Sp(10), Alignment: layout.Center, Shaper: t.Theme.Shaper}
		if i == ie.instrumentDragList.TrackerList.Selected() {
			grabhandle.Text = ":::"
		}
		label := func(gtx C) D {
			name, level, ok := (*tracker.Instruments)(t.Model).Item(i)
			if !ok {
				labelStyle := LabelStyle{Text: "", ShadeColor: black, Color: white, FontSize: unit.Sp(12), Shaper: t.Theme.Shaper}
				return layout.Center.Layout(gtx, labelStyle.Layout)
			}
			k := byte(255 - level*127)
			color := color.NRGBA{R: 255, G: k, B: 255, A: 255}
			if i == ie.instrumentDragList.TrackerList.Selected() {
				for _, ev := range ie.nameEditor.Events() {
					_, ok := ev.(widget.SubmitEvent)
					if ok {
						ie.instrumentDragList.Focus()
						continue
					}
				}
				if n := name; n != ie.nameEditor.Text() {
					ie.nameEditor.SetText(n)
				}
				editor := material.Editor(t.Theme, ie.nameEditor, "Instr")
				editor.Color = color
				editor.HintColor = instrumentNameHintColor
				editor.TextSize = unit.Sp(12)
				editor.Font = labelDefaultFont
				dims := layout.Center.Layout(gtx, func(gtx C) D {
					defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
					key.InputOp{Tag: &ie.nameEditor, Keys: globalKeys}.Add(gtx.Ops)
					return editor.Layout(gtx)
				})
				ie.nameString.Set(ie.nameEditor.Text())
				return dims
			}
			if name == "" {
				name = "Instr"
			}
			labelStyle := LabelStyle{Text: name, ShadeColor: black, Color: color, Font: labelDefaultFont, FontSize: unit.Sp(12), Shaper: t.Theme.Shaper}
			return layout.Center.Layout(gtx, labelStyle.Layout)
		}
		return layout.Inset{Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(grabhandle.Layout),
				layout.Rigid(label),
			)
		})
	}

	color := inactiveLightSurfaceColor
	if ie.wasFocused {
		color = activeLightSurfaceColor
	}
	instrumentList := FilledDragList(t.Theme, ie.instrumentDragList, element, nil)
	instrumentList.SelectedColor = color
	instrumentList.HoverColor = instrumentHoverColor
	instrumentList.ScrollBarWidth = unit.Dp(6)

	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
	key.InputOp{Tag: ie.instrumentDragList, Keys: "↓|⏎|⌤"}.Add(gtx.Ops)

	for _, event := range gtx.Events(ie.instrumentDragList) {
		switch e := event.(type) {
		case key.Event:
			switch e.State {
			case key.Press:
				switch e.Name {
				case key.NameDownArrow:
					ie.unitDragList.Focus()
				case key.NameReturn, key.NameEnter:
					ie.nameEditor.Focus()
					l := len(ie.nameEditor.Text())
					ie.nameEditor.SetCaret(l, l)
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
	addUnitBtnStyle := ActionIcon(t.Theme, ie.addUnitBtn, icons.ContentAdd, "Add unit (Enter)")
	addUnitBtnStyle.IconButtonStyle.Color = t.Theme.ContrastFg
	addUnitBtnStyle.IconButtonStyle.Background = t.Theme.Fg
	addUnitBtnStyle.IconButtonStyle.Inset = layout.UniformInset(unit.Dp(4))

	index := 0
	var units [256]tracker.UnitListItem
	(*tracker.Units)(t.Model).Iterate(func(item tracker.UnitListItem) (ok bool) {
		units[index] = item
		index++
		return index <= 256
	})
	count := intMin(ie.unitDragList.TrackerList.Count(), 256)

	element := func(gtx C, i int) D {
		gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(unit.Dp(120)), gtx.Dp(unit.Dp(20))))
		if i < 0 || i >= count {
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}
		u := units[i]
		var color color.NRGBA = white
		f := labelDefaultFont

		var stackText string
		stackText = strconv.FormatInt(int64(u.StackAfter), 10)
		if u.StackNeed > u.StackBefore {
			color = errorColor
			(*tracker.Alerts)(t.Model).AddNamed("UnitNeedsInputs", fmt.Sprintf("%v needs at least %v input signals, got %v", u.Type, u.StackNeed, u.StackBefore), tracker.Error)
		} else if i == count-1 && u.StackAfter != 0 {
			color = warningColor
			(*tracker.Alerts)(t.Model).AddNamed("InstrumentLeavesSignals", fmt.Sprintf("Instrument leaves %v signal(s) on the stack", u.StackAfter), tracker.Warning)
		}
		if u.Disabled {
			color = disabledTextColor
			f.Style = font.Italic
		}

		stackLabel := LabelStyle{Text: stackText, ShadeColor: black, Color: mediumEmphasisTextColor, Font: labelDefaultFont, FontSize: unit.Sp(12), Shaper: t.Theme.Shaper}
		rightMargin := layout.Inset{Right: unit.Dp(10)}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Flexed(1, func(gtx C) D {
				if i == ie.unitDragList.TrackerList.Selected() {
					for _, ev := range ie.searchEditor.Events() {
						_, ok := ev.(widget.SubmitEvent)
						if ok {
							txt := ""
							ie.unitDragList.Focus()
							if text := ie.searchEditor.Text(); text != "" {
								for _, n := range sointu.UnitNames {
									if strings.HasPrefix(n, ie.searchEditor.Text()) {
										txt = n
										break
									}
								}
							}
							t.Units().SetSelectedType(txt)
							t.UnitSearching().Bool().Set(false)
							continue
						}
					}
					editor := material.Editor(t.Theme, ie.searchEditor, "---")
					editor.Color = color
					editor.HintColor = instrumentNameHintColor
					editor.TextSize = unit.Sp(12)
					editor.Font = f

					defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
					key.InputOp{Tag: &ie.searchEditor, Keys: globalKeys}.Add(gtx.Ops)
					txt := u.Type
					str := tracker.String{StringData: (*tracker.UnitSearch)(t.Model)}
					if t.UnitSearching().Value() {
						txt = str.Value()
					}
					if ie.searchEditor.Text() != txt {
						ie.searchEditor.SetText(txt)
					}
					ret := editor.Layout(gtx)
					if ie.searchEditor.Text() != txt {
						str.Set(ie.searchEditor.Text())
					}
					return ret
				} else {
					unitNameLabel := LabelStyle{Text: u.Type, ShadeColor: black, Color: color, Font: f, FontSize: unit.Sp(12), Shaper: t.Theme.Shaper}
					if unitNameLabel.Text == "" {
						unitNameLabel.Text = "---"
					}
					return unitNameLabel.Layout(gtx)
				}
			}),
			layout.Rigid(func(gtx C) D {
				return rightMargin.Layout(gtx, stackLabel.Layout)
			}),
		)
	}

	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	unitList := FilledDragList(t.Theme, ie.unitDragList, element, nil)
	return Surface{Gray: 30, Focus: ie.wasFocused}.Layout(gtx, func(gtx C) D {
		return layout.Stack{Alignment: layout.SE}.Layout(gtx,
			layout.Expanded(func(gtx C) D {
				defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
				key.InputOp{Tag: ie.unitDragList, Keys: "→|⏎|Ctrl-⏎|⌫|⎋"}.Add(gtx.Ops)
				for _, event := range gtx.Events(ie.unitDragList) {
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
								ie.searchEditor.Focus()
								l := len(ie.searchEditor.Text())
								ie.searchEditor.SetCaret(l, l)
							case key.NameReturn:
								t.Model.AddUnit(e.Modifiers.Contain(key.ModCtrl)).Do()
								ie.searchEditor.SetText("")
								ie.searchEditor.Focus()
								l := len(ie.searchEditor.Text())
								ie.searchEditor.SetCaret(l, l)
							}
						}
					}
				}
				gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(unit.Dp(120)), gtx.Constraints.Max.Y))
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

func clamp(i, min, max int) int {
	if i < min {
		return min
	}
	if i > max {
		return max
	}
	return i
}
