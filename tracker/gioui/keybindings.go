package gioui

import (
	"bytes"
	_ "embed"
	"fmt"
	"strconv"
	"strings"

	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"github.com/vsariola/sointu/tracker"
	"gopkg.in/yaml.v3"
)

type (
	KeyAction string

	KeyBinding struct {
		Key                                        string
		Shortcut, Ctrl, Command, Shift, Alt, Super bool
		Action                                     string
	}
)

var keyBindingMap = map[key.Event]string{}
var keyActionMap = map[KeyAction]string{} // holds an informative string of the first key bound to an action

//go:embed keybindings.yml
var defaultKeyBindings []byte

func init() {
	var keyBindings, userKeybindings []KeyBinding
	dec := yaml.NewDecoder(bytes.NewReader(defaultKeyBindings))
	dec.KnownFields(true)
	if err := dec.Decode(&keyBindings); err != nil {
		panic(fmt.Errorf("failed to unmarshal default keybindings: %w", err))
	}
	if err := ReadCustomConfig("keybindings.yml", &userKeybindings); err == nil {
		keyBindings = append(keyBindings, userKeybindings...)
	}

	for _, kb := range keyBindings {
		var mods key.Modifiers
		if kb.Shortcut {
			mods |= key.ModShortcut
		}
		if kb.Ctrl {
			mods |= key.ModCtrl
		}
		if kb.Command {
			mods |= key.ModCommand
		}
		if kb.Shift {
			mods |= key.ModShift
		}
		if kb.Alt {
			mods |= key.ModAlt
		}
		if kb.Super {
			mods |= key.ModSuper
		}

		keyEvent := key.Event{Name: key.Name(kb.Key), Modifiers: mods, State: key.Press}
		action, ok := keyBindingMap[keyEvent] // if this key has been previously bound, remove it from the hint map
		if ok {
			delete(keyActionMap, KeyAction(action))
		}
		if kb.Action == "" { // unbind
			delete(keyBindingMap, keyEvent)
		} else { // bind
			keyBindingMap[keyEvent] = kb.Action
			// last binding of the some action wins for displaying the hint
			modString := strings.Replace(mods.String(), "-", "+", -1)
			text := kb.Key
			if modString != "" {
				text = modString + "+" + text
			}
			keyActionMap[KeyAction(kb.Action)] = text
		}
	}
}

func makeHint(hint, format, action string) string {
	if keyActionMap[KeyAction(action)] != "" {
		return hint + fmt.Sprintf(format, keyActionMap[KeyAction(action)])
	}
	return hint
}

