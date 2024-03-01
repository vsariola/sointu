package gioui

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/vsariola/sointu/tracker"
)

type Dialog struct {
	BtnAlt     *ActionClickable
	BtnOk      *ActionClickable
	BtnCancel  *ActionClickable
	tag        bool
	keyFilters []event.Filter
}

type DialogStyle struct {
	dialog      *Dialog
	Title       string
	Text        string
	Inset       layout.Inset
	TextInset   layout.Inset
	AltStyle    material.ButtonStyle
	OkStyle     material.ButtonStyle
	CancelStyle material.ButtonStyle
	Shaper      *text.Shaper
}

func NewDialog(ok, alt, cancel tracker.Action) *Dialog {
	ret := &Dialog{
		BtnOk:     NewActionClickable(ok),
		BtnAlt:    NewActionClickable(alt),
		BtnCancel: NewActionClickable(cancel),
	}

	return ret
}

func ConfirmDialog(gtx C, th *material.Theme, dialog *Dialog, title, text string) DialogStyle {
	ret := DialogStyle{
		dialog:      dialog,
		Title:       title,
		Text:        text,
		Inset:       layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(20), Right: unit.Dp(20)},
		TextInset:   layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12)},
		AltStyle:    ActionButton(gtx, th, dialog.BtnAlt, "Alt"),
		OkStyle:     ActionButton(gtx, th, dialog.BtnOk, "Ok"),
		CancelStyle: ActionButton(gtx, th, dialog.BtnCancel, "Cancel"),
		Shaper:      th.Shaper,
	}
	return ret
}

func (d *Dialog) handleKeysForButton(gtx C, btn, next, prev *ActionClickable) {
	for {
		e, ok := gtx.Event(
			key.Filter{Focus: &btn.Clickable, Name: key.NameLeftArrow},
			key.Filter{Focus: &btn.Clickable, Name: key.NameRightArrow},
			key.Filter{Focus: &btn.Clickable, Name: key.NameEscape},
			key.Filter{Focus: &btn.Clickable, Name: key.NameTab, Optional: key.ModShift},
		)
		if !ok {
			break
		}
		if e, ok := e.(key.Event); ok && e.State == key.Press {
			switch {
			case e.Name == key.NameLeftArrow || (e.Name == key.NameTab && e.Modifiers.Contain(key.ModShift)):
				gtx.Execute(key.FocusCmd{Tag: &prev.Clickable})
			case e.Name == key.NameRightArrow || (e.Name == key.NameTab && !e.Modifiers.Contain(key.ModShift)):
				gtx.Execute(key.FocusCmd{Tag: &next.Clickable})
			case e.Name == key.NameEscape:
				d.BtnCancel.Action.Do()
			}
		}
	}
}

func (d *Dialog) handleKeys(gtx C) {
	if d.BtnAlt.Action.Allowed() {
		d.handleKeysForButton(gtx, d.BtnAlt, d.BtnCancel, d.BtnOk)
		d.handleKeysForButton(gtx, d.BtnCancel, d.BtnOk, d.BtnAlt)
		d.handleKeysForButton(gtx, d.BtnOk, d.BtnAlt, d.BtnCancel)
	} else {
		d.handleKeysForButton(gtx, d.BtnOk, d.BtnCancel, d.BtnCancel)
		d.handleKeysForButton(gtx, d.BtnCancel, d.BtnOk, d.BtnOk)
	}
}

func (d *DialogStyle) Layout(gtx C) D {
	if !gtx.Source.Focused(&d.dialog.BtnOk.Clickable) && !gtx.Source.Focused(&d.dialog.BtnCancel.Clickable) && !gtx.Source.Focused(&d.dialog.BtnAlt.Clickable) {
		gtx.Execute(key.FocusCmd{Tag: &d.dialog.BtnCancel.Clickable})
	}
	d.dialog.handleKeys(gtx)
	paint.Fill(gtx.Ops, dialogBgColor)
	text := func(gtx C) D {
		return d.TextInset.Layout(gtx, LabelStyle{Text: d.Text, Color: highEmphasisTextColor, Font: labelDefaultFont, FontSize: unit.Sp(14), Shaper: d.Shaper}.Layout)
	}
	visible := true
	return layout.Center.Layout(gtx, func(gtx C) D {
		return Popup(&visible).Layout(gtx, func(gtx C) D {
			return d.Inset.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(Label(d.Title, highEmphasisTextColor, d.Shaper)),
					layout.Rigid(text),
					layout.Rigid(func(gtx C) D {
						return layout.E.Layout(gtx, func(gtx C) D {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(120))
							if d.dialog.BtnAlt.Action.Allowed() {
								return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
									layout.Rigid(d.OkStyle.Layout),
									layout.Rigid(d.AltStyle.Layout),
									layout.Rigid(d.CancelStyle.Layout),
								)
							}
							return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
								layout.Rigid(d.OkStyle.Layout),
								layout.Rigid(d.CancelStyle.Layout),
							)
						})
					}),
				)
			})
		})
	})
}
