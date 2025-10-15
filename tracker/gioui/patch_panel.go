package gioui

import (
	"bytes"
	"image"
	"image/color"
	"io"
	"strconv"

	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/text"
	"gioui.org/unit"
	"github.com/vsariola/sointu/tracker"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type (
	PatchPanel struct {
		instrList  InstrumentList
		tools      InstrumentTools
		unitList   UnitList
		unitEditor UnitEditor
	}

	InstrumentList struct {
		instrumentDragList *DragList
		nameEditor         *Editor

		octave            *NumericUpDownState
		enlargeBtn        *Clickable
		linkInstrTrackBtn *Clickable
		newInstrumentBtn  *Clickable

		octaveHint              string
		linkDisabledHint        string
		linkEnabledHint         string
		enlargeHint, shrinkHint string
		addInstrumentHint       string
	}

	InstrumentTools struct {
		EditorTab  *Clickable
		PresetsTab *Clickable
		CommentTab *Clickable

		Voices              *NumericUpDownState
		splitInstrumentBtn  *Clickable
		commentExpandBtn    *Clickable
		commentEditor       *Editor
		soloBtn             *Clickable
		muteBtn             *Clickable
		presetMenuBtn       *Clickable
		presetMenu          MenuState
		presetMenuItems     []ActionMenuItem
		saveInstrumentBtn   *Clickable
		loadInstrumentBtn   *Clickable
		copyInstrumentBtn   *Clickable
		deleteInstrumentBtn *Clickable

		muteHint, unmuteHint string
		soloHint, unsoloHint string
		expandCommentHint    string
		collapseCommentHint  string
		splitInstrumentHint  string
		deleteInstrumentHint string
	}
)

// PatchPanel methods

func NewPatchPanel(model *tracker.Model) *PatchPanel {
	return &PatchPanel{
		instrList:  MakeInstrList(model),
		tools:      MakeInstrumentTools(model),
		unitList:   MakeUnitList(model),
		unitEditor: *NewUnitEditor(model),
	}
}

func (pp *PatchPanel) Layout(gtx C) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(pp.instrList.Layout),
		layout.Rigid(pp.tools.Layout),
		layout.Flexed(1, func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(pp.unitList.Layout),
				layout.Flexed(1, pp.unitEditor.Layout),
			)
		}))
}

func (pp *PatchPanel) Tags(level int, yield TagYieldFunc) bool {
	return pp.instrList.Tags(level, yield) &&
		pp.tools.Tags(level, yield) &&
		pp.unitList.Tags(level, yield) &&
		pp.unitEditor.Tags(level, yield)
}

// TreeFocused returns true if any of the tags in the patch panel is focused
func (pp *PatchPanel) TreeFocused(gtx C) bool {
	return !pp.Tags(0, func(_ int, tag event.Tag) bool {
		return !gtx.Focused(tag)
	})
}

// InstrumentTools methods

func MakeInstrumentTools(m *tracker.Model) InstrumentTools {
	ret := InstrumentTools{
		EditorTab:            new(Clickable),
		PresetsTab:           new(Clickable),
		CommentTab:           new(Clickable),
		Voices:               NewNumericUpDownState(),
		deleteInstrumentBtn:  new(Clickable),
		splitInstrumentBtn:   new(Clickable),
		copyInstrumentBtn:    new(Clickable),
		saveInstrumentBtn:    new(Clickable),
		loadInstrumentBtn:    new(Clickable),
		commentExpandBtn:     new(Clickable),
		presetMenuBtn:        new(Clickable),
		soloBtn:              new(Clickable),
		muteBtn:              new(Clickable),
		presetMenuItems:      []ActionMenuItem{},
		commentEditor:        NewEditor(false, false, text.Start),
		expandCommentHint:    makeHint("Expand comment", " (%s)", "CommentExpandedToggle"),
		collapseCommentHint:  makeHint("Collapse comment", " (%s)", "CommentExpandedToggle"),
		deleteInstrumentHint: makeHint("Delete\ninstrument", "\n(%s)", "DeleteInstrument"),
		muteHint:             makeHint("Mute", " (%s)", "MuteToggle"),
		unmuteHint:           makeHint("Unmute", " (%s)", "MuteToggle"),
		soloHint:             makeHint("Solo", " (%s)", "SoloToggle"),
		unsoloHint:           makeHint("Unsolo", " (%s)", "SoloToggle"),
		splitInstrumentHint:  makeHint("Split instrument", " (%s)", "SplitInstrument"),
	}
	for index, name := range m.IterateInstrumentPresets {
		ret.presetMenuItems = append(ret.presetMenuItems, MenuItem(m.LoadPreset(index), name, "", icons.ImageAudiotrack))
	}
	return ret
}

