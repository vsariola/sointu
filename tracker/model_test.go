package tracker_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/vsariola/sointu/tracker"
	"github.com/vsariola/sointu/vm"
)

type NullContext struct{}

func (NullContext) NextEvent() (event tracker.MIDINoteEvent, ok bool) {
	return tracker.MIDINoteEvent{}, false
}

func (NullContext) BPM() (bpm float64, ok bool) {
	return 0, false
}

type modelFuzzState struct {
	model     *tracker.Model
	clipboard []byte
}

func (s *modelFuzzState) Iterate(yield func(string, func(p string, t *testing.T)) bool, seed int) {
	// Ints
	s.IterateInt("InstrumentVoices", s.model.InstrumentVoices().Int(), yield, seed)
	s.IterateInt("TrackVoices", s.model.TrackVoices().Int(), yield, seed)
	s.IterateInt("SongLength", s.model.SongLength().Int(), yield, seed)
	s.IterateInt("BPM", s.model.BPM().Int(), yield, seed)
	s.IterateInt("RowsPerPattern", s.model.RowsPerPattern().Int(), yield, seed)
	s.IterateInt("RowsPerBeat", s.model.RowsPerBeat().Int(), yield, seed)
	s.IterateInt("Step", s.model.Step().Int(), yield, seed)
	s.IterateInt("Octave", s.model.Octave().Int(), yield, seed)
	// Lists
	s.IterateList("Instruments", s.model.Instruments().List(), yield, seed)
	s.IterateList("Units", s.model.Units().List(), yield, seed)
	s.IterateList("Tracks", s.model.Tracks().List(), yield, seed)
	s.IterateList("OrderRows", s.model.OrderRows().List(), yield, seed)
	s.IterateList("NoteRows", s.model.NoteRows().List(), yield, seed)
	s.IterateList("UnitSearchResults", s.model.SearchResults().List(), yield, seed)
	s.IterateBool("Panic", s.model.Panic().Bool(), yield, seed)
	s.IterateBool("Recording", s.model.IsRecording().Bool(), yield, seed)
	s.IterateBool("Playing", s.model.Playing().Bool(), yield, seed)
	s.IterateBool("InstrEnlarged", s.model.InstrEnlarged().Bool(), yield, seed)
	s.IterateBool("Effect", s.model.Effect().Bool(), yield, seed)
	s.IterateBool("CommentExpanded", s.model.CommentExpanded().Bool(), yield, seed)
	s.IterateBool("NoteTracking", s.model.NoteTracking().Bool(), yield, seed)
	// Strings
	s.IterateString("FilePath", s.model.FilePath().String(), yield, seed)
	s.IterateString("InstrumentName", s.model.InstrumentName().String(), yield, seed)
	s.IterateString("InstrumentComment", s.model.InstrumentComment().String(), yield, seed)
	s.IterateString("UnitSearchText", s.model.UnitSearch().String(), yield, seed)
	// Actions
	s.IterateAction("AddTrack", s.model.AddTrack(), yield, seed)
	s.IterateAction("DeleteTrack", s.model.DeleteTrack(), yield, seed)
	s.IterateAction("AddInstrument", s.model.AddInstrument(), yield, seed)
	s.IterateAction("DeleteInstrument", s.model.DeleteInstrument(), yield, seed)
	s.IterateAction("AddUnitAfter", s.model.AddUnit(false), yield, seed)
	s.IterateAction("AddUnitBefore", s.model.AddUnit(true), yield, seed)
	s.IterateAction("DeleteUnit", s.model.DeleteUnit(), yield, seed)
	s.IterateAction("ClearUnit", s.model.ClearUnit(), yield, seed)
	s.IterateAction("Undo", s.model.Undo(), yield, seed)
	s.IterateAction("Redo", s.model.Redo(), yield, seed)
	s.IterateAction("RemoveUnused", s.model.RemoveUnused(), yield, seed)
	s.IterateAction("AddSemitone", s.model.AddSemitone(), yield, seed)
	s.IterateAction("SubtractSemitone", s.model.SubtractSemitone(), yield, seed)
	s.IterateAction("AddOctave", s.model.AddOctave(), yield, seed)
	s.IterateAction("SubtractOctave", s.model.SubtractOctave(), yield, seed)
	s.IterateAction("EditNoteOff", s.model.EditNoteOff(), yield, seed)
	s.IterateAction("Rewind", s.model.Rewind(), yield, seed)
	s.IterateAction("AddOrderRowAfter", s.model.AddOrderRow(false), yield, seed)
	s.IterateAction("AddOrderRowBefore", s.model.AddOrderRow(true), yield, seed)
	s.IterateAction("DeleteOrderRowForward", s.model.DeleteOrderRow(false), yield, seed)
	s.IterateAction("DeleteOrderRowBackward", s.model.DeleteOrderRow(true), yield, seed)
	// Tables
	s.IterateTable("Order", s.model.Order().Table(), yield, seed)
	s.IterateTable("Notes", s.model.Notes().Table(), yield, seed)
}

