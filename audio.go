package sointu

type AudioSink interface {
	WriteAudio(buffer []float32) error
	Close() error
}

type AudioContext interface {
	Output() AudioSink
	Close() error
}
