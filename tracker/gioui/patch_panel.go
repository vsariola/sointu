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
		instrList    InstrumentList
		tools        InstrumentTools
		instrProps   InstrumentProperties
		instrPresets InstrumentPresets
		instrEditor  InstrumentEditor
		*tracker.Model
	}

	InstrumentList struct {
		instrumentDragList *DragList
		nameEditor         *Editor
	}

	InstrumentTools struct {
		EditorTab  *Clickable
		PresetsTab *Clickable
		CommentTab *Clickable

		saveInstrumentBtn   *Clickable
		loadInstrumentBtn   *Clickable
		copyInstrumentBtn   *Clickable
		deleteInstrumentBtn *Clickable

		octave            *NumericUpDownState
		enlargeBtn        *Clickable
		linkInstrTrackBtn *Clickable
		newInstrumentBtn  *Clickable

		octaveHint              string
		linkDisabledHint        string
		linkEnabledHint         string
		enlargeHint, shrinkHint string
		addInstrumentHint       string

		deleteInstrumentHint string
	}
)

// PatchPanel methods

func NewPatchPanel(model *tracker.Model) *PatchPanel {
	return &PatchPanel{
		instrEditor:  *NewInstrumentEditor(model),
		instrList:    MakeInstrList(model),
		tools:        MakeInstrumentTools(model),
		instrProps:   *NewInstrumentProperties(),
		instrPresets: *NewInstrumentPresets(model),
		Model:        model,
	}
}

func (pp *PatchPanel) Layout(gtx C) D {
	tr := TrackerFromContext(gtx)
	bottom := func(gtx C) D {
		switch {
		case tr.InstrComment().Value():
			return pp.instrProps.layout(gtx)
		case tr.InstrPresets().Value():
			return pp.instrPresets.layout(gtx)
		default: // editor
			return pp.instrEditor.layout(gtx)
		}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(pp.instrList.Layout),
		layout.Rigid(pp.tools.Layout),
		layout.Flexed(1, bottom),
	)
}

func (pp *PatchPanel) BottomTags(level int, yield TagYieldFunc) bool {
	switch {
	case pp.InstrComment().Value():
		return pp.instrProps.Tags(level, yield)
	case pp.InstrPresets().Value():
		return pp.instrPresets.Tags(level, yield)
	default: // editor
		return pp.instrEditor.Tags(level, yield)
	}
}

func (pp *PatchPanel) Tags(level int, yield TagYieldFunc) bool {
	return pp.instrList.Tags(level, yield) &&
		pp.tools.Tags(level, yield) &&
		pp.BottomTags(level, yield)
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
		deleteInstrumentBtn:  new(Clickable),
		copyInstrumentBtn:    new(Clickable),
		saveInstrumentBtn:    new(Clickable),
		loadInstrumentBtn:    new(Clickable),
		deleteInstrumentHint: makeHint("Delete\ninstrument", "\n(%s)", "DeleteInstrument"),
		octave:               NewNumericUpDownState(),
		enlargeBtn:           new(Clickable),
		linkInstrTrackBtn:    new(Clickable),
		newInstrumentBtn:     new(Clickable),
		octaveHint:           makeHint("Octave down", " (%s)", "OctaveNumberInputSubtract") + makeHint(" or up", " (%s)", "OctaveNumberInputAdd"),
		linkDisabledHint:     makeHint("Instrument-Track\nlinking disabled", "\n(%s)", "LinkInstrTrackToggle"),
		linkEnabledHint:      makeHint("Instrument-Track\nlinking enabled", "\n(%s)", "LinkInstrTrackToggle"),
		enlargeHint:          makeHint("Enlarge", " (%s)", "InstrEnlargedToggle"),
		shrinkHint:           makeHint("Shrink", " (%s)", "InstrEnlargedToggle"),
		addInstrumentHint:    makeHint("Add\ninstrument", "\n(%s)", "AddInstrument"),
	}
	return ret
}