func (it *InstrumentTools) Layout(gtx C) D {
	t := TrackerFromContext(gtx)
	it.update(gtx, t)
	voicesLabel := Label(t.Theme, &t.Theme.InstrumentEditor.Voices, "Voices")

	editorBtn := TabBtn(t.Model.InstrEditor(), t.Theme, it.EditorTab, "Editor", "")
	presetsBtn := TabBtn(t.Model.InstrPresets(), t.Theme, it.PresetsTab, "Presets", "")
	commentBtn := TabBtn(t.Model.InstrComment(), t.Theme, it.CommentTab, "Comment", "")

	splitInstrumentBtn := ActionIconBtn(t.SplitInstrument(), t.Theme, it.splitInstrumentBtn, icons.CommunicationCallSplit, it.splitInstrumentHint)
	soloBtn := ToggleIconBtn(t.Solo(), t.Theme, it.soloBtn, icons.SocialGroup, icons.SocialPerson, it.soloHint, it.unsoloHint)
	muteBtn := ToggleIconBtn(t.Mute(), t.Theme, it.muteBtn, icons.AVVolumeUp, icons.AVVolumeOff, it.muteHint, it.unmuteHint)
	//	saveInstrumentBtn := IconBtn(t.Theme, &t.Theme.IconButton.Enabled, it.saveInstrumentBtn, icons.ContentSave, "Save instrument")
	//	loadInstrumentBtn := IconBtn(t.Theme, &t.Theme.IconButton.Enabled, it.loadInstrumentBtn, icons.FileFolderOpen, "Load instrument")
	copyInstrumentBtn := IconBtn(t.Theme, &t.Theme.IconButton.Enabled, it.copyInstrumentBtn, icons.ContentContentCopy, "Copy instrument")
	deleteInstrumentBtn := ActionIconBtn(t.DeleteInstrument(), t.Theme, it.deleteInstrumentBtn, icons.ActionDelete, it.deleteInstrumentHint)
	instrumentVoices := NumUpDown(t.Model.InstrumentVoices(), t.Theme, it.Voices, "Number of voices for this instrument")
	btns := func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(layout.Spacer{Width: 6}.Layout),
			layout.Rigid(editorBtn.Layout),
			layout.Rigid(presetsBtn.Layout),
			layout.Rigid(commentBtn.Layout),
			layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
			layout.Rigid(voicesLabel.Layout),
			layout.Rigid(layout.Spacer{Width: 4}.Layout),
			layout.Rigid(instrumentVoices.Layout),
			layout.Rigid(splitInstrumentBtn.Layout),
			layout.Rigid(soloBtn.Layout),
			layout.Rigid(muteBtn.Layout),
			/*layout.Rigid(func(gtx C) D {
				presetBtn := IconBtn(t.Theme, &t.Theme.IconButton.Enabled, it.presetMenuBtn, icons.NavigationMenu, "Load preset")
				dims := presetBtn.Layout(gtx)
				op.Offset(image.Pt(0, dims.Size.Y)).Add(gtx.Ops)
				m := Menu(t.Theme, &it.presetMenu)
				m.Style = &t.Theme.Menu.Preset
				m.Layout(gtx, it.presetMenuItems...)
				return dims
			}),*/
			//			layout.Rigid(saveInstrumentBtn.Layout),
			//			layout.Rigid(loadInstrumentBtn.Layout),
			layout.Rigid(copyInstrumentBtn.Layout),
			layout.Rigid(deleteInstrumentBtn.Layout),
		)
	}
	/*comment := func(gtx C) D {
		defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
		ret := layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
			return it.commentEditor.Layout(gtx, t.InstrumentComment(), t.Theme, &t.Theme.InstrumentEditor.InstrumentComment, "Comment")
		})
		return ret
	}*/
	return Surface{Gray: 37, Focus: t.PatchPanel.TreeFocused(gtx)}.Layout(gtx, btns)
}

