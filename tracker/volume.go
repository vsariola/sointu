package tracker

import (
	"errors"
	"math"

	"github.com/vsariola/sointu"
)

// Volume represents an average and peak volume measurement, in decibels. 0 dB =
// signal level of +-1.
type Volume struct {
	Average [2]float64
	Peak    [2]float64
}

// Analyze updates Average and Peak fields, by analyzing the given buffer.
//
// Internally, it first converts the signal to decibels (0 dB = +-1). Then, the
// average volume level is computed by smoothing the decibel values with a
// exponentially decaying average, with a time constant tau (in seconds).
// Typical value could be 0.3 (seconds).
//
// Peak volume detection is similar exponential smoothing, but the time
// constants for attack and release are different. Generally attack << release.
// Typical values could be attack 1.5e-3 and release 1.5 (seconds)
//
// minVolume and maxVolume are hard limits in decibels to prevent negative
// infinities for volumes
func (v *Volume) Analyze(buffer sointu.AudioBuffer, tau float64, attack float64, release float64, minVolume float64, maxVolume float64) error {
	alpha := 1 - math.Exp(-1.0/(tau*44100)) // from https://en.wikipedia.org/wiki/Exponential_smoothing
	alphaAttack := 1 - math.Exp(-1.0/(attack*44100))
	alphaRelease := 1 - math.Exp(-1.0/(release*44100))
	var err error
	for j := 0; j < 2; j++ {
		for i := 0; i < len(buffer); i++ {
			sample2 := float64(buffer[i][j] * buffer[i][j])
			if math.IsNaN(sample2) {
				if err == nil {
					err = errors.New("NaN detected in master output")
				}
				continue
			}
			dB := 10 * math.Log10(float64(sample2))
			if dB < minVolume || math.IsNaN(dB) {
				dB = minVolume
			}
			if dB > maxVolume {
				dB = maxVolume
			}
			v.Average[j] += (dB - v.Average[j]) * alpha
			alphaPeak := alphaAttack
			if dB < v.Peak[j] {
				alphaPeak = alphaRelease
			}
			v.Peak[j] += (dB - v.Peak[j]) * alphaPeak
		}
	}
	return err
}
