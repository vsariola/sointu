package tracker

import "github.com/vsariola/sointu"

var defaultSong = sointu.Song{
	BPM: 100,
	Patterns: [][]byte{
		{64, 0, 68, 0, 32, 0, 0, 0, 75, 0, 78, 0, 0, 0, 0, 0},
		{0, 0, 64, 0, 68, 0, 32, 0, 0, 0, 75, 0, 78, 0, 0, 0},
	},
	Tracks: []sointu.Track{
		{NumVoices: 1, Sequence: []byte{0}},
		{NumVoices: 1, Sequence: []byte{1}},
	},
	Patch: sointu.Patch{
		Instruments: []sointu.Instrument{{NumVoices: 2, Units: []sointu.Unit{
			{Type: "envelope", Parameters: map[string]int{"stereo": 0, "attack": 32, "decay": 32, "sustain": 64, "release": 64, "gain": 128}},
			{Type: "oscillator", Parameters: map[string]int{"stereo": 0, "transpose": 64, "detune": 64, "phase": 0, "color": 96, "shape": 64, "gain": 128, "type": sointu.Sine}},
			{Type: "mulp", Parameters: map[string]int{"stereo": 0}},
			{Type: "envelope", Parameters: map[string]int{"stereo": 0, "attack": 32, "decay": 32, "sustain": 64, "release": 64, "gain": 128}},
			{Type: "oscillator", Parameters: map[string]int{"stereo": 0, "transpose": 72, "detune": 64, "phase": 64, "color": 64, "shape": 96, "gain": 128, "type": sointu.Sine}},
			{Type: "mulp", Parameters: map[string]int{"stereo": 0}},
			{Type: "out", Parameters: map[string]int{"stereo": 1, "gain": 128}},
		}}}},
}