func (it *InstrumentTools) update(gtx C, tr *Tracker) {
	for it.copyInstrumentBtn.Clicked(gtx) {
		if contents, ok := tr.Instruments().List().CopyElements(); ok {
			gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(bytes.NewReader(contents))})
			tr.Alerts().Add("Instrument copied to clipboard", tracker.Info)
		}
	}
	for it.saveInstrumentBtn.Clicked(gtx) {
		writer, err := tr.Explorer.CreateFile(tr.InstrumentName().Value() + ".yml")
		if err != nil {
			continue
		}
		tr.SaveInstrument(writer)
	}
	for it.loadInstrumentBtn.Clicked(gtx) {
		reader, err := tr.Explorer.ChooseFile(".yml", ".json", ".4ki", ".4kp")
		if err != nil {
			continue
		}
		tr.LoadInstrument(reader)
	}
	for it.presetMenuBtn.Clicked(gtx) {
		it.presetMenu.visible = true
	}
	for it.commentEditor.Update(gtx, tr.InstrumentComment()) != EditorEventNone {
		tr.PatchPanel.instrList.instrumentDragList.Focus()
	}
}

func (it *InstrumentTools) Tags(level int, yield TagYieldFunc) bool {
	return true
}

// InstrumentList methods

func MakeInstrList(model *tracker.Model) InstrumentList {
	return InstrumentList{
		instrumentDragList: NewDragList(model.Instruments().List(), layout.Horizontal),
		nameEditor:         NewEditor(true, true, text.Middle),
		octave:             NewNumericUpDownState(),
		enlargeBtn:         new(Clickable),
		linkInstrTrackBtn:  new(Clickable),
		newInstrumentBtn:   new(Clickable),
		octaveHint:         makeHint("Octave down", " (%s)", "OctaveNumberInputSubtract") + makeHint(" or up", " (%s)", "OctaveNumberInputAdd"),
		linkDisabledHint:   makeHint("Instrument-Track\nlinking disabled", "\n(%s)", "LinkInstrTrackToggle"),
		linkEnabledHint:    makeHint("Instrument-Track\nlinking enabled", "\n(%s)", "LinkInstrTrackToggle"),
		enlargeHint:        makeHint("Enlarge", " (%s)", "InstrEnlargedToggle"),
		shrinkHint:         makeHint("Shrink", " (%s)", "InstrEnlargedToggle"),
		addInstrumentHint:  makeHint("Add\ninstrument", "\n(%s)", "AddInstrument"),
	}
}

