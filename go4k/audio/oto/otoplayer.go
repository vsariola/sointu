package oto

import (
	"fmt"
	"github.com/hajimehoshi/oto"
	"github.com/vsariola/sointu/go4k/audio"
)

// OtoPlayer wraps github.com/hajimehoshi/oto to play sointu-style float32[] audio
type OtoPlayer struct {
	context *oto.Context
	player  *oto.Player
}

// Play implements the audio.Player interface for OtoPlayer
func (o *OtoPlayer) Play(floatBuffer []float32) (err error) {
	if byteBuffer, err := audio.FloatBufferTo16BitLE(floatBuffer); err != nil {
		return fmt.Errorf("error writing to player: %w", err)
	} else if _, err := o.player.Write(byteBuffer); err != nil {
		return fmt.Errorf("error writing to player: %w", err)
	} else {
		fmt.Printf("%#v\n", floatBuffer[0:100])
		fmt.Printf("%#v\n", byteBuffer[0:200])
	}

	return nil
}

// Close disposes of resources
func (o *OtoPlayer) Close() error {
	if err := o.player.Close(); err != nil {
		return fmt.Errorf("error closing player: %w", err)
	}
	if err := o.context.Close(); err != nil {
		return fmt.Errorf("error closing oto context: %w", err)
	}
	return nil
}

const otoBufferSize = 8192

// NewPlayer creates and initializes a new OtoPlayer
func NewPlayer() (*OtoPlayer, error) {
	context, err := oto.NewContext(44100, 2, 2, otoBufferSize)
	if err != nil {
		return nil, fmt.Errorf("cannot create oto context: %w", err)
	}
	player := context.NewPlayer()

	return &OtoPlayer{
		context: context,
		player:  player,
	}, nil
}