func (s *modelFuzzState) IterateInt(name string, i tracker.Int, yield func(string, func(p string, t *testing.T)) bool, seed int) {
	r := i.Range()
	yield(name+".Set", func(p string, t *testing.T) {
		i.Set(seed%(r.Max-r.Min+10) - 5 + r.Min)
	})
	yield(name+".Value", func(p string, t *testing.T) {
		if v := i.Value(); v < r.Min || v > r.Max {
			r := i.Range()
			t.Errorf("Path: %s %s value out of range [%d,%d]: %d", p, name, r.Min, r.Max, v)
		}
	})
}

func (s *modelFuzzState) IterateAction(name string, a tracker.Action, yield func(string, func(p string, t *testing.T)) bool, seed int) {
	yield(name+".Do", func(p string, t *testing.T) {
		a.Do()
	})
}

func (s *modelFuzzState) IterateBool(name string, b tracker.Bool, yield func(string, func(p string, t *testing.T)) bool, seed int) {
	yield(name+".Set", func(p string, t *testing.T) {
		b.Set(seed%2 == 0)
	})
	yield(name+".Toggle", func(p string, t *testing.T) {
		b.Toggle()
	})
}

func (s *modelFuzzState) IterateString(name string, str tracker.String, yield func(string, func(p string, t *testing.T)) bool, seed int) {
	yield(name+".Set", func(p string, t *testing.T) {
		str.Set(fmt.Sprintf("%d", seed))
	})
}

func (s *modelFuzzState) IterateList(name string, l tracker.List, yield func(string, func(p string, t *testing.T)) bool, seed int) {
	yield(name+".SetSelected", func(p string, t *testing.T) {
		l.SetSelected(seed%50 - 16)
	})
	yield(name+".Count", func(p string, t *testing.T) {
		if c := l.Count(); c > 0 {
			if l.Selected() < 0 || l.Selected() >= c {
				t.Errorf("Path: %s %s selected out of range: %d", p, name, l.Selected())
			}
		} else {
			if l.Selected() != 0 {
				t.Errorf("Path: %s %s selected out of range: %d", p, name, l.Selected())
			}
		}
	})
	yield(name+".SetSelected2", func(p string, t *testing.T) {
		l.SetSelected2(seed%50 - 16)
	})
	yield(name+".Count2", func(p string, t *testing.T) {
		if c := l.Count(); c > 0 {
			if l.Selected2() < 0 || l.Selected2() >= c {
				t.Errorf("Path: %s List selected2 out of range: %d", p, l.Selected2())
			}
		} else {
			if l.Selected2() != 0 {
				t.Errorf("Path: %s List selected2 out of range: %d", p, l.Selected2())
			}
		}
	})
	yield(name+".MoveElements", func(p string, t *testing.T) {
		l.MoveElements(seed%2*2 - 1)
	})
	yield(name+".DeleteElementsForward", func(p string, t *testing.T) {
		l.DeleteElements(false)
	})
	yield(name+".DeleteElementsBackward", func(p string, t *testing.T) {
		l.DeleteElements(true)
	})
	yield(name+".CopyElements", func(p string, t *testing.T) {
		s.clipboard, _ = l.CopyElements()
	})
	yield(name+".PasteElements", func(p string, t *testing.T) {
		l.PasteElements(s.clipboard)
	})
}

