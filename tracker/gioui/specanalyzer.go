package gioui

import (
	"fmt"
	"math"
	"strconv"

	"gioui.org/layout"
	"gioui.org/unit"
	"github.com/vsariola/sointu/tracker"
)

type (
	SpectrumState struct {
		resolutionNumber *NumericUpDownState
		speed            *NumericUpDownState
		chnModeBtn       *Clickable
		plot             *Plot
	}
)

const (
	SpectrumDbMin = -60
	SpectrumDbMax = 12
)

func NewSpectrumState() *SpectrumState {
	return &SpectrumState{
		plot:             NewPlot(plotRange{-3.8, 0}, plotRange{SpectrumDbMax, SpectrumDbMin}, SpectrumDbMin),
		resolutionNumber: NewNumericUpDownState(),
		speed:            NewNumericUpDownState(),
		chnModeBtn:       new(Clickable),
	}
}

func (s *SpectrumState) Layout(gtx C) D {
	s.Update(gtx)
	t := TrackerFromContext(gtx)
	leftSpacer := layout.Spacer{Width: unit.Dp(6), Height: unit.Dp(36)}.Layout
	rightSpacer := layout.Spacer{Width: unit.Dp(6)}.Layout

	var chnModeTxt string = "???"
	switch tracker.SpecChnMode(t.Model.Spectrum().Channels().Value()) {
	case tracker.SpecChnModeSum:
		chnModeTxt = "Sum"
	case tracker.SpecChnModeSeparate:
		chnModeTxt = "Separate"
	}

	resolution := NumUpDown(t.Model.Spectrum().Resolution(), t.Theme, s.resolutionNumber, "Resolution")
	chnModeBtn := Btn(t.Theme, &t.Theme.Button.Text, s.chnModeBtn, chnModeTxt, "Channel mode")
	speed := NumUpDown(t.Model.Spectrum().Speed(), t.Theme, s.speed, "Speed")

	numchns := 0
	speclen := len(t.Model.Spectrum().Result()[0])
	if speclen > 0 {
		numchns = 1
		if len(t.Model.Spectrum().Result()[1]) == speclen {
			numchns = 2
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D {
			biquad, biquadok := t.Model.Spectrum().BiquadCoeffs()
			data := func(chn int, xr plotRange) (yr plotRange, ok bool) {
				if chn == 2 {
					if xr.a >= 0 {
						return plotRange{}, false
					}
					ya := math.Log10(float64(biquad.Gain(float32(math.Pi*math.Pow(10, float64(xr.a)))))) * 20
					yb := math.Log10(float64(biquad.Gain(float32(math.Pi*math.Pow(10, float64(xr.b)))))) * 20
					return plotRange{float32(ya), float32(yb)}, true
				}
				if chn >= numchns {
					return plotRange{}, false
				}
				xr.a = float32(math.Pow(10, float64(xr.a)))
				xr.b = float32(math.Pow(10, float64(xr.b)))
				w1, f1 := math.Modf(float64(xr.a)*float64(speclen) - 1) // -1 cause we don't have the DC bin there
				w2, f2 := math.Modf(float64(xr.b)*float64(speclen) - 1) // -1 cause we don't have the DC bin there
				x1 := max(int(w1), 0)
				x2 := min(int(w2), speclen-1)
				if x1 > x2 {
					return plotRange{}, false
				}
				y1 := float32(math.Inf(-1))
				y2 := float32(math.Inf(+1))
				switch {
				case x2 <= x1+1 && x2 < speclen-1: // perform smoothstep interpolation when we are overlapping only a few bins
					l := t.Model.Spectrum().Result()[chn][x1]
					r := t.Model.Spectrum().Result()[chn][x1+1]
					y1 = smoothInterpolate(l, r, float32(f1))
					l = t.Model.Spectrum().Result()[chn][x2]
					r = t.Model.Spectrum().Result()[chn][x2+1]
					y2 = smoothInterpolate(l, r, float32(f2))
					y1, y2 = max(y1, y2), min(y1, y2)
				default:
					for i := x1; i <= x2; i++ {
						sample := t.Model.Spectrum().Result()[chn][i]
						y1 = max(y1, sample)
						y2 = min(y2, sample)
					}
				}
				y1 = softplus((y1-SpectrumDbMin)/5)*5 + SpectrumDbMin // we "squash" the low volumes so the -Inf dB becomes -SpectrumDbMin
				y2 = softplus((y2-SpectrumDbMin)/5)*5 + SpectrumDbMin

				return plotRange{y1, y2}, true
			}
			xticks := func(r plotRange, count int, yield func(pos float32, label string)) {
				type pair struct {
					freq  float64
					label string
				}
				const offset = 0.343408593803857 // log10(22050/10000)
				const startdiv = 3 * (1 << 8)
				step := nextPowerOfTwo(int(float64(r.b-r.a)*startdiv/float64(count)) + 1)
				start := int(math.Floor(float64(r.a+offset) * startdiv / float64(step)))
				end := int(math.Ceil(float64(r.b+offset) * startdiv / float64(step)))
				for i := start; i <= end; i++ {
					lognormfreq := float32(i*step)/startdiv - offset
					freq := math.Pow(10, float64(lognormfreq)) * 22050
					df := freq * math.Log(10) * float64(step) / startdiv // this is roughly the difference in Hz between the ticks currently
					rounding := int(math.Floor(math.Log10(df)))
					r := math.Pow(10, float64(rounding))
					freq = math.Round(freq/r) * r
					tickpos := float32(math.Log10(freq / 22050))
					if rounding >= 3 {
						yield(tickpos, fmt.Sprintf("%.0f kHz", freq/1000))
					} else {
						yield(tickpos, fmt.Sprintf("%s Hz", strconv.FormatFloat(freq, 'f', -rounding, 64)))
					}
				}
			}
			yticks := func(r plotRange, count int, yield func(pos float32, label string)) {
				step := 3
				var start, end int
				for {
					start = int(math.Ceil(float64(r.b) / float64(step)))
					end = int(math.Floor(float64(r.a) / float64(step)))
					if end-start+1 <= count*4 { // we use 4x density for the y-lines in the spectrum
						break
					}
					step *= 2
				}
				for i := start; i <= end; i++ {
					yield(float32(i*step), strconv.Itoa(i*step))
				}
			}
			n := numchns
			if biquadok {
				n = 3
			}
			return s.plot.Layout(gtx, data, xticks, yticks, float32(math.NaN()), n)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(leftSpacer),
				layout.Rigid(Label(t.Theme, &t.Theme.SongPanel.RowHeader, "Resolution").Layout),
				layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
				layout.Rigid(resolution.Layout),
				layout.Rigid(rightSpacer),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(leftSpacer),
				layout.Rigid(Label(t.Theme, &t.Theme.SongPanel.RowHeader, "Speed").Layout),
				layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
				layout.Rigid(speed.Layout),
				layout.Rigid(rightSpacer),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(leftSpacer),
				layout.Rigid(Label(t.Theme, &t.Theme.SongPanel.RowHeader, "Channels").Layout),
				layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
				layout.Rigid(chnModeBtn.Layout),
				layout.Rigid(rightSpacer),
			)
		}),
	)
}

func softplus(f float32) float32 {
	return float32(math.Log(1 + math.Exp(float64(f))))
}

func smoothInterpolate(a, b float32, t float32) float32 {
	t = t * t * (3 - 2*t)
	return (1-t)*a + t*b
}

func nextPowerOfTwo(v int) int {
	if v <= 0 {
		return 1
	}
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v |= v >> 32
	v++
	return v
}

func (s *SpectrumState) Update(gtx C) {
	t := TrackerFromContext(gtx)
	for s.chnModeBtn.Clicked(gtx) {
		t.Model.Spectrum().Channels().SetValue((t.Model.Spectrum().Channels().Value() + 1) % int(tracker.NumSpecChnModes))
	}
	s.resolutionNumber.Update(gtx, t.Model.Spectrum().Resolution())
	s.speed.Update(gtx, t.Model.Spectrum().Speed())
}
