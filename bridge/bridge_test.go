package bridge_test

import (
	"bytes"
	"encoding/binary"
	"github.com/vsariola/sointu/bridge"
	"io/ioutil"
	"path"
	"runtime"
	"testing"
)

const BPM = 100
const SAMPLE_RATE = 44100
const TOTAL_ROWS = 16
const SAMPLES_PER_ROW = SAMPLE_RATE * 4 * 60 / (BPM * 16)

const su_max_samples = SAMPLES_PER_ROW * TOTAL_ROWS

// const bufsize = su_max_samples * 2

func TestBridge(t *testing.T) {
	commands := [2048]bridge.Opcode{
		bridge.Envelope, bridge.Envelope, bridge.Out.Stereo(), bridge.Advance}
	values := [16384]byte{64, 64, 64, 80, 128, // envelope 1
		95, 64, 64, 80, 128, // envelope 2
		128}
	s := bridge.NewSynthState()
	// memcpy(synthState->Commands, commands, sizeof(commands));
	s.SetCommands(commands)
	// memcpy(synthState->Values, values, sizeof(values));
	s.SetValues(values)
	// synthState->RandSeed = 1;
	// initialized in NewSynthState
	// synthState->RowLen = INT32_MAX;
	// synthState->NumVoices = 1;
	s.NumVoices = 1
	// synthState->Synth.Voices[0].Note = 64;
	s.Synth.Voices[0].Note = 64
	// retval = su_render_samples(buffer, su_max_samples / 2, synthState);
	buffer := make([]float32, su_max_samples)
	remaining := s.Render(buffer)
	if remaining > 0 {
		t.Fatalf("could not render full buffer, %v bytes remaining, expected <= 0", remaining)
	}
	// synthState->Synth.Voices[0].Release++;
	s.Synth.Voices[0].Release++
	sbuffer := make([]float32, su_max_samples)
	remaining = s.Render(sbuffer)
	if remaining > 0 {
		t.Fatalf("could not render second full buffer, %v bytes remaining, expected <= 0", remaining)
	}
	buffer = append(buffer, sbuffer...)
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