func (il *InstrumentList) Layout(gtx C) D {
	t := TrackerFromContext(gtx)
	il.update(gtx, t)
	octave := NumUpDown(t.Model.Octave(), t.Theme, t.OctaveNumberInput, "Octave")
	linkInstrTrackBtn := ToggleIconBtn(t.Model.LinkInstrTrack(), t.Theme, il.linkInstrTrackBtn, icons.NotificationSyncDisabled, icons.NotificationSync, il.linkDisabledHint, il.linkEnabledHint)
	instrEnlargedBtn := ToggleIconBtn(t.Model.InstrEnlarged(), t.Theme, il.enlargeBtn, icons.NavigationFullscreen, icons.NavigationFullscreenExit, il.enlargeHint, il.shrinkHint)
	addInstrumentBtn := ActionIconBtn(t.Model.AddInstrument(), t.Theme, il.newInstrumentBtn, icons.ContentAdd, il.addInstrumentHint)
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(
		gtx,
		layout.Flexed(1, il.actualLayout),
		layout.Rigid(layout.Spacer{Width: 10}.Layout),
		layout.Rigid(Label(t.Theme, &t.Theme.InstrumentEditor.Octave, "Octave").Layout),
		layout.Rigid(layout.Spacer{Width: 4}.Layout),
		layout.Rigid(octave.Layout),
		layout.Rigid(linkInstrTrackBtn.Layout),
		layout.Rigid(instrEnlargedBtn.Layout),
		layout.Rigid(addInstrumentBtn.Layout),
	)
}

func (il *InstrumentList) actualLayout(gtx C) D {
	t := TrackerFromContext(gtx)
	gtx.Constraints.Max.Y = gtx.Dp(36)
	gtx.Constraints.Min.Y = gtx.Dp(36)
	element := func(gtx C, i int) D {
		grabhandle := Label(t.Theme, &t.Theme.InstrumentEditor.InstrumentList.Number, strconv.Itoa(i+1))
		label := func(gtx C) D {
			name, level, mute, ok := (*tracker.Instruments)(t.Model).Item(i)
			if !ok {
				labelStyle := Label(t.Theme, &t.Theme.InstrumentEditor.InstrumentList.Number, "")
				return layout.Center.Layout(gtx, labelStyle.Layout)
			}
			s := t.Theme.InstrumentEditor.InstrumentList.NameMuted
			if !mute {
				s = t.Theme.InstrumentEditor.InstrumentList.Name
				k := byte(255 - level*127)
				s.Color = color.NRGBA{R: 255, G: k, B: 255, A: 255}
			}
			if i == il.instrumentDragList.TrackerList.Selected() {
				for il.nameEditor.Update(gtx, t.InstrumentName()) != EditorEventNone {
					il.instrumentDragList.Focus()
				}
				return layout.Center.Layout(gtx, func(gtx C) D {
					defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
					return il.nameEditor.Layout(gtx, t.InstrumentName(), t.Theme, &s, "Instr")
				})
			}
			if name == "" {
				name = "Instr"
			}
			l := s.AsLabelStyle()
			return layout.Center.Layout(gtx, Label(t.Theme, &l, name).Layout)
		}
		return layout.Center.Layout(gtx, func(gtx C) D {
			return layout.Inset{Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(grabhandle.Layout),
					layout.Rigid(label),
				)
			})
		})
	}
	instrumentList := FilledDragList(t.Theme, il.instrumentDragList)
	instrumentList.ScrollBar = t.Theme.InstrumentEditor.InstrumentList.ScrollBar
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()
	dims := instrumentList.Layout(gtx, element, nil)
	gtx.Constraints = layout.Exact(dims.Size)
	instrumentList.LayoutScrollBar(gtx)
	return dims
}

func (il *InstrumentList) update(gtx C, t *Tracker) {
	for {
		event, ok := gtx.Event(
			key.Filter{Focus: il.instrumentDragList, Name: key.NameDownArrow},
			key.Filter{Focus: il.instrumentDragList, Name: key.NameReturn},
			key.Filter{Focus: il.instrumentDragList, Name: key.NameEnter},
		)
		if !ok {
			break
		}
		if e, ok := event.(key.Event); ok && e.State == key.Press {
			switch e.Name {
			case key.NameDownArrow:
				t.PatchPanel.unitList.dragList.Focus()
			case key.NameReturn, key.NameEnter:
				il.nameEditor.Focus()
			}
		}
	}
}

func (il *InstrumentList) Tags(level int, yield TagYieldFunc) bool {
	return yield(level, il.instrumentDragList)
}