func (it *InstrumentTools) Layout(gtx C) D {
	t := TrackerFromContext(gtx)
	it.update(gtx, t)
	editorBtn := TabBtn(t.Model.InstrEditor(), t.Theme, it.EditorTab, "Editor", "")
	presetsBtn := TabBtn(t.Model.InstrPresets(), t.Theme, it.PresetsTab, "Presets", "")
	commentBtn := TabBtn(t.Model.InstrComment(), t.Theme, it.CommentTab, "Properties", "")
	octave := NumUpDown(t.Model.Octave(), t.Theme, t.OctaveNumberInput, "Octave")
	linkInstrTrackBtn := ToggleIconBtn(t.Model.LinkInstrTrack(), t.Theme, it.linkInstrTrackBtn, icons.NotificationSyncDisabled, icons.NotificationSync, it.linkDisabledHint, it.linkEnabledHint)
	instrEnlargedBtn := ToggleIconBtn(t.Model.InstrEnlarged(), t.Theme, it.enlargeBtn, icons.NavigationFullscreen, icons.NavigationFullscreenExit, it.enlargeHint, it.shrinkHint)
	addInstrumentBtn := ActionIconBtn(t.Model.AddInstrument(), t.Theme, it.newInstrumentBtn, icons.ContentAdd, it.addInstrumentHint)

	saveInstrumentBtn := IconBtn(t.Theme, &t.Theme.IconButton.Enabled, it.saveInstrumentBtn, icons.ContentSave, "Save instrument")
	loadInstrumentBtn := IconBtn(t.Theme, &t.Theme.IconButton.Enabled, it.loadInstrumentBtn, icons.FileFolderOpen, "Load instrument")
	copyInstrumentBtn := IconBtn(t.Theme, &t.Theme.IconButton.Enabled, it.copyInstrumentBtn, icons.ContentContentCopy, "Copy instrument")
	deleteInstrumentBtn := ActionIconBtn(t.DeleteInstrument(), t.Theme, it.deleteInstrumentBtn, icons.ActionDelete, it.deleteInstrumentHint)
	btns := func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(layout.Spacer{Width: 6}.Layout),
			layout.Rigid(editorBtn.Layout),
			layout.Rigid(presetsBtn.Layout),
			layout.Rigid(commentBtn.Layout),
			layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
			layout.Rigid(layout.Spacer{Width: 4}.Layout),
			layout.Rigid(Label(t.Theme, &t.Theme.InstrumentEditor.Octave, "Octave").Layout),
			layout.Rigid(octave.Layout),
			layout.Rigid(linkInstrTrackBtn.Layout),
			layout.Rigid(instrEnlargedBtn.Layout),
			layout.Rigid(copyInstrumentBtn.Layout),
			layout.Rigid(saveInstrumentBtn.Layout),
			layout.Rigid(loadInstrumentBtn.Layout),
			layout.Rigid(deleteInstrumentBtn.Layout),
			layout.Rigid(addInstrumentBtn.Layout),
		)
	}
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
}

func (it *InstrumentTools) Tags(level int, yield TagYieldFunc) bool {
	return true
}

// InstrumentList methods

func MakeInstrList(model *tracker.Model) InstrumentList {
	return InstrumentList{
		instrumentDragList: NewDragList(model.Instruments().List(), layout.Horizontal),
		nameEditor:         NewEditor(true, true, text.Middle),
	}
}

func (il *InstrumentList) Layout(gtx C) D {
	t := TrackerFromContext(gtx)
	il.update(gtx, t)
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
				var tagged Tagged
				switch {
				case t.InstrComment().Value():
					tagged = &t.PatchPanel.instrProps
				case t.InstrPresets().Value():
					tagged = &t.PatchPanel.instrPresets
				default: // editor
					tagged = &t.PatchPanel.instrEditor
				}
				if tag, ok := firstTag(tagged); ok {
					gtx.Execute(key.FocusCmd{Tag: tag})
				}
			case key.NameReturn, key.NameEnter:
				il.nameEditor.Focus()
			}
		}
	}
}

func (il *InstrumentList) Tags(level int, yield TagYieldFunc) bool {
	return yield(level, il.instrumentDragList)
}
