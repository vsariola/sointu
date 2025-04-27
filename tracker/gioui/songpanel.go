package gioui

import (
	"fmt"
	"image"
	"image/color"
	"strconv"

	"gioui.org/gesture"
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
	MenuBar []widget.Clickable
	Menus   []Menu

	SongSettingsExpander *Expander
	ScopeExpander        *Expander
	LoudnessExpander     *Expander
	PeakExpander         *Expander

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

		SongSettingsExpander: &Expander{Expanded: true},
		ScopeExpander:        &Expander{},
		LoudnessExpander:     &Expander{},
		PeakExpander:         &Expander{},
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
			return t.SongSettingsExpander.Layout(gtx, tr.Theme, "Song",
				func(gtx C) D {
					return LabelStyle{Text: strconv.Itoa(tr.BPM().Value()) + " BPM", Color: mediumEmphasisTextColor, Alignment: layout.W, FontSize: tr.Theme.TextSize * 14.0 / 16.0, Shaper: tr.Theme.Shaper}.Layout(gtx)
				},
				func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return layoutSongOptionRow(gtx, tr.Theme, "BPM", NumericUpDown(tr.Theme, t.BPM, "Song Length").Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return layoutSongOptionRow(gtx, tr.Theme, "Song length", NumericUpDown(tr.Theme, t.SongLength, "Song Length").Layout)
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
					)
				})
		}),
		layout.Rigid(func(gtx C) D {
			return t.LoudnessExpander.Layout(gtx, tr.Theme, "Loudness",
				func(gtx C) D {
					return LabelStyle{Text: fmt.Sprintf("%.1f dB", tr.Model.DetectorResult().Loudness[tracker.LoudnessShortTerm]), Color: mediumEmphasisTextColor, Alignment: layout.W, FontSize: tr.Theme.TextSize * 14.0 / 16.0, Shaper: tr.Theme.Shaper}.Layout(gtx)
				},
				func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return layoutSongOptionRow(gtx, tr.Theme, "Momentary", dbLabel(tr.Theme, tr.Model.DetectorResult().Loudness[tracker.LoudnessMomentary]).Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return layoutSongOptionRow(gtx, tr.Theme, "Short term", dbLabel(tr.Theme, tr.Model.DetectorResult().Loudness[tracker.LoudnessShortTerm]).Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return layoutSongOptionRow(gtx, tr.Theme, "Integrated", dbLabel(tr.Theme, tr.Model.DetectorResult().Loudness[tracker.LoudnessIntegrated]).Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return layoutSongOptionRow(gtx, tr.Theme, "Max. momentary", dbLabel(tr.Theme, tr.Model.DetectorResult().Loudness[tracker.LoudnessMaxMomentary]).Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return layoutSongOptionRow(gtx, tr.Theme, "Max. short term", dbLabel(tr.Theme, tr.Model.DetectorResult().Loudness[tracker.LoudnessMaxShortTerm]).Layout)
						}),
					)
				},
			)
		}),
		layout.Rigid(func(gtx C) D {
			return t.PeakExpander.Layout(gtx, tr.Theme, "Peaks",
				func(gtx C) D {
					maxPeak := max(tr.Model.DetectorResult().Peaks[tracker.PeakShortTerm][0], tr.Model.DetectorResult().Peaks[tracker.PeakShortTerm][1])
					return dbLabel(tr.Theme, maxPeak).Layout(gtx)
				},
				func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						// no need to show momentary peak, it does not have too much meaning
						layout.Rigid(func(gtx C) D {
							return layoutSongOptionRow(gtx, tr.Theme, "Short term L", dbLabel(tr.Theme, tr.Model.DetectorResult().Peaks[tracker.PeakShortTerm][0]).Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return layoutSongOptionRow(gtx, tr.Theme, "Short term R", dbLabel(tr.Theme, tr.Model.DetectorResult().Peaks[tracker.PeakShortTerm][1]).Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return layoutSongOptionRow(gtx, tr.Theme, "Integrated L", dbLabel(tr.Theme, tr.Model.DetectorResult().Peaks[tracker.PeakIntegrated][0]).Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return layoutSongOptionRow(gtx, tr.Theme, "Integrated R", dbLabel(tr.Theme, tr.Model.DetectorResult().Peaks[tracker.PeakIntegrated][1]).Layout)
						}),
					)
				},
			)
		}),
		layout.Flexed(1, func(gtx C) D {
			return t.ScopeExpander.Layout(gtx, tr.Theme, "Oscilloscope", func(gtx C) D { return D{} }, scopeStyle.Layout)
		}),
		layout.Rigid(func(gtx C) D {
			labelStyle := LabelStyle{Text: version.VersionOrHash, FontSize: unit.Sp(12), Color: mediumEmphasisTextColor, Shaper: tr.Theme.Shaper}
			return labelStyle.Layout(gtx)
		}),
	)
}

