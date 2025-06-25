package gioui

import (
	"math"

	"gioui.org/io/event"
	"gioui.org/io/key"
)

type TagYieldFunc func(level int, tag event.Tag) bool

// FocusNext navigates to the next focusable tag in the tracker. If stepInto is
// true, it will focus the next tag regardless of its depth; otherwise it will
// focus the next tag at the current level or shallower.
func (t *Tracker) FocusNext(gtx C, stepInto bool) {
	_, next := t.findPrevNext(gtx, stepInto)
	if next != nil {
		gtx.Execute(key.FocusCmd{Tag: next})
	}
}

// FocusPrev navigates to the previous focusable tag in the tracker. If stepInto
// is true, it will focus the previous tag regardless of its depth; otherwise it
// will focus the previous tag at the current level or shallower.
func (t *Tracker) FocusPrev(gtx C, stepInto bool) {
	prev, _ := t.findPrevNext(gtx, stepInto)
	if prev != nil {
		gtx.Execute(key.FocusCmd{Tag: prev})
	}
}

func (t *Tracker) findPrevNext(gtx C, stepInto bool) (prev, next event.Tag) {
	var first, last event.Tag
	found := false
	maxLevel := math.MaxInt
	if !stepInto {
		if level, ok := t.findFocusedLevel(gtx); ok {
			maxLevel = level // limit to the current focused tag's level
		}
	}
	t.Tags(0, func(l int, t event.Tag) bool {
		if l > maxLevel || t == nil {
			return true // skip tags that are too deep or nils
		}
		if first == nil {
			first = t
		}
		if found && next == nil {
			next = t
		}
		if gtx.Focused(t) {
			found = true
		}
		if !found {
			prev = t
		}
		last = t
		return true
	})
	if next == nil {
		next = first
	}
	if prev == nil {
		prev = last
	}
	return prev, next
}

func (t *Tracker) findFocusedLevel(gtx C) (level int, ok bool) {
	t.Tags(0, func(l int, t event.Tag) bool {
		if gtx.Focused(t) {
			level = l
			ok = true
			return false // stop when we find the focused tag
		}
		return true // continue searching
	})
	return level, ok
}
