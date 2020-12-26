package compiler

import (
	"fmt"

	"github.com/vsariola/sointu"
)

type SongMacros struct {
	Song              *sointu.Song
	VoiceTrackBitmask int
	MaxSamples        int
}

func NewSongMacros(s *sointu.Song) *SongMacros {
	maxSamples := s.SamplesPerRow() * s.TotalRows()
	p := SongMacros{Song: s, MaxSamples: maxSamples}
	trackVoiceNumber := 0
	for _, t := range s.Tracks {
		for b := 0; b < t.NumVoices-1; b++ {
			p.VoiceTrackBitmask += 1 << trackVoiceNumber
			trackVoiceNumber++
		}
		trackVoiceNumber++ // set all bits except last one
	}
	return &p
}

func (p *SongMacros) NumDelayLines() string {
	total := 0
	for _, instr := range p.Song.Patch.Instruments {
		for _, unit := range instr.Units {
			if unit.Type == "delay" {
				total += unit.Parameters["count"] * (1 + unit.Parameters["stereo"])
			}
		}
	}
	return fmt.Sprintf("%v", total)
}
