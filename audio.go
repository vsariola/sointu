package sointu

// AudioSink represents something where we can send audio e.g. audio output.
// WriteAudio should block if not ready to accept audio e.g. buffer full.
type AudioSink interface {
	WriteAudio(buffer []float32) error
	Close() error
}

// AudioContext represents the low-level audio drivers. There should be at most
// one AudioContext at a time. The interface is implemented at least by
// oto.OtoContext, but in future we could also mock it.
//
// AudioContext is used to create one or more AudioSinks with Output(); each can
// be used to output separate sound & closed when done.
type AudioContext interface {
	Output() AudioSink
	Close() error
}
