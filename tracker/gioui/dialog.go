package gioui

import (
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/vsariola/sointu/tracker"
)

type Dialog struct {
	BtnAlt    *ActionClickable
	BtnOk     *ActionClickable
	BtnCancel *ActionClickable
	tag       bool
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
	return &Dialog{
		BtnOk:     NewActionClickable(ok),
		BtnAlt:    NewActionClickable(alt),
		BtnCancel: NewActionClickable(cancel),
	}
}

func ConfirmDialog(th *material.Theme, dialog *Dialog, title, text string) DialogStyle {
	ret := DialogStyle{
		dialog:      dialog,
		Title:       title,
		Text:        text,
		Inset:       layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(20), Right: unit.Dp(20)},
		TextInset:   layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12)},
		AltStyle:    ActionButton(th, dialog.BtnAlt, "Alt"),
		OkStyle:     ActionButton(th, dialog.BtnOk, "Ok"),
		CancelStyle: ActionButton(th, dialog.BtnCancel, "Cancel"),
		Shaper:      th.Shaper,
	}
	return ret
}

func (d *DialogStyle) Layout(gtx C) D {
	if !d.dialog.BtnOk.Clickable.Focused() && !d.dialog.BtnCancel.Clickable.Focused() && !d.dialog.BtnAlt.Clickable.Focused() {
		d.dialog.BtnCancel.Clickable.Focus()
	}
	paint.Fill(gtx.Ops, dialogBgColor)
	text := func(gtx C) D {
		return d.TextInset.Layout(gtx, LabelStyle{Text: d.Text, Color: highEmphasisTextColor, Font: labelDefaultFont, FontSize: unit.Sp(14), Shaper: d.Shaper}.Layout)
	}
	for _, e := range gtx.Events(&d.dialog.tag) {
		if e, ok := e.(key.Event); ok && e.State == key.Press {
			d.command(e)
		}
	}
	visible := true
	return layout.Center.Layout(gtx, func(gtx C) D {
		return Popup(&visible).Layout(gtx, func(gtx C) D {
			key.InputOp{Tag: &d.dialog.tag, Keys: "⎋|←|→|Tab"}.Add(gtx.Ops)
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

func (d *DialogStyle) command(e key.Event) {
	switch e.Name {
	case key.NameEscape:
		d.dialog.BtnCancel.Action.Do()
	case key.NameLeftArrow:
		switch {
		case d.dialog.BtnOk.Clickable.Focused():
			d.dialog.BtnCancel.Clickable.Focus()
		case d.dialog.BtnCancel.Clickable.Focused():
			if d.dialog.BtnAlt.Action.Allowed() {
				d.dialog.BtnAlt.Clickable.Focus()
			} else {
				d.dialog.BtnOk.Clickable.Focus()
			}
		case d.dialog.BtnAlt.Clickable.Focused():
			d.dialog.BtnOk.Clickable.Focus()
		}
	case key.NameRightArrow, key.NameTab:
		switch {
		case d.dialog.BtnOk.Clickable.Focused():
			if d.dialog.BtnAlt.Action.Allowed() {
				d.dialog.BtnAlt.Clickable.Focus()
			} else {
				d.dialog.BtnCancel.Clickable.Focus()
			}
		case d.dialog.BtnCancel.Clickable.Focused():
			d.dialog.BtnOk.Clickable.Focus()
		case d.dialog.BtnAlt.Clickable.Focused():
			d.dialog.BtnCancel.Clickable.Focus()
		}
	}
}
