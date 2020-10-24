package song_test

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"path"
	"runtime"
	"testing"

	"github.com/vsariola/sointu/bridge"
	"github.com/vsariola/sointu/song"
)

const BPM = 100
const SAMPLE_RATE = 44100
const TOTAL_ROWS = 16
const SAMPLES_PER_ROW = SAMPLE_RATE * 4 * 60 / (BPM * 16)

const su_max_samples = SAMPLES_PER_ROW * TOTAL_ROWS

// const bufsize = su_max_samples * 2

func TestSongRender(t *testing.T) {
	patch := []bridge.Instrument{
		bridge.Instrument{1, []bridge.Unit{
			bridge.Unit{bridge.Envelope, []byte{32, 32, 64, 64, 128}},
			bridge.Unit{bridge.Oscillat, []byte{64, 64, 0, 96, 64, 128, 0x40}},
			bridge.Unit{bridge.Mulp, []byte{}},
			bridge.Unit{bridge.Envelope, []byte{32, 32, 64, 64, 128}},
			bridge.Unit{bridge.Oscillat, []byte{72, 64, 64, 64, 96, 128, 0x40}},
			bridge.Unit{bridge.Mulp, []byte{}},
			bridge.Unit{bridge.Out.Stereo(), []byte{128}},
		}}}
	patterns := [][]byte{{64, 0, 68, 0, 32, 0, 0, 0, 75, 0, 78, 0, 0, 0, 0, 0}}
	tracks := []song.Track{song.Track{1, []byte{0}}}
	song, err := song.NewSong(100, patterns, tracks, patch)
	if err != nil {
		t.Fatalf("NewSong failed: %v", err)
	}
	buffer, err := song.Render()
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