func dbLabel(th *material.Theme, value tracker.Decibel) LabelStyle {
	color := mediumEmphasisTextColor
	if value >= 0 {
		color = errorColor
	}
	return LabelStyle{
		Text:      fmt.Sprintf("%.1f dB", value),
		Color:     color,
		Alignment: layout.W,
		FontSize:  th.TextSize * 14.0 / 16.0,
		Shaper:    th.Shaper,
	}
}

func layoutSongOptionRow(gtx C, th *material.Theme, label string, widget layout.Widget) D {
	leftSpacer := layout.Spacer{Width: unit.Dp(6), Height: unit.Dp(24)}.Layout
	rightSpacer := layout.Spacer{Width: unit.Dp(6)}.Layout

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(leftSpacer),
		layout.Rigid(LabelStyle{Text: label, Color: mediumEmphasisTextColor, Alignment: layout.W, FontSize: th.TextSize * 14.0 / 16.0, Shaper: th.Shaper}.Layout),
		layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
		layout.Rigid(widget),
		layout.Rigid(rightSpacer),
	)
}

type Expander struct {
	Expanded bool
	click    gesture.Click
}

func (e *Expander) Update(gtx C) {
	for ev, ok := e.click.Update(gtx.Source); ok; ev, ok = e.click.Update(gtx.Source) {
		switch ev.Kind {
		case gesture.KindClick:
			e.Expanded = !e.Expanded
		}
	}
}

func (e *Expander) Layout(gtx C, th *material.Theme, title string, smallWidget, largeWidget layout.Widget) D {
	e.Update(gtx)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D { return e.layoutHeader(gtx, th, title, smallWidget) }),
		layout.Rigid(func(gtx C) D {
			if e.Expanded {
				return largeWidget(gtx)
			}
			return D{}
		}),
		layout.Rigid(func(gtx C) D {
			px := max(gtx.Dp(unit.Dp(1)), 1)
			paint.FillShape(gtx.Ops, color.NRGBA{255, 255, 255, 3}, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, px)).Op())
			return D{Size: image.Pt(gtx.Constraints.Max.X, px)}
		}),
	)
}

func (e *Expander) layoutHeader(gtx C, th *material.Theme, title string, smallWidget layout.Widget) D {
	return layout.Background{}.Layout(gtx,
		func(gtx C) D {
			defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)).Push(gtx.Ops).Pop()
			// add click op
			e.click.Add(gtx.Ops)
			return D{Size: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}
		},
		func(gtx C) D {
			leftSpacer := layout.Spacer{Width: unit.Dp(6), Height: unit.Dp(24)}.Layout
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(leftSpacer),
				layout.Rigid(LabelStyle{Text: title, Color: disabledTextColor, Alignment: layout.W, FontSize: th.TextSize * 14.0 / 16.0, Shaper: th.Shaper}.Layout),
				layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
				layout.Rigid(func(gtx C) D {
					if !e.Expanded {
						return smallWidget(gtx)
					}
					return D{}
				}),
				layout.Rigid(func(gtx C) D {
					// draw icon
					icon := icons.NavigationExpandMore
					if e.Expanded {
						icon = icons.NavigationExpandLess
					}
					gtx.Constraints.Min = image.Pt(gtx.Dp(unit.Dp(24)), gtx.Dp(unit.Dp(24)))
					return widgetForIcon(icon).Layout(gtx, th.Palette.Fg)
				}),
			)
		},
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
