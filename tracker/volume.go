package tracker

import (
	"math"
	"sync"
	"time"

	"github.com/viterin/vek/vek32"
	"github.com/vsariola/sointu"
)

type (
	SignalAnalyzer struct {
		loudness Decibel
		peak     [2]Decibel
		waveForm RingBuffer[[2]float32]

		pool            sync.Pool
		audioBufferChan chan *sointu.AudioBuffer
	}

	RingBuffer[T any] struct {
		buffer []T
		cursor int
	}

	WeightingType int

	Decibel float32

	SignalProcessMsg struct {
		trigger bool
		data    *sointu.AudioBuffer
		action  func()
	}

	SignalResultMsg struct {
		avgVolume  Volume
		peakVolume Volume
		waveForm   *sointu.AudioBuffer
	}

	// VolumeAnalyzer measures the volume in an AudioBuffer, in decibels relative to
	// full scale (0 dB = signal level of +-1)
	VolumeAnalyzer struct {
		Level   Volume  // current volume level of left and right channels
		Attack  float64 // attack time constant in seconds
		Release float64 // release time constant in seconds
		Min     float64 // minimum volume in decibels
		Max     float64 // maximum volume in decibels
	}

	loudnessDetector struct {
		weighting  weighting
		windowTime time.Duration
		states     [2][3]biquadState
		windows    [2]RingBuffer[float32]
		tmp, tmp2  []float32
	}

	biquadState struct {
		x1, x2, y1, y2 float32
	}

	biquadCoeff struct {
		b0, b1, b2, a1, a2 float32
	}

	weighting struct {
		coeffs []biquadCoeff
		offset float32
	}

	peakDetector struct {
		oversampling bool
		windowTime   time.Duration
		states       [2]oversamplerState
		windows      [2]RingBuffer[float32]
		tmp, tmp2    []float32
	}

	oversamplerState struct {
		history   [11]float32
		tmp, tmp2 []float32
	}
)

const (
	KWeighting WeightingType = iota
	AWeighting
	CWeighting
	NoWeighting
)

func (r *RingBuffer[T]) WriteWrap(values []T) {
	r.cursor = (r.cursor + len(values)) % len(r.buffer)
	a := min(len(values), r.cursor)                 // how many values to copy before the cursor
	b := min(len(values)-a, len(r.buffer)-r.cursor) // how many values to copy to the end of the buffer
	copy(r.buffer[r.cursor-a:r.cursor], values[len(values)-a:])
	copy(r.buffer[len(r.buffer)-b:], values[len(values)-a-b:])
}

func (a *SignalAnalyzer) Process(buffer sointu.AudioBuffer) {
	s := a.pool.Get().(*sointu.AudioBuffer)
	*s = (*s)[:0]
	*s = append(*s, buffer...)
	select {
	case a.audioBufferChan <- s:
	default:
		a.pool.Put(s)
	}
}

/*
From matlab:
f = getFilter(weightingFilter('A-weighting','SampleRate',44100)); f.Numerator, f.Denominator
for i = 1:size(f.Numerator,1); fprintf("b0: %.16f, b1: %.16f, b2: %.16f, a1: %.16f, a2: %.16f\n",f.Numerator(i,:),f.Denominator(i,2:end)); end
f = getFilter(weightingFilter('C-weighting','SampleRate',44100)); f.Numerator, f.Denominator
for i = 1:size(f.Numerator,1); fprintf("b0: %.16f, b1: %.16f, b2: %.16f, a1: %.16f, a2: %.16f\n",f.Numerator(i,:),f.Denominator(i,2:end)); end
f = getFilter(weightingFilter('k-weighting','SampleRate',44100)); f.Numerator, f.Denominator
for i = 1:size(f.Numerator,1); fprintf("b0: %.16f, b1: %.16f, b2: %.16f, a1: %.16f, a2: %.16f\n",f.Numerator(i,:),f.Denominator(i,2:end)); end
*/
var weightings = map[WeightingType]weighting{
	AWeighting: {coeffs: []biquadCoeff{
		{b0: 1, b1: 2, b2: 1, a1: -0.1405360824207108, a2: 0.0049375976155402},
		{b0: 1, b1: -2, b2: 1, a1: -1.8849012174287920, a2: 0.8864214718161675},
		{b0: 1, b1: -2, b2: 1, a1: -1.9941388812663283, a2: 0.9941474694445309},
	}, offset: 0},
	CWeighting: {coeffs: []biquadCoeff{
		{b0: 1, b1: 2, b2: 1, a1: -0.1405360824207108, a2: 0.0049375976155402},
		{b0: 1, b1: -2, b2: 1, a1: -1.9941388812663283, a2: 0.9941474694445309},
	}, offset: 0},
	KWeighting: {coeffs: []biquadCoeff{
		{b0: 1.5308412300503476, b1: -2.6509799951547293, b2: 1.1690790799215869, a1: -1.6636551132560204, a2: 0.7125954280732254},
		{b0: 0.9995600645425144, b1: -1.9991201290850289, b2: 0.9995600645425144, a1: -1.9891696736297957, a2: 0.9891990357870394},
	}, offset: -0.691}, // offset is to make up for the fact that K-weighting has slightly above unity gain at 1 kHz
	NoWeighting: {coeffs: []biquadCoeff{}, offset: 0},
}

