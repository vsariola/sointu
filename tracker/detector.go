package tracker

import (
	"math"

	"github.com/viterin/vek/vek32"
	"github.com/vsariola/sointu"
)

const MAX_INTEGRATED_DATA = 10 * 60 * 60 // 1 hour of samples at 10 Hz (100 ms per sample)
// In the detector, we clamp the signal levels to +-MAX_SIGNAL_AMPLITUDE to
// avoid Inf results. This is 240 dBFS. max float32 is about 3.4e38, so squaring
// the amplitude values gives 1e24, and adding 4410 of those together (when
// taking the mean) gives a value < 1e37, which is still < max float32.
const MAX_SIGNAL_AMPLITUDE = 1e12

// Detector returns a DetectorModel which provides access to the detector
// settings and results.
func (m *Model) Detector() *DetectorModel { return (*DetectorModel)(m) }

type DetectorModel Model

// Result returns the latest DetectorResult from the detector.
func (m *DetectorModel) Result() DetectorResult { return m.detectorResult }

type (
	DetectorResult struct {
		Loudness LoudnessResult
		Peaks    PeakResult
	}
	LoudnessResult [NumLoudnessTypes]Decibel
	PeakResult     [NumPeakTypes][2]Decibel
	Decibel        float32
	LoudnessType   int
	PeakType       int
)

const (
	LoudnessMomentary LoudnessType = iota
	LoudnessShortTerm
	LoudnessMaxMomentary
	LoudnessMaxShortTerm
	LoudnessIntegrated
	NumLoudnessTypes
)

const (
	PeakMomentary PeakType = iota
	PeakShortTerm
	PeakIntegrated
	NumPeakTypes
)

// Weighting returns an Int property for setting the detector weighting type.
func (m *DetectorModel) Weighting() Int { return MakeInt((*detectorWeighting)(m)) }

type detectorWeighting Model

func (v *detectorWeighting) Value() int { return int(v.weightingType) }
func (v *detectorWeighting) SetValue(value int) bool {
	v.weightingType = WeightingType(value)
	TrySend(v.broker.ToDetector, MsgToDetector{HasWeightingType: true, WeightingType: WeightingType(value)})
	return true
}
func (v *detectorWeighting) Range() RangeInclusive {
	return RangeInclusive{0, int(NumWeightingTypes) - 1}
}
func (v *detectorWeighting) StringOf(value int) string {
	switch WeightingType(value) {
	case KWeighting:
		return "K-weighting (LUFS)"
	case AWeighting:
		return "A-weighting"
	case CWeighting:
		return "C-weighting"
	case NoWeighting:
		return "No weighting (RMS)"
	default:
		return "Unknown"
	}
}

type WeightingType int

const (
	KWeighting WeightingType = iota
	AWeighting
	CWeighting
	NoWeighting
	NumWeightingTypes
)

// Oversampling returns a Bool property for setting whether the peak detector
// uses oversampling to calculate true peaks, or just sample peaks if not.
func (m *DetectorModel) Oversampling() Bool { return MakeBool((*detectorOversampling)(m)) }

type detectorOversampling Model

func (m *detectorOversampling) Value() bool { return m.oversampling }
func (m *detectorOversampling) SetValue(val bool) {
	m.oversampling = val
	TrySend(m.broker.ToDetector, MsgToDetector{HasOversampling: true, Oversampling: val})
}

