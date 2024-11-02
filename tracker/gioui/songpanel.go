package gioui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/vsariola/sointu/tracker"
	"github.com/vsariola/sointu/version"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type SongPanel struct {
	MenuBar        []widget.Clickable
	Menus          []Menu
	BPM            *NumberInput
	RowsPerPattern *NumberInput
	RowsPerBeat    *NumberInput
	Step           *NumberInput
	SongLength     *NumberInput

	RewindBtn  *ActionClickable
	PlayingBtn *BoolClickable
	RecordBtn  *BoolClickable
	FollowBtn  *BoolClickable
	PanicBtn   *BoolClickable
	LoopBtn    *BoolClickable

	Scope *Oscilloscope

	// File menu items
	fileMenuItems  []MenuItem
	NewSong        tracker.Action
	OpenSongFile   tracker.Action
	SaveSongFile   tracker.Action
	SaveSongAsFile tracker.Action
	ExportWav      tracker.Action
	Quit           tracker.Action

	// Edit menu items
	editMenuItems []MenuItem

	// Hints
	rewindHint                  string
	playHint, stopHint          string
	recordHint, stopRecordHint  string
	followOnHint, followOffHint string
	panicHint                   string
	loopOffHint, loopOnHint     string

	// Midi menu items
	midiMenuItems []MenuItem
}

func NewSongPanel(model *tracker.Model) *SongPanel {
	ret := &SongPanel{
		MenuBar:        make([]widget.Clickable, 3),
		Menus:          make([]Menu, 3),
		BPM:            NewNumberInput(model.BPM().Int()),
		RowsPerPattern: NewNumberInput(model.RowsPerPattern().Int()),
		RowsPerBeat:    NewNumberInput(model.RowsPerBeat().Int()),
		Step:           NewNumberInput(model.Step().Int()),
		SongLength:     NewNumberInput(model.SongLength().Int()),
		PanicBtn:       NewBoolClickable(model.Panic().Bool()),
		LoopBtn:        NewBoolClickable(model.LoopToggle().Bool()),
		RecordBtn:      NewBoolClickable(model.IsRecording().Bool()),
		FollowBtn:      NewBoolClickable(model.Follow().Bool()),
		PlayingBtn:     NewBoolClickable(model.Playing().Bool()),
		RewindBtn:      NewActionClickable(model.PlaySongStart()),
		Scope:          NewOscilloscope(model),
	}
	ret.fileMenuItems = []MenuItem{
		{IconBytes: icons.ContentClear, Text: "New Song", ShortcutText: keyActionMap["NewSong"], Doer: model.NewSong()},
		{IconBytes: icons.FileFolder, Text: "Open Song", ShortcutText: keyActionMap["OpenSong"], Doer: model.OpenSong()},
		{IconBytes: icons.ContentSave, Text: "Save Song", ShortcutText: keyActionMap["SaveSong"], Doer: model.SaveSong()},
		{IconBytes: icons.ContentSave, Text: "Save Song As...", ShortcutText: keyActionMap["SaveSongAs"], Doer: model.SaveSongAs()},
		{IconBytes: icons.ImageAudiotrack, Text: "Export Wav...", ShortcutText: keyActionMap["ExportWav"], Doer: model.Export()},
	}
	if canQuit {
		ret.fileMenuItems = append(ret.fileMenuItems, MenuItem{IconBytes: icons.ActionExitToApp, Text: "Quit", ShortcutText: keyActionMap["Quit"], Doer: model.Quit()})
	}
	ret.editMenuItems = []MenuItem{
		{IconBytes: icons.ContentUndo, Text: "Undo", ShortcutText: keyActionMap["Undo"], Doer: model.Undo()},
		{IconBytes: icons.ContentRedo, Text: "Redo", ShortcutText: keyActionMap["Redo"], Doer: model.Redo()},
		{IconBytes: icons.ImageCrop, Text: "Remove unused data", ShortcutText: keyActionMap["RemoveUnused"], Doer: model.RemoveUnused()},
	}
	for input := range model.MIDI.InputDevices {
		ret.midiMenuItems = append(ret.midiMenuItems, MenuItem{
			IconBytes: icons.ImageControlPoint,
			Text:      input.String(),
			Doer:      model.SelectMidiInput(input),
		})
	}
	ret.rewindHint = makeHint("Rewind", "\n(%s)", "PlaySongStartUnfollow")
	ret.playHint = makeHint("Play", " (%s)", "PlayCurrentPosUnfollow")
	ret.stopHint = makeHint("Stop", " (%s)", "StopPlaying")
	ret.panicHint = makeHint("Panic", " (%s)", "PanicToggle")
	ret.recordHint = makeHint("Record", " (%s)", "RecordingToggle")
	ret.stopRecordHint = makeHint("Stop", " (%s)", "RecordingToggle")
	ret.followOnHint = makeHint("Follow on", " (%s)", "FollowToggle")
	ret.followOffHint = makeHint("Follow off", " (%s)", "FollowToggle")
	ret.loopOffHint = makeHint("Loop off", " (%s)", "LoopToggle")
	ret.loopOnHint = makeHint("Loop on", " (%s)", "LoopToggle")

	return ret
}

