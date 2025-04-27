package tracker

import (
	"math"

	"github.com/viterin/vek/vek32"
	"github.com/vsariola/sointu"
)

type (
	Detector struct {
		broker           *Broker
		loudnessDetector loudnessDetector
		peakDetector     peakDetector
	}

	WeightingType int
	LoudnessType  int
	PeakType      int

	Decibel float32

	LoudnessResult [NumLoudnessTypes]Decibel
	PeakResult     [NumPeakTypes][2]Decibel

	DetectorResult struct {
		Loudness LoudnessResult
		Peaks    PeakResult
	}

	loudnessDetector struct {
		weighting       weighting
		states          [2][3]biquadState
		powers          [2]RingBuffer[float32] // 0 = momentary, 1 = short-term
		averagedPowers  [2][]float32
		maxPowers       [2]float32
		integratedPower float32
		tmp, tmp2       []float32
		tmpbool         []bool
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
		states       [2]oversamplerState
		windows      [2][2]RingBuffer[float32]
		maxPower     [2]float32
		tmp, tmp2    []float32
	}

	oversamplerState struct {
		history   [11]float32
		tmp, tmp2 []float32
	}
)

const (
	LoudnessMomentary LoudnessType = iota
	LoudnessShortTerm
	LoudnessMaxMomentary
	LoudnessMaxShortTerm
	LoudnessIntegrated
	NumLoudnessTypes
)

const MAX_INTEGRATED_DATA = 10 * 60 * 60 // 1 hour of samples at 10 Hz (100 ms per sample)

const (
	PeakMomentary PeakType = iota
	PeakShortTerm
	PeakIntegrated
	NumPeakTypes
)

const (
	KWeighting WeightingType = iota
	AWeighting
	CWeighting
	NoWeighting
	NumWeightingTypes
)

func NewDetector(b *Broker) *Detector {
	return &Detector{
		broker:           b,
		loudnessDetector: makeLoudnessDetector(KWeighting),
		peakDetector:     makePeakDetector(true),
	}
}

func (s *Detector) Run() {
	var chunkHistory sointu.AudioBuffer
	for msg := range s.broker.ToDetector {
		if msg.Reset {
			s.loudnessDetector.reset()
			s.peakDetector.reset()
		}
		if msg.Quit {
			return
		}
		if msg.HasWeightingType {
			s.loudnessDetector.weighting = weightings[WeightingType(msg.WeightingType)]
			s.loudnessDetector.reset()
		}
		if msg.HasOversampling {
			s.peakDetector.oversampling = msg.Oversampling
			s.peakDetector.reset()
		}

		switch data := msg.Data.(type) {
		case *sointu.AudioBuffer:
			buf := *data
			for {
				var chunk sointu.AudioBuffer
				if len(chunkHistory) > 0 && len(chunkHistory) < 4410 {
					l := min(len(buf), 4410-len(chunkHistory))
					chunkHistory = append(chunkHistory, buf[:l]...)
					if len(chunkHistory) < 4410 {
						break
					}
					chunk = chunkHistory
					buf = buf[l:]
				} else {
					if len(buf) >= 4410 {
						chunk = buf[:4410]
						buf = buf[4410:]
					} else {
						chunkHistory = chunkHistory[:0]
						chunkHistory = append(chunkHistory, buf...)
						break
					}
				}
				trySend(s.broker.ToModel, MsgToModel{
					HasDetectorResult: true,
					DetectorResult: DetectorResult{
						Loudness: s.loudnessDetector.update(chunk),
						Peaks:    s.peakDetector.update(chunk),
					},
				})
			}
			s.broker.PutAudioBuffer(data)
		case func():
			data()
		}
	}
}

// Close may theoretically block if the broker is full, but it should not happen in practice
func (s *Detector) Close() {
	s.broker.ToDetector <- MsgToDetector{Quit: true}
}

func makeLoudnessDetector(weighting WeightingType) loudnessDetector {
	return loudnessDetector{
		weighting: weightings[weighting],
		powers: [2]RingBuffer[float32]{
			{Buffer: make([]float32, 4)},  // momentary loudness
			{Buffer: make([]float32, 30)}, // short-term loudness
		},
	}
}