type (
	detector struct {
		broker           *Broker
		loudnessDetector loudnessDetector
		peakDetector     peakDetector
		chunker          chunker
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

	weighting []biquadCoeff

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

func runDetector(b *Broker) {
	s := &detector{
		broker:           b,
		loudnessDetector: makeLoudnessDetector(KWeighting),
		peakDetector:     makePeakDetector(true),
	}
	for {
		select {
		case <-s.broker.CloseDetector:
			close(s.broker.FinishedDetector)
			return
		case msg := <-s.broker.ToDetector:
			s.handleMsg(msg)
		}
	}
}

func (s *detector) handleMsg(msg MsgToDetector) {
	if msg.Reset {
		s.loudnessDetector.reset()
		s.peakDetector.reset()
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
		s.chunker.Process(buf, 4410, 0, func(chunk sointu.AudioBuffer) {
			TrySend(s.broker.ToModel, MsgToModel{
				HasDetectorResult: true,
				DetectorResult: DetectorResult{
					Loudness: s.loudnessDetector.update(chunk),
					Peaks:    s.peakDetector.update(chunk),
				},
			})
		})
		s.broker.PutAudioBuffer(data)
	}
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
From matlab: (we bake in the scale values to the numerator coefficients)
weightings = {'A-weighting','C-weighting','k-weighting'}
for j = 1:3
disp(weightings{j})
f = getFilter(weightingFilter(weightings{j},'SampleRate',44100)); f.Numerator, f.Denominator, f.ScaleValues
if j == 3 % k-weighting has non-zero gain at 1 kHz, so normalize it to 0 dB by scaling the first filter
[h,w] = freqz(f,[1000,1000],44100);
g = abs(h(1));
fprintf("Gain %f dB\n", 20*log10(abs(h(1))));
f.Numerator(1,:) = f.Numerator(1,:)/g;
end
for i = 1:size(f.Numerator,1); fprintf("b0: %.16f, b1: %.16f, b2: %.16f, a1: %.16f, a2: %.16f\n",f.Numerator(i,:)*f.ScaleValues(i),f.Denominator(i,2:end)); end
end
*/
var weightings = map[WeightingType]weighting{
	AWeighting: {
		{b0: 0.2556115104436430, b1: 0.5112230208872860, b2: 0.2556115104436430, a1: -0.1405360824207108, a2: 0.0049375976155402},
		{b0: 1, b1: -2, b2: 1, a1: -1.8849012174287920, a2: 0.8864214718161675},
		{b0: 1, b1: -2, b2: 1, a1: -1.9941388812663283, a2: 0.9941474694445309},
	},
	CWeighting: {
		{b0: 0.2170124955461332, b1: 0.4340249910922664, b2: 0.2170124955461332, a1: -0.1405360824207108, a2: 0.0049375976155402},
		{b0: 1, b1: -2, b2: 1, a1: -1.9941388812663283, a2: 0.9941474694445309},
	},
	KWeighting: {
		{b0: 1.4128568659906546, b1: -2.4466647580657646, b2: 1.0789762991286349, a1: -1.6636551132560204, a2: 0.7125954280732254},
		{b0: 0.9995600645425144, b1: -1.9991201290850289, b2: 0.9995600645425144, a1: -1.9891696736297957, a2: 0.9891990357870394},
	},
	NoWeighting: {},
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
	for chn := range 2 {
		// deinterleave the channels
		for i := range chunk {
			d.tmp[i] = removeNaNsAndClamp(chunk[i][chn])
		}
		// filter the signal with the weighting filter
		for k := range d.weighting {
			d.states[chn][k].Filter(d.tmp[:len(chunk)], d.weighting[k])
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
		ret[i+int(LoudnessMomentary)] = powerToDecibel(mean) // we assume the LoudnessMomentary is followed by LoudnessShortTerm
		ret[i+int(LoudnessMaxMomentary)] = powerToDecibel(d.maxPowers[i])
	}
	if len(d.averagedPowers[0])%10 == 0 { // every 10 samples of 100 ms i.e. every 1 s, we recalculate the integrated power
		absThreshold := decibelToPower(-70) // -70 dB is the first threshold
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
	ret[LoudnessIntegrated] = powerToDecibel(d.integratedPower)
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
	// reset the biquad states
	d.states = [2][3]biquadState{}
	d.integratedPower = 0
}

func removeNaNsAndClamp(s float32) float32 {
	if s != s { // NaN
		return 0
	}
	return min(max(s, -MAX_SIGNAL_AMPLITUDE), MAX_SIGNAL_AMPLITUDE)
}

func powerToDecibel(power float32) Decibel {
	return Decibel(float32(10 * math.Log10(float64(power))))
}

func amplitudeToDecibel(amplitude float32) Decibel {
	return Decibel(float32(20 * math.Log10(float64(amplitude))))
}

func decibelToPower(loudness Decibel) float32 {
	return (float32)(math.Pow(10, (float64(loudness))/10))
}

func (state *biquadState) Filter(buffer []float32, coeff biquadCoeff) {
	s := *state
	for i := range buffer {
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
			d.tmp[i] = removeNaNsAndClamp(buf[i][chn])
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
			ret[i+int(PeakMomentary)][chn] = amplitudeToDecibel(windowPeak)
		}
		if d.maxPower[chn] < p {
			d.maxPower[chn] = p
		}
		ret[int(PeakIntegrated)][chn] = amplitudeToDecibel(d.maxPower[chn])
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

// chunker maintains a buffer of audio data. Its Process method appends an input
// buffer to the buffer and calls a callback function with chunks of specified
// length and overlap. The remaining data is kept in the buffer for the next
// call.
type chunker struct {
	buffer sointu.AudioBuffer
}

// Process appends input to the internal buffer and calls cb with chunks of
// windowLen length and overlap overlap. The remaining data is kept in the
// internal buffer.
func (c *chunker) Process(input sointu.AudioBuffer, windowLen, overlap int, cb func(sointu.AudioBuffer)) {
	c.buffer = append(c.buffer, input...)
	b := c.buffer
	for len(b) >= windowLen {
		cb(b[:windowLen])
		b = b[windowLen-overlap:]
	}
	copy(c.buffer, b)
	c.buffer = c.buffer[:len(b)]
}
