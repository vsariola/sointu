package gioui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
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
	"golang.org/x/exp/shiny/materialdesign/icons"
	"gopkg.in/yaml.v3"
)

var instrumentPointerTag = false

func (t *Tracker) layoutInstruments(gtx C) D {
	for _, ev := range gtx.Events(&instrumentPointerTag) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		if e.Type == pointer.Press && (t.EditMode() != tracker.EditUnits && t.EditMode() != tracker.EditParameters) {
			t.SetEditMode(tracker.EditUnits)
		}
	}
	rect := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
	pointer.Rect(rect).Add(gtx.Ops)
	pointer.InputOp{Tag: &instrumentPointerTag,
		Types: pointer.Press,
	}.Add(gtx.Ops)
	for t.NewInstrumentBtn.Clicked() {
		t.AddInstrument(true)
	}
	btnStyle := IconButton(t.Theme, t.NewInstrumentBtn, icons.ContentAdd, t.CanAddInstrument())
	spy, spiedGtx := eventx.Enspy(gtx)
	ret := layout.Flex{Axis: layout.Vertical}.Layout(spiedGtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{}.Layout(
				gtx,
				layout.Flexed(1, func(gtx C) D {
					return layout.Stack{}.Layout(gtx,
						layout.Stacked(t.layoutInstrumentNames),
						layout.Expanded(func(gtx C) D {
							return t.InstrumentScrollBar.Layout(gtx, unit.Dp(6), len(t.Song().Patch), &t.InstrumentDragList.List.Position)
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.E.Layout(gtx, btnStyle.Layout)
				}),
			)
		}),
		layout.Rigid(t.layoutInstrumentHeader),
		layout.Flexed(1, t.layoutInstrumentEditor))
	for _, group := range spy.AllEvents() {
		for _, event := range group.Items {
			switch e := event.(type) {
			case key.Event:
				if e.Name == key.NameEscape {
					key.FocusOp{}.Add(gtx.Ops)
				}
			}
		}
	}
	return ret
}

func (t *Tracker) layoutInstrumentHeader(gtx C) D {
	header := func(gtx C) D {
		collapseIcon := icons.NavigationExpandLess
		if t.InstrumentExpanded {
			collapseIcon = icons.NavigationExpandMore
		}

		instrumentExpandBtnStyle := IconButton(t.Theme, t.InstrumentExpandBtn, collapseIcon, true)
		copyInstrumentBtnStyle := IconButton(t.Theme, t.CopyInstrumentBtn, icons.ContentContentCopy, true)
		saveInstrumentBtnStyle := IconButton(t.Theme, t.SaveInstrumentBtn, icons.ContentSave, true)
		loadInstrumentBtnStyle := IconButton(t.Theme, t.LoadInstrumentBtn, icons.FileFolderOpen, true)
		deleteInstrumentBtnStyle := IconButton(t.Theme, t.DeleteInstrumentBtn, icons.ActionDelete, t.CanDeleteInstrument())

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
				layout.Rigid(instrumentExpandBtnStyle.Layout),
				layout.Rigid(saveInstrumentBtnStyle.Layout),
				layout.Rigid(loadInstrumentBtnStyle.Layout),
				layout.Rigid(copyInstrumentBtnStyle.Layout),
				layout.Rigid(deleteInstrumentBtnStyle.Layout))
		}
		for t.InstrumentExpandBtn.Clicked() {
			t.InstrumentExpanded = !t.InstrumentExpanded
			if !t.InstrumentExpanded {
				key.FocusOp{Tag: nil}.Add(gtx.Ops) // clear focus
			}
		}
		if t.InstrumentExpanded || t.InstrumentCommentEditor.Focused() { // we draw once the widget after it manages to lose focus
			if t.InstrumentCommentEditor.Text() != t.Instrument().Comment {
				t.InstrumentCommentEditor.SetText(t.Instrument().Comment)
			}
			editorStyle := material.Editor(t.Theme, t.InstrumentCommentEditor, "Comment")
			editorStyle.Color = highEmphasisTextColor

			spy, spiedGtx := eventx.Enspy(gtx)
			ret := layout.Flex{Axis: layout.Vertical}.Layout(spiedGtx,
				layout.Rigid(header),
				layout.Rigid(func(gtx C) D {
					return layout.UniformInset(unit.Dp(6)).Layout(gtx, editorStyle.Layout)
				}),
			)
			for _, group := range spy.AllEvents() {
				for _, event := range group.Items {
					switch e := event.(type) {
					case key.Event:
						if e.Name == key.NameEscape {
							key.FocusOp{}.Add(gtx.Ops)
						}
					}
				}
			}
			t.SetInstrumentComment(t.InstrumentCommentEditor.Text())
			return ret
		}
		return header(gtx)
	}
	for t.CopyInstrumentBtn.Clicked() {
		contents, err := yaml.Marshal(t.Instrument())
		if err == nil {
			clipboard.WriteOp{Text: string(contents)}.Add(gtx.Ops)
			t.Alert.Update("Instrument copied to clipboard", Notify, time.Second*3)
		}
	}
	for t.DeleteInstrumentBtn.Clicked() {
		t.ConfirmInstrDelete.Visible = true
	}
	for t.ConfirmInstrDelete.BtnOk.Clicked() {
		t.DeleteInstrument(false)
		t.ConfirmInstrDelete.Visible = false
	}
	for t.ConfirmInstrDelete.BtnCancel.Clicked() {
		t.ConfirmInstrDelete.Visible = false
	}
	for t.SaveInstrumentBtn.Clicked() {
		t.SaveInstrument()
	}

	for t.LoadInstrumentBtn.Clicked() {
		t.LoadInstrument()
	}
	return Surface{Gray: 37, Focus: t.EditMode() == tracker.EditUnits || t.EditMode() == tracker.EditParameters}.Layout(gtx, header)
}