func makePeakDetector(oversampling bool) peakDetector {
	return peakDetector{
		oversampling: oversampling,
		windows: [2][2]RingBuffer[float32]{
			{{Buffer: make([]float32, 4)}, {Buffer: make([]float32, 4)}},   // momentary peaks
			{{Buffer: make([]float32, 30)}, {Buffer: make([]float32, 30)}}, // short-term peaks
		},
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

// according to https://tech.ebu.ch/docs/tech/tech3341.pdf
// we have two sliding windows: momentary loudness = last 400 ms, short-term loudness = last 3 s
// display:
//
//	momentary loudness = last analyzed 400 ms blcok
//	short-term loudness = last analyzed 3 s block
//
// every 100 ms, we collect one data point of the momentary loudness (starting to play song again resets the data blocks)
// then:
//
//	integrated loudness = the blocks are gated, and the average loudness of the gated blocks is calculated
//	maximum momentary loudness = maximum of all the momentary blocks
//	maximum short-term loudness = maximum of all the short-term blocks
func (d *loudnessDetector) update(chunk sointu.AudioBuffer) LoudnessResult {
	l := max(len(chunk), MAX_INTEGRATED_DATA)
	setSliceLength(&d.tmp, l)
	setSliceLength(&d.tmp2, l)
	setSliceLength(&d.tmpbool, l)
	var total float32
	for chn := 0; chn < 2; chn++ {
		// deinterleave the channels
		for i := 0; i < len(chunk); i++ {
			d.tmp[i] = chunk[i][chn]
		}
		// filter the signal with the weighting filter
		for k := 0; k < len(d.weighting.coeffs); k++ {
			d.states[chn][k].Filter(d.tmp[:len(chunk)], d.weighting.coeffs[k])
		}
		// square the samples
		res := vek32.Mul_Into(d.tmp2, d.tmp[:len(chunk)], d.tmp[:len(chunk)])
		// calculate the mean and add it to the total
		total += vek32.Mean(res)
	}
	var ret [NumLoudnessTypes]Decibel
	for i := range d.powers {
		d.powers[i].WriteWrapSingle(total) // these are sliding windows of 4 and 30 power measurements (400 ms and 3 s aka momentary and short-term windows)
		mean := vek32.Mean(d.powers[i].Buffer)
		if len(d.averagedPowers[i]) < MAX_INTEGRATED_DATA { // we need to have some limit on how much data we keep
			d.averagedPowers[i] = append(d.averagedPowers[i], mean)
		}
		if d.maxPowers[i] < mean {
			d.maxPowers[i] = mean
		}
		ret[i+int(LoudnessMomentary)] = power2loudness(mean, d.weighting.offset) // we assume the LoudnessMomentary is followed by LoudnessShortTerm
		ret[i+int(LoudnessMaxMomentary)] = power2loudness(d.maxPowers[i], d.weighting.offset)
	}
	if len(d.averagedPowers[0])%10 == 0 { // every 10 samples of 100 ms i.e. every 1 s, we recalculate the integrated power
		absThreshold := loudness2power(-70, d.weighting.offset) // -70 dB is the first threshold
		b := vek32.GtNumber_Into(d.tmpbool, d.averagedPowers[0], absThreshold)
		m2 := vek32.Select_Into(d.tmp, d.averagedPowers[0], b)
		if len(m2) > 0 {
			relThreshold := vek32.Mean(m2) / 10 // the relative threshold is 10 dB below the mean of the values above the absolute threshold
			b2 := vek32.GtNumber_Into(d.tmpbool, m2, relThreshold)
			m3 := vek32.Select_Into(d.tmp2, m2, b2)
			if len(m3) > 0 {
				d.integratedPower = vek32.Mean(m3)
			}
		}
	}
	ret[LoudnessIntegrated] = power2loudness(d.integratedPower, d.weighting.offset)
	return ret
}

func (d *loudnessDetector) reset() {
	for i := range d.powers {
		d.powers[i].Cursor = 0
		l := len(d.powers[i].Buffer)
		d.powers[i].Buffer = d.powers[i].Buffer[:0]
		d.powers[i].Buffer = append(d.powers[i].Buffer, make([]float32, l)...)
		d.averagedPowers[i] = d.averagedPowers[i][:0]
		d.maxPowers[i] = 0
	}
	d.integratedPower = 0
}

func power2loudness(power, offset float32) Decibel {
	return Decibel(float32(10*math.Log10(float64(power))) + offset)
}

func loudness2power(loudness Decibel, offset float32) float32 {
	return (float32)(math.Pow(10, (float64(loudness)-float64(offset))/10))
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

func setSliceLength[T any](slice *[]T, length int) {
	if len(*slice) < length {
		*slice = append(*slice, make([]T, length-len(*slice))...)
	}
	*slice = (*slice)[:length]
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
// y should be at least 4 times the length of x
func (s *oversamplerState) Oversample(x []float32, y []float32) []float32 {
	if len(s.tmp) < len(x) {
		s.tmp = append(s.tmp, make([]float32, len(x)-len(s.tmp))...)
	}
	if len(s.tmp2) < len(x) {
		s.tmp2 = append(s.tmp2, make([]float32, len(x)-len(s.tmp2))...)
	}
	for q, coeffs := range oversamplingCoeffs {
		// tmp2 will be conv(o[q],x)
		r := vek32.Zeros_Into(s.tmp2, len(x))
		for j, c := range coeffs {
			vek32.MulNumber_Into(s.tmp[:j], s.history[11-j:11], c) // convolution might pull values before x[0], so we need to use history for that
			vek32.MulNumber_Into(s.tmp[j:], x[:len(x)-j], c)
			vek32.Add_Inplace(r, s.tmp[:len(x)])
		}
		// interleave the phases
		for p, v := range r {
			y[p*4+q] = v
		}
	}
	z := min(len(x), 11)
	copy(s.history[:11-z], s.history[z:11])
	copy(s.history[11-z:], x[len(x)-z:])
	return y[:len(x)*4]
}

// we should perform the peak detection also momentary (last 400 ms), short term
// (last 3 s), and integrated (whole song) for display purposes, we can use
// always last arrived data for the integrated peak, we can use the maximum of
// all the peaks so far (there is no need show "maximum short term true peak" or
// "maximum momentary true peak" because they are same as the maximum for entire song)
//
// display:
//
//	momentary true peak
//	short-term true peak
//	integrated true peak
func (d *peakDetector) update(buf sointu.AudioBuffer) (ret PeakResult) {
	if len(d.tmp) < len(buf) {
		d.tmp = append(d.tmp, make([]float32, len(buf)-len(d.tmp))...)
	}
	len4 := 4 * len(buf)
	if len(d.tmp2) < len4 {
		d.tmp2 = append(d.tmp2, make([]float32, len4-len(d.tmp2))...)
	}
	for chn := range 2 {
		// deinterleave the channels
		for i := range buf {
			d.tmp[i] = buf[i][chn]
		}
		// 4x oversample the signal
		var o []float32
		if d.oversampling {
			o = d.states[chn].Oversample(d.tmp[:len(buf)], d.tmp2)
		} else {
			o = d.tmp[:len(buf)]
		}
		// take absolute value of the oversampled signal
		vek32.Abs_Inplace(o)
		p := vek32.Max(o)
		// find the maximum value in the window
		for i := range d.windows {
			d.windows[i][chn].WriteWrapSingle(p)
			windowPeak := vek32.Max(d.windows[i][chn].Buffer)
			ret[i+int(PeakMomentary)][chn] = Decibel(20 * math.Log10(float64(windowPeak)))
		}
		if d.maxPower[chn] < p {
			d.maxPower[chn] = p
		}
		ret[int(PeakIntegrated)][chn] = Decibel(20 * math.Log10(float64(d.maxPower[chn])))
	}
	return
}

func (d *peakDetector) reset() {
	for chn := range 2 {
		d.states[chn].history = [11]float32{}
		for i := range d.windows[chn] {
			d.windows[i][chn].Cursor = 0
			l := len(d.windows[i][chn].Buffer)
			d.windows[i][chn].Buffer = d.windows[i][chn].Buffer[:0]
			d.windows[i][chn].Buffer = append(d.windows[i][chn].Buffer, make([]float32, l)...)
		}
		d.maxPower[chn] = 0
	}
}
