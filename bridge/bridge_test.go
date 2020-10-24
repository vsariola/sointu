package bridge_test

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"path"
	"runtime"
	"testing"

	"github.com/vsariola/sointu/bridge"
)

const BPM = 100
const SAMPLE_RATE = 44100
const TOTAL_ROWS = 16
const SAMPLES_PER_ROW = SAMPLE_RATE * 4 * 60 / (BPM * 16)

const su_max_samples = SAMPLES_PER_ROW * TOTAL_ROWS

// const bufsize = su_max_samples * 2

func TestBridge(t *testing.T) {
	s := bridge.NewSynthState()
	s.SetPatch([]bridge.Instrument{
		bridge.Instrument{1, []bridge.Unit{
			bridge.Unit{bridge.Envelope, []byte{64, 64, 64, 80, 128}},
			bridge.Unit{bridge.Envelope, []byte{95, 64, 64, 80, 128}},
			bridge.Unit{bridge.Out.Stereo(), []byte{128}},
		}},
	})
	s.Trigger(0, 64)
	s.SamplesPerRow = SAMPLES_PER_ROW * 8 // this song is two blocks of 8 rows, release before second block start
	buffer := make([]float32, 2*su_max_samples)
	n, err := s.Render(buffer, 2, func() {
		s.Release(0)
	})
	if n < su_max_samples {
		t.Fatalf("could not fill the whole buffer, %v samples rendered, %v expected", n, su_max_samples)
	}
	_, filename, _, _ := runtime.Caller(0)
	expectedb, err := ioutil.ReadFile(path.Join(path.Dir(filename), "..", "tests", "expected_output", "test_render_samples.raw"))
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
			t.Errorf("byte mismatch @ %v, got %v, expected %v", i, v, expectedb[i])
		}
	}
}