func (d *loudnessDetector) update(buf sointu.AudioBuffer) Decibel {
	if len(d.tmp) < len(buf) {
		d.tmp = append(d.tmp, make([]float32, len(buf)-len(d.tmp))...)
	}
	sqLen := min(len(d.windows[0].buffer), len(buf)) // there's no need to square more samples than the window size
	if len(d.tmp2) < sqLen {
		d.tmp2 = append(d.tmp2, make([]float32, sqLen-len(buf))...)
	}
	var total float32
	for chn := 0; chn < 2; chn++ {
		// deinterleave the channels
		for i := 0; i < len(buf); i++ {
			d.tmp[i] = buf[i][chn]
		}
		// filter the signal with the weighting filter
		for k := 0; k < len(d.weighting.coeffs); k++ {
			d.states[chn][k].Filter(d.tmp[:len(buf)], d.weighting.coeffs[k])
		}
		// square the last sqLen samples of the signal
		vek32.Mul_Into(d.tmp2[:sqLen], d.tmp[len(buf)-sqLen:len(buf)], d.tmp[len(buf)-sqLen:len(buf)])
		// write the squared signal to the window
		d.windows[chn].WriteWrap(d.tmp2[:sqLen])
		total += vek32.Mean(d.windows[chn].buffer)
	}
	return Decibel(float32(10*math.Log10(float64(total))) + d.weighting.offset)
}

func (state *biquadState) Filter(buffer []float32, coeff biquadCoeff) {
	s := *state
	for i := 0; i < len(buffer); i++ {
		x := buffer[i]
		y := coeff.b0*x + coeff.b1*s.x1 + coeff.b2*s.x2 - coeff.a1*s.y1 - coeff.a2*s.y2
		s.x2, s.x1 = s.x1, x
		s.y2, s.y1 = s.y1, y
		buffer[i] = y
	}
	*state = s
}

// ref: https://www.itu.int/dms_pubrec/itu-r/rec/bs/R-REC-BS.1770-5-202311-I!!PDF-E.pdf
var oversamplingCoeffs = [4][12]float32{
	{0.0017089843750, 0.0109863281250, -0.0196533203125, 0.0332031250000, -0.0594482421875, 0.1373291015625, 0.9721679687500, -0.1022949218750, 0.0476074218750, -0.0266113281250, 0.0148925781250, -0.0083007812500},
	{-0.0291748046875, 0.0292968750000, -0.0517578125000, 0.0891113281250, -0.1665039062500, 0.4650878906250, 0.7797851562500, -0.2003173828125, 0.1015625000000, -0.0582275390625, 0.0330810546875, -0.0189208984375},
	{-0.0189208984375, 0.0330810546875, -0.058227539062, 0.1015625000000, -0.200317382812, 0.7797851562500, 0.4650878906250, -0.166503906250, 0.0891113281250, -0.051757812500, 0.0292968750000, -0.0291748046875},
	{-0.0083007812500, 0.0148925781250, -0.0266113281250, 0.0476074218750, -0.1022949218750, 0.9721679687500, 0.1373291015625, -0.0594482421875, 0.0332031250000, -0.0196533203125, 0.0109863281250, 0.0017089843750},
}

// u[k] = x[k/4] if k%4 == 0, 0 otherwise
// y[k] = sum_{i=0}^{47} h[i] * u[k-i]
// h[i] = o[i%4][i/4]
// k = p*4+q, q=0..3
// y[p*4+q] = sum_{j=0}^{11} sum_{i=0}^{3} h[j*4+i] * u[p*4+q-j*4-i] = ...
// (q-i)%4 == 0 ==> i = q
// ... = sum_{j=0}^{11} o[q][j] * x[p-j]
// y should be 4 times the length of x
func (s *oversamplerState) Oversample(x []float32, y []float32) {
	if len(s.tmp) < len(x) {
		s.tmp = append(s.tmp, make([]float32, len(x)-len(s.tmp))...)
	}
	s.tmp = s.tmp[:len(x)]
	if len(s.tmp2) < len(x) {
		s.tmp2 = append(s.tmp2, make([]float32, len(x)-len(s.tmp2))...)
	}
	s.tmp2 = s.tmp2[:len(x)]
	for q, coeffs := range oversamplingCoeffs {
		// tmp2 will be conv(o[q],x)
		vek32.Zeros_Into(s.tmp2, len(s.tmp2))
		for j, c := range coeffs {
			vek32.MulNumber_Into(s.tmp[:j], s.history[11-j:11], c) // convolution might pull values before x[0], so we need to use history for that
			vek32.MulNumber_Into(s.tmp[j:], x[:len(x)-j], c)
			vek32.Add_Inplace(s.tmp2, s.tmp)
		}
		// interleave the phases
		for p := range s.tmp2 {
			y[p*4+q] = s.tmp2[p]
		}
	}
	z := min(len(x), 11)
	copy(s.history[:11-z], s.history[z:11])
	copy(s.history[11-z:], x[len(x)-z:])
}