func (s *SongPanel) Layout(gtx C, t *Tracker) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return s.layoutMenuBar(gtx, t)
		}),
		layout.Rigid(func(gtx C) D {
			return s.layoutSongOptions(gtx, t)
		}),
	)
}

func (t *SongPanel) layoutMenuBar(gtx C, tr *Tracker) D {
	gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(36))
	gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(36))

	panicBtnStyle := ToggleIcon(gtx, tr.Theme, t.PanicBtn, icons.AlertErrorOutline, icons.AlertError, t.panicHint, t.panicHint)
	if t.PanicBtn.Bool.Value() {
		panicBtnStyle.IconButtonStyle.Color = errorColor
	}
	menuLayouts := []layout.FlexChild{
		layout.Rigid(tr.layoutMenu(gtx, "File", &t.MenuBar[0], &t.Menus[0], unit.Dp(200), t.fileMenuItems...)),
		layout.Rigid(tr.layoutMenu(gtx, "Edit", &t.MenuBar[1], &t.Menus[1], unit.Dp(200), t.editMenuItems...)),
	}
	if len(t.midiMenuItems) > 0 {
		menuLayouts = append(
			menuLayouts,
			layout.Rigid(tr.layoutMenu(gtx, "MIDI", &t.MenuBar[2], &t.Menus[2], unit.Dp(200), t.midiMenuItems...)),
		)
	}
	menuLayouts = append(menuLayouts, layout.Flexed(1, func(gtx C) D {
		return layout.E.Layout(gtx, panicBtnStyle.Layout)
	}))
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx, menuLayouts...)
}

func (t *SongPanel) layoutSongOptions(gtx C, tr *Tracker) D {
	paint.FillShape(gtx.Ops, songSurfaceColor, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())

	in := layout.UniformInset(unit.Dp(1))

	rewindBtnStyle := ActionIcon(gtx, tr.Theme, t.RewindBtn, icons.AVFastRewind, t.rewindHint)
	playBtnStyle := ToggleIcon(gtx, tr.Theme, t.PlayingBtn, icons.AVPlayArrow, icons.AVStop, t.playHint, t.stopHint)
	recordBtnStyle := ToggleIcon(gtx, tr.Theme, t.RecordBtn, icons.AVFiberManualRecord, icons.AVFiberSmartRecord, t.recordHint, t.stopRecordHint)
	noteTrackBtnStyle := ToggleIcon(gtx, tr.Theme, t.FollowBtn, icons.ActionSpeakerNotesOff, icons.ActionSpeakerNotes, t.followOffHint, t.followOnHint)
	loopBtnStyle := ToggleIcon(gtx, tr.Theme, t.LoopBtn, icons.NavigationArrowForward, icons.AVLoop, t.loopOffHint, t.loopOnHint)

	scopeStyle := LineOscilloscope(t.Scope, tr.SignalAnalyzer().Waveform(), tr.Theme)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("LEN:", white, tr.Theme.Shaper)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					numStyle := NumericUpDown(tr.Theme, t.SongLength, "Song length")
					gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
					dims := in.Layout(gtx, numStyle.Layout)
					return dims
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("BPM:", white, tr.Theme.Shaper)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					numStyle := NumericUpDown(tr.Theme, t.BPM, "Beats per minute")
					gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
					dims := in.Layout(gtx, numStyle.Layout)
					return dims
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("RPP:", white, tr.Theme.Shaper)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					numStyle := NumericUpDown(tr.Theme, t.RowsPerPattern, "Rows per pattern")
					gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
					dims := in.Layout(gtx, numStyle.Layout)
					return dims
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("RPB:", white, tr.Theme.Shaper)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					numStyle := NumericUpDown(tr.Theme, t.RowsPerBeat, "Rows per beat")
					gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(20))
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
					dims := in.Layout(gtx, numStyle.Layout)
					return dims
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(Label("STP:", white, tr.Theme.Shaper)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					numStyle := NumericUpDown(tr.Theme, t.Step, "Cursor step")
					numStyle.UnitsPerStep = unit.Dp(20)
					dims := in.Layout(gtx, numStyle.Layout)
					return dims
				}),
			)
		}),
		layout.Rigid(VuMeter{Loudness: tr.Model.DetectorResult().Loudness[tracker.LoudnessShortTerm], Peak: tr.Model.DetectorResult().Peaks[tracker.PeakMomentary], Range: 100}.Layout),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(rewindBtnStyle.Layout),
				layout.Rigid(playBtnStyle.Layout),
				layout.Rigid(recordBtnStyle.Layout),
				layout.Rigid(noteTrackBtnStyle.Layout),
				layout.Rigid(loopBtnStyle.Layout),
			)
		}),
		layout.Flexed(1, scopeStyle.Layout),
		layout.Rigid(func(gtx C) D {
			labelStyle := LabelStyle{Text: version.VersionOrHash, FontSize: unit.Sp(12), Color: mediumEmphasisTextColor, Shaper: tr.Theme.Shaper}
			return labelStyle.Layout(gtx)
		}),
	)
}
