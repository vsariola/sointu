package bridge_test

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"path"
	"runtime"
	"testing"

	"github.com/vsariola/sointu/go4k"
	"github.com/vsariola/sointu/go4k/bridge"
)

func TestBridge(t *testing.T) {
	patch := go4k.Patch{
		Instruments: []go4k.Instrument{
			go4k.Instrument{1, []go4k.Unit{
				go4k.Unit{Type: "envelope", Parameters: map[string]int{"stereo": 0, "attack": 64, "decay": 64, "sustain": 64, "release": 80, "gain": 128}},
				go4k.Unit{Type: "envelope", Parameters: map[string]int{"stereo": 0, "attack": 95, "decay": 64, "sustain": 64, "release": 80, "gain": 128}},
				go4k.Unit{Type: "out", Parameters: map[string]int{"stereo": 1, "gain": 128}},
			}}}}

	synth, err := bridge.Synth(patch)
	if err != nil {
		t.Fatalf("bridge compile error: %v", err)
	}
	synth.Trigger(0, 64)
	buffer := make([]float32, 2*su_max_samples)
	err = go4k.Render(synth, buffer[:len(buffer)/2])
	if err != nil {
		t.Fatalf("first render gave an error")
	}
	synth.Release(0)
	err = go4k.Render(synth, buffer[len(buffer)/2:])
	if err != nil {
		t.Fatalf("first render gave an error")
	}
	_, filename, _, _ := runtime.Caller(0)
	expectedb, err := ioutil.ReadFile(path.Join(path.Dir(filename), "..", "..", "tests", "expected_output", "test_render_samples.raw"))
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
			break
		}
	}
}

func TestStackUnderflow(t *testing.T) {
	patch := go4k.Patch{
		Instruments: []go4k.Instrument{
			go4k.Instrument{1, []go4k.Unit{
				go4k.Unit{Type: "pop", Parameters: map[string]int{}},
			}}}}
	synth, err := bridge.Synth(patch)
	if err != nil {
		t.Fatalf("bridge compile error: %v", err)
	}
	buffer := make([]float32, 2)
	err = go4k.Render(synth, buffer)
	if err == nil {
		t.Fatalf("rendering should have failed due to stack underflow")
	}
}

func TestStackBalancing(t *testing.T) {
	patch := go4k.Patch{
		Instruments: []go4k.Instrument{
			go4k.Instrument{1, []go4k.Unit{
				go4k.Unit{Type: "push", Parameters: map[string]int{}},
			}}}}
	synth, err := bridge.Synth(patch)
	if err != nil {
		t.Fatalf("bridge compile error: %v", err)
	}
	buffer := make([]float32, 2)
	err = go4k.Render(synth, buffer)
	if err == nil {
		t.Fatalf("rendering should have failed due to unbalanced stack push/pop")
	}
}

func TestStackOverflow(t *testing.T) {
	patch := go4k.Patch{
		Instruments: []go4k.Instrument{
			go4k.Instrument{1, []go4k.Unit{
				go4k.Unit{Type: "loadval", Parameters: map[string]int{"value": 128}},
				go4k.Unit{Type: "loadval", Parameters: map[string]int{"value": 128}},
				go4k.Unit{Type: "loadval", Parameters: map[string]int{"value": 128}},
				go4k.Unit{Type: "loadval", Parameters: map[string]int{"value": 128}},
				go4k.Unit{Type: "loadval", Parameters: map[string]int{"value": 128}},
				go4k.Unit{Type: "loadval", Parameters: map[string]int{"value": 128}},
				go4k.Unit{Type: "loadval", Parameters: map[string]int{"value": 128}},
				go4k.Unit{Type: "loadval", Parameters: map[string]int{"value": 128}},
				go4k.Unit{Type: "loadval", Parameters: map[string]int{"value": 128}},
				go4k.Unit{Type: "pop", Parameters: map[string]int{}},
				go4k.Unit{Type: "pop", Parameters: map[string]int{}},
				go4k.Unit{Type: "pop", Parameters: map[string]int{}},
				go4k.Unit{Type: "pop", Parameters: map[string]int{}},
				go4k.Unit{Type: "pop", Parameters: map[string]int{}},
				go4k.Unit{Type: "pop", Parameters: map[string]int{}},
				go4k.Unit{Type: "pop", Parameters: map[string]int{}},
				go4k.Unit{Type: "pop", Parameters: map[string]int{}},
				go4k.Unit{Type: "pop", Parameters: map[string]int{}},
			}}}}
	synth, err := bridge.Synth(patch)
	if err != nil {
		t.Fatalf("bridge compile error: %v", err)
	}
	buffer := make([]float32, 2)
	err = go4k.Render(synth, buffer)
	if err == nil {
		t.Fatalf("rendering should have failed due to stack overflow, despite balanced push/pops")
	}
}

func TestDivideByZero(t *testing.T) {
	patch := go4k.Patch{
		Instruments: []go4k.Instrument{
			go4k.Instrument{1, []go4k.Unit{
				go4k.Unit{Type: "loadval", Parameters: map[string]int{"value": 128}},
				go4k.Unit{Type: "invgain", Parameters: map[string]int{"invgain": 0}},
				go4k.Unit{Type: "pop", Parameters: map[string]int{}},
			}}}}
	synth, err := bridge.Synth(patch)
	if err != nil {
		t.Fatalf("bridge compile error: %v", err)
	}
	buffer := make([]float32, 2)
	err = go4k.Render(synth, buffer)
	if err == nil {
		t.Fatalf("rendering should have failed due to divide by zero")
	}
}
