package compiler

import (
	"github.com/vsariola/sointu"
)

type SongMacros struct {
	Song              *sointu.Song
	VoiceTrackBitmask []byte
	HasVoiceTrack     bool
	MaxSamples        int
}

func NewSongMacros(s *sointu.Song) *SongMacros {
	maxSamples := s.SamplesPerRow() * s.Score.LengthInRows()
	p := SongMacros{Song: s, MaxSamples: maxSamples}
	trackVoiceNumber := 0
	p.VoiceTrackBitmask = make([]byte, (s.Score.NumVoices()+7)/8)
	for _, t := range s.Score.Tracks {
		if t.NumVoices > 0 {
			p.HasVoiceTrack = true
		}
		for b := 0; b < t.NumVoices-1; b++ {
			p.VoiceTrackBitmask[trackVoiceNumber/8] += 1 << (trackVoiceNumber % 8)
			trackVoiceNumber++
		}
		trackVoiceNumber++ // set all bits except last one
	}
	return &p
}
