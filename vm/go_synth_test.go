package vm_test

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
	"gopkg.in/yaml.v3"
)

const errorThreshold = 1e-2

func TestAllRegressionTests(t *testing.T) {
	_, myname, _, _ := runtime.Caller(0)
	files, err := filepath.Glob(path.Join(path.Dir(myname), "..", "tests", "*.yml"))
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
			buffer, err := sointu.Play(vm.GoSynther{}, song, nil)
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

var defaultUnits = map[string]sointu.Unit{
	"envelope":   {Type: "envelope", Parameters: map[string]int{"stereo": 0, "attack": 64, "decay": 64, "sustain": 64, "release": 64, "gain": 64}},
	"oscillator": {Type: "oscillator", Parameters: map[string]int{"stereo": 0, "transpose": 64, "detune": 64, "phase": 0, "color": 64, "shape": 64, "gain": 64, "type": sointu.Sine}},
	"noise":      {Type: "noise", Parameters: map[string]int{"stereo": 0, "shape": 64, "gain": 64}},
	"mulp":       {Type: "mulp", Parameters: map[string]int{"stereo": 0}},
	"mul":        {Type: "mul", Parameters: map[string]int{"stereo": 0}},
	"add":        {Type: "add", Parameters: map[string]int{"stereo": 0}},
	"addp":       {Type: "addp", Parameters: map[string]int{"stereo": 0}},
	"push":       {Type: "push", Parameters: map[string]int{"stereo": 0}},
	"pop":        {Type: "pop", Parameters: map[string]int{"stereo": 0}},
	"xch":        {Type: "xch", Parameters: map[string]int{"stereo": 0}},
	"receive":    {Type: "receive", Parameters: map[string]int{"stereo": 0}},
	"loadnote":   {Type: "loadnote", Parameters: map[string]int{"stereo": 0}},
	"loadval":    {Type: "loadval", Parameters: map[string]int{"stereo": 0, "value": 64}},
	"pan":        {Type: "pan", Parameters: map[string]int{"stereo": 0, "panning": 64}},
	"gain":       {Type: "gain", Parameters: map[string]int{"stereo": 0, "gain": 64}},
	"invgain":    {Type: "invgain", Parameters: map[string]int{"stereo": 0, "invgain": 64}},
	"dbgain":     {Type: "dbgain", Parameters: map[string]int{"stereo": 0, "decibels": 64}},
	"crush":      {Type: "crush", Parameters: map[string]int{"stereo": 0, "resolution": 64}},
	"clip":       {Type: "clip", Parameters: map[string]int{"stereo": 0}},
	"hold":       {Type: "hold", Parameters: map[string]int{"stereo": 0, "holdfreq": 64}},
	"distort":    {Type: "distort", Parameters: map[string]int{"stereo": 0, "drive": 64}},
	"filter":     {Type: "filter", Parameters: map[string]int{"stereo": 0, "frequency": 64, "resonance": 64, "lowpass": 1, "bandpass": 0, "highpass": 0}},
	"out":        {Type: "out", Parameters: map[string]int{"stereo": 1, "gain": 64}},
	"outaux":     {Type: "outaux", Parameters: map[string]int{"stereo": 1, "outgain": 64, "auxgain": 64}},
	"aux":        {Type: "aux", Parameters: map[string]int{"stereo": 1, "gain": 64, "channel": 2}},
	"delay": {Type: "delay",
		Parameters: map[string]int{"damp": 0, "dry": 128, "feedback": 96, "notetracking": 2, "pregain": 40, "stereo": 0},
		VarArgs:    []int{48}},
	"in":         {Type: "in", Parameters: map[string]int{"stereo": 1, "channel": 2}},
	"speed":      {Type: "speed", Parameters: map[string]int{}},
	"compressor": {Type: "compressor", Parameters: map[string]int{"stereo": 0, "attack": 64, "release": 64, "invgain": 64, "threshold": 64, "ratio": 64}},
	"send":       {Type: "send", Parameters: map[string]int{"stereo": 0, "amount": 128, "voice": 0, "unit": 0, "port": 0, "sendpop": 1}},
	"sync":       {Type: "sync", Parameters: map[string]int{}},
	"eq":         {Type: "eq", Parameters: map[string]int{"stereo": 0, "freq": 1000, "q": 10, "gain": 64}},
}

var defaultInstrument = sointu.Instrument{
	Name:      "Instr",
	NumVoices: 1,
	Units: []sointu.Unit{
		defaultUnits["envelope"],
		defaultUnits["oscillator"],
		defaultUnits["mulp"],
		defaultUnits["pan"],
		defaultUnits["outaux"],
	},
}

func TestDisabledWithInstrument(t *testing.T) {
	// Compile the default instrument and then add a version of every unit to
	// that, but disabled, and make sure the compilation produces identical
	// results
	patch := sointu.Patch{defaultInstrument}
	features := vm.NecessaryFeaturesFor(patch)
	byteCode, err := vm.NewBytecode(patch, features, 120)
	if err != nil {
		t.Fatalf("vm.NewBytecode failed: %v", err)
	}
	for _, unit := range defaultUnits {
		units := []sointu.Unit{}
		u := unit.Copy()
		u.Disabled = true
		u.ID = 1000
		units = append(units, u)
		units = append(units, defaultInstrument.Units...)
		u2 := unit.Copy()
		u2.Disabled = true
		u2.ID = 1001
		units = append(units, u2)

		patch2 := sointu.Patch{sointu.Instrument{Name: "Instr", NumVoices: 1, Units: units}}
		features2 := vm.NecessaryFeaturesFor(patch2)
		byteCode2, err := vm.NewBytecode(patch2, features2, 120)
		if err != nil {
			t.Fatalf("vm.NewBytecode failed: %v", err)
		}
		if !reflect.DeepEqual(features, features2) {
			t.Fatalf("disabled unit %v produced different FeatureSet", unit.Type)
		}
		if !reflect.DeepEqual(byteCode, byteCode2) {
			t.Fatalf("disabled unit %v produced different Bytecode", unit.Type)
		}
	}
}

func TestDisabled(t *testing.T) {
	patch := sointu.Patch{sointu.Instrument{Name: "Instr", NumVoices: 1, Units: []sointu.Unit{}}}
	features := vm.NecessaryFeaturesFor(patch)
	byteCode, err := vm.NewBytecode(patch, features, 120)
	if err != nil {
		t.Fatalf("vm.NewBytecode failed: %v", err)
	}
	for _, unit := range defaultUnits {
		u := unit.Copy()
		u.Disabled = true
		u.ID = 1000
		u2 := unit.Copy()
		u2.Disabled = true
		u2.ID = 1001
		patch2 := sointu.Patch{sointu.Instrument{Name: "Instr", NumVoices: 1, Units: []sointu.Unit{u, u2}}}
		features2 := vm.NecessaryFeaturesFor(patch2)
		byteCode2, err := vm.NewBytecode(patch2, features2, 120)
		if err != nil {
			t.Fatalf("vm.NewBytecode failed: %v", err)
		}
		if !reflect.DeepEqual(features, features2) {
			t.Fatalf("disabled unit %v produced different FeatureSet", unit.Type)
		}
		if !reflect.DeepEqual(byteCode, byteCode2) {
			t.Fatalf("disabled unit %v produced different bytecode", unit.Type)
		}
	}
}

func TestStackUnderflow(t *testing.T) {
	patch := sointu.Patch{sointu.Instrument{NumVoices: 1, Units: []sointu.Unit{
		{Type: "pop", Parameters: map[string]int{}},
	}}}
	synth, err := vm.GoSynther{}.Synth(patch, 120)
	if err != nil {
		t.Fatalf("bridge compile error: %v", err)
	}
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
	synth, err := vm.GoSynther{}.Synth(patch, 120)
	if err != nil {
		t.Fatalf("bridge compile error: %v", err)
	}
	buffer := make(sointu.AudioBuffer, 1)
	err = buffer.Fill(synth)
	if err == nil {
		t.Fatalf("rendering should have failed due to unbalanced stack push/pop")
	}
}

func compareToRawFloat32(t *testing.T, buffer sointu.AudioBuffer, rawname string) {
	_, filename, _, _ := runtime.Caller(0)
	expectedb, err := ioutil.ReadFile(path.Join(path.Dir(filename), "..", "tests", "expected_output", rawname))
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
	firsterr := -1
	errs := 0
	for i, v := range expected[1 : len(expected)-1] {
		for j, s := range v {
			if math.IsNaN(float64(buffer[i][j])) || (math.Abs(float64(s-buffer[i][j])) > errorThreshold &&
				math.Abs(float64(s-buffer[i+1][j])) > errorThreshold && math.Abs(float64(s-buffer[i+2][j])) > errorThreshold) {
				errs++
				if firsterr == -1 {
					firsterr = i
				}
				if errs > 200 { // we are again quite liberal with rounding errors, as different platforms have minor differences in floating point rounding
					t.Fatalf("more than 200 errors bigger than %v detected, first at sample position %v", errorThreshold, firsterr)
				}
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
