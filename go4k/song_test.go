package go4k_test

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"path"
	"runtime"
	"testing"

	"github.com/vsariola/sointu/go4k"
	"github.com/vsariola/sointu/go4k/bridge"
	// TODO: test the song using a mocks instead
)

const BPM = 100
const SAMPLE_RATE = 44100
const TOTAL_ROWS = 16
const SAMPLES_PER_ROW = SAMPLE_RATE * 4 * 60 / (BPM * 16)

const su_max_samples = SAMPLES_PER_ROW * TOTAL_ROWS

// const bufsize = su_max_samples * 2

func TestPlayer(t *testing.T) {
	patch := go4k.Patch{
		Instruments: []go4k.Instrument{go4k.Instrument{1, []go4k.Unit{
			go4k.Unit{"envelope", map[string]int{"stereo": 0, "attack": 32, "decay": 32, "sustain": 64, "release": 64, "gain": 128}},
			go4k.Unit{"oscillator", map[string]int{"stereo": 0, "transpose": 64, "detune": 64, "phase": 0, "color": 96, "shape": 64, "gain": 128, "type": go4k.Sine, "lfo": 0, "unison": 0}},
			go4k.Unit{"mulp", map[string]int{"stereo": 0}},
			go4k.Unit{"envelope", map[string]int{"stereo": 0, "attack": 32, "decay": 32, "sustain": 64, "release": 64, "gain": 128}},
			go4k.Unit{"oscillator", map[string]int{"stereo": 0, "transpose": 72, "detune": 64, "phase": 64, "color": 64, "shape": 96, "gain": 128, "type": go4k.Sine, "lfo": 0, "unison": 0}},
			go4k.Unit{"mulp", map[string]int{"stereo": 0}},
			go4k.Unit{"out", map[string]int{"stereo": 1, "gain": 128}},
		}}},
		DelayTimes:    []int{},
		SampleOffsets: []go4k.SampleOffset{}}
	patterns := [][]byte{{64, 0, 68, 0, 32, 0, 0, 0, 75, 0, 78, 0, 0, 0, 0, 0}}
	tracks := []go4k.Track{go4k.Track{1, []byte{0}}}
	song := go4k.Song{100, patterns, tracks, patch}
	synth, err := bridge.Synth(patch)
	if err != nil {
		t.Fatalf("Compiling patch failed: %v", err)
	}
	buffer, err := go4k.Play(synth, song)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	_, filename, _, _ := runtime.Caller(0)
	expectedb, err := ioutil.ReadFile(path.Join(path.Dir(filename), "..", "tests", "expected_output", "test_oscillat_sine.raw"))
	if err != nil {
		t.Fatalf("cannot read expected: %v", err)
	}
	var createdbuf bytes.Buffer
	err = binary.Write(&createdbuf, binary.LittleEndian, buffer)
	if err != nil {
		t.Fatalf("error converting buffer: %v", err)
	}
	createdb := createdbuf.Bytes()
	if len(createdb) != len(expectedb) {
		t.Fatalf("buffer length mismatch, got %v, expected %v", len(createdb), len(expectedb))
	}
	for i, v := range createdb {
		if expectedb[i] != v {
			t.Fatalf("byte mismatch @ %v, got %v, expected %v", i, v, expectedb[i])
		}
	}
}
