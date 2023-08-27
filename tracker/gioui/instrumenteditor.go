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
	"gioui.org/op/clip"
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
	newInstrumentBtn    *TipClickable
	enlargeBtn          *TipClickable
	deleteInstrumentBtn *TipClickable
	copyInstrumentBtn   *TipClickable
	saveInstrumentBtn   *TipClickable
	loadInstrumentBtn   *TipClickable
	addUnitBtn          *TipClickable
	commentExpandBtn    *TipClickable
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
		newInstrumentBtn:    new(TipClickable),
		enlargeBtn:          new(TipClickable),
		deleteInstrumentBtn: new(TipClickable),
		copyInstrumentBtn:   new(TipClickable),
		saveInstrumentBtn:   new(TipClickable),
		loadInstrumentBtn:   new(TipClickable),
		addUnitBtn:          new(TipClickable),
		commentExpandBtn:    new(TipClickable),
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
	area := clip.Rect(rect).Push(gtx.Ops)
	pointer.InputOp{Tag: &ie.tag,
		Types: pointer.Press,
	}.Add(gtx.Ops)
	area.Pop()

	enlargeTip := "Enlarge"
	icon := icons.NavigationFullscreen
	if t.InstrEnlarged() {
		icon = icons.NavigationFullscreenExit
		enlargeTip = "Shrink"
	}
	fullscreenBtnStyle := IconButton(t.Theme, ie.enlargeBtn, icon, true, enlargeTip)
	for ie.enlargeBtn.Clickable.Clicked() {
		t.SetInstrEnlarged(!t.InstrEnlarged())
	}
	for ie.newInstrumentBtn.Clickable.Clicked() {
		t.AddInstrument(true)
	}
	octave := func(gtx C) D {
		in := layout.UniformInset(unit.Dp(1))
		t.OctaveNumberInput.Value = t.Octave()
		numStyle := NumericUpDown(t.Theme, t.OctaveNumberInput, 0, 9, "Octave down (<) or up (>)")
		dims := in.Layout(gtx, numStyle.Layout)
		t.SetOctave(t.OctaveNumberInput.Value)
		return dims
	}
	newBtnStyle := IconButton(t.Theme, ie.newInstrumentBtn, icons.ContentAdd, t.CanAddInstrument(), "Add\ninstrument")
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
		commentTip := "Collapse comment"
		if !ie.commentExpanded {
			collapseIcon = icons.NavigationExpandMore
			commentTip = "Expand comment"
		}

		commentExpandBtnStyle := IconButton(t.Theme, ie.commentExpandBtn, collapseIcon, true, commentTip)
		copyInstrumentBtnStyle := IconButton(t.Theme, ie.copyInstrumentBtn, icons.ContentContentCopy, true, "Copy instrument")
		saveInstrumentBtnStyle := IconButton(t.Theme, ie.saveInstrumentBtn, icons.ContentSave, true, "Save instrument")
		loadInstrumentBtnStyle := IconButton(t.Theme, ie.loadInstrumentBtn, icons.FileFolderOpen, true, "Load instrument")
		deleteInstrumentBtnStyle := IconButton(t.Theme, ie.deleteInstrumentBtn, icons.ActionDelete, t.CanDeleteInstrument(), "Delete\ninstrument")

		header := func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(Label("Voices: ", white)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					maxRemain := t.MaxInstrumentVoices()
					t.InstrumentVoices.Value = t.Instrument().NumVoices
					numStyle := NumericUpDown(t.Theme, t.InstrumentVoices, 0, maxRemain, "Number of voices for this instrument")
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
		for ie.commentExpandBtn.Clickable.Clicked() {
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
	for ie.copyInstrumentBtn.Clickable.Clicked() {
		contents, err := yaml.Marshal(t.Instrument())
		if err == nil {
			clipboard.WriteOp{Text: string(contents)}.Add(gtx.Ops)
			t.Alert.Update("Instrument copied to clipboard", Notify, time.Second*3)
		}
	}
	for ie.deleteInstrumentBtn.Clickable.Clicked() {
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
	for ie.saveInstrumentBtn.Clickable.Clicked() {
		t.SaveInstrument()
	}

	for ie.loadInstrumentBtn.Clickable.Clicked() {
		t.LoadInstrument()
	}
	return Surface{Gray: 37, Focus: ie.wasFocused}.Layout(gtx, header)
}

func (ie *InstrumentEditor) layoutInstrumentNames(gtx C, t *Tracker) D {
	element := func(gtx C, i int) D {
		gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(36))
		gtx.Constraints.Min.X = gtx.Dp(unit.Dp(30))
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
				editor.TextSize = unit.Sp(12)
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
	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
	key.InputOp{Tag: ie.instrumentDragList, Keys: "↓"}.Add(gtx.Ops)

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
				}
			}
		}
	}

	dims := instrumentList.Layout(gtx)

	if t.InstrIndex() != ie.instrumentDragList.SelectedItem {
		t.SetInstrIndex(ie.instrumentDragList.SelectedItem)
		op.InvalidateOp{}.Add(gtx.Ops)
	}
	return dims
}
func (ie *InstrumentEditor) layoutInstrumentEditor(gtx C, t *Tracker) D {
	for ie.addUnitBtn.Clickable.Clicked() {
		t.AddUnit(true)
		ie.unitDragList.Focus()
	}
	addUnitBtnStyle := IconButton(t.Theme, ie.addUnitBtn, icons.ContentAdd, true, "Add unit (Ctrl+Enter)")
	addUnitBtnStyle.IconButtonStyle.Color = t.Theme.ContrastFg
	addUnitBtnStyle.IconButtonStyle.Background = t.Theme.Fg
	addUnitBtnStyle.IconButtonStyle.Inset = layout.UniformInset(unit.Dp(4))

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
		gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(unit.Dp(120)), gtx.Dp(unit.Dp(20))))
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
			unitName = editor.Layout
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

	defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
	unitList := FilledDragList(t.Theme, ie.unitDragList, len(units), element, t.SwapUnits)
	return Surface{Gray: 30, Focus: ie.wasFocused}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Stack{Alignment: layout.SE}.Layout(gtx,
					layout.Expanded(func(gtx C) D {
						defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
						key.InputOp{Tag: ie.unitDragList, Keys: "→|⏎|⌫|⌦|⎋|Ctrl-⏎|Ctrl-C|Ctrl-X"}.Add(gtx.Ops)
						for _, event := range gtx.Events(ie.unitDragList) {
							switch e := event.(type) {
							case key.Event:
								switch e.State {
								case key.Press:
									switch e.Name {
									case key.NameEscape:
										ie.instrumentDragList.Focus()
									case key.NameRightArrow:
										ie.paramEditor.Focus()
									case key.NameDeleteBackward:
										t.SetUnitType("")
										ie.unitTypeEditor.Focus()
										l := len(ie.unitTypeEditor.Text())
										ie.unitTypeEditor.SetCaret(l, l)
									case key.NameDeleteForward:
										t.DeleteUnits(true, ie.unitDragList.SelectedItem, ie.unitDragList.SelectedItem2)
										ie.unitDragList.SelectedItem2 = t.UnitIndex()
									case "X":
										units := t.DeleteUnits(true, ie.unitDragList.SelectedItem, ie.unitDragList.SelectedItem2)
										ie.unitDragList.SelectedItem2 = t.UnitIndex()
										contents, err := yaml.Marshal(units)
										if err == nil {
											clipboard.WriteOp{Text: string(contents)}.Add(gtx.Ops)
											t.Alert.Update("Unit(s) cut to clipboard", Notify, time.Second*3)
										}
									case "C":
										a := clamp(ie.unitDragList.SelectedItem, 0, len(t.Instrument().Units)-1)
										b := clamp(ie.unitDragList.SelectedItem2, 0, len(t.Instrument().Units)-1)
										if a > b {
											a, b = b, a
										}
										units := t.Instrument().Units[a : b+1]
										contents, err := yaml.Marshal(units)
										if err == nil {
											clipboard.WriteOp{Text: string(contents)}.Add(gtx.Ops)
											t.Alert.Update("Unit(s) copied to clipboard", Notify, time.Second*3)
										}
									case key.NameReturn:
										if e.Modifiers.Contain(key.ModShortcut) {
											t.AddUnit(true)
											ie.unitDragList.SelectedItem2 = ie.unitDragList.SelectedItem
											ie.unitTypeEditor.SetText("")
										}
										ie.unitTypeEditor.Focus()
										l := len(ie.unitTypeEditor.Text())
										ie.unitTypeEditor.SetCaret(l, l)
									}
								}
							}
						}
						ie.unitDragList.SelectedItem = t.UnitIndex()
						dims := unitList.Layout(gtx)
						if t.UnitIndex() != ie.unitDragList.SelectedItem {
							t.SetUnitIndex(ie.unitDragList.SelectedItem)
							ie.unitTypeEditor.SetText(t.Unit().Type)
						}
						return dims
					}),
					layout.Stacked(func(gtx C) D {
						margin := layout.Inset{Right: unit.Dp(20), Bottom: unit.Dp(1)}
						return margin.Layout(gtx, addUnitBtnStyle.Layout)
					}),
					layout.Expanded(func(gtx C) D {
						return ie.unitScrollBar.Layout(gtx, unit.Dp(10), len(t.Instrument().Units), &ie.unitDragList.List.Position)
					}))
			}),
			layout.Rigid(ie.paramEditor.Bind(t)))
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
