package gioui

import (
	"math"
	"strconv"

	"gioui.org/layout"
	"gioui.org/unit"
	"github.com/vsariola/sointu/tracker"
)

type (
	OscilloscopeState struct {
		onceBtn              *Clickable
		wrapBtn              *Clickable
		lengthInBeatsNumber  *NumericUpDownState
		triggerChannelNumber *NumericUpDownState
		plot                 *Plot
	}

	Oscilloscope struct {
		Theme *Theme
		Model *tracker.ScopeModel
		State *OscilloscopeState
	}
)

func NewOscilloscope(model *tracker.Model) *OscilloscopeState {
	return &OscilloscopeState{
		plot:                 NewPlot(plotRange{0, 1}, plotRange{-1, 1}, 0),
		onceBtn:              new(Clickable),
		wrapBtn:              new(Clickable),
		lengthInBeatsNumber:  NewNumericUpDownState(),
		triggerChannelNumber: NewNumericUpDownState(),
	}
}

func Scope(th *Theme, m *tracker.ScopeModel, st *OscilloscopeState) Oscilloscope {
	return Oscilloscope{
		Theme: th,
		Model: m,
		State: st,
	}
}

func (s *Oscilloscope) Layout(gtx C) D {
	t := TrackerFromContext(gtx)
	leftSpacer := layout.Spacer{Width: unit.Dp(6), Height: unit.Dp(24)}.Layout
	rightSpacer := layout.Spacer{Width: unit.Dp(6)}.Layout

	triggerChannel := NumUpDown(s.Model.TriggerChannel(), s.Theme, s.State.triggerChannelNumber, "Trigger channel")
	lengthInBeats := NumUpDown(s.Model.LengthInBeats(), s.Theme, s.State.lengthInBeatsNumber, "Buffer length in beats")

	onceBtn := ToggleBtn(s.Model.Once(), s.Theme, s.State.onceBtn, "Once", "Trigger once on next event")
	wrapBtn := ToggleBtn(s.Model.Wrap(), s.Theme, s.State.wrapBtn, "Wrap", "Wrap buffer when full")

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D {
			w := s.Model.Waveform()
			cx := float32(w.Cursor) / float32(len(w.Buffer))

			data := func(chn int, xr plotRange) (yr plotRange, ok bool) {
				x1 := max(int(xr.a*float32(len(w.Buffer))), 0)
				x2 := min(int(xr.b*float32(len(w.Buffer))), len(w.Buffer)-1)
				if x1 > x2 {
					return plotRange{}, false
				}
				y1 := float32(math.Inf(-1))
				y2 := float32(math.Inf(+1))
				for i := x1; i <= x2; i++ {
					sample := w.Buffer[i][chn]
					y1 = max(y1, sample)
					y2 = min(y2, sample)
				}
				return plotRange{-y1, -y2}, true
			}

			rpb := max(t.Model.RowsPerBeat().Value(), 1)
			xticks := func(r plotRange, count int, yield func(pos float32, label string)) {
				l := s.Model.LengthInBeats().Value() * rpb
				a := max(int(math.Ceil(float64(r.a*float32(l)))), 0)
				b := min(int(math.Floor(float64(r.b*float32(l)))), l)
				step := 1
				n := rpb
				for (b-a+1)/step > count {
					step *= n
					n = 2
				}
				a = (a / step) * step
				for i := a; i <= b; i += step {
					if i%rpb == 0 {
						beat := i / rpb
						yield(float32(i)/float32(l), strconv.Itoa(beat))
					} else {
						yield(float32(i)/float32(l), "")
					}
				}
			}
			yticks := func(r plotRange, count int, yield func(pos float32, label string)) {
				yield(-1, "")
				yield(1, "")
			}

			return s.State.plot.Layout(gtx, data, xticks, yticks, cx, 2)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(leftSpacer),
				layout.Rigid(Label(s.Theme, &s.Theme.SongPanel.RowHeader, "Trigger").Layout),
				layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
				layout.Rigid(onceBtn.Layout),
				layout.Rigid(triggerChannel.Layout),
				layout.Rigid(rightSpacer),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(leftSpacer),
				layout.Rigid(Label(s.Theme, &s.Theme.SongPanel.RowHeader, "Buffer").Layout),
				layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
				layout.Rigid(wrapBtn.Layout),
				layout.Rigid(lengthInBeats.Layout),
				layout.Rigid(rightSpacer),
			)
		}),
	)
}