func (d *peakDetector) update(buf sointu.AudioBuffer) (ret [2]Decibel) {
	if len(d.tmp) < len(buf) {
		d.tmp = append(d.tmp, make([]float32, len(buf)-len(d.tmp))...)
	}
	d.tmp = d.tmp[:len(buf)]
	len4 := 4 * len(buf)
	if len(d.tmp2) < len4 {
		d.tmp2 = append(d.tmp2, make([]float32, len4-len(buf))...)
	}
	d.tmp2 = d.tmp2[:len4]
	absLen := min(len(d.windows[0].buffer), len(d.tmp2))
	for chn := 0; chn < 2; chn++ {
		// deinterleave the channels
		for i := 0; i < len(buf); i++ {
			d.tmp[i] = buf[i][chn]
		}
		// 4x oversample the signal
		d.states[chn].Oversample(d.tmp, d.tmp2)
		// take absolute value of the oversampled signal
		a := d.tmp2[len(d.tmp2)-absLen : len(d.tmp2)]
		vek32.Abs_Inplace(a)
		d.windows[chn].WriteWrap(a)
		// find the maximum value in the window
		max := vek32.Max(d.windows[chn].buffer)
		ret[chn] = Decibel(float32(20 * math.Log10(float64(max))))
	}
	return
}

func NewSignalAnalyzer() *SignalAnalyzer {
	s := &SignalAnalyzer{pool: sync.Pool{
		New: func() any {
			s := make(sointu.AudioBuffer, 0)
			return &s
		},
	},
		resultChan:   make(chan SignalResultMsg, 16),
		processChan:  make(chan SignalProcessMsg, 16),
		avgAnalyzer:  VolumeAnalyzer{Attack: 0.3, Release: 0.3, Min: -100, Max: 20},
		peakAnalyzer: VolumeAnalyzer{Attack: 1e-4, Release: 1, Min: -100, Max: 20},
		length:       4096,
		skipping:     20,
	}
	go func() {
		waveform := make(sointu.AudioBuffer, 0, 44100)
		for msg := range s.processChan {
			if msg.trigger && s.triggering {
				waveform = waveform[:0]
			}
			if msg.action != nil {
				msg.action()
			}
			var result *sointu.AudioBuffer = nil
			if msg.data != nil {
				s.avgAnalyzer.Update(*msg.data)
				s.peakAnalyzer.Update(*msg.data)
				j := 0
				for i := 0; i < len(*msg.data); i++ {
					if s.skipIndex > 0 {
						s.skipIndex--
						continue
					}
					s.skipIndex = s.skipping
					(*msg.data)[j] = (*msg.data)[i]
					j++
				}
				*msg.data = (*msg.data)[:j]
				space := s.length - len(waveform)
				if s.triggering {
					if space <= 0 {
						goto skip
					}
				} else {
					missingSpace := len(*msg.data) - space
					if missingSpace > 0 {
						move := min(len(waveform), missingSpace)
						copy(waveform, waveform[move:])
						waveform = waveform[:len(waveform)-move]
						space += move
					}
				}
				if len(*msg.data) > space {
					*msg.data = (*msg.data)[:space]
				}
				waveform = append(waveform, *msg.data...)
				result = msg.data
				*result = (*result)[:0]
				*result = append(*result, waveform...)
			}
		skip:
			select {
			case s.resultChan <- SignalResultMsg{
				avgVolume:  s.avgAnalyzer.Level,
				peakVolume: s.peakAnalyzer.Level,
				waveForm:   result,
			}:
			default:
				if result != nil {
					s.pool.Put(result)
				}
			}
		}
	}()
	return s
}

// Process is thread safe
func (s *SignalAnalyzer) Process(buffer sointu.AudioBuffer) {
	buf := s.pool.Get().(*sointu.AudioBuffer)
	*buf = (*buf)[:0]
	*buf = append(*buf, buffer...)
	select {
	case s.processChan <- SignalProcessMsg{data: buf}:
	default:
		s.pool.Put(buf)
	}
}
