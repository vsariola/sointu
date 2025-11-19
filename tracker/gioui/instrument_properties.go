package gioui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type (
	InstrumentProperties struct {
		nameEditor          *Editor
		commentEditor       *Editor
		list                *layout.List
		soloBtn             *Clickable
		muteBtn             *Clickable
		threadBtns          [4]*Clickable
		soloHint            string
		unsoloHint          string
		muteHint            string
		unmuteHint          string
		voices              *NumericUpDownState
		splitInstrumentBtn  *Clickable
		splitInstrumentHint string
	}
)

func NewInstrumentProperties() *InstrumentProperties {
	ret := &InstrumentProperties{
		list:               &layout.List{Axis: layout.Vertical},
		nameEditor:         NewEditor(true, true, text.Start),
		commentEditor:      NewEditor(false, false, text.Start),
		soloBtn:            new(Clickable),
		muteBtn:            new(Clickable),
		voices:             NewNumericUpDownState(),
		splitInstrumentBtn: new(Clickable),
		threadBtns:         [4]*Clickable{new(Clickable), new(Clickable), new(Clickable), new(Clickable)},
	}
	ret.soloHint = makeHint("Solo", " (%s)", "SoloToggle")
	ret.unsoloHint = makeHint("Unsolo", " (%s)", "SoloToggle")
	ret.muteHint = makeHint("Mute", " (%s)", "MuteToggle")
	ret.unmuteHint = makeHint("Unmute", " (%s)", "MuteToggle")
	ret.splitInstrumentHint = makeHint("Split instrument", " (%s)", "SplitInstrument")
	return ret
}

func (ip *InstrumentProperties) Tags(level int, yield TagYieldFunc) bool {
	return yield(level, &ip.commentEditor.widgetEditor)
}

// layout
func (ip *InstrumentProperties) layout(gtx C) D {
	// get tracker from values
	tr := TrackerFromContext(gtx)
	voiceLine := func(gtx C) D {
		splitInstrumentBtn := ActionIconBtn(tr.SplitInstrument(), tr.Theme, ip.splitInstrumentBtn, icons.CommunicationCallSplit, ip.splitInstrumentHint)
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				instrumentVoices := NumUpDown(tr.Model.InstrumentVoices(), tr.Theme, ip.voices, "Number of voices for this instrument")
				return instrumentVoices.Layout(gtx)
			}),
			layout.Rigid(splitInstrumentBtn.Layout),
		)
	}

	thread1btn := ToggleIconBtn(tr.Thread1(), tr.Theme, ip.threadBtns[0], icons.ImageCropSquare, icons.ImageFilter1, "Do not render instrument on thread 1", "Render instrument on thread 1")
	thread2btn := ToggleIconBtn(tr.Thread2(), tr.Theme, ip.threadBtns[1], icons.ImageCropSquare, icons.ImageFilter2, "Do not render instrument on thread 2", "Render instrument on thread 2")
	thread3btn := ToggleIconBtn(tr.Thread3(), tr.Theme, ip.threadBtns[2], icons.ImageCropSquare, icons.ImageFilter3, "Do not render instrument on thread 3", "Render instrument on thread 3")
	thread4btn := ToggleIconBtn(tr.Thread4(), tr.Theme, ip.threadBtns[3], icons.ImageCropSquare, icons.ImageFilter4, "Do not render instrument on thread 4", "Render instrument on thread 4")

	threadbtnline := func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(thread1btn.Layout),
			layout.Rigid(thread2btn.Layout),
			layout.Rigid(thread3btn.Layout),
			layout.Rigid(thread4btn.Layout),
		)
	}

	return ip.list.Layout(gtx, 11, func(gtx C, index int) D {
		switch index {
		case 0:
			return layoutInstrumentPropertyLine(gtx, "Name", func(gtx C) D {
				return ip.nameEditor.Layout(gtx, tr.InstrumentName(), tr.Theme, &tr.Theme.InstrumentEditor.InstrumentComment, "Instr")
			})
		case 2:
			return layoutInstrumentPropertyLine(gtx, "Voices", voiceLine)
		case 4:
			muteBtn := ToggleIconBtn(tr.Mute(), tr.Theme, ip.muteBtn, icons.ToggleCheckBoxOutlineBlank, icons.ToggleCheckBox, ip.muteHint, ip.unmuteHint)
			return layoutInstrumentPropertyLine(gtx, "Mute", muteBtn.Layout)
		case 6:
			soloBtn := ToggleIconBtn(tr.Solo(), tr.Theme, ip.soloBtn, icons.ToggleCheckBoxOutlineBlank, icons.ToggleCheckBox, ip.soloHint, ip.unsoloHint)
			return layoutInstrumentPropertyLine(gtx, "Solo", soloBtn.Layout)
		case 8:
			return layoutInstrumentPropertyLine(gtx, "Thread", threadbtnline)
		case 10:
			return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
				return ip.commentEditor.Layout(gtx, tr.InstrumentComment(), tr.Theme, &tr.Theme.InstrumentEditor.InstrumentComment, "Comment")
			})
		default: // odd valued list items are dividers
			px := max(gtx.Dp(unit.Dp(1)), 1)
			paint.FillShape(gtx.Ops, color.NRGBA{255, 255, 255, 3}, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, px)).Op())
			return D{Size: image.Pt(gtx.Constraints.Max.X, px)}
		}
	})
}

func layoutInstrumentPropertyLine(gtx C, text string, content layout.Widget) D {
	tr := TrackerFromContext(gtx)
	gtx.Constraints.Max.X = min(gtx.Dp(300), gtx.Constraints.Max.X)
	label := Label(tr.Theme, &tr.Theme.InstrumentEditor.Properties.Label, text)
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(layout.Spacer{Width: 6, Height: 36}.Layout),
		layout.Rigid(label.Layout),
		layout.Flexed(1, func(gtx C) D { return layout.Dimensions{Size: gtx.Constraints.Min} }),
		layout.Rigid(content),
	)
}
