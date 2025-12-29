package tracker

import (
	"math"
	"math/cmplx"

	"github.com/viterin/vek/vek32"
	"github.com/vsariola/sointu"
)

type (
	SpecAnalyzer struct {
		settings SpecAnSettings
		broker   *Broker
		chunker  chunker
		temp     specTemp
	}

	SpecAnSettings struct {
		ChnMode    SpecChnMode
		Smooth     int
		Resolution int
	}

	SpecChnMode int
	Spectrum    [2][]float32

	specTemp struct {
		power      [2][]float32
		window     []float32    // window weighting function
		normFactor float32      // normalization factor, to account for the windowing
		bitPerm    []int        // bit-reversal permutation table
		tmpC       []complex128 // temporary buffer for FFT
		tmp1, tmp2 []float32    // temporary buffers for processing
	}

	BiquadCoeffs struct {
		b0, b1, b2 float32
		a0, a1, a2 float32
	}

	SpecAnEnabled Model
)

const (
	SpecResolutionMin = -3
	SpecResolutionMax = 3
)

const (
	SpecSpeedMin = -3
	SpecSpeedMax = 3
)

const (
	SpecChnModeSum      SpecChnMode = iota // calculate a single combined spectrum for both channels
	SpecChnModeSeparate                    // calculate separate spectrums for left and right channels
	NumSpecChnModes
)

func (m *Model) SpecAnEnabled() Bool { return MakeEnabledBool((*simpleBool)(&m.specAnEnabled)) }

func NewSpecAnalyzer(broker *Broker) *SpecAnalyzer {
	ret := &SpecAnalyzer{broker: broker}
	ret.init(SpecAnSettings{})
	return ret
}

func (m *Model) BiquadCoeffs() (coeffs BiquadCoeffs, ok bool) {
	i := m.d.InstrIndex
	u := m.d.UnitIndex
	if i < 0 || i >= len(m.d.Song.Patch) || u < 0 || u >= len(m.d.Song.Patch[i].Units) {
		return BiquadCoeffs{}, false
	}
	switch m.d.Song.Patch[i].Units[u].Type {
	case "filter":
		p := m.d.Song.Patch[i].Units[u].Parameters
		f := float32(p["frequency"]) / 128
		f *= f
		r := float32(p["resonance"]) / 128
		// The equations for the filter are:
		//   s1[n+1] = s1[n] + f*s2[n]
		//   h = u - s1[n+1] - r*s2[n]
		//   s2[n+1] = s2[n] + f*h = s2[n] + f*(u-s1[n]-f*s2[n]-r*s2[n]) = - f*s1[n]+(1-f*r-f*f)*s2[n] + f*u
		//   y_low[n] = s1[n+1], y_band[n] = s2[n+1], y_high[n] = -s1[n+1]-r*s2[n]+u
		// This gives state space representation
		//   s(n+1) = A*s(n)+B*u, where A = [1 f;-f 1-f*r-f*f] and B = [0;f]
		//   y(n) = C*s(n)+D*u, where
		//   C_low = [z 0], C_band = [0 z], C_high = [-z -r], D_high = [1] (note we use those z:s in C to account for those 1 sample time shifts)
		// The transfer function is then H(z) = C*(zI-A)^-1*B + D
		//   z*I-A = [z-1 -f; f z+f*r+f*f-1]
		// Calculate (zI-A)^-1*B:
		//   (z*I-A)^-1*B = 1/det * [z+f*r+f*f-1 f; -f z-1] * [0;f] = 1/det * f * [f; z-1], where
		//     det = (z+f*r+f*f-1)*(z-1)+f^2 = z*z+z*f*r+z*f*f-z-z-f*r-f*f+1+f^2 = z*z + (r*f+f*f-2)*z + 1-f*r = a0*z^2 + a1*z + a2
		//   Low: [z 0]*f*[f;z-1] / det = f*f*z / det = b1 * z / det
		//   Band: [0 z]*f*[f;z-1] / det = (f*z^2-f*z) / det = (b0*z^2 + b1*z) / det
		//   High: [-z -r]*f*[f;z-1] / det + 1 = ((-f*f-r*f)*z+r*f)/det + 1 = ((-f*f-r*f)*z+r*f+det)/det = (z^2-2*z+1)/det = (b0*z^2 + b1*z + b2)/det
		// Negative versions have only b coefficients negated
		var a0 float32 = 1
		var a1 float32 = r*f + f*f - 2
		var a2 float32 = 1 - f*r
		var b0, b1, b2 float32
		b1 += f * f * float32(p["lowpass"])
		b0 += f * float32(p["bandpass"])
		b1 -= f * float32(p["bandpass"])
		b0 += float32(p["highpass"])
		b1 += -2 * float32(p["highpass"])
		b2 += float32(p["highpass"])
		return BiquadCoeffs{a0: a0, a1: a1, a2: a2, b0: b0, b1: b1, b2: b2}, true
	case "belleq":
		f := float32(m.d.Song.Patch[i].Units[u].Parameters["frequency"]) / 128
		band := float32(m.d.Song.Patch[i].Units[u].Parameters["bandwidth"]) / 128
		gain := float32(m.d.Song.Patch[i].Units[u].Parameters["gain"]) / 128
		omega0 := 2 * f * f
		alpha := float32(math.Sin(float64(omega0))) * 2 * band
		A := float32(math.Pow(2, float64(gain-.5)*6.643856189774724))
		u, v := alpha*A, alpha/A
		return BiquadCoeffs{
			b0: 1 + u,
			b1: -2 * float32(math.Cos(float64(omega0))),
			b2: 1 - u,
			a0: 1 + v,
			a1: -2 * float32(math.Cos(float64(omega0))),
			a2: 1 - v,
		}, true
	default:
		return BiquadCoeffs{}, false
	}
}

