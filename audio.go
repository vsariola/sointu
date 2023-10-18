package sointu

// AudioBuffer is a buffer of stereo audio samples of variable length, each
// sample represented by a slice of [2]float32. [0] is left channel, [1] is
// right
type AudioBuffer [][2]float32

// AudioOutput represents something where we can send audio e.g. audio output.
// WriteAudio should block if not ready to accept audio e.g. buffer full.
type AudioOutput interface {
	WriteAudio(buffer AudioBuffer) error
	Close() error
}

// AudioContext represents the low-level audio drivers. There should be at most
// one AudioContext at a time. The interface is implemented at least by
// oto.OtoContext, but in future we could also mock it.
//
// AudioContext is used to create one or more AudioOutputs with Output(); each
// can be used to output separate sound & closed when done.
type AudioContext interface {
	Output() AudioOutput
	Close() error
}
