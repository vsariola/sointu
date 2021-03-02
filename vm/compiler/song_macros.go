package compiler

import (
	"github.com/vsariola/sointu"
)

type SongMacros struct {
	Song              *sointu.Song
	VoiceTrackBitmask int
	MaxSamples        int
}

func NewSongMacros(s *sointu.Song) *SongMacros {
	maxSamples := s.SamplesPerRow() * s.Score.LengthInRows()
	p := SongMacros{Song: s, MaxSamples: maxSamples}
	trackVoiceNumber := 0
	for _, t := range s.Score.Tracks {
		for b := 0; b < t.NumVoices-1; b++ {
			p.VoiceTrackBitmask += 1 << trackVoiceNumber
			trackVoiceNumber++
		}
		trackVoiceNumber++ // set all bits except last one
	}
	return &p
}
