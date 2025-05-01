package gioui

import (
	"image/color"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type (
	// Editor wraps a widget.Editor and adds some additional key event filters,
	// to prevent key presses from flowing through to the rest of the
	// application while editing (particularly: to prevent triggering notes
	// while editing).
	Editor struct {
		Editor       widget.Editor
		filters      []event.Filter
		requestFocus bool
	}

	EditorStyle struct {
		Color     color.NRGBA
		HintColor color.NRGBA
		Font      font.Font
		TextSize  unit.Sp
	}
)

func NewEditor(e widget.Editor) *Editor {
	ret := &Editor{
		Editor: e,
	}
	for c := 'A'; c <= 'Z'; c++ {
		ret.filters = append(ret.filters, key.Filter{Name: key.Name(c), Focus: &ret.Editor})
	}
	for c := '0'; c <= '9'; c++ {
		ret.filters = append(ret.filters, key.Filter{Name: key.Name(c), Focus: &ret.Editor})
	}
	ret.filters = append(ret.filters, key.Filter{Name: key.NameSpace, Focus: &ret.Editor})
	ret.filters = append(ret.filters, key.Filter{Name: key.NameEscape, Focus: &ret.Editor})
	return ret
}

func (s *EditorStyle) AsLabelStyle() LabelStyle {
	return LabelStyle{
		Color:    s.Color,
		Font:     s.Font,
		TextSize: s.TextSize,
	}
}

func MaterialEditor(th *Theme, style *EditorStyle, editor *Editor, hint string) material.EditorStyle {
	ret := material.Editor(&th.Material, &editor.Editor, hint)
	ret.Font = style.Font
	ret.TextSize = style.TextSize
	ret.Color = style.Color
	ret.HintColor = style.HintColor
	return ret
}

func (e *Editor) SetText(s string) {
	if e.Editor.Text() != s {
		e.Editor.SetText(s)
	}
}

func (e *Editor) Text() string {
	return e.Editor.Text()
}

func (e *Editor) Submitted(gtx C) bool {
	for {
		ev, ok := e.Editor.Update(gtx)
		if !ok {
			break
		}
		_, ok = ev.(widget.SubmitEvent)
		if ok {
			return true
		}
	}
	return false
}

func (e *Editor) Cancelled(gtx C) bool {
	for {
		event, ok := gtx.Event(e.filters...)
		if !ok {
			break
		}
		if e, ok := event.(key.Event); ok && e.State == key.Press && e.Name == key.NameEscape {
			return true
		}
	}
	return false
}

func (e *Editor) Focus() {
	e.requestFocus = true
}
