package gioui

import (
	"time"

	"github.com/vsariola/sointu/tracker"
)

type (
	// Keyboard is used to associate the keys of a keyboard (e.g. computer or a
	// MIDI keyboard) to currently playing notes. You can use any type T to
	// identify each key; T should be a comparable type.
	Keyboard[T comparable] struct {
		broker  *tracker.Broker
		pressed map[T]tracker.NoteEvent
	}
)

func MakeKeyboard[T comparable](broker *tracker.Broker) Keyboard[T] {
	return Keyboard[T]{
		broker:  broker,
		pressed: make(map[T]tracker.NoteEvent),
	}
}

func (t *Keyboard[T]) Press(key T, ev tracker.NoteEvent) {
	if _, ok := t.pressed[key]; ok {
		return // already playing a note with this key, do not send a new event
	}
	t.Release(key) // unset any previous note
	if ev.Note > 1 {
		ev.Source = t // set the source to this keyboard
		ev.On = true
		ev.Timestamp = t.now()
		if tracker.TrySend(t.broker.ToPlayer, any(&ev)) {
			t.pressed[key] = ev
		}
	}
}

func (t *Keyboard[T]) Release(key T) {
	if ev, ok := t.pressed[key]; ok {
		ev.Timestamp = t.now()
		ev.On = false // the pressed contains the event we need to send to release the note
		tracker.TrySend(t.broker.ToPlayer, any(&ev))
		delete(t.pressed, key)
	}
}

func (t *Keyboard[T]) ReleaseAll() {
	for key := range t.pressed {
		t.Release(key)
	}
}

func (t *Keyboard[T]) now() int64 {
	return time.Now().UnixMilli() * 441 / 10 // convert to 44100Hz frames
}
