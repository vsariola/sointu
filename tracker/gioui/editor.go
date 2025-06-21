package gioui

import (
	"image/color"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/vsariola/sointu/tracker"
)

type (
	// Editor wraps a widget.Editor and adds some additional key event filters,
	// to prevent key presses from flowing through to the rest of the
	// application while editing (particularly: to prevent triggering notes
	// while editing).
	Editor struct {
		widgetEditor widget.Editor
		filters      []event.Filter
		requestFocus bool
	}

	EditorStyle struct {
		Color     color.NRGBA
		HintColor color.NRGBA
		Font      font.Font
		TextSize  unit.Sp
	}

	EditorEvent int
)

const (
	EditorEventNone EditorEvent = iota
	EditorEventSubmit
	EditorEventCancel
)

func NewEditor(singleLine, submit bool, alignment text.Alignment) *Editor {
	ret := &Editor{widgetEditor: widget.Editor{SingleLine: singleLine, Submit: submit, Alignment: alignment}}
	for c := 'A'; c <= 'Z'; c++ {
		ret.filters = append(ret.filters, key.Filter{Name: key.Name(c), Focus: &ret.widgetEditor, Optional: key.ModAlt | key.ModShift | key.ModShortcut})
	}
	for c := '0'; c <= '9'; c++ {
		ret.filters = append(ret.filters, key.Filter{Name: key.Name(c), Focus: &ret.widgetEditor, Optional: key.ModAlt | key.ModShift | key.ModShortcut})
	}
	ret.filters = append(ret.filters, key.Filter{Name: key.NameSpace, Focus: &ret.widgetEditor, Optional: key.ModAlt | key.ModShift | key.ModShortcut})
	ret.filters = append(ret.filters, key.Filter{Name: key.NameEscape, Focus: &ret.widgetEditor, Optional: key.ModAlt | key.ModShift | key.ModShortcut})
	return ret
}

func (s *EditorStyle) AsLabelStyle() LabelStyle {
	return LabelStyle{
		Color:    s.Color,
		Font:     s.Font,
		TextSize: s.TextSize,
	}
}

func (e *Editor) Layout(gtx C, str tracker.String, th *Theme, style *EditorStyle, hint string) D {
	for e.Update(gtx, str) != EditorEventNone {
		// just consume all events if the user did not consume them
	}
	if e.widgetEditor.Text() != str.Value() {
		e.widgetEditor.SetText(str.Value())
	}
	me := material.Editor(&th.Material, &e.widgetEditor, hint)
	me.Font = style.Font
	me.TextSize = style.TextSize
	me.Color = style.Color
	me.HintColor = style.HintColor
	return me.Layout(gtx)
}

func (e *Editor) Update(gtx C, str tracker.String) EditorEvent {
	if e.requestFocus {
		e.requestFocus = false
		gtx.Execute(key.FocusCmd{Tag: &e.widgetEditor})
		l := len(e.widgetEditor.Text())
		e.widgetEditor.SetCaret(l, l)
	}
	for {
		ev, ok := e.widgetEditor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.ChangeEvent); ok {
			str.SetValue(e.widgetEditor.Text())
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			return EditorEventSubmit
		}
	}
	for {
		event, ok := gtx.Event(e.filters...)
		if !ok {
			break
		}
		if e, ok := event.(key.Event); ok && e.State == key.Press && e.Name == key.NameEscape {
			return EditorEventCancel
		}
	}
	return EditorEventNone
}

func (e *Editor) Focus() {
	e.requestFocus = true
}
