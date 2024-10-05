package oto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/ebitengine/oto/v3"
	"github.com/vsariola/sointu"
)

const latency = 2048 // in samples at 44100 Hz = ~46 ms

type (
	OtoContext oto.Context

	OtoPlayer struct {
		player *oto.Player
		reader *OtoReader
	}

	OtoReader struct {
		audioSource sointu.AudioSource
		tmpBuffer   sointu.AudioBuffer
		waitGroup   sync.WaitGroup
		err         error
		errMutex    sync.RWMutex
	}
)

func NewContext() (*OtoContext, error) {
	op := oto.NewContextOptions{}
	op.SampleRate = 44100
	op.ChannelCount = 2
	op.Format = oto.FormatFloat32LE
	context, readyChan, err := oto.NewContext(&op)
	if err != nil {
		return nil, fmt.Errorf("cannot create oto context: %w", err)
	}
	<-readyChan
	return (*OtoContext)(context), nil
}

func (c *OtoContext) Play(r sointu.AudioSource) sointu.CloserWaiter {
	reader := &OtoReader{audioSource: r}
	reader.waitGroup.Add(1)
	player := (*oto.Context)(c).NewPlayer(reader)
	player.SetBufferSize(latency * 8)
	player.Play()
	return OtoPlayer{player: player, reader: reader}
}

func (o OtoPlayer) Wait() {
	o.reader.waitGroup.Wait()
}

func (o OtoPlayer) Close() error {
	o.reader.closeWithError(errors.New("OtoPlayer was closed"))
	return o.player.Close()
}

func (o *OtoReader) Read(b []byte) (n int, err error) {
	o.errMutex.RLock()
	if o.err != nil {
		o.errMutex.RUnlock()
		return 0, o.err
	}
	o.errMutex.RUnlock()
	if len(b)%8 != 0 {
		return o.closeWithError(fmt.Errorf("oto: Read buffer length must be a multiple of 8"))
	}
	samples := len(b) / 8
	if samples > len(o.tmpBuffer) {
		o.tmpBuffer = append(o.tmpBuffer, make(sointu.AudioBuffer, samples-len(o.tmpBuffer))...)
	} else if samples < len(o.tmpBuffer) {
		o.tmpBuffer = o.tmpBuffer[:samples]
	}
	err = o.audioSource.ReadAudio(o.tmpBuffer)
	if err != nil {
		return o.closeWithError(err)
	}
	for i := range o.tmpBuffer {
		binary.LittleEndian.PutUint32(b[i*8:], math.Float32bits(o.tmpBuffer[i][0]))
		binary.LittleEndian.PutUint32(b[i*8+4:], math.Float32bits(o.tmpBuffer[i][1]))
	}
	return samples * 8, nil
}

func (o *OtoReader) closeWithError(err error) (int, error) {
	o.errMutex.Lock()
	defer o.errMutex.Unlock()
	if o.err == nil {
		o.err = err
		o.waitGroup.Done()
	}
	return 0, err
}
