package tracker

import (
	"github.com/vsariola/sointu"
)

type Processor struct {
	*Player
	playerProcessContext PlayerProcessContext
	uiProcessor          EventProcessor
}

func NewProcessor(player *Player, context PlayerProcessContext, uiProcessor EventProcessor) *Processor {
	return &Processor{player, context, uiProcessor}
}

func (p *Processor) ReadAudio(buf sointu.AudioBuffer) error {
	p.Player.Process(buf, p.playerProcessContext, p.uiProcessor)
	return nil
}
