package gioui

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"

	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gopkg.in/yaml.v2"
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

func loadCustomKeyBindings() []KeyBinding {
	var keyBindings []KeyBinding
	_, err := ReadCustomConfigYml("keybindings.yml", &keyBindings)
	if err != nil {
		return nil
	}
	if len(keyBindings) == 0 {
		return nil
	}
	return keyBindings
}

//go:embed keybindings.yml
var defaultKeyBindingsYaml []byte

func loadDefaultKeyBindings() []KeyBinding {
	var keyBindings []KeyBinding
	err := yaml.Unmarshal(defaultKeyBindingsYaml, &keyBindings)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal keybindings: %w", err))
	}
	return keyBindings
}

func init() {
	keyBindings := loadDefaultKeyBindings()
	keyBindings = append(keyBindings, loadCustomKeyBindings()...)

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
		t.JammingReleased(e)
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
		t.Follow().Bool().Set(true)
		t.PlayCurrentPos().Do()
	case "PlayCurrentPosUnfollow":
		t.Follow().Bool().Set(false)
		t.PlayCurrentPos().Do()
	case "PlaySongStartFollow":
		t.Follow().Bool().Set(true)
		t.PlaySongStart().Do()
	case "PlaySongStartUnfollow":
		t.Follow().Bool().Set(false)
		t.PlaySongStart().Do()
	case "PlaySelectedFollow":
		t.Follow().Bool().Set(true)
		t.PlaySelected().Do()
	case "PlaySelectedUnfollow":
		t.Follow().Bool().Set(false)
		t.PlaySelected().Do()
	case "PlayLoopFollow":
		t.Follow().Bool().Set(true)
		t.PlayFromLoopStart().Do()
	case "PlayLoopUnfollow":
		t.Follow().Bool().Set(false)
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
	// Booleans
	case "PanicToggle":
		t.Panic().Bool().Toggle()
	case "RecordingToggle":
		t.IsRecording().Bool().Toggle()
	case "PlayingToggleFollow":
		t.Follow().Bool().Set(true)
		t.Playing().Bool().Toggle()
	case "PlayingToggleUnfollow":
		t.Follow().Bool().Set(false)
		t.Playing().Bool().Toggle()
	case "InstrEnlargedToggle":
		t.InstrEnlarged().Bool().Toggle()
	case "LinkInstrTrackToggle":
		t.LinkInstrTrack().Bool().Toggle()
	case "CommentExpandedToggle":
		t.CommentExpanded().Bool().Toggle()
	case "FollowToggle":
		t.Follow().Bool().Toggle()
	case "UnitDisabledToggle":
		t.UnitDisabled().Bool().Toggle()
	case "LoopToggle":
		t.LoopToggle().Bool().Toggle()
	case "UniquePatternsToggle":
		t.UniquePatterns().Bool().Toggle()
	case "MuteToggle":
		t.Mute().Bool().Toggle()
	case "SoloToggle":
		t.Solo().Bool().Toggle()
	// Integers
	case "InstrumentVoicesAdd":
		t.Model.InstrumentVoices().Int().Add(1)
	case "InstrumentVoicesSubtract":
		t.Model.InstrumentVoices().Int().Add(-1)
	case "TrackVoicesAdd":
		t.TrackVoices().Int().Add(1)
	case "TrackVoicesSubtract":
		t.TrackVoices().Int().Add(-1)
	case "SongLengthAdd":
		t.SongLength().Int().Add(1)
	case "SongLengthSubtract":
		t.SongLength().Int().Add(-1)
	case "BPMAdd":
		t.BPM().Int().Add(1)
	case "BPMSubtract":
		t.BPM().Int().Add(-1)
	case "RowsPerPatternAdd":
		t.RowsPerPattern().Int().Add(1)
	case "RowsPerPatternSubtract":
		t.RowsPerPattern().Int().Add(-1)
	case "RowsPerBeatAdd":
		t.RowsPerBeat().Int().Add(1)
	case "RowsPerBeatSubtract":
		t.RowsPerBeat().Int().Add(-1)
	case "StepAdd":
		t.Step().Int().Add(1)
	case "StepSubtract":
		t.Step().Int().Add(-1)
	case "OctaveAdd":
		t.Octave().Int().Add(1)
	case "OctaveSubtract":
		t.Octave().Int().Add(-1)
	// Other miscellaneous
	case "Paste":
		gtx.Execute(clipboard.ReadCmd{Tag: t})
	case "OrderEditorFocus":
		t.OrderEditor.scrollTable.Focus()
	case "TrackEditorFocus":
		t.TrackEditor.scrollTable.Focus()
	case "InstrumentEditorFocus":
		t.InstrumentEditor.Focus()
	case "FocusPrev":
		switch {
		case t.OrderEditor.scrollTable.Focused():
			t.InstrumentEditor.unitEditor.sliderList.Focus()
		case t.TrackEditor.scrollTable.Focused():
			t.OrderEditor.scrollTable.Focus()
		case t.InstrumentEditor.Focused():
			if t.InstrumentEditor.enlargeBtn.Bool.Value() {
				t.InstrumentEditor.unitEditor.sliderList.Focus()
			} else {
				t.TrackEditor.scrollTable.Focus()
			}
		default:
			t.InstrumentEditor.Focus()
		}
	case "FocusNext":
		switch {
		case t.OrderEditor.scrollTable.Focused():
			t.TrackEditor.scrollTable.Focus()
		case t.TrackEditor.scrollTable.Focused():
			t.InstrumentEditor.Focus()
		case t.InstrumentEditor.Focused():
			t.InstrumentEditor.unitEditor.sliderList.Focus()
		default:
			if t.InstrumentEditor.enlargeBtn.Bool.Value() {
				t.InstrumentEditor.Focus()
			} else {
				t.OrderEditor.scrollTable.Focus()
			}
		}
	default:
		if action[:4] == "Note" {
			val, err := strconv.Atoi(string(action[4:]))
			if err != nil {
				break
			}
			t.JammingPressed(e, val-12)
		}
	}
}

func (t *Tracker) JammingPressed(e key.Event, val int) byte {
	if _, ok := t.KeyPlaying[e.Name]; !ok {
		n := noteAsValue(t.OctaveNumberInput.Int.Value(), val)
		instr := t.InstrumentEditor.instrumentDragList.TrackerList.Selected()
		t.KeyPlaying[e.Name] = t.InstrNoteOn(instr, n)
		return n
	}
	return 0
}

func (t *Tracker) JammingReleased(e key.Event) bool {
	if noteID, ok := t.KeyPlaying[e.Name]; ok {
		noteID.NoteOff()
		delete(t.KeyPlaying, e.Name)
		return true
	}
	return false
}
