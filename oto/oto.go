package oto

import (
	"fmt"

	"github.com/hajimehoshi/oto"
	"github.com/vsariola/sointu"
)

type OtoContext oto.Context
type OtoOutput struct {
	player    *oto.Player
	tmpBuffer []byte
}

func (c *OtoContext) Output() sointu.AudioSink {
	return &OtoOutput{player: (*oto.Context)(c).NewPlayer(), tmpBuffer: make([]byte, 0)}
}

const otoBufferSize = 8192

// NewPlayer creates and initializes a new OtoPlayer
func NewContext() (*OtoContext, error) {
	context, err := oto.NewContext(44100, 2, 2, otoBufferSize)
	if err != nil {
		return nil, fmt.Errorf("cannot create oto context: %w", err)
	}
	return (*OtoContext)(context), nil
}

func (c *OtoContext) Close() error {
	if err := (*oto.Context)(c).Close(); err != nil {
		return fmt.Errorf("cannot close oto context: %w", err)
	}
	return nil
}

// Play implements the audio.Player interface for OtoPlayer
func (o *OtoOutput) WriteAudio(floatBuffer []float32) (err error) {
	// we reuse the old capacity tmpBuffer by setting its length to zero. then,
	// we save the tmpBuffer so we can reuse it next time
	o.tmpBuffer = FloatBufferTo16BitLE(floatBuffer, o.tmpBuffer[:0])
	if _, err := o.player.Write(o.tmpBuffer); err != nil {
		return fmt.Errorf("cannot write to player: %w", err)
	}
	return nil
}

// Close disposes of resources
func (o *OtoOutput) Close() error {
	if err := o.player.Close(); err != nil {
		return fmt.Errorf("cannot close oto player: %w", err)
	}
	return nil
}