func (c *BiquadCoeffs) Gain(omega float32) float32 {
	e := cmplx.Rect(1, -float64(omega))
	return float32(cmplx.Abs((complex(float64(c.b0), 0) + complex(float64(c.b1), 0)*e + complex(float64(c.b2), 0)*(e*e)) /
		(complex(float64(c.a0), 0) + complex(float64(c.a1), 0)*e + complex(float64(c.a2), 0)*e*e)))
}

func (s *SpecAnalyzer) Run() {
	for {
		select {
		case <-s.broker.CloseSpecAn:
			close(s.broker.FinishedSpecAn)
			return
		case msg := <-s.broker.ToSpecAn:
			s.handleMsg(msg)
		}
	}
}

func (s *SpecAnalyzer) handleMsg(msg MsgToSpecAn) {
	if msg.HasSettings {
		s.init(msg.SpecSettings)
	}
	switch m := msg.Data.(type) {
	case *sointu.AudioBuffer:
		buf := *m
		l := len(s.temp.window)
		// 50% overlap with the windows
		s.chunker.Process(buf, l, l>>1, func(chunk sointu.AudioBuffer) {
			TrySend(s.broker.ToModel, MsgToModel{Data: s.update(chunk)})
		})
		s.broker.PutAudioBuffer(m)
	default:
		// unknown message type; ignore
	}
}

func (a *SpecAnalyzer) init(s SpecAnSettings) {
	s.Resolution = min(max(s.Resolution, SpecResolutionMin), SpecResolutionMax) + 10
	a.settings = s
	n := 1 << s.Resolution
	a.temp = specTemp{
		power:   [2][]float32{make([]float32, n/2), make([]float32, n/2)},
		window:  make([]float32, n),
		bitPerm: make([]int, n),
		tmpC:    make([]complex128, n),
		tmp1:    make([]float32, n),
		tmp2:    make([]float32, n),
	}
	for i := range n {
		// Hanning window
		w := float32(0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1))))
		a.temp.window[i] = w
		a.temp.normFactor += w
		// initialize the bit-reversal permutation table
		a.temp.bitPerm[i] = i
	}
	// compute the bit-reversal permutation
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit

		if i < j {
			a.temp.bitPerm[i], a.temp.bitPerm[j] = a.temp.bitPerm[j], a.temp.bitPerm[i]
		}
	}
}

func (s *SpecAnalyzer) update(buf sointu.AudioBuffer) *Spectrum {
	ret := s.broker.GetSpectrum()
	switch s.settings.ChnMode {
	case SpecChnModeSeparate:
		s.process(buf, 0)
		s.process(buf, 1)
		ret[0] = append(ret[0], s.temp.power[0]...)
		ret[1] = append(ret[1], s.temp.power[1]...)
	case SpecChnModeSum:
		s.process(buf, 0)
		s.process(buf, 1)
		ret[0] = append(ret[0], s.temp.power[0]...)
		vek32.Add_Inplace(ret[0], s.temp.power[1])
	}
	// convert to decibels
	for c := range 2 {
		vek32.Log10_Inplace(ret[c])
		vek32.MulNumber_Inplace(ret[c], 10)
	}
	return ret
}

func (sd *SpecAnalyzer) process(buf sointu.AudioBuffer, channel int) {
	for i := range buf { // de-interleave
		sd.temp.tmp1[i] = removeNaNsAndClamp(buf[i][channel])
	}
	vek32.Mul_Inplace(sd.temp.tmp1, sd.temp.window)                // apply windowing
	vek32.Gather_Into(sd.temp.tmp2, sd.temp.tmp1, sd.temp.bitPerm) // bit-reversal permutation
	// convert into complex numbers
	c := sd.temp.tmpC
	for i := range c {
		c[i] = complex(float64(sd.temp.tmp2[i]), 0)
	}
	// FFT
	n := len(c)
	for len := 2; len <= n; len <<= 1 {
		ang := 2 * math.Pi / float64(len)
		wlen := complex(math.Cos(ang), math.Sin(ang))
		for i := 0; i < n; i += len {
			w := complex(1, 0)
			for j := 0; j < len/2; j++ {
				u := c[i+j]
				v := c[i+j+len/2] * w
				c[i+j] = u + v
				c[i+j+len/2] = u - v
				w *= wlen
			}
		}
	}
	// take absolute values of the first half, including nyquist frequency but excluding DC
	m := n / 2
	t1 := sd.temp.tmp1[:m]
	t2 := sd.temp.tmp2[:m]
	for i := 0; i < m; i++ {
		t1[i] = float32(cmplx.Abs(c[1+i])) // do not include DC
	}
	// square the amplitudes to get power
	vek32.Mul_Into(t2, t1, t1)
	vek32.DivNumber_Inplace(t2, sd.temp.normFactor*sd.temp.normFactor) // normalize for windowing
	// Since we are using a real-valued FFT, we need to double the values except for Nyquist (and DC, but we don't have that here)
	vek32.MulNumber_Inplace(t2[:m-1], 2)
	// calculate difference to current spectrum and add back, multiplied by smoothing factor
	vek32.Sub_Inplace(t2, sd.temp.power[channel])
	alpha := float32(math.Pow(2, float64(sd.settings.Smooth-SpecSpeedMax)))
	vek32.MulNumber_Inplace(t2, alpha)
	vek32.Add_Inplace(sd.temp.power[channel], t2)
}
