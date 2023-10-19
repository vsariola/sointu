package gioui

import (
	"image"
	"image/color"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type Menu struct {
	Visible   bool
	clickable widget.Clickable
	tags      []bool
	clicks    []int
	hover     int
	list      layout.List
	scrollBar ScrollBar
}

type MenuStyle struct {
	Menu          *Menu
	Title         string
	IconColor     color.NRGBA
	TextColor     color.NRGBA
	ShortCutColor color.NRGBA
	FontSize      unit.Sp
	IconSize      unit.Dp
	HoverColor    color.NRGBA
}

type MenuItem struct {
	IconBytes    []byte
	Text         string
	ShortcutText string
	Disabled     bool
}

func (m *Menu) Clicked() (int, bool) {
	if len(m.clicks) == 0 {
		return 0, false
	}
	first := m.clicks[0]
	for i := 1; i < len(m.clicks); i++ {
		m.clicks[i-1] = m.clicks[i]
	}
	m.clicks = m.clicks[:len(m.clicks)-1]
	return first, true
}

func (m *MenuStyle) Layout(gtx C, items ...MenuItem) D {
	contents := func(gtx C) D {
		for i := range items {
			// make sure we have a tag for every item
			for len(m.Menu.tags) <= i {
				m.Menu.tags = append(m.Menu.tags, false)
			}
			// handle pointer events for this item
			for _, ev := range gtx.Events(&m.Menu.tags[i]) {
				e, ok := ev.(pointer.Event)
				if !ok {
					continue
				}
				switch e.Type {
				case pointer.Press:
					m.Menu.clicks = append(m.Menu.clicks, i)
					m.Menu.Visible = false
				case pointer.Enter:
					m.Menu.hover = i + 1
				case pointer.Leave:
					if m.Menu.hover == i+1 {
						m.Menu.hover = 0
					}
				}
			}
		}
		m.Menu.list.Axis = layout.Vertical
		m.Menu.scrollBar.Axis = layout.Vertical
		return layout.Stack{Alignment: layout.SE}.Layout(gtx,
			layout.Expanded(func(gtx C) D {
				return m.Menu.list.Layout(gtx, len(items), func(gtx C, i int) D {
					defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
					var macro op.MacroOp
					item := &items[i]
					if i == m.Menu.hover-1 && !item.Disabled {
						macro = op.Record(gtx.Ops)
					}
					icon := widgetForIcon(item.IconBytes)
					iconColor := m.IconColor
					if item.Disabled {
						iconColor = mediumEmphasisTextColor
					}
					iconInset := layout.Inset{Left: unit.Dp(12), Right: unit.Dp(6)}
					textLabel := LabelStyle{Text: item.Text, FontSize: m.FontSize, Color: m.TextColor}
					if item.Disabled {
						textLabel.Color = mediumEmphasisTextColor
					}
					shortcutLabel := LabelStyle{Text: item.ShortcutText, FontSize: m.FontSize, Color: m.ShortCutColor}
					shortcutInset := layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12), Bottom: unit.Dp(2), Top: unit.Dp(2)}
					dims := layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return iconInset.Layout(gtx, func(gtx C) D {
								p := gtx.Dp(unit.Dp(m.IconSize))
								gtx.Constraints.Min = image.Pt(p, p)
								return icon.Layout(gtx, iconColor)
							})
						}),
						layout.Rigid(textLabel.Layout),
						layout.Flexed(1, func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Max.X, 1)} }),
						layout.Rigid(func(gtx C) D {
							return shortcutInset.Layout(gtx, shortcutLabel.Layout)
						}),
					)
					if i == m.Menu.hover-1 && !item.Disabled {
						recording := macro.Stop()
						paint.FillShape(gtx.Ops, m.HoverColor, clip.Rect{
							Max: image.Pt(dims.Size.X, dims.Size.Y),
						}.Op())
						recording.Add(gtx.Ops)
					}
					if !item.Disabled {
						rect := image.Rect(0, 0, dims.Size.X, dims.Size.Y)
						area := clip.Rect(rect).Push(gtx.Ops)
						pointer.InputOp{Tag: &m.Menu.tags[i],
							Types: pointer.Press | pointer.Enter | pointer.Leave,
						}.Add(gtx.Ops)
						area.Pop()
					}
					return dims
				})
			}),
			layout.Expanded(func(gtx C) D {
				return m.Menu.scrollBar.Layout(gtx, unit.Dp(10), len(items), &m.Menu.list.Position)
			}),
		)
	}
	popup := Popup(&m.Menu.Visible)
	popup.NE = unit.Dp(0)
	popup.ShadowN = unit.Dp(0)
	popup.NW = unit.Dp(0)
	return popup.Layout(gtx, contents)
}

func PopupMenu(th *material.Theme, menu *Menu) MenuStyle {
	return MenuStyle{
		Menu:          menu,
		IconColor:     white,
		TextColor:     white,
		ShortCutColor: mediumEmphasisTextColor,
		FontSize:      unit.Sp(16),
		IconSize:      unit.Dp(16),
		HoverColor:    menuHoverColor,
	}
}
