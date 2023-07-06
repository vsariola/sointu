package gioui

import (
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"
	"time"

	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/eventx"
	"github.com/vsariola/sointu/tracker"
	"github.com/vsariola/sointu/vm"
	"golang.org/x/exp/shiny/materialdesign/icons"
	"gopkg.in/yaml.v3"
)

type InstrumentEditor struct {
	newInstrumentBtn    *widget.Clickable
	enlargeBtn          *widget.Clickable
	deleteInstrumentBtn *widget.Clickable
	copyInstrumentBtn   *widget.Clickable
	saveInstrumentBtn   *widget.Clickable
	loadInstrumentBtn   *widget.Clickable
	addUnitBtn          *widget.Clickable
	commentExpandBtn    *widget.Clickable
	commentEditor       *widget.Editor
	nameEditor          *widget.Editor
	unitTypeEditor      *widget.Editor
	instrumentDragList  *DragList
	instrumentScrollBar *ScrollBar
	unitDragList        *DragList
	unitScrollBar       *ScrollBar
	confirmInstrDelete  *Dialog
	paramEditor         *ParamEditor
	stackUse            []int
	tag                 bool
	wasFocused          bool
	commentExpanded     bool
	voiceStates         [vm.MAX_VOICES]float32
}

func NewInstrumentEditor() *InstrumentEditor {
	return &InstrumentEditor{
		newInstrumentBtn:    new(widget.Clickable),
		enlargeBtn:          new(widget.Clickable),
		deleteInstrumentBtn: new(widget.Clickable),
		copyInstrumentBtn:   new(widget.Clickable),
		saveInstrumentBtn:   new(widget.Clickable),
		loadInstrumentBtn:   new(widget.Clickable),
		addUnitBtn:          new(widget.Clickable),
		commentExpandBtn:    new(widget.Clickable),
		commentEditor:       new(widget.Editor),
		nameEditor:          &widget.Editor{SingleLine: true, Submit: true, Alignment: text.Middle},
		unitTypeEditor:      &widget.Editor{SingleLine: true, Submit: true, Alignment: text.Start},
		instrumentDragList:  &DragList{List: &layout.List{Axis: layout.Horizontal}, HoverItem: -1},
		instrumentScrollBar: &ScrollBar{Axis: layout.Horizontal},
		unitDragList:        &DragList{List: &layout.List{Axis: layout.Vertical}, HoverItem: -1},
		unitScrollBar:       &ScrollBar{Axis: layout.Vertical},
		confirmInstrDelete:  new(Dialog),
		paramEditor:         NewParamEditor(),
	}
}

func (t *InstrumentEditor) ExpandComment() {
	t.commentExpanded = true
}

func (ie *InstrumentEditor) Focus() {
	ie.unitDragList.Focus()
}

func (ie *InstrumentEditor) Focused() bool {
	return ie.unitDragList.focused
}

func (ie *InstrumentEditor) ChildFocused() bool {
	return ie.paramEditor.Focused() || ie.instrumentDragList.Focused() || ie.commentEditor.Focused() || ie.nameEditor.Focused() || ie.unitTypeEditor.Focused()
}

