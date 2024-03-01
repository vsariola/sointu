package gioui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/vsariola/sointu/tracker"
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

	RewindBtn    *ActionClickable
	PlayingBtn   *BoolClickable
	RecordBtn    *BoolClickable
	NoteTracking *BoolClickable
	PanicBtn     *BoolClickable
	LoopBtn      *BoolClickable

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
}

func NewSongPanel(model *tracker.Model) *SongPanel {
	ret := &SongPanel{
		MenuBar:        make([]widget.Clickable, 2),
		Menus:          make([]Menu, 2),
		BPM:            NewNumberInput(model.BPM().Int()),
		RowsPerPattern: NewNumberInput(model.RowsPerPattern().Int()),
		RowsPerBeat:    NewNumberInput(model.RowsPerBeat().Int()),
		Step:           NewNumberInput(model.Step().Int()),
		SongLength:     NewNumberInput(model.SongLength().Int()),
		PanicBtn:       NewBoolClickable(model.Panic().Bool()),
		LoopBtn:        NewBoolClickable(model.LoopToggle().Bool()),
		RecordBtn:      NewBoolClickable(model.IsRecording().Bool()),
		NoteTracking:   NewBoolClickable(model.NoteTracking().Bool()),
		PlayingBtn:     NewBoolClickable(model.Playing().Bool()),
		RewindBtn:      NewActionClickable(model.Rewind()),
	}
	ret.fileMenuItems = []MenuItem{
		{IconBytes: icons.ContentClear, Text: "New Song", ShortcutText: shortcutKey + "N", Doer: model.NewSong()},
		{IconBytes: icons.FileFolder, Text: "Open Song", ShortcutText: shortcutKey + "O", Doer: model.OpenSong()},
		{IconBytes: icons.ContentSave, Text: "Save Song", ShortcutText: shortcutKey + "S", Doer: model.SaveSong()},
		{IconBytes: icons.ContentSave, Text: "Save Song As...", Doer: model.SaveSongAs()},
		{IconBytes: icons.ImageAudiotrack, Text: "Export Wav...", Doer: model.Export()},
	}
	if canQuit {
		ret.fileMenuItems = append(ret.fileMenuItems, MenuItem{IconBytes: icons.ActionExitToApp, Text: "Quit", Doer: model.Quit()})
	}
	ret.editMenuItems = []MenuItem{
		{IconBytes: icons.ContentUndo, Text: "Undo", ShortcutText: shortcutKey + "Z", Doer: model.Undo()},
		{IconBytes: icons.ContentRedo, Text: "Redo", ShortcutText: shortcutKey + "Y", Doer: model.Redo()},
		{IconBytes: icons.ImageCrop, Text: "Remove unused data", Doer: model.RemoveUnused()},
	}
	return ret
}

const shortcutKey = "Ctrl+"

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

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(tr.layoutMenu(gtx, "File", &t.MenuBar[0], &t.Menus[0], unit.Dp(200), t.fileMenuItems...)),
		layout.Rigid(tr.layoutMenu(gtx, "Edit", &t.MenuBar[1], &t.Menus[1], unit.Dp(200), t.editMenuItems...)),
	)
}

func (t *SongPanel) layoutSongOptions(gtx C, tr *Tracker) D {
	paint.FillShape(gtx.Ops, songSurfaceColor, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())

	in := layout.UniformInset(unit.Dp(1))

	panicBtnStyle := ToggleButton(gtx, tr.Theme, t.PanicBtn, "Panic (F12)")
	rewindBtnStyle := ActionIcon(gtx, tr.Theme, t.RewindBtn, icons.AVFastRewind, "Rewind\n(F5)")
	playBtnStyle := ToggleIcon(gtx, tr.Theme, t.PlayingBtn, icons.AVPlayArrow, icons.AVStop, "Play (F6 / Space)", "Stop (F6 / Space)")
	recordBtnStyle := ToggleIcon(gtx, tr.Theme, t.RecordBtn, icons.AVFiberManualRecord, icons.AVFiberSmartRecord, "Record (F7)", "Stop (F7)")
	noteTrackBtnStyle := ToggleIcon(gtx, tr.Theme, t.NoteTracking, icons.ActionSpeakerNotesOff, icons.ActionSpeakerNotes, "Follow\nOff\n(F8)", "Follow\nOn\n(F8)")
	loopBtnStyle := ToggleIcon(gtx, tr.Theme, t.LoopBtn, icons.NavigationArrowForward, icons.AVLoop, "Loop\nOff\n(Ctrl+L)", "Loop\nOn\n(Ctrl+L)")

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
		layout.Rigid(VuMeter{AverageVolume: tr.Model.AverageVolume(), PeakVolume: tr.Model.PeakVolume(), Range: 100}.Layout),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(rewindBtnStyle.Layout),
				layout.Rigid(playBtnStyle.Layout),
				layout.Rigid(recordBtnStyle.Layout),
				layout.Rigid(noteTrackBtnStyle.Layout),
				layout.Rigid(loopBtnStyle.Layout),
			)
		}),
		layout.Rigid(panicBtnStyle.Layout),
	)
}
