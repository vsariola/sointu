package tracker

import (
	"math"
)

// Volume represents an average and peak volume measurement, in decibels. 0 dB =
// signal level of +-1.
type Volume struct {
	Average [2]float32
	Peak    [2]float32
}

// VuAnalyzer receives stereo from the bc channel and converts these into peak &
// average volume measurements, and pushes Volume values into the vc channel.
// The pushes are nonblocking so if e.g. a GUI does not have enough time to
// process redraw the volume meter, the values is just skipped. Thus, the vc
// chan should have a capacity of at least 1 (!).
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
// minVolume is just a hard limit for the vuanalyzer volumes, in decibels, just to
// prevent negative infinities for volumes
func VuAnalyzer(tau float64, attack float64, release float64, minVolume float32, bc <-chan []float32, vc chan<- Volume) {
	v := Volume{Average: [2]float32{minVolume, minVolume}, Peak: [2]float32{minVolume, minVolume}}
	alpha := 1 - float32(math.Exp(-1.0/(tau*44100))) // from https://en.wikipedia.org/wiki/Exponential_smoothing
	alphaAttack := 1 - float32(math.Exp(-1.0/(attack*44100)))
	alphaRelease := 1 - float32(math.Exp(-1.0/(release*44100)))
	for buffer := range bc {
		for j := 0; j < 2; j++ {
			for i := 0; i < len(buffer); i += 2 {
				sample2 := float64(buffer[i+j] * buffer[i+j])
				if math.IsNaN(sample2) {
					sample2 = float64(minVolume)
				}
				dB := float32(10 * math.Log10(float64(sample2)))
				if dB < minVolume {
					dB = minVolume
				}
				v.Average[j] += (dB - v.Average[j]) * alpha
				alphaPeak := alphaAttack
				if dB < v.Peak[j] {
					alphaPeak = alphaRelease
				}
				v.Peak[j] += (dB - v.Peak[j]) * alphaPeak
			}
		}
		select {
		case vc <- v:
		default:
		}
	}
}
