package tracker

import "github.com/vsariola/sointu"

var defaultInstrument = sointu.Instrument{
	NumVoices: 1,
	Units: []sointu.Unit{
		{Type: "envelope", Parameters: map[string]int{"stereo": 1, "attack": 32, "decay": 32, "sustain": 64, "release": 64, "gain": 64}},
		{Type: "oscillator", Parameters: map[string]int{"stereo": 1, "transpose": 64, "detune": 64, "phase": 0, "color": 128, "shape": 64, "gain": 64, "type": sointu.Sine}},
		{Type: "mulp", Parameters: map[string]int{"stereo": 1}},
		{Type: "out", Parameters: map[string]int{"stereo": 1, "gain": 64}},
	},
}

var defaultSong = sointu.Song{
	BPM:            100,
	RowsPerPattern: 16,
	Tracks: []sointu.Track{
		{NumVoices: 2, Sequence: []byte{0, 0, 0, 1}, Patterns: [][]byte{{64, 0, 68, 0, 32, 0, 0, 0, 75, 0, 78, 0, 0, 0, 0, 0}, {64, 0, 68, 0, 32, 0, 0, 0, 75, 0, 75, 0, 75, 0, 80, 0}}},
		{NumVoices: 2, Sequence: []byte{0, 0, 0, 1}, Patterns: [][]byte{{0, 0, 64, 0, 68, 0, 32, 0, 0, 0, 75, 0, 78, 0, 0, 0}, {32, 0, 64, 0, 68, 0, 32, 0, 0, 0, 75, 0, 68, 0, 68, 0}}},
	},
	Patch: sointu.Patch{
		Instruments: []sointu.Instrument{{NumVoices: 4, Units: []sointu.Unit{
			{Type: "envelope", Parameters: map[string]int{"stereo": 1, "attack": 32, "decay": 32, "sustain": 64, "release": 64, "gain": 64}},
			{Type: "oscillator", Parameters: map[string]int{"stereo": 1, "transpose": 64, "detune": 64, "phase": 0, "color": 128, "shape": 64, "gain": 64, "type": sointu.Sine}},
			{Type: "mulp", Parameters: map[string]int{"stereo": 1}},
			{Type: "delay",
				Parameters: map[string]int{"damp": 0, "dry": 128, "feedback": 96, "notetracking": 0, "pregain": 40, "stereo": 1},
				VarArgs: []int{1116, 1188, 1276, 1356, 1422, 1492, 1556, 1618,
					1140, 1212, 1300, 1380, 1446, 1516, 1580, 1642,
				}},
			{Type: "out", Parameters: map[string]int{"stereo": 1, "gain": 64}},
		}}}},
}
