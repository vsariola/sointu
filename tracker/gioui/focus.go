package gioui

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
)

type TagYieldFunc func(level int, tag event.Tag) bool

func (t *Tracker) FocusNext(gtx C, maxLevel int) {
	var focused, first, next event.Tag
	yield := func(level int, tag event.Tag) bool {
		if first == nil {
			first = tag // remember the first tag
		}
		if focused != nil && level <= maxLevel {
			next = tag
			return false // we're done
		}
		if gtx.Source.Focused(tag) {
			focused = tag
		}
		return true
	}
	if t.Tags(0, yield) {
		t.Tags(0, yield) // run it twice to ensure we find the next tag after the focused one
	}
	if next == nil {
		next = first // if we didn't find a next tag, use the first one
	}
	if next != nil {
		gtx.Execute(key.FocusCmd{Tag: next})
	}
}

func (t *Tracker) FocusPrev(gtx C, maxLevel int) {
	var prev, first event.Tag
	yield := func(level int, tag event.Tag) bool {
		if first == nil {
			first = tag // remember the first tag
		}
		if gtx.Source.Focused(tag) {
			if prev != nil {
				return false // we're done
			}
		} else if level <= maxLevel {
			prev = tag
		}
		return true
	}
	if t.Tags(0, yield) {
		t.Tags(0, yield) // run it twice to ensure we find the previous tag before the focused one
	}
	if prev == nil {
		prev = first // if we didn't find a next tag, use the first one
	}
	if prev != nil {
		gtx.Execute(key.FocusCmd{Tag: prev})
	}
}