func (t *Tracker) layoutInstrumentNames(gtx C) D {
	element := func(gtx C, i int) D {
		gtx.Constraints.Min.Y = gtx.Px(unit.Dp(36))
		gtx.Constraints.Min.X = gtx.Px(unit.Dp(30))
		grabhandle := LabelStyle{Text: "", ShadeColor: black, Color: white, FontSize: unit.Sp(10), Alignment: layout.Center}
		if i == t.InstrIndex() {
			grabhandle.Text = ":::"
		}
		label := func(gtx C) D {
			c := 0.0
			voice := t.Song().Patch.FirstVoiceForInstrument(i)
			for j := 0; j < t.Song().Patch[i].NumVoices; j++ {
				released, event := t.player.VoiceState(voice)
				vc := math.Exp(-float64(event)/15000) * .5
				if !released {
					vc += .5
				}
				if c < vc {
					c = vc
				}
				voice++
			}
			k := byte(255 - c*127)
			color := color.NRGBA{R: 255, G: k, B: 255, A: 255}
			if i == t.InstrIndex() {
				for _, ev := range t.InstrumentNameEditor.Events() {
					_, ok := ev.(widget.SubmitEvent)
					if ok {
						t.InstrumentNameEditor = &widget.Editor{SingleLine: true, Submit: true, Alignment: text.Middle} // TODO: is there any other way to defocus the editor
						break
					}
				}
				if n := t.Instrument().Name; n != t.InstrumentNameEditor.Text() {
					t.InstrumentNameEditor.SetText(n)
				}
				editor := material.Editor(t.Theme, t.InstrumentNameEditor, "Instr")
				editor.Color = color
				editor.HintColor = instrumentNameHintColor
				editor.TextSize = unit.Dp(12)
				dims := layout.Center.Layout(gtx, editor.Layout)
				t.SetInstrumentName(t.InstrumentNameEditor.Text())
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
	if t.EditMode() == tracker.EditUnits || t.EditMode() == tracker.EditParameters {
		color = activeLightSurfaceColor
	}
	instrumentList := FilledDragList(t.Theme, t.InstrumentDragList, len(t.Song().Patch), element, t.SwapInstruments)
	instrumentList.SelectedColor = color
	instrumentList.HoverColor = instrumentHoverColor

	t.InstrumentDragList.SelectedItem = t.InstrIndex()
	defer op.Save(gtx.Ops).Load()
	pointer.PassOp{Pass: true}.Add(gtx.Ops)
	dims := instrumentList.Layout(gtx)
	if t.InstrIndex() != t.InstrumentDragList.SelectedItem {
		t.SetInstrIndex(t.InstrumentDragList.SelectedItem)
		op.InvalidateOp{}.Add(gtx.Ops)
	}
	return dims
}
func (t *Tracker) layoutInstrumentEditor(gtx C) D {
	for t.AddUnitBtn.Clicked() {
		t.AddUnit(true)
	}
	addUnitBtnStyle := material.IconButton(t.Theme, t.AddUnitBtn, widgetForIcon(icons.ContentAdd))
	addUnitBtnStyle.Color = t.Theme.ContrastFg
	addUnitBtnStyle.Background = t.Theme.Fg
	addUnitBtnStyle.Inset = layout.UniformInset(unit.Dp(4))

	units := t.Instrument().Units
	for len(t.StackUse) < len(units) {
		t.StackUse = append(t.StackUse, 0)
	}

	stackHeight := 0
	for i, u := range units {
		stackHeight += u.StackChange()
		t.StackUse[i] = stackHeight
	}

	element := func(gtx C, i int) D {
		gtx.Constraints = layout.Exact(image.Pt(gtx.Px(unit.Dp(120)), gtx.Px(unit.Dp(20))))
		u := units[i]
		unitNameLabel := LabelStyle{Text: u.Type, ShadeColor: black, Color: white, Font: labelDefaultFont, FontSize: unit.Sp(12)}
		if unitNameLabel.Text == "" {
			unitNameLabel.Text = "---"
			unitNameLabel.Alignment = layout.Center
		}
		var stackText string
		if i < len(t.StackUse) {
			stackText = strconv.FormatInt(int64(t.StackUse[i]), 10)
			var prevStackUse int
			if i > 0 {
				prevStackUse = t.StackUse[i-1]
			}
			if stackNeed := u.StackNeed(); stackNeed > prevStackUse {
				unitNameLabel.Color = errorColor
				typeString := u.Type
				if u.Parameters["stereo"] == 1 {
					typeString += " (stereo)"
				}
				t.Alert.Update(fmt.Sprintf("%v needs at least %v input signals, got %v", typeString, stackNeed, prevStackUse), Error, 0)
			} else if i == len(units)-1 && t.StackUse[i] != 0 {
				unitNameLabel.Color = warningColor
				t.Alert.Update(fmt.Sprintf("Instrument leaves %v signal(s) on the stack", t.StackUse[i]), Warning, 0)
			}
		}
		stackLabel := LabelStyle{Text: stackText, ShadeColor: black, Color: mediumEmphasisTextColor, Font: labelDefaultFont, FontSize: unit.Sp(12)}
		rightMargin := layout.Inset{Right: unit.Dp(10)}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Flexed(1, unitNameLabel.Layout),
			layout.Rigid(func(gtx C) D {
				return rightMargin.Layout(gtx, stackLabel.Layout)
			}),
		)
	}

	unitList := FilledDragList(t.Theme, t.UnitDragList, len(units), element, t.SwapUnits)

	if t.EditMode() == tracker.EditUnits {
		unitList.SelectedColor = cursorColor
	}

	t.UnitDragList.SelectedItem = t.UnitIndex()
	return Surface{Gray: 30, Focus: t.EditMode() == tracker.EditUnits || t.EditMode() == tracker.EditParameters}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Stack{Alignment: layout.SE}.Layout(gtx,
					layout.Expanded(func(gtx C) D {
						dims := unitList.Layout(gtx)
						if t.UnitIndex() != t.UnitDragList.SelectedItem {
							t.SetUnitIndex(t.UnitDragList.SelectedItem)
							t.SetEditMode(tracker.EditUnits)
							op.InvalidateOp{}.Add(gtx.Ops)
						}
						return dims
					}),
					layout.Expanded(func(gtx C) D {
						return t.UnitScrollBar.Layout(gtx, unit.Dp(10), len(t.Instrument().Units), &t.UnitDragList.List.Position)
					}),
					layout.Stacked(func(gtx C) D {
						margin := layout.Inset{Right: unit.Dp(20), Bottom: unit.Dp(1)}
						return margin.Layout(gtx, addUnitBtnStyle.Layout)
					}))
			}),
			layout.Rigid(t.layoutUnitEditor))
	})
}
