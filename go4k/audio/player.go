package audio

type Player interface {
	Play(buffer []float32) (err error)
	Close()
}