// KeyEvent handles incoming key events and returns true if repaint is needed.
func (t *Tracker) KeyEvent(e key.Event, gtx C) {
	if e.State == key.Release {
		t.KeyNoteMap.Release(e.Name)
		return
	}
	action, ok := keyBindingMap[e]
	if !ok {
		return
	}
	switch action {
	// Actions
	case "AddTrack":
		t.Track().Add().Do()
	case "DeleteTrack":
		t.Track().Delete().Do()
	case "AddInstrument":
		t.Instrument().Add().Do()
	case "DeleteInstrument":
		t.Instrument().Delete().Do()
	case "AddUnitAfter":
		t.Unit().Add(false).Do()
	case "AddUnitBefore":
		t.Unit().Add(true).Do()
	case "DeleteUnit":
		t.Unit().Delete().Do()
	case "ClearUnit":
		t.Unit().Clear().Do()
	case "Undo":
		t.History().Undo().Do()
	case "Redo":
		t.History().Redo().Do()
	case "AddSemitone":
		t.Note().AddSemitone().Do()
	case "SubtractSemitone":
		t.Note().SubtractSemitone().Do()
	case "AddOctave":
		t.Note().AddOctave().Do()
	case "SubtractOctave":
		t.Note().SubtractOctave().Do()
	case "EditNoteOff":
		t.Note().NoteOff().Do()
	case "RemoveUnused":
		t.Order().RemoveUnusedPatterns().Do()
	case "PlayCurrentPosFollow":
		t.Play().IsFollowing().SetValue(true)
		t.Play().FromCurrentPos().Do()
	case "PlayCurrentPosUnfollow":
		t.Play().IsFollowing().SetValue(false)
		t.Play().FromCurrentPos().Do()
	case "PlaySongStartFollow":
		t.Play().IsFollowing().SetValue(true)
		t.Play().FromBeginning().Do()
	case "PlaySongStartUnfollow":
		t.Play().IsFollowing().SetValue(false)
		t.Play().FromBeginning().Do()
	case "PlaySelectedFollow":
		t.Play().IsFollowing().SetValue(true)
		t.Play().FromSelected().Do()
	case "PlaySelectedUnfollow":
		t.Play().IsFollowing().SetValue(false)
		t.Play().FromSelected().Do()
	case "PlayLoopFollow":
		t.Play().IsFollowing().SetValue(true)
		t.Play().FromLoopBeginning().Do()
	case "PlayLoopUnfollow":
		t.Play().IsFollowing().SetValue(false)
		t.Play().FromLoopBeginning().Do()
	case "StopPlaying":
		t.Play().Stop().Do()
	case "AddOrderRowBefore":
		t.Order().AddRow(true).Do()
	case "AddOrderRowAfter":
		t.Order().AddRow(false).Do()
	case "DeleteOrderRowBackwards":
		t.Order().DeleteRow(true).Do()
	case "DeleteOrderRowForwards":
		t.Order().DeleteRow(false).Do()
	case "NewSong":
		t.Song().New().Do()
	case "OpenSong":
		t.Song().Open().Do()
	case "Quit":
		if canQuit {
			t.RequestQuit().Do()
		}
	case "SaveSong":
		t.Song().Save().Do()
	case "SaveSongAs":
		t.Song().SaveAs().Do()
	case "ExportWav":
		t.Song().Export().Do()
	case "ExportFloat":
		t.Song().ExportFloat().Do()
	case "ExportInt16":
		t.Song().ExportInt16().Do()
	case "SplitTrack":
		t.Track().Split().Do()
	case "SplitInstrument":
		t.Instrument().Split().Do()
	case "ShowManual":
		t.ShowManual().Do()
	case "AskHelp":
		t.AskHelp().Do()
	case "ReportBug":
		t.ReportBug().Do()
	case "ShowLicense":
		t.ShowLicense().Do()
	// Booleans
	case "PanicToggle":
		t.Play().Panicked().Toggle()
	case "RecordingToggle":
		t.Play().IsRecording().Toggle()
	case "PlayingToggleFollow":
		t.Play().IsFollowing().SetValue(true)
		t.Play().Started().Toggle()
	case "PlayingToggleUnfollow":
		t.Play().IsFollowing().SetValue(false)
		t.Play().Started().Toggle()
	case "InstrEnlargedToggle":
		t.Play().TrackerHidden().Toggle()
	case "LinkInstrTrackToggle":
		t.Track().LinkInstrument().Toggle()
	case "FollowToggle":
		t.Play().IsFollowing().Toggle()
	case "UnitDisabledToggle":
		t.Unit().Disabled().Toggle()
	case "LoopToggle":
		t.Play().IsLooping().Toggle()
	case "UniquePatternsToggle":
		t.Note().UniquePatterns().Toggle()
	case "MuteToggle":
		t.Instrument().Mute().Toggle()
	case "SoloToggle":
		t.Instrument().Solo().Toggle()
	// Integers
	case "InstrumentVoicesAdd":
		t.Instrument().Voices().Add(1)
	case "InstrumentVoicesSubtract":
		t.Instrument().Voices().Add(-1)
	case "TrackVoicesAdd":
		t.Track().Voices().Add(1)
	case "TrackVoicesSubtract":
		t.Track().Voices().Add(-1)
	case "SongLengthAdd":
		t.Song().Length().Add(1)
	case "SongLengthSubtract":
		t.Song().Length().Add(-1)
	case "BPMAdd":
		t.Song().BPM().Add(1)
	case "BPMSubtract":
		t.Song().BPM().Add(-1)
	case "RowsPerPatternAdd":
		t.Song().RowsPerPattern().Add(1)
	case "RowsPerPatternSubtract":
		t.Song().RowsPerPattern().Add(-1)
	case "RowsPerBeatAdd":
		t.Song().RowsPerBeat().Add(1)
	case "RowsPerBeatSubtract":
		t.Song().RowsPerBeat().Add(-1)
	case "StepAdd":
		t.Note().Step().Add(1)
	case "StepSubtract":
		t.Note().Step().Add(-1)
	case "OctaveAdd":
		t.Note().Octave().Add(1)
	case "OctaveSubtract":
		t.Note().Octave().Add(-1)
	// Other miscellaneous
	case "Paste":
		gtx.Execute(clipboard.ReadCmd{Tag: t})
	case "OrderEditorFocus":
		t.Play().TrackerHidden().SetValue(false)
		gtx.Execute(key.FocusCmd{Tag: t.OrderEditor.scrollTable})
	case "TrackEditorFocus":
		t.Play().TrackerHidden().SetValue(false)
		gtx.Execute(key.FocusCmd{Tag: t.TrackEditor.scrollTable})
	case "InstrumentListFocus":
		gtx.Execute(key.FocusCmd{Tag: t.PatchPanel.instrList.instrumentDragList})
	case "UnitListFocus":
		var tag event.Tag
		t.PatchPanel.BottomTags(0, func(level int, t event.Tag) bool {
			tag = t
			return false
		})
		gtx.Execute(key.FocusCmd{Tag: tag})
	case "FocusPrev":
		t.FocusPrev(gtx, false)
	case "FocusPrevInto":
		t.FocusPrev(gtx, true)
	case "FocusNext":
		t.FocusNext(gtx, false)
	case "FocusNextInto":
		t.FocusNext(gtx, true)
	case "MIDIRefresh":
		t.MIDI().Refresh().Do()
	case "ToggleMIDIInputtingNotes":
		t.MIDI().InputtingNotes().Toggle()
	default:
		if len(action) > 4 && action[:4] == "Note" {
			val, err := strconv.Atoi(string(action[4:]))
			if err != nil {
				break
			}
			instr := t.Model.Instrument().List().Selected()
			n := noteAsValue(t.Model.Note().Octave().Value(), val-12)
			t.KeyNoteMap.Press(e.Name, tracker.NoteEvent{Channel: instr, Note: n})
		}
	}
}
