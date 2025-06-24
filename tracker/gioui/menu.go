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

type (
	// MenuState is the part of the menu that needs to be retained between frames.
	MenuState struct {
		visible   bool
		tags      []bool
		hover     int
		list      layout.List
		scrollBar ScrollBar
	}

	// MenuStyle is the style for a menu that is stored in the theme.yml.
	MenuStyle struct {
		Text     LabelStyle
		Shortcut LabelStyle
		Disabled color.NRGBA
		Hover    color.NRGBA
		Width    unit.Dp
		Height   unit.Dp
	}

	// ActionMenuItem is a menu item that has an icon, text, shortcut and an action.
	ActionMenuItem struct {
		Icon     []byte
		Text     string
		Shortcut string
		Action   tracker.Action
	}

	// MenuWidget has a Layout method to display a menu
	MenuWidget struct {
		Theme *Theme
		State *MenuState
		Style *MenuStyle
	}

	// MenuButton displayes a button with text that opens a menu when clicked.
	MenuButton struct {
		Theme     *Theme
		Title     string
		Style     *ButtonStyle
		Clickable *Clickable
		MenuState *MenuState
		Width     unit.Dp
	}
)

func Menu(th *Theme, state *MenuState) MenuWidget {
	return MenuWidget{
		Theme: th,
		State: state,
		Style: &th.Menu.Main,
	}
}

func MenuItem(act tracker.Action, text, shortcut string, icon []byte) ActionMenuItem {
	return ActionMenuItem{
		Icon:     icon,
		Text:     text,
		Shortcut: shortcut,
		Action:   act,
	}
}

func MenuBtn(th *Theme, ms *MenuState, cl *Clickable, title string) MenuButton {
	return MenuButton{
		Theme:     th,
		Title:     title,
		Clickable: cl,
		MenuState: ms,
		Style:     &th.Button.Menu,
	}
}

func (m *MenuWidget) Layout(gtx C, items ...ActionMenuItem) D {
	// unfortunately, there was no way to include items into the MenuWidget
	// without causing heap escapes, so they are passed as a parameter to the Layout
	m.update(gtx, items...)
	popup := Popup(m.Theme, &m.State.visible)
	popup.NE = unit.Dp(0)
	popup.ShadowN = unit.Dp(0)
	popup.NW = unit.Dp(0)
	return popup.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Max.X = gtx.Dp(m.Style.Width)
		gtx.Constraints.Max.Y = gtx.Dp(m.Style.Height)
		m.State.list.Axis = layout.Vertical
		m.State.scrollBar.Axis = layout.Vertical
		return layout.Stack{Alignment: layout.SE}.Layout(gtx,
			layout.Expanded(func(gtx C) D {
				return m.State.list.Layout(gtx, len(items), func(gtx C, i int) D {
					defer op.Offset(image.Point{}).Push(gtx.Ops).Pop()
					var macro op.MacroOp
					item := &items[i]
					if i == m.State.hover-1 && item.Action.Enabled() {
						macro = op.Record(gtx.Ops)
					}
					icon := m.Theme.Icon(item.Icon)
					iconColor := m.Style.Text.Color
					iconInset := layout.Inset{Left: unit.Dp(12), Right: unit.Dp(6)}
					textLabel := Label(m.Theme, &m.Style.Text, item.Text)
					shortcutLabel := Label(m.Theme, &m.Style.Shortcut, item.Shortcut)
					if !item.Action.Enabled() {
						iconColor = m.Style.Disabled
						textLabel.Color = m.Style.Disabled
						shortcutLabel.Color = m.Style.Disabled
					}
					shortcutInset := layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12), Bottom: unit.Dp(2), Top: unit.Dp(2)}
					dims := layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return iconInset.Layout(gtx, func(gtx C) D {
								p := gtx.Dp(unit.Dp(m.Style.Text.TextSize))
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
					if i == m.State.hover-1 && item.Action.Enabled() {
						recording := macro.Stop()
						paint.FillShape(gtx.Ops, m.Style.Hover, clip.Rect{
							Max: image.Pt(dims.Size.X, dims.Size.Y),
						}.Op())
						recording.Add(gtx.Ops)
					}
					if item.Action.Enabled() {
						rect := image.Rect(0, 0, dims.Size.X, dims.Size.Y)
						area := clip.Rect(rect).Push(gtx.Ops)
						event.Op(gtx.Ops, &m.State.tags[i])
						area.Pop()
					}
					return dims
				})
			}),
			layout.Expanded(func(gtx C) D {
				return m.State.scrollBar.Layout(gtx, &m.Theme.ScrollBar, len(items), &m.State.list.Position)
			}),
		)
	})
}

func (m *MenuWidget) update(gtx C, items ...ActionMenuItem) {
	for i, item := range items {
		// make sure we have a tag for every item
		for len(m.State.tags) <= i {
			m.State.tags = append(m.State.tags, false)
		}
		// handle pointer events for this item
		for {
			ev, ok := gtx.Event(pointer.Filter{
				Target: &m.State.tags[i],
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
				item.Action.Do()
				m.State.visible = false
			case pointer.Enter:
				m.State.hover = i + 1
			case pointer.Leave:
				if m.State.hover == i+1 {
					m.State.hover = 0
				}
			}
		}
	}
}

func (mb MenuButton) Layout(gtx C, items ...ActionMenuItem) D {
	for mb.Clickable.Clicked(gtx) {
		mb.MenuState.visible = true
	}
	btn := Btn(mb.Theme, mb.Style, mb.Clickable, mb.Title, "")
	dims := btn.Layout(gtx)
	defer op.Offset(image.Pt(0, dims.Size.Y)).Push(gtx.Ops).Pop()
	m := Menu(mb.Theme, mb.MenuState)
	m.Layout(gtx, items...)
	return dims
}
