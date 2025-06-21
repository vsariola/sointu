package gioui

import (
	"image"
	"image/color"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/vsariola/sointu/tracker"
)

type Menu struct {
	Visible   bool
	tags      []bool
	clicks    []int
	hover     int
	list      layout.List
	scrollBar ScrollBar
}

type MenuStyle struct {
	Menu          *Menu
	Title         string
	ShortCutColor color.NRGBA
	HoverColor    color.NRGBA
	Theme         *Theme
	LabelStyle    LabelStyle
	Disabled      color.NRGBA
}

type MenuItem struct {
	IconBytes    []byte
	Text         string
	ShortcutText string
	Doer         tracker.Action
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
		for i, item := range items {
			// make sure we have a tag for every item
			for len(m.Menu.tags) <= i {
				m.Menu.tags = append(m.Menu.tags, false)
			}
			// handle pointer events for this item
			for {
				ev, ok := gtx.Event(pointer.Filter{
					Target: &m.Menu.tags[i],
					Kinds:  pointer.Press | pointer.Enter | pointer.Leave,
				})
				if !ok {
					break
				}
				e, ok := ev.(pointer.Event)
				if !ok {
					continue
				}
				switch e.Kind {
				case pointer.Press:
					item.Doer.Do()
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
					if i == m.Menu.hover-1 && item.Doer.Enabled() {
						macro = op.Record(gtx.Ops)
					}
					icon := m.Theme.Icon(item.IconBytes)
					iconColor := m.LabelStyle.Color
					iconInset := layout.Inset{Left: unit.Dp(12), Right: unit.Dp(6)}
					textLabel := Label(m.Theme, &m.Theme.Menu.Text, item.Text)
					shortcutLabel := Label(m.Theme, &m.Theme.Menu.Text, item.ShortcutText)
					shortcutLabel.Color = m.ShortCutColor
					if !item.Doer.Enabled() {
						iconColor = m.Disabled
						textLabel.Color = m.Disabled
						shortcutLabel.Color = m.Disabled
					}
					shortcutInset := layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12), Bottom: unit.Dp(2), Top: unit.Dp(2)}
					dims := layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return iconInset.Layout(gtx, func(gtx C) D {
								p := gtx.Dp(unit.Dp(m.LabelStyle.TextSize))
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
					if i == m.Menu.hover-1 && item.Doer.Enabled() {
						recording := macro.Stop()
						paint.FillShape(gtx.Ops, m.HoverColor, clip.Rect{
							Max: image.Pt(dims.Size.X, dims.Size.Y),
						}.Op())
						recording.Add(gtx.Ops)
					}
					if item.Doer.Enabled() {
						rect := image.Rect(0, 0, dims.Size.X, dims.Size.Y)
						area := clip.Rect(rect).Push(gtx.Ops)
						event.Op(gtx.Ops, &m.Menu.tags[i])
						area.Pop()
					}
					return dims
				})
			}),
			layout.Expanded(func(gtx C) D {
				return m.Menu.scrollBar.Layout(gtx, &m.Theme.ScrollBar, len(items), &m.Menu.list.Position)
			}),
		)
	}
	popup := Popup(m.Theme, &m.Menu.Visible)
	popup.NE = unit.Dp(0)
	popup.ShadowN = unit.Dp(0)
	popup.NW = unit.Dp(0)
	return popup.Layout(gtx, contents)
}

func PopupMenu(th *Theme, s *LabelStyle, menu *Menu) MenuStyle {
	return MenuStyle{
		Menu:          menu,
		ShortCutColor: th.Menu.ShortCut,
		LabelStyle:    *s,
		HoverColor:    th.Menu.Hover,
		Disabled:      th.Menu.Disabled,
		Theme:         th,
	}
}

func (tr *Tracker) layoutMenu(gtx C, title string, clickable *Clickable, menu *Menu, width unit.Dp, items ...MenuItem) layout.Widget {
	for clickable.Clicked(gtx) {
		menu.Visible = true
	}
	m := PopupMenu(tr.Theme, &tr.Theme.Menu.Text, menu)
	return func(gtx C) D {
		defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
		btn := Btn(tr.Theme, &tr.Theme.Button.Menu, clickable, title, "")
		dims := btn.Layout(gtx)
		op.Offset(image.Pt(0, dims.Size.Y)).Add(gtx.Ops)
		gtx.Constraints.Max.X = gtx.Dp(width)
		gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(300))
		m.Layout(gtx, items...)
		return dims
	}
}
