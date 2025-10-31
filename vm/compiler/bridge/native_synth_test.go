package bridge_test

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm/compiler/bridge"
	"gopkg.in/yaml.v3"
	// TODO: test the song using a mocks instead
)

const BPM = 100
const SAMPLE_RATE = 44100
const TOTAL_ROWS = 16
const SAMPLES_PER_ROW = SAMPLE_RATE * 4 * 60 / (BPM * 16)

const su_max_samples = SAMPLES_PER_ROW * TOTAL_ROWS

// const bufsize = su_max_samples * 2

func TestEmptyPatch(t *testing.T) {
	patch := sointu.Patch{}
	tracks := []sointu.Track{{NumVoices: 0, Order: []int{0}, Patterns: []sointu.Pattern{{64, 0, 68, 0, 32, 0, 0, 0, 75, 0, 78, 0, 0, 0, 0, 0}}}}
	song := sointu.Song{BPM: 100, RowsPerBeat: 4, Score: sointu.Score{RowsPerPattern: 16, Length: 1, Tracks: tracks}, Patch: patch}
	// make sure that the empty patch does not crash the synth
	sointu.Play(bridge.NativeSynther{}, song, nil)
}

func TestUpdatingEmptyPatch(t *testing.T) {
	patch := sointu.Patch{sointu.Instrument{NumVoices: 1, Units: []sointu.Unit{
		{Type: "envelope", Parameters: map[string]int{"stereo": 0, "attack": 64, "decay": 64, "sustain": 64, "release": 80, "gain": 128}},
		{Type: "envelope", Parameters: map[string]int{"stereo": 0, "attack": 95, "decay": 64, "sustain": 64, "release": 80, "gain": 128}},
		{Type: "out", Parameters: map[string]int{"stereo": 1, "gain": 128}},
	}}}
	tracks := []sointu.Track{{NumVoices: 0, Order: []int{0}, Patterns: []sointu.Pattern{{64, 0, 68, 0, 32, 0, 0, 0, 75, 0, 78, 0, 0, 0, 0, 0}}}}
	song := sointu.Song{BPM: 100, RowsPerBeat: 4, Score: sointu.Score{RowsPerPattern: 16, Length: 1, Tracks: tracks}, Patch: patch}
	synth, err := bridge.NativeSynther{}.Synth(patch, song.BPM)
	if err != nil {
		t.Fatalf("Synth creation failed: %v", err)
	}
	synth.Update(sointu.Patch{}, song.BPM)
	buffer := make(sointu.AudioBuffer, su_max_samples)
	err = buffer[:len(buffer)/2].Fill(synth)
	if err != nil {
		t.Fatalf("render gave an error: %v", err)
	}
}

