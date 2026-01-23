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
		t.AddTrack().Do()
	case "DeleteTrack":
		t.DeleteTrack().Do()
	case "AddInstrument":
		t.AddInstrument().Do()
	case "DeleteInstrument":
		t.DeleteInstrument().Do()
	case "AddUnitAfter":
		t.AddUnit(false).Do()
	case "AddUnitBefore":
		t.AddUnit(true).Do()
	case "DeleteUnit":
		t.DeleteUnit().Do()
	case "ClearUnit":
		t.ClearUnit().Do()
	case "Undo":
		t.Undo().Do()
	case "Redo":
		t.Redo().Do()
	case "AddSemitone":
		t.AddSemitone().Do()
	case "SubtractSemitone":
		t.SubtractSemitone().Do()
	case "AddOctave":
		t.AddOctave().Do()
	case "SubtractOctave":
		t.SubtractOctave().Do()
	case "EditNoteOff":
		t.EditNoteOff().Do()
	case "RemoveUnused":
		t.RemoveUnused().Do()
	case "PlayCurrentPosFollow":
		t.Follow().SetValue(true)
		t.PlayCurrentPos().Do()
	case "PlayCurrentPosUnfollow":
		t.Follow().SetValue(false)
		t.PlayCurrentPos().Do()
	case "PlaySongStartFollow":
		t.Follow().SetValue(true)
		t.PlaySongStart().Do()
	case "PlaySongStartUnfollow":
		t.Follow().SetValue(false)
		t.PlaySongStart().Do()
	case "PlaySelectedFollow":
		t.Follow().SetValue(true)
		t.PlaySelected().Do()
	case "PlaySelectedUnfollow":
		t.Follow().SetValue(false)
		t.PlaySelected().Do()
	case "PlayLoopFollow":
		t.Follow().SetValue(true)
		t.PlayFromLoopStart().Do()
	case "PlayLoopUnfollow":
		t.Follow().SetValue(false)
		t.PlayFromLoopStart().Do()
	case "StopPlaying":
		t.StopPlaying().Do()
	case "AddOrderRowBefore":
		t.AddOrderRow(true).Do()
	case "AddOrderRowAfter":
		t.AddOrderRow(false).Do()
	case "DeleteOrderRowBackwards":
		t.DeleteOrderRow(true).Do()
	case "DeleteOrderRowForwards":
		t.DeleteOrderRow(false).Do()
	case "NewSong":
		t.NewSong().Do()
	case "OpenSong":
		t.OpenSong().Do()
	case "Quit":
		if canQuit {
			t.RequestQuit().Do()
		}
	case "SaveSong":
		t.SaveSong().Do()
	case "SaveSongAs":
		t.SaveSongAs().Do()
	case "ExportWav":
		t.Export().Do()
	case "ExportFloat":
		t.ExportFloat().Do()
	case "ExportInt16":
		t.ExportInt16().Do()
	case "SplitTrack":
		t.SplitTrack().Do()
	case "SplitInstrument":
		t.SplitInstrument().Do()
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
		t.Panic().Toggle()
	case "RecordingToggle":
		t.IsRecording().Toggle()
	case "PlayingToggleFollow":
		t.Follow().SetValue(true)
		t.Playing().Toggle()
	case "PlayingToggleUnfollow":
		t.Follow().SetValue(false)
		t.Playing().Toggle()
	case "InstrEnlargedToggle":
		t.InstrEnlarged().Toggle()
	case "LinkInstrTrackToggle":
		t.LinkInstrTrack().Toggle()
	case "FollowToggle":
		t.Follow().Toggle()
	case "UnitDisabledToggle":
		t.UnitDisabled().Toggle()
	case "LoopToggle":
		t.LoopToggle().Toggle()
	case "UniquePatternsToggle":
		t.UniquePatterns().Toggle()
	case "MuteToggle":
		t.Mute().Toggle()
	case "SoloToggle":
		t.Solo().Toggle()
	// Integers
	case "InstrumentVoicesAdd":
		t.Model.InstrumentVoices().Add(1)
	case "InstrumentVoicesSubtract":
		t.Model.InstrumentVoices().Add(-1)
	case "TrackVoicesAdd":
		t.TrackVoices().Add(1)
	case "TrackVoicesSubtract":
		t.TrackVoices().Add(-1)
	case "SongLengthAdd":
		t.SongLength().Add(1)
	case "SongLengthSubtract":
		t.SongLength().Add(-1)
	case "BPMAdd":
		t.BPM().Add(1)
	case "BPMSubtract":
		t.BPM().Add(-1)
	case "RowsPerPatternAdd":
		t.RowsPerPattern().Add(1)
	case "RowsPerPatternSubtract":
		t.RowsPerPattern().Add(-1)
	case "RowsPerBeatAdd":
		t.RowsPerBeat().Add(1)
	case "RowsPerBeatSubtract":
		t.RowsPerBeat().Add(-1)
	case "StepAdd":
		t.Step().Add(1)
	case "StepSubtract":
		t.Step().Add(-1)
	case "OctaveAdd":
		t.Octave().Add(1)
	case "OctaveSubtract":
		t.Octave().Add(-1)
	// Other miscellaneous
	case "Paste":
		gtx.Execute(clipboard.ReadCmd{Tag: t})
	case "OrderEditorFocus":
		t.InstrEnlarged().SetValue(false)
		gtx.Execute(key.FocusCmd{Tag: t.OrderEditor.scrollTable})
	case "TrackEditorFocus":
		t.InstrEnlarged().SetValue(false)
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
	default:
		if len(action) > 4 && action[:4] == "Note" {
			val, err := strconv.Atoi(string(action[4:]))
			if err != nil {
				break
			}
			instr := t.Model.Instruments().Selected()
			n := noteAsValue(t.Model.Octave().Value(), val-12)
			t.KeyNoteMap.Press(e.Name, tracker.NoteEvent{Channel: instr, Note: n})
		}
	}
}
