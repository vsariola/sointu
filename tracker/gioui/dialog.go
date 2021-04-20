package gioui

import (
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type Dialog struct {
	Visible   bool
	BtnAlt    widget.Clickable
	BtnOk     widget.Clickable
	BtnCancel widget.Clickable
}

type DialogStyle struct {
	dialog      *Dialog
	Text        string
	Inset       layout.Inset
	ShowAlt     bool
	AltStyle    material.ButtonStyle
	OkStyle     material.ButtonStyle
	CancelStyle material.ButtonStyle
}

func ConfirmDialog(th *material.Theme, dialog *Dialog, text string) DialogStyle {
	ret := DialogStyle{
		dialog:      dialog,
		Text:        text,
		Inset:       layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(20), Right: unit.Dp(20)},
		AltStyle:    HighEmphasisButton(th, &dialog.BtnAlt, "Alt"),
		OkStyle:     HighEmphasisButton(th, &dialog.BtnOk, "Ok"),
		CancelStyle: HighEmphasisButton(th, &dialog.BtnCancel, "Cancel"),
	}
	return ret
}

func (d *DialogStyle) Layout(gtx C) D {
	if d.dialog.Visible {
		paint.Fill(gtx.Ops, dialogBgColor)
		return layout.Center.Layout(gtx, func(gtx C) D {
			return Popup(&d.dialog.Visible).Layout(gtx, func(gtx C) D {
				return d.Inset.Layout(gtx, func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(Label(d.Text, highEmphasisTextColor)),
						layout.Rigid(func(gtx C) D {
							gtx.Constraints.Min.X = gtx.Px(unit.Dp(120))
							if d.ShowAlt {
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
						}),
					)
				})
			})
		})
	}
	return D{}
}