func TestOscillatSine(t *testing.T) {
	patch := sointu.Patch{sointu.Instrument{NumVoices: 1, Units: []sointu.Unit{
		{Type: "envelope", Parameters: map[string]int{"stereo": 0, "attack": 32, "decay": 32, "sustain": 64, "release": 64, "gain": 128}},
		{Type: "oscillator", Parameters: map[string]int{"stereo": 0, "transpose": 64, "detune": 64, "phase": 0, "color": 96, "shape": 64, "gain": 128, "type": sointu.Sine, "lfo": 0, "unison": 0}},
		{Type: "mulp", Parameters: map[string]int{"stereo": 0}},
		{Type: "envelope", Parameters: map[string]int{"stereo": 0, "attack": 32, "decay": 32, "sustain": 64, "release": 64, "gain": 128}},
		{Type: "oscillator", Parameters: map[string]int{"stereo": 0, "transpose": 72, "detune": 64, "phase": 64, "color": 64, "shape": 96, "gain": 128, "type": sointu.Sine, "lfo": 0, "unison": 0}},
		{Type: "mulp", Parameters: map[string]int{"stereo": 0}},
		{Type: "out", Parameters: map[string]int{"stereo": 1, "gain": 128}},
	}}}
	tracks := []sointu.Track{{NumVoices: 1, Order: []int{0}, Patterns: []sointu.Pattern{{64, 0, 68, 0, 32, 0, 0, 0, 75, 0, 78, 0, 0, 0, 0, 0}}}}
	song := sointu.Song{BPM: 100, RowsPerBeat: 4, Score: sointu.Score{RowsPerPattern: 16, Length: 1, Tracks: tracks}, Patch: patch}
	buffer, err := sointu.Play(bridge.NativeSynther{}, song, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	compareToRawFloat32(t, buffer, "test_oscillat_sine.raw")
}

func TestRenderSamples(t *testing.T) {
	patch := sointu.Patch{sointu.Instrument{NumVoices: 1, Units: []sointu.Unit{
		{Type: "envelope", Parameters: map[string]int{"stereo": 0, "attack": 64, "decay": 64, "sustain": 64, "release": 80, "gain": 128}},
		{Type: "envelope", Parameters: map[string]int{"stereo": 0, "attack": 95, "decay": 64, "sustain": 64, "release": 80, "gain": 128}},
		{Type: "out", Parameters: map[string]int{"stereo": 1, "gain": 128}},
	}}}

	synth, err := bridge.Synth(patch, 120)
	if err != nil {
		t.Fatalf("bridge compile error: %v", err)
	}
	defer synth.Close()
	synth.Trigger(0, 64)
	buffer := make(sointu.AudioBuffer, su_max_samples)
	err = buffer[:len(buffer)/2].Fill(synth)
	if err != nil {
		t.Fatalf("first render gave an error")
	}
	synth.Release(0)
	err = buffer[len(buffer)/2:].Fill(synth)
	if err != nil {
		t.Fatalf("first render gave an error")
	}
	compareToRawFloat32(t, buffer, "test_render_samples.raw")
}

func TestAllRegressionTests(t *testing.T) {
	_, myname, _, _ := runtime.Caller(0)
	files, err := filepath.Glob(path.Join(path.Dir(myname), "..", "..", "..", "tests", "*.yml"))
	if err != nil {
		t.Fatalf("cannot glob files in the test directory: %v", err)
	}
	for _, filename := range files {
		basename := filepath.Base(filename)
		testname := strings.TrimSuffix(basename, path.Ext(basename))
		t.Run(testname, func(t *testing.T) {
			if runtime.GOOS != "windows" && strings.Contains(testname, "sample") {
				t.Skip("Samples (gm.dls) available only on Windows")
				return
			}
			asmcode, err := ioutil.ReadFile(filename)
			if err != nil {
				t.Fatalf("cannot read the .asm file: %v", filename)
			}
			var song sointu.Song
			err = yaml.Unmarshal(asmcode, &song)
			if err != nil {
				t.Fatalf("could not parse the .yml file: %v", err)
			}
			buffer, err := sointu.Play(bridge.NativeSynther{}, song, nil)
			buffer = buffer[:song.Score.LengthInRows()*song.SamplesPerRow()] // extend to the nominal length always.
			if err != nil {
				t.Fatalf("Play failed: %v", err)
			}
			if os.Getenv("SOINTU_TEST_SAVE_OUTPUT") == "YES" {
				outputpath := path.Join(path.Dir(myname), "actual_output")
				if _, err := os.Stat(outputpath); os.IsNotExist(err) {
					os.Mkdir(outputpath, 0755)
				}
				outFileName := path.Join(path.Dir(myname), "actual_output", testname+".raw")
				outfile, err := os.OpenFile(outFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
				defer outfile.Close()
				if err != nil {
					t.Fatalf("Creating file failed: %v", err)
				}
				var createdbuf bytes.Buffer
				err = binary.Write(&createdbuf, binary.LittleEndian, buffer)
				if err != nil {
					t.Fatalf("error converting buffer: %v", err)
				}
				_, err = outfile.Write(createdbuf.Bytes())
				if err != nil {
					log.Fatal(err)
				}
			}
			compareToRawFloat32(t, buffer, testname+".raw")
		})
	}
}

func TestStackUnderflow(t *testing.T) {
	patch := sointu.Patch{sointu.Instrument{NumVoices: 1, Units: []sointu.Unit{
		{Type: "pop", Parameters: map[string]int{}},
	}}}
	synth, err := bridge.Synth(patch, 120)
	if err != nil {
		t.Fatalf("bridge compile error: %v", err)
	}
	defer synth.Close()
	buffer := make(sointu.AudioBuffer, 1)
	err = buffer.Fill(synth)
	if err == nil {
		t.Fatalf("rendering should have failed due to stack underflow")
	}
}

func TestStackBalancing(t *testing.T) {
	patch := sointu.Patch{
		sointu.Instrument{NumVoices: 1, Units: []sointu.Unit{
			{Type: "push", Parameters: map[string]int{}},
		}}}
	synth, err := bridge.Synth(patch, 120)
	if err != nil {
		t.Fatalf("bridge compile error: %v", err)
	}
	defer synth.Close()
	buffer := make(sointu.AudioBuffer, 1)
	err = buffer.Fill(synth)
	if err == nil {
		t.Fatalf("rendering should have failed due to unbalanced stack push/pop")
	}
}

func TestStackOverflow(t *testing.T) {
	patch := sointu.Patch{
		sointu.Instrument{NumVoices: 1, Units: []sointu.Unit{
			{Type: "loadval", Parameters: map[string]int{"value": 128}},
			{Type: "loadval", Parameters: map[string]int{"value": 128}},
			{Type: "loadval", Parameters: map[string]int{"value": 128}},
			{Type: "loadval", Parameters: map[string]int{"value": 128}},
			{Type: "loadval", Parameters: map[string]int{"value": 128}},
			{Type: "loadval", Parameters: map[string]int{"value": 128}},
			{Type: "loadval", Parameters: map[string]int{"value": 128}},
			{Type: "loadval", Parameters: map[string]int{"value": 128}},
			{Type: "loadval", Parameters: map[string]int{"value": 128}},
			{Type: "pop", Parameters: map[string]int{}},
			{Type: "pop", Parameters: map[string]int{}},
			{Type: "pop", Parameters: map[string]int{}},
			{Type: "pop", Parameters: map[string]int{}},
			{Type: "pop", Parameters: map[string]int{}},
			{Type: "pop", Parameters: map[string]int{}},
			{Type: "pop", Parameters: map[string]int{}},
			{Type: "pop", Parameters: map[string]int{}},
			{Type: "pop", Parameters: map[string]int{}},
		}}}
	synth, err := bridge.Synth(patch, 120)
	if err != nil {
		t.Fatalf("bridge compile error: %v", err)
	}
	defer synth.Close()
	buffer := make(sointu.AudioBuffer, 1)
	err = buffer.Fill(synth)
	if err == nil {
		t.Fatalf("rendering should have failed due to stack overflow, despite balanced push/pops")
	}
}

func TestDivideByZero(t *testing.T) {
	patch := sointu.Patch{sointu.Instrument{NumVoices: 1, Units: []sointu.Unit{
		{Type: "loadval", Parameters: map[string]int{"value": 128}},
		{Type: "invgain", Parameters: map[string]int{"invgain": 0}},
		{Type: "pop", Parameters: map[string]int{}},
	}}}
	synth, err := bridge.Synth(patch, 120)
	if err != nil {
		t.Fatalf("bridge compile error: %v", err)
	}
	defer synth.Close()
	buffer := make(sointu.AudioBuffer, 1)
	err = buffer.Fill(synth)
	if err == nil {
		t.Fatalf("rendering should have failed due to divide by zero")
	}
}

func compareToRawFloat32(t *testing.T, buffer sointu.AudioBuffer, rawname string) {
	_, filename, _, _ := runtime.Caller(0)
	expectedb, err := ioutil.ReadFile(path.Join(path.Dir(filename), "..", "..", "..", "tests", "expected_output", rawname))
	if err != nil {
		t.Fatalf("cannot read expected: %v", err)
	}
	expected := make(sointu.AudioBuffer, len(expectedb)/8)
	buf := bytes.NewReader(expectedb)
	err = binary.Read(buf, binary.LittleEndian, &expected)
	if err != nil {
		t.Fatalf("error converting expected buffer: %v", err)
	}
	if len(expected) != len(buffer) {
		t.Fatalf("buffer length mismatch, got %v, expected %v", len(buffer), len(expected))
	}
	for i, v := range expected {
		for j, s := range v {
			if math.IsNaN(float64(buffer[i][j])) || math.Abs(float64(s-buffer[i][j])) > 1e-6 {
				t.Fatalf("error bigger than 1e-6 detected, at sample position %v", i)
			}
		}
	}
}

func compareToRawInt16(t *testing.T, buffer []int16, rawname string) {
	_, filename, _, _ := runtime.Caller(0)
	expectedb, err := ioutil.ReadFile(path.Join(path.Dir(filename), "..", "tests", "expected_output", rawname))
	if err != nil {
		t.Fatalf("cannot read expected: %v", err)
	}
	expected := make([]int16, len(expectedb)/2)
	buf := bytes.NewReader(expectedb)
	err = binary.Read(buf, binary.LittleEndian, &expected)
	if err != nil {
		t.Fatalf("error converting expected buffer: %v", err)
	}
	if len(expected) != len(buffer) {
		t.Fatalf("buffer length mismatch, got %v, expected %v", len(buffer), len(expected))
	}
	for i, v := range expected {
		if math.IsNaN(float64(buffer[i])) || v != buffer[i] {
			t.Fatalf("error at sample position %v", i)
		}
	}
}

func convertToInt16Buffer(buffer []float32) []int16 {
	int16Buffer := make([]int16, len(buffer))
	for i, v := range buffer {
		int16Buffer[i] = int16(math.Round(math.Min(math.Max(float64(v), -1.0), 1.0) * 32767))
	}
	return int16Buffer
}