func (s *modelFuzzState) IterateTable(name string, table tracker.Table, yield func(string, func(p string, t *testing.T)) bool, seed int) {
	yield(name+".SetCursor", func(p string, t *testing.T) {
		table.SetCursor(tracker.Point{seed % 16, seed * 1337 % 16})
	})
	yield(name+".SetCursor2", func(p string, t *testing.T) {
		table.SetCursor2(tracker.Point{seed % 16, seed * 1337 % 16})
	})
	yield(name+".Cursor", func(p string, t *testing.T) {
		if c := table.Cursor(); c.X < 0 || (c.X >= table.Width() && table.Width() > 0) || c.Y < 0 || (c.Y >= table.Height() && table.Height() > 0) {
			t.Errorf("Path: %s Table cursor out of range: %v", p, c)
		}
	})
	yield(name+".Cursor2", func(p string, t *testing.T) {
		if c := table.Cursor2(); c.X < 0 || (c.X >= table.Width() && table.Width() > 0) || c.Y < 0 || (c.Y >= table.Height() && table.Height() > 0) {
			t.Errorf("Path: %s Table cursor2 out of range: %v", p, c)
		}
	})
	yield(name+".SetCursorX", func(p string, t *testing.T) {
		table.SetCursorX(seed % 16)
	})
	yield(name+".SetCursorY", func(p string, t *testing.T) {
		table.SetCursorY(seed % 16)
	})
	yield(name+".MoveCursor", func(p string, t *testing.T) {
		table.MoveCursor(seed%2*2-1, seed%2*2-1)
	})
	yield(name+".Copy", func(p string, t *testing.T) {
		s.clipboard, _ = table.Copy()
	})
	yield(name+".Paste", func(p string, t *testing.T) {
		table.Paste(s.clipboard)
	})
	yield(name+".Clear", func(p string, t *testing.T) {
		table.Clear()
	})
	yield(name+".Fill", func(p string, t *testing.T) {
		table.Fill(seed % 16)
	})
	yield(name+".Add", func(p string, t *testing.T) {
		table.Add(seed % 16)
	})
}

func FuzzModel(f *testing.F) {
	seed := make([]byte, 1)
	for i := range seed {
		seed[i] = byte(i)
	}
	f.Add(seed)
	f.Fuzz(func(t *testing.T, slice []byte) {
		reader := bytes.NewReader(slice)
		synther := vm.GoSynther{}
		model, player := tracker.NewModelPlayer(synther, "")
		buf := make([][2]float32, 2048)
		closeChan := make(chan struct{})
		go func() {
		loop:
			for {
				select {
				case <-closeChan:
					break loop
				default:
					ctx := NullContext{}
					player.Process(buf, ctx)
				}
			}
		}()
		state := modelFuzzState{model: model}
		count := 0
		state.Iterate(func(n string, f func(p string, t *testing.T)) bool {
			count++
			return true
		}, 0)
		totalPath := ""
		for m, err := binary.ReadVarint(reader); err == nil; m, err = binary.ReadVarint(reader) {
			seed := int(m)
			index := seed % count
			state.Iterate(func(n string, f func(p string, t *testing.T)) bool {
				if index == 0 {
					totalPath += n + ". "
					f(totalPath, t)
				}
				index--
				return index > 0
			}, seed)
		}
		closeChan <- struct{}{}
	})
}
