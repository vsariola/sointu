package tracker

import (
	"errors"
	"math"
	"sync"

	"github.com/vsariola/sointu"
)

type (
	SignalAnalyzer struct {
		pool sync.Pool
		// these should be only used in the GUI thread
		avgVolume  Volume
		peakVolume Volume
		waveForm   *sointu.AudioBuffer

		resultChan  chan SignalResultMsg
		processChan chan SignalProcessMsg

		// these should be only used in the signal analyzer goroutine
		avgAnalyzer, peakAnalyzer VolumeAnalyzer
		triggering                bool
		length                    int
		skipping                  int
		skipIndex                 int
	}

	Volume [2]float64

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
)

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

// SetTriggering is thread safe
func (s *SignalAnalyzer) SetTriggering(value bool) {
	select {
	case s.processChan <- SignalProcessMsg{action: func() {
		s.triggering = value
	}}:
	default:
	}
}

// SetTriggering is thread safe
func (s *SignalAnalyzer) SetLength(length int) {
	select {
	case s.processChan <- SignalProcessMsg{action: func() {
		s.length = length
	}}:
	default:
	}
}

// SetTriggering is thread safe
func (s *SignalAnalyzer) SetSkipping(skipping int) {
	select {
	case s.processChan <- SignalProcessMsg{action: func() {
		if skipping >= 0 {
			s.skipping = skipping
		}
	}}:
	default:
	}
}

// Trigger is thread safe
func (s *SignalAnalyzer) Trigger() {
	select {
	case s.processChan <- SignalProcessMsg{trigger: true}:
	default:
	}
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

// Close must be called to stop the signal analyzer goroutine
func (s *SignalAnalyzer) Close() {
	close(s.processChan)
}

// This should be called only in the GUI thread
func (s *SignalAnalyzer) Update(msg SignalResultMsg) {
	s.avgVolume = msg.avgVolume
	s.peakVolume = msg.peakVolume
	if msg.waveForm != nil {
		if s.waveForm != nil {
			s.pool.Put(s.waveForm)
		}
		s.waveForm = msg.waveForm
	}
}

var nanError = errors.New("NaN detected in master output")

// Update updates the Level field, by analyzing the given buffer.
//
// Internally, it first converts the signal to decibels (0 dB = +-1). Then, the
// average volume level is computed by smoothing the decibel values with a
// exponentially decaying average, with a time constant Attack (in seconds) if
// the decibel value is greater than current level and time constant Decay (in
// seconds) if the decibel value is less than current level.
//
// Typical time constants for average level detection would be 0.3 seconds for
// both attack and release. For peak level detection, attack could be 1.5e-3 and
// release 1.5 (seconds)
//
// MinVolume and MaxVolume are hard limits in decibels to prevent negative
// infinities for volumes
func (v *VolumeAnalyzer) Update(buffer sointu.AudioBuffer) (err error) {
	// from https://en.wikipedia.org/wiki/Exponential_smoothing
	alphaAttack := 1 - math.Exp(-1.0/(v.Attack*44100))
	alphaRelease := 1 - math.Exp(-1.0/(v.Release*44100))
	for j := 0; j < 2; j++ {
		for i := 0; i < len(buffer); i++ {
			sample2 := float64(buffer[i][j] * buffer[i][j])
			if math.IsNaN(sample2) {
				if err == nil {
					err = nanError
				}
				continue
			}
			dB := 10 * math.Log10(sample2)
			if dB < v.Min || math.IsNaN(dB) {
				dB = v.Min
			}
			if dB > v.Max {
				dB = v.Max
			}
			a := alphaAttack
			if dB < v.Level[j] {
				a = alphaRelease
			}
			v.Level[j] += (dB - v.Level[j]) * a
		}
	}
	return err
}
