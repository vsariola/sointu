package oto

import (
	"fmt"

	"github.com/hajimehoshi/oto"
	"github.com/vsariola/sointu"
)

type OtoContext oto.Context
type OtoOutput oto.Player

func (c *OtoContext) Output() sointu.AudioSink {
	return (*OtoOutput)((*oto.Context)(c).NewPlayer())
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
	if byteBuffer, err := FloatBufferTo16BitLE(floatBuffer); err != nil {
		return fmt.Errorf("cannot convert buffer to bytes: %w", err)
	} else if _, err := (*oto.Player)(o).Write(byteBuffer); err != nil {
		return fmt.Errorf("cannot write to player: %w", err)
	}
	return nil
}

// Close disposes of resources
func (o *OtoOutput) Close() error {
	if err := (*oto.Player)(o).Close(); err != nil {
		return fmt.Errorf("cannot close oto player: %w", err)
	}
	return nil
}
