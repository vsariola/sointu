package tracker

import "github.com/vsariola/sointu/go4k"

var defaultSong = go4k.Song{
	BPM: 100,
	Patterns: [][]byte{
		{64, 0, 68, 0, 32, 0, 0, 0, 75, 0, 78, 0, 0, 0, 0, 0},
		{0, 0, 64, 0, 68, 0, 32, 0, 0, 0, 75, 0, 78, 0, 0, 0},
	},
	Tracks: []go4k.Track{
		{NumVoices: 1, Sequence: []byte{0}},
		{NumVoices: 1, Sequence: []byte{1}},
	},
	SongLength: 0,
	Patch: go4k.Patch{
		go4k.Instrument{NumVoices: 2, Units: []go4k.Unit{
			{"envelope", map[string]int{"stereo": 0, "attack": 32, "decay": 32, "sustain": 64, "release": 64, "gain": 128}, []int{}},
			{"oscillator", map[string]int{"stereo": 0, "transpose": 64, "detune": 64, "phase": 0, "color": 96, "shape": 64, "gain": 128, "type": go4k.Sine}, []int{}},
			{"mulp", map[string]int{"stereo": 0}, []int{}},
			{"envelope", map[string]int{"stereo": 0, "attack": 32, "decay": 32, "sustain": 64, "release": 64, "gain": 128}, []int{}},
			{"oscillator", map[string]int{"stereo": 0, "transpose": 72, "detune": 64, "phase": 64, "color": 64, "shape": 96, "gain": 128, "type": go4k.Sine}, []int{}},
			{"mulp", map[string]int{"stereo": 0}, []int{}},
			{"out", map[string]int{"stereo": 1, "gain": 128}, []int{}},
		}},
	},
}
