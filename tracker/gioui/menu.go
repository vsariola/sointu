package gioui

import (
	"image"
	"image/color"

	"gioui.org/io/event"
	"gioui.org/io/key"
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
		tags      []bool
		hover     int
		hoverOk   bool
		list      layout.List
		scrollBar ScrollBar

		tag     bool
		visible bool

		itemTmp []menuItem
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

	// MenuWidget has a Layout method to display a menu
	MenuWidget struct {
		State *MenuState
		Style *MenuStyle
	}
)

func Menu(state *MenuState) MenuWidget                     { return MenuWidget{State: state} }
func (w MenuWidget) WithStyle(style *MenuStyle) MenuWidget { w.Style = style; return w }

func (ms *MenuState) Tags(level int, yield TagYieldFunc) bool {
	if ms.visible {
		return yield(level, &ms.tag)
	}
	return true
}

// MenuChild describes one or more menu items; if MenuChild is an Action or
// Bool, it's one item per child, but Ints are treated as enumerations and
// create one item per different possible values of the int.
type MenuChild struct {
	Icon     []byte
	Text     string
	Shortcut string

	kind   menuChildKind
	action tracker.Action
	bool   tracker.Bool
	int    tracker.Int
	widget layout.Widget // these should be passive separators and such
}

type menuChildKind int

const (
	menuChildAction menuChildKind = iota
	menuChildBool
	menuChildInt
	menuChildList
	menuChildWidget
)

func ActionMenuChild(act tracker.Action, text, shortcut string, icon []byte) MenuChild {
	return MenuChild{
		Icon:     icon,
		Text:     text,
		Shortcut: shortcut,

		kind:   menuChildAction,
		action: act,
	}
}

func BoolMenuChild(b tracker.Bool, text, shortcut string, icon []byte) MenuChild {
	return MenuChild{
		Icon:     icon,
		Text:     text,
		Shortcut: shortcut,

		kind: menuChildBool,
		bool: b,
	}
}

func IntMenuChild(i tracker.Int, text, shortcut string, icon []byte) MenuChild {
	return MenuChild{
		Icon:     icon,
		Text:     text,
		Shortcut: shortcut,
		kind:     menuChildInt,
		int:      i,
	}
}

// Layout the menu with the given items
func (m MenuWidget) Layout(gtx C, children ...MenuChild) D {
	t := TrackerFromContext(gtx)
	if m.Style == nil {
		m.Style = &t.Theme.Menu.Main
	}
	// unfortunately, there was no way to include items into the MenuWidget
	// without causing heap escapes, so they are passed as a parameter to the Layout
	m.State.itemTmp = m.State.itemTmp[:0]
	for i, c := range children {
		switch c.kind {
		case menuChildAction:
			m.State.itemTmp = append(m.State.itemTmp, menuItem{childIndex: i, icon: c.Icon, text: c.Text, shortcut: c.Shortcut, enabled: c.enabled()})
		case menuChildBool:
			mi := menuItem{childIndex: i, text: c.Text, shortcut: c.Shortcut, enabled: c.enabled()}
			if c.bool.Value() {
				mi.icon = c.Icon
			}
			m.State.itemTmp = append(m.State.itemTmp, mi)
		case menuChildInt:
			for i := c.int.Range().Min; i <= c.int.Range().Max; i++ {
				mi := menuItem{childIndex: i, text: c.int.StringOf(i), value: i, enabled: c.enabled()}
				if c.int.Value() == i {
					mi.icon = c.Icon
				}
				if i == c.int.Range().Min {
					mi.shortcut = c.Shortcut
				}
				m.State.itemTmp = append(m.State.itemTmp, mi)
			}
		}
	}
	m.update(gtx, children, m.State.itemTmp)
	listItem := func(gtx C, i int) D {
		item := m.State.itemTmp[i]
		icon := t.Theme.Icon(item.icon)
		iconColor := m.Style.Text.Color
		iconInset := layout.Inset{Left: unit.Dp(12), Right: unit.Dp(6)}
		textLabel := Label(t.Theme, &m.Style.Text, item.text)
		shortcutLabel := Label(t.Theme, &m.Style.Shortcut, item.shortcut)
		if !item.enabled {
			iconColor = m.Style.Disabled
			textLabel.Color = m.Style.Disabled
			shortcutLabel.Color = m.Style.Disabled
		}
		shortcutInset := layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12), Bottom: unit.Dp(2), Top: unit.Dp(2)}
		fg := func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
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
		}
		bg := func(gtx C) D {
			rect := clip.Rect{Max: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}
			if item.enabled && m.State.hoverOk && m.State.hover == i {
				paint.FillShape(gtx.Ops, m.Style.Hover, rect.Op())
			}
			if item.enabled {
				area := rect.Push(gtx.Ops)
				event.Op(gtx.Ops, &m.State.tags[i])
				area.Pop()
			}
			return D{Size: rect.Max}
		}
		return layout.Background{}.Layout(gtx, bg, fg)
	}
	menuList := func(gtx C) D {
		gtx.Constraints.Max.X = gtx.Dp(m.Style.Width)
		gtx.Constraints.Max.Y = gtx.Dp(m.Style.Height)
		r := clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops)
		event.Op(gtx.Ops, &m.State.tag)
		r.Pop()
		m.State.list.Axis = layout.Vertical
		m.State.scrollBar.Axis = layout.Vertical
		return layout.Stack{Alignment: layout.SE}.Layout(gtx,
			layout.Expanded(func(gtx C) D { return m.State.list.Layout(gtx, len(m.State.itemTmp), listItem) }),
			layout.Expanded(func(gtx C) D {
				return m.State.scrollBar.Layout(gtx, &t.Theme.ScrollBar, len(m.State.itemTmp), &m.State.list.Position)
			}),
		)
	}
	popup := Popup(t.Theme, &m.State.visible)
	popup.Style = &t.Theme.Popup.Menu
	return popup.Layout(gtx, menuList)
}

