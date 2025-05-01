package gioui

import (
	"image/color"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/vsariola/sointu/tracker"
)

type Dialog struct {
	BtnAlt    widget.Clickable
	BtnOk     widget.Clickable
	BtnCancel widget.Clickable

	ok, alt, cancel tracker.Action
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
	Theme       *Theme
}

func NewDialog(ok, alt, cancel tracker.Action) *Dialog {
	ret := &Dialog{ok: ok, alt: alt, cancel: cancel}

	return ret
}

func ConfirmDialog(gtx C, th *Theme, dialog *Dialog, title, text string) DialogStyle {
	ret := DialogStyle{
		dialog:      dialog,
		Title:       title,
		Text:        text,
		Inset:       layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(20), Right: unit.Dp(20)},
		TextInset:   layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12)},
		AltStyle:    material.Button(&th.Material, &dialog.BtnAlt, "Alt"),
		OkStyle:     material.Button(&th.Material, &dialog.BtnOk, "Ok"),
		CancelStyle: material.Button(&th.Material, &dialog.BtnCancel, "Cancel"),
		Theme:       th,
	}
	for _, b := range [...]*material.ButtonStyle{&ret.AltStyle, &ret.OkStyle, &ret.CancelStyle} {
		b.Background = color.NRGBA{}
		b.Inset = layout.UniformInset(unit.Dp(6))
		b.Color = th.Material.Palette.Fg
	}
	return ret
}

func (d *Dialog) handleKeysForButton(gtx C, btn, next, prev *widget.Clickable) {
	for {
		e, ok := gtx.Event(
			key.Filter{Focus: btn, Name: key.NameLeftArrow},
			key.Filter{Focus: btn, Name: key.NameRightArrow},
			key.Filter{Focus: btn, Name: key.NameEscape},
			key.Filter{Focus: btn, Name: key.NameTab, Optional: key.ModShift},
		)
		if !ok {
			break
		}
		if e, ok := e.(key.Event); ok && e.State == key.Press {
			switch {
			case e.Name == key.NameLeftArrow || (e.Name == key.NameTab && e.Modifiers.Contain(key.ModShift)):
				gtx.Execute(key.FocusCmd{Tag: prev})
			case e.Name == key.NameRightArrow || (e.Name == key.NameTab && !e.Modifiers.Contain(key.ModShift)):
				gtx.Execute(key.FocusCmd{Tag: next})
			case e.Name == key.NameEscape:
				d.cancel.Do()
			}
		}
	}
}

func (d *Dialog) handleKeys(gtx C) {
	for d.BtnOk.Clicked(gtx) {
		d.ok.Do()
	}
	for d.BtnAlt.Clicked(gtx) {
		d.alt.Do()
	}
	for d.BtnCancel.Clicked(gtx) {
		d.cancel.Do()
	}
	if d.alt.Allowed() {
		d.handleKeysForButton(gtx, &d.BtnAlt, &d.BtnCancel, &d.BtnOk)
		d.handleKeysForButton(gtx, &d.BtnCancel, &d.BtnOk, &d.BtnAlt)
		d.handleKeysForButton(gtx, &d.BtnOk, &d.BtnAlt, &d.BtnCancel)
	} else {
		d.handleKeysForButton(gtx, &d.BtnOk, &d.BtnCancel, &d.BtnCancel)
		d.handleKeysForButton(gtx, &d.BtnCancel, &d.BtnOk, &d.BtnOk)
	}
}

func (d *DialogStyle) Layout(gtx C) D {
	if !gtx.Source.Focused(&d.dialog.BtnOk) && !gtx.Source.Focused(&d.dialog.BtnCancel) && !gtx.Source.Focused(&d.dialog.BtnAlt) {
		gtx.Execute(key.FocusCmd{Tag: &d.dialog.BtnCancel})
	}
	d.dialog.handleKeys(gtx)
	paint.Fill(gtx.Ops, d.Theme.Dialog.Bg)
	visible := true
	return layout.Center.Layout(gtx, func(gtx C) D {
		return Popup(d.Theme, &visible).Layout(gtx, func(gtx C) D {
			return d.Inset.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(Label(d.Theme, &d.Theme.Dialog.Title, d.Title).Layout),
					layout.Rigid(Label(d.Theme, &d.Theme.Dialog.Text, d.Text).Layout),
					layout.Rigid(func(gtx C) D {
						return layout.E.Layout(gtx, func(gtx C) D {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(120))
							if d.dialog.alt.Allowed() {
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
