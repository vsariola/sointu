package gioui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
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

	PanicBtn *BoolClickable
	Scope    *Oscilloscope
	PlayBar  *PlayBar

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

	panicHint string
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
		Scope:          NewOscilloscope(model),
		PlayBar:        NewPlayBar(model),
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
	ret.panicHint = makeHint("Panic", " (%s)", "PanicToggle")
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

	scopeStyle := LineOscilloscope(t.Scope, tr.SignalAnalyzer().Waveform(), tr.Theme)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Background{}.Layout(gtx,
				func(gtx C) D {
					// push defer clip op
					defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)).Push(gtx.Ops).Pop()
					paint.FillShape(gtx.Ops, songSurfaceColor, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())
					return D{Size: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}
				},
				func(gtx C) D {
					return t.PlayBar.Layout(gtx, tr.Theme)
				},
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layoutSongOptionRow(gtx, tr.Theme, "Song length", NumericUpDown(tr.Theme, t.SongLength, "Song Length").Layout)
		}),
		layout.Rigid(func(gtx C) D {
			return layoutSongOptionRow(gtx, tr.Theme, "BPM", NumericUpDown(tr.Theme, t.BPM, "Song Length").Layout)
		}),
		layout.Rigid(func(gtx C) D {
			return layoutSongOptionRow(gtx, tr.Theme, "Rows per pat", NumericUpDown(tr.Theme, t.RowsPerPattern, "Rows per pattern").Layout)
		}),
		layout.Rigid(func(gtx C) D {
			return layoutSongOptionRow(gtx, tr.Theme, "Rows per beat", NumericUpDown(tr.Theme, t.RowsPerBeat, "Rows per beat").Layout)
		}),
		layout.Rigid(func(gtx C) D {
			return layoutSongOptionRow(gtx, tr.Theme, "Cursor step", NumericUpDown(tr.Theme, t.Step, "Cursor step").Layout)
		}),
		layout.Rigid(VuMeter{Loudness: tr.Model.DetectorResult().Loudness[tracker.LoudnessShortTerm], Peak: tr.Model.DetectorResult().Peaks[tracker.PeakMomentary], Range: 100}.Layout),
		layout.Flexed(1, scopeStyle.Layout),
		layout.Rigid(func(gtx C) D {
			labelStyle := LabelStyle{Text: version.VersionOrHash, FontSize: unit.Sp(12), Color: mediumEmphasisTextColor, Shaper: tr.Theme.Shaper}
			return labelStyle.Layout(gtx)
		}),
	)
}

func layoutSongOptionRow(gtx C, th *material.Theme, label string, widget layout.Widget) D {
	leftSpacer := layout.Spacer{Width: unit.Dp(6), Height: unit.Dp(24)}.Layout
	rightSpacer := layout.Spacer{Width: unit.Dp(6)}.Layout

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(leftSpacer),
		layout.Rigid(LabelStyle{Text: label, Color: disabledTextColor, Alignment: layout.W, FontSize: th.TextSize * 14.0 / 16.0, Shaper: th.Shaper}.Layout),
		layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
		layout.Rigid(widget),
		layout.Rigid(rightSpacer),
	)
}

type PlayBar struct {
	RewindBtn  *ActionClickable
	PlayingBtn *BoolClickable
	RecordBtn  *BoolClickable
	FollowBtn  *BoolClickable
	LoopBtn    *BoolClickable
	// Hints
	rewindHint                  string
	playHint, stopHint          string
	recordHint, stopRecordHint  string
	followOnHint, followOffHint string
	loopOffHint, loopOnHint     string
}

func NewPlayBar(model *tracker.Model) *PlayBar {
	ret := &PlayBar{
		LoopBtn:    NewBoolClickable(model.LoopToggle().Bool()),
		RecordBtn:  NewBoolClickable(model.IsRecording().Bool()),
		FollowBtn:  NewBoolClickable(model.Follow().Bool()),
		PlayingBtn: NewBoolClickable(model.Playing().Bool()),
		RewindBtn:  NewActionClickable(model.PlaySongStart()),
	}
	ret.rewindHint = makeHint("Rewind", "\n(%s)", "PlaySongStartUnfollow")
	ret.playHint = makeHint("Play", " (%s)", "PlayCurrentPosUnfollow")
	ret.stopHint = makeHint("Stop", " (%s)", "StopPlaying")
	ret.recordHint = makeHint("Record", " (%s)", "RecordingToggle")
	ret.stopRecordHint = makeHint("Stop", " (%s)", "RecordingToggle")
	ret.followOnHint = makeHint("Follow on", " (%s)", "FollowToggle")
	ret.followOffHint = makeHint("Follow off", " (%s)", "FollowToggle")
	ret.loopOffHint = makeHint("Loop off", " (%s)", "LoopToggle")
	ret.loopOnHint = makeHint("Loop on", " (%s)", "LoopToggle")
	return ret
}

func (pb *PlayBar) Layout(gtx C, th *material.Theme) D {
	rewindBtnStyle := ActionIcon(gtx, th, pb.RewindBtn, icons.AVFastRewind, pb.rewindHint)
	playBtnStyle := ToggleIcon(gtx, th, pb.PlayingBtn, icons.AVPlayArrow, icons.AVStop, pb.playHint, pb.stopHint)
	recordBtnStyle := ToggleIcon(gtx, th, pb.RecordBtn, icons.AVFiberManualRecord, icons.AVFiberSmartRecord, pb.recordHint, pb.stopRecordHint)
	noteTrackBtnStyle := ToggleIcon(gtx, th, pb.FollowBtn, icons.ActionSpeakerNotesOff, icons.ActionSpeakerNotes, pb.followOffHint, pb.followOnHint)
	loopBtnStyle := ToggleIcon(gtx, th, pb.LoopBtn, icons.NavigationArrowForward, icons.AVLoop, pb.loopOffHint, pb.loopOnHint)

	return Surface{Gray: 37}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, playBtnStyle.Layout),
			layout.Rigid(rewindBtnStyle.Layout),
			layout.Rigid(recordBtnStyle.Layout),
			layout.Rigid(noteTrackBtnStyle.Layout),
			layout.Rigid(loopBtnStyle.Layout),
		)
	})
}
