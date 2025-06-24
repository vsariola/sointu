package gioui

import (
	"fmt"
	"image/color"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/vsariola/sointu/tracker"
)

const DIALOG_MAX_BTNS = 3

type (
	// DialogState is the state that needs to be retained between frames
	DialogState struct {
		Clickables [DIALOG_MAX_BTNS]widget.Clickable

		visible bool // this is used to control the visibility of the dialog
	}

	// DialogStyle is the style for a dialog that is store in the theme.yml
	DialogStyle struct {
		TitleInset  layout.Inset
		TextInset   layout.Inset
		ButtonStyle ButtonStyle
		Title       LabelStyle
		Text        LabelStyle
		Bg          color.NRGBA
		Buttons     ButtonStyle
	}

	// Dialog is the widget with a Layout method that can be used to display a dialog.
	Dialog struct {
		Theme   *Theme
		State   *DialogState
		Style   *DialogStyle
		Btns    [DIALOG_MAX_BTNS]DialogButton
		NumBtns int
		Title   string
		Text    string
	}

	DialogButton struct {
		Text   string
		Action tracker.Action
	}
)

func MakeDialog(th *Theme, d *DialogState, title, text string, btns ...DialogButton) Dialog {
	ret := Dialog{
		Theme: th,
		Style: &th.Dialog,
		State: d,
		Title: title,
		Text:  text,
	}
	if len(btns) > DIALOG_MAX_BTNS {
		panic(fmt.Sprintf("too many buttons for dialog: %d, max is %d", len(btns), DIALOG_MAX_BTNS))
	}
	copy(ret.Btns[:], btns)
	ret.NumBtns = len(btns)
	d.visible = true
	return ret
}

func DialogBtn(text string, action tracker.Action) DialogButton {
	return DialogButton{Text: text, Action: action}
}

func (d *Dialog) Layout(gtx C) D {
	anyFocused := false
	for i := 0; i < d.NumBtns; i++ {
		anyFocused = anyFocused || gtx.Source.Focused(&d.State.Clickables[i])
	}
	if !anyFocused {
		gtx.Execute(key.FocusCmd{Tag: &d.State.Clickables[d.NumBtns-1]})
	}
	d.handleKeys(gtx)
	paint.Fill(gtx.Ops, d.Style.Bg)
	return layout.Center.Layout(gtx, func(gtx C) D {
		return Popup(d.Theme, &d.State.visible).Layout(gtx, func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return d.Style.TitleInset.Layout(gtx, Label(d.Theme, &d.Style.Title, d.Title).Layout)
				}),
				layout.Rigid(func(gtx C) D {
					return d.Style.TextInset.Layout(gtx, Label(d.Theme, &d.Style.Text, d.Text).Layout)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.E.Layout(gtx, func(gtx C) D {
						var fcs [DIALOG_MAX_BTNS]layout.FlexChild
						var actBtns [DIALOG_MAX_BTNS]material.ButtonStyle
						for i := 0; i < d.NumBtns; i++ {
							actBtns[i] = material.Button(&d.Theme.Material, &d.State.Clickables[i], d.Btns[i].Text)
							actBtns[i].Background = d.Style.Buttons.Background
							actBtns[i].Color = d.Style.Buttons.Color
							actBtns[i].TextSize = d.Style.Buttons.TextSize
							actBtns[i].Font = d.Style.Buttons.Font
							actBtns[i].Inset = d.Style.Buttons.Inset
							actBtns[i].CornerRadius = d.Style.Buttons.CornerRadius
						}
						// putting this inside these inside the for loop
						// cause heap escapes, so that's why this ugliness;
						// remember to update if you change the
						// DIAOLG_MAX_BTNS constant
						fcs[0] = layout.Rigid(actBtns[0].Layout)
						fcs[1] = layout.Rigid(actBtns[1].Layout)
						fcs[2] = layout.Rigid(actBtns[2].Layout)
						gtx.Constraints.Min.Y = gtx.Dp(d.Style.Buttons.Height)
						return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx, fcs[:d.NumBtns]...)
					})
				}),
			)
		})
	})
}

func (d *Dialog) handleKeys(gtx C) {
	for i := 0; i < d.NumBtns; i++ {
		for d.State.Clickables[i].Clicked(gtx) {
			d.Btns[i].Action.Do()
		}
		d.handleKeysForButton(gtx, (i+d.NumBtns-1)%d.NumBtns, i, (i+1)%d.NumBtns)
	}
}

func (d *Dialog) handleKeysForButton(gtx C, prev, cur, next int) {
	cPrev := &d.State.Clickables[prev]
	cCur := &d.State.Clickables[cur]
	cNext := &d.State.Clickables[next]
	for {
		e, ok := gtx.Event(
			key.Filter{Focus: cCur, Name: key.NameLeftArrow},
			key.Filter{Focus: cCur, Name: key.NameRightArrow},
			key.Filter{Focus: cCur, Name: key.NameEscape},
			key.Filter{Focus: cCur, Name: key.NameTab, Optional: key.ModShift},
		)
		if !ok {
			break
		}
		if e, ok := e.(key.Event); ok && e.State == key.Press {
			switch {
			case e.Name == key.NameLeftArrow || (e.Name == key.NameTab && e.Modifiers.Contain(key.ModShift)):
				gtx.Execute(key.FocusCmd{Tag: cPrev})
			case e.Name == key.NameRightArrow || (e.Name == key.NameTab && !e.Modifiers.Contain(key.ModShift)):
				gtx.Execute(key.FocusCmd{Tag: cNext})
			case e.Name == key.NameEscape:
				d.Btns[d.NumBtns-1].Action.Do() // last button is always the cancel button
			}
		}
	}
}