func (ie *InstrumentEditor) Layout(gtx C, t *Tracker) D {
	ie.wasFocused = ie.Focused() || ie.ChildFocused()
	for _, e := range gtx.Events(&ie.tag) {
		switch e.(type) {
		case pointer.Event:
			ie.unitDragList.Focus()
		}
	}
	rect := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
	pointer.Rect(rect).Add(gtx.Ops)
	pointer.InputOp{Tag: &ie.tag,
		Types: pointer.Press,
	}.Add(gtx.Ops)

	var icon []byte
	if t.InstrEnlarged() {
		icon = icons.NavigationFullscreenExit
	} else {
		icon = icons.NavigationFullscreen
	}
	fullscreenBtnStyle := IconButton(t.Theme, ie.enlargeBtn, icon, true)
	for ie.enlargeBtn.Clicked() {
		t.SetInstrEnlarged(!t.InstrEnlarged())
	}
	for ie.newInstrumentBtn.Clicked() {
		t.AddInstrument(true)
	}
	octave := func(gtx C) D {
		in := layout.UniformInset(unit.Dp(1))
		t.OctaveNumberInput.Value = t.Octave()
		numStyle := NumericUpDown(t.Theme, t.OctaveNumberInput, 0, 9)
		gtx.Constraints.Min.Y = gtx.Px(unit.Dp(20))
		gtx.Constraints.Min.X = gtx.Px(unit.Dp(70))
		dims := in.Layout(gtx, numStyle.Layout)
		t.SetOctave(t.OctaveNumberInput.Value)
		return dims
	}
	newBtnStyle := IconButton(t.Theme, ie.newInstrumentBtn, icons.ContentAdd, t.CanAddInstrument())
	ret := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{}.Layout(
				gtx,
				layout.Flexed(1, func(gtx C) D {
					return layout.Stack{}.Layout(gtx,
						layout.Stacked(func(gtx C) D {
							return ie.layoutInstrumentNames(gtx, t)
						}),
						layout.Expanded(func(gtx C) D {
							return ie.instrumentScrollBar.Layout(gtx, unit.Dp(6), len(t.Song().Patch), &ie.instrumentDragList.List.Position)
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					inset := layout.UniformInset(unit.Dp(6))
					return inset.Layout(gtx, func(gtx C) D {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(Label("OCT:", white)),
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
			return ie.layoutInstrumentEditor(gtx, t)
		}))
	return ret
}

func (ie *InstrumentEditor) layoutInstrumentHeader(gtx C, t *Tracker) D {
	header := func(gtx C) D {
		collapseIcon := icons.NavigationExpandLess
		if !ie.commentExpanded {
			collapseIcon = icons.NavigationExpandMore
		}

		commentExpandBtnStyle := IconButton(t.Theme, ie.commentExpandBtn, collapseIcon, true)
		copyInstrumentBtnStyle := IconButton(t.Theme, ie.copyInstrumentBtn, icons.ContentContentCopy, true)
		saveInstrumentBtnStyle := IconButton(t.Theme, ie.saveInstrumentBtn, icons.ContentSave, true)
		loadInstrumentBtnStyle := IconButton(t.Theme, ie.loadInstrumentBtn, icons.FileFolderOpen, true)
		deleteInstrumentBtnStyle := IconButton(t.Theme, ie.deleteInstrumentBtn, icons.ActionDelete, t.CanDeleteInstrument())

		header := func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(Label("Voices: ", white)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					maxRemain := t.MaxInstrumentVoices()
					t.InstrumentVoices.Value = t.Instrument().NumVoices
					numStyle := NumericUpDown(t.Theme, t.InstrumentVoices, 0, maxRemain)
					gtx.Constraints.Min.Y = gtx.Px(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Px(unit.Dp(70))
					dims := numStyle.Layout(gtx)
					t.SetInstrumentVoices(t.InstrumentVoices.Value)
					return dims
				}),
				layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
				layout.Rigid(commentExpandBtnStyle.Layout),
				layout.Rigid(saveInstrumentBtnStyle.Layout),
				layout.Rigid(loadInstrumentBtnStyle.Layout),
				layout.Rigid(copyInstrumentBtnStyle.Layout),
				layout.Rigid(deleteInstrumentBtnStyle.Layout))
		}
		for ie.commentExpandBtn.Clicked() {
			ie.commentExpanded = !ie.commentExpanded
			if !ie.commentExpanded {
				key.FocusOp{Tag: &ie.tag}.Add(gtx.Ops) // clear focus
			}
		}
		if ie.commentExpanded || ie.commentEditor.Focused() { // we draw once the widget after it manages to lose focus
			if ie.commentEditor.Text() != t.Instrument().Comment {
				ie.commentEditor.SetText(t.Instrument().Comment)
			}
			editorStyle := material.Editor(t.Theme, ie.commentEditor, "Comment")
			editorStyle.Color = highEmphasisTextColor

			ret := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(header),
				layout.Rigid(func(gtx C) D {
					spy, spiedGtx := eventx.Enspy(gtx)
					ret := layout.UniformInset(unit.Dp(6)).Layout(spiedGtx, editorStyle.Layout)
					for _, group := range spy.AllEvents() {
						for _, event := range group.Items {
							switch e := event.(type) {
							case key.Event:
								if e.Name == key.NameEscape {
									ie.instrumentDragList.Focus()
								}
							}
						}
					}
					return ret
				}),
			)
			t.SetInstrumentComment(ie.commentEditor.Text())
			return ret
		}
		return header(gtx)
	}
	for ie.copyInstrumentBtn.Clicked() {
		contents, err := yaml.Marshal(t.Instrument())
		if err == nil {
			clipboard.WriteOp{Text: string(contents)}.Add(gtx.Ops)
			t.Alert.Update("Instrument copied to clipboard", Notify, time.Second*3)
		}
	}
	for ie.deleteInstrumentBtn.Clicked() {
		if t.CanDeleteInstrument() {
			dialogStyle := ConfirmDialog(t.Theme, ie.confirmInstrDelete, "Are you sure you want to delete this instrument?")
			ie.confirmInstrDelete.Visible = true
			t.ModalDialog = dialogStyle.Layout
		}
	}
	for ie.confirmInstrDelete.BtnOk.Clicked() {
		t.DeleteInstrument(false)
		t.ModalDialog = nil
	}
	for ie.confirmInstrDelete.BtnCancel.Clicked() {
		t.ModalDialog = nil
	}
	for ie.saveInstrumentBtn.Clicked() {
		t.SaveInstrument()
	}

	for ie.loadInstrumentBtn.Clicked() {
		t.LoadInstrument()
	}
	return Surface{Gray: 37, Focus: ie.wasFocused}.Layout(gtx, header)
}

func (ie *InstrumentEditor) layoutInstrumentNames(gtx C, t *Tracker) D {
	element := func(gtx C, i int) D {
		gtx.Constraints.Min.Y = gtx.Px(unit.Dp(36))
		gtx.Constraints.Min.X = gtx.Px(unit.Dp(30))
		grabhandle := LabelStyle{Text: "", ShadeColor: black, Color: white, FontSize: unit.Sp(10), Alignment: layout.Center}
		if i == t.InstrIndex() {
			grabhandle.Text = ":::"
		}
		label := func(gtx C) D {
			c := float32(0.0)
			voice := t.Song().Patch.FirstVoiceForInstrument(i)
			loopMax := t.Song().Patch[i].NumVoices
			if loopMax > vm.MAX_VOICES {
				loopMax = vm.MAX_VOICES
			}
			for j := 0; j < loopMax; j++ {
				vc := ie.voiceStates[voice]
				if c < vc {
					c = vc
				}
				voice++
			}
			k := byte(255 - c*127)
			color := color.NRGBA{R: 255, G: k, B: 255, A: 255}
			if i == t.InstrIndex() {
				for _, ev := range ie.nameEditor.Events() {
					_, ok := ev.(widget.SubmitEvent)
					if ok {
						ie.instrumentDragList.Focus()
						continue
					}
				}
				if n := t.Instrument().Name; n != ie.nameEditor.Text() {
					ie.nameEditor.SetText(n)
				}
				editor := material.Editor(t.Theme, ie.nameEditor, "Instr")
				editor.Color = color
				editor.HintColor = instrumentNameHintColor
				editor.TextSize = unit.Dp(12)
				dims := layout.Center.Layout(gtx, editor.Layout)
				t.SetInstrumentName(ie.nameEditor.Text())
				return dims
			}
			text := t.Song().Patch[i].Name
			if text == "" {
				text = "Instr"
			}
			labelStyle := LabelStyle{Text: text, ShadeColor: black, Color: color, FontSize: unit.Sp(12)}
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
	instrumentList := FilledDragList(t.Theme, ie.instrumentDragList, len(t.Song().Patch), element, t.SwapInstruments)
	instrumentList.SelectedColor = color
	instrumentList.HoverColor = instrumentHoverColor

	ie.instrumentDragList.SelectedItem = t.InstrIndex()
	defer op.Save(gtx.Ops).Load()
	pointer.PassOp{Pass: true}.Add(gtx.Ops)
	spy, spiedGtx := eventx.Enspy(gtx)
	dims := instrumentList.Layout(spiedGtx)
	for _, group := range spy.AllEvents() {
		for _, event := range group.Items {
			switch e := event.(type) {
			case key.Event:
				if e.Modifiers.Contain(key.ModShortcut) {
					continue
				}
				if !ie.nameEditor.Focused() {
					switch e.State {
					case key.Press:
						switch e.Name {
						case key.NameDownArrow:
							ie.unitDragList.Focus()
						case key.NameReturn, key.NameEnter:
							ie.nameEditor.Focus()
						}
						t.JammingPressed(e)
					case key.Release:
						t.JammingReleased(e)
					}
				}
			}
		}
	}
	if t.InstrIndex() != ie.instrumentDragList.SelectedItem {
		t.SetInstrIndex(ie.instrumentDragList.SelectedItem)
		op.InvalidateOp{}.Add(gtx.Ops)
	}
	return dims
}
func (ie *InstrumentEditor) layoutInstrumentEditor(gtx C, t *Tracker) D {
	for ie.addUnitBtn.Clicked() {
		t.AddUnit(true)
		ie.unitDragList.Focus()
	}
	addUnitBtnStyle := material.IconButton(t.Theme, ie.addUnitBtn, widgetForIcon(icons.ContentAdd))
	addUnitBtnStyle.Color = t.Theme.ContrastFg
	addUnitBtnStyle.Background = t.Theme.Fg
	addUnitBtnStyle.Inset = layout.UniformInset(unit.Dp(4))

	units := t.Instrument().Units
	for len(ie.stackUse) < len(units) {
		ie.stackUse = append(ie.stackUse, 0)
	}

	stackHeight := 0
	for i, u := range units {
		stackHeight += u.StackChange()
		ie.stackUse[i] = stackHeight
	}

	element := func(gtx C, i int) D {
		gtx.Constraints = layout.Exact(image.Pt(gtx.Px(unit.Dp(120)), gtx.Px(unit.Dp(20))))
		u := units[i]
		var color color.NRGBA = white

		var stackText string
		if i < len(ie.stackUse) {
			stackText = strconv.FormatInt(int64(ie.stackUse[i]), 10)
			var prevStackUse int
			if i > 0 {
				prevStackUse = ie.stackUse[i-1]
			}
			if stackNeed := u.StackNeed(); stackNeed > prevStackUse {
				color = errorColor
				typeString := u.Type
				if u.Parameters["stereo"] == 1 {
					typeString += " (stereo)"
				}
				t.Alert.Update(fmt.Sprintf("%v needs at least %v input signals, got %v", typeString, stackNeed, prevStackUse), Error, 0)
			} else if i == len(units)-1 && ie.stackUse[i] != 0 {
				color = warningColor
				t.Alert.Update(fmt.Sprintf("Instrument leaves %v signal(s) on the stack", ie.stackUse[i]), Warning, 0)
			}
		}

		var unitName layout.Widget
		if i == t.UnitIndex() {
			for _, ev := range ie.unitTypeEditor.Events() {
				_, ok := ev.(widget.SubmitEvent)
				if ok {
					ie.unitDragList.Focus()
					if text := ie.unitTypeEditor.Text(); text != "" {
						for _, n := range tracker.UnitTypeNames {
							if strings.HasPrefix(n, ie.unitTypeEditor.Text()) {
								t.SetUnitType(n)
								break
							}
						}
					} else {
						t.SetUnitType("")
					}
					continue
				}
			}
			if !ie.unitTypeEditor.Focused() && !ie.paramEditor.Focused() && ie.unitTypeEditor.Text() != t.Unit().Type {
				ie.unitTypeEditor.SetText(t.Unit().Type)
			}
			editor := material.Editor(t.Theme, ie.unitTypeEditor, "---")
			editor.Color = color
			editor.HintColor = instrumentNameHintColor
			editor.TextSize = unit.Sp(12)
			editor.Font = labelDefaultFont
			unitName = func(gtx C) D {
				spy, spiedGtx := eventx.Enspy(gtx)
				ret := editor.Layout(spiedGtx)
				for _, group := range spy.AllEvents() {
					for _, event := range group.Items {
						switch e := event.(type) {
						case key.Event:
							if e.Name == key.NameEscape {
								ie.unitDragList.Focus()
							}
						}
					}
				}
				return ret
			}
		} else {
			unitNameLabel := LabelStyle{Text: u.Type, ShadeColor: black, Color: color, Font: labelDefaultFont, FontSize: unit.Sp(12)}
			if unitNameLabel.Text == "" {
				unitNameLabel.Text = "---"
			}
			unitName = unitNameLabel.Layout
		}

		stackLabel := LabelStyle{Text: stackText, ShadeColor: black, Color: mediumEmphasisTextColor, Font: labelDefaultFont, FontSize: unit.Sp(12)}
		rightMargin := layout.Inset{Right: unit.Dp(10)}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Flexed(1, unitName),
			layout.Rigid(func(gtx C) D {
				return rightMargin.Layout(gtx, stackLabel.Layout)
			}),
		)
	}

	defer op.Save(gtx.Ops).Load()
	pointer.PassOp{Pass: true}.Add(gtx.Ops)
	unitList := FilledDragList(t.Theme, ie.unitDragList, len(units), element, t.SwapUnits)
	ie.unitDragList.SelectedItem = t.UnitIndex()
	return Surface{Gray: 30, Focus: ie.wasFocused}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Stack{Alignment: layout.SE}.Layout(gtx,
					layout.Expanded(func(gtx C) D {
						spy, spiedGtx := eventx.Enspy(gtx)
						dims := unitList.Layout(spiedGtx)
						prevUnitIndex := t.UnitIndex()
						if t.UnitIndex() != ie.unitDragList.SelectedItem {
							t.SetUnitIndex(ie.unitDragList.SelectedItem)
							ie.unitTypeEditor.SetText(t.Unit().Type)
						}
						if ie.unitDragList.Focused() {
							for _, group := range spy.AllEvents() {
								for _, event := range group.Items {
									switch e := event.(type) {
									case key.Event:
										switch e.State {
										case key.Press:
											switch e.Name {
											case key.NameUpArrow:
												if prevUnitIndex == 0 {
													ie.instrumentDragList.Focus()
												}
											case key.NameRightArrow:
												ie.paramEditor.Focus()
											case key.NameDeleteBackward:
												t.SetUnitType("")
												ie.unitTypeEditor.Focus()
												l := len(ie.unitTypeEditor.Text())
												ie.unitTypeEditor.SetCaret(l, l)
											case key.NameDeleteForward:
												t.DeleteUnit(true)
											case key.NameReturn:
												if e.Modifiers.Contain(key.ModShortcut) {
													t.AddUnit(true)
												}
												ie.unitTypeEditor.Focus()
												l := len(ie.unitTypeEditor.Text())
												ie.unitTypeEditor.SetCaret(l, l)
											}
											if e.Modifiers.Contain(key.ModShortcut) {
												continue
											}
											t.JammingPressed(e)
										case key.Release:
											t.JammingReleased(e)
										}
									}
								}
							}
						}
						return dims
					}),
					layout.Expanded(func(gtx C) D {
						return ie.unitScrollBar.Layout(gtx, unit.Dp(10), len(t.Instrument().Units), &ie.unitDragList.List.Position)
					}),
					layout.Stacked(func(gtx C) D {
						margin := layout.Inset{Right: unit.Dp(20), Bottom: unit.Dp(1)}
						return margin.Layout(gtx, addUnitBtnStyle.Layout)
					}))
			}),
			layout.Rigid(ie.paramEditor.Bind(t)))
	})
}