type menuItem struct {
	childIndex     int
	value          int
	icon           []byte
	text, shortcut string
	enabled        bool
}

func (m *MenuWidget) update(gtx C, children []MenuChild, items []menuItem) {
	// handle keyboard events for the menu
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: &m.State.tag},
			key.Filter{Focus: &m.State.tag, Name: key.NameUpArrow},
			key.Filter{Focus: &m.State.tag, Name: key.NameDownArrow},
			key.Filter{Focus: &m.State.tag, Name: key.NameEnter},
			key.Filter{Focus: &m.State.tag, Name: key.NameReturn},
		)
		if !ok {
			break
		}
		switch e := ev.(type) {
		case key.Event:
			if e.State != key.Press {
				continue
			}
			switch e.Name {
			case key.NameUpArrow:
				if !m.State.hoverOk {
					m.State.hover = 0 // if nothing is selected, select the first item before starting to move backwards
				}
				for i := 1; i < len(items); i++ {
					idx := (m.State.hover - i + len(items)) % len(items)
					child := &children[items[idx].childIndex]
					if child.enabled() {
						m.State.hover = idx
						m.State.hoverOk = true
						break
					}
				}
			case key.NameDownArrow:
				if !m.State.hoverOk {
					m.State.hover = len(items) - 1 // if nothing is selected, select the last item before starting to move backwards
				}
				for i := 1; i < len(items); i++ {
					idx := (m.State.hover + i) % len(items)
					child := &children[items[idx].childIndex]
					if child.enabled() {
						m.State.hover = idx
						m.State.hoverOk = true
						break
					}
				}
			case key.NameEnter, key.NameReturn:
				if m.State.hoverOk && m.State.hover >= 0 && m.State.hover < len(items) {
					m.activateItem(items[m.State.hover], children)
				}
			}
		case key.FocusEvent:
			if !m.State.hoverOk {
				m.State.hover = 0
			}
			m.State.hoverOk = e.Focus
		}
	}
	for i := range items {
		// make sure we have a tag for every item
		for len(m.State.tags) <= i {
			m.State.tags = append(m.State.tags, false)
		}
		// handle pointer events for this item
		for {
			ev, ok := gtx.Event(pointer.Filter{Target: &m.State.tags[i], Kinds: pointer.Press | pointer.Enter | pointer.Leave})
			if !ok {
				break
			}
			e, ok := ev.(pointer.Event)
			if !ok {
				continue
			}
			switch e.Kind {
			case pointer.Press:
				m.activateItem(items[i], children)
			case pointer.Enter:
				m.State.hover = i
				m.State.hoverOk = true
				if !gtx.Focused(&m.State.tag) {
					gtx.Execute(key.FocusCmd{Tag: &m.State.tag})
				}
			case pointer.Leave:
				if m.State.hover == i {
					m.State.hoverOk = false
				}
			}
		}
	}
}

func (m *MenuWidget) activateItem(item menuItem, children []MenuChild) {
	if item.childIndex < 0 || item.childIndex >= len(children) {
		return
	}
	child := &children[item.childIndex]
	if !child.enabled() {
		return
	}
	switch child.kind {
	case menuChildAction:
		child.action.Do()
	case menuChildBool:
		child.bool.Toggle()
	case menuChildInt:
		child.int.SetValue(item.value)
	}
	m.State.visible = false
}

func (c *MenuChild) enabled() bool {
	switch c.kind {
	case menuChildAction:
		return c.action.Enabled()
	case menuChildBool:
		return c.bool.Enabled()
	case menuChildWidget:
		return false // the widget are passive separators and such
	default:
		return true
	}
}

// MenuButton displays a button with text that opens a menu when clicked.
type MenuButton struct {
	Title     string
	Style     *ButtonStyle
	Clickable *Clickable
	MenuState *MenuState
	Width     unit.Dp
}

func MenuBtn(ms *MenuState, cl *Clickable, title string) MenuButton {
	return MenuButton{MenuState: ms, Clickable: cl, Title: title}
}

func (mb MenuButton) WithStyle(style *ButtonStyle) MenuButton { mb.Style = style; return mb }

func (mb MenuButton) Layout(gtx C, children ...MenuChild) D {
	for mb.Clickable.Clicked(gtx) {
		mb.MenuState.visible = true
		gtx.Execute(key.FocusCmd{Tag: &mb.MenuState.tag})
	}
	t := TrackerFromContext(gtx)
	if mb.Style == nil {
		mb.Style = &t.Theme.Button.Menu
	}
	btn := Btn(t.Theme, mb.Style, mb.Clickable, mb.Title, "")
	dims := btn.Layout(gtx)
	if mb.MenuState.visible {
		defer op.Offset(image.Pt(0, dims.Size.Y)).Push(gtx.Ops).Pop()
		m := Menu(mb.MenuState)
		m.Layout(gtx, children...)
	}
	return dims
}
