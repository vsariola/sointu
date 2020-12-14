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

	"github.com/vsariola/sointu/go4k"
	"github.com/vsariola/sointu/go4k/bridge"
	"gopkg.in/yaml.v2"
)

func TestAllAsmFiles(t *testing.T) {
	_, myname, _, _ := runtime.Caller(0)
	files, err := filepath.Glob(path.Join(path.Dir(myname), "..", "..", "tests", "*.yml"))
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
			var song go4k.Song
			err = yaml.Unmarshal(asmcode, &song)
			if err != nil {
				t.Fatalf("could not parse the .yml file: %v", err)
			}
			synth, err := bridge.Synth(song.Patch)
			if err != nil {
				t.Fatalf("Compiling patch failed: %v", err)
			}
			buffer, err := go4k.Play(synth, song)
			buffer = buffer[:song.TotalRows()*song.SamplesPerRow()*2] // extend to the nominal length always.
			if err != nil {
				t.Fatalf("Play failed: %v", err)
			}
			if os.Getenv("GO4K_TEST_SAVE_OUTPUT") == "YES" {
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
			if song.Output16Bit {
				int16Buffer := convertToInt16Buffer(buffer)
				compareToRawInt16(t, int16Buffer, testname+".raw")
			} else {
				compareToRawFloat32(t, buffer, testname+".raw")
			}
		})
	}
}

func compareToRawFloat32(t *testing.T, buffer []float32, rawname string) {
	_, filename, _, _ := runtime.Caller(0)
	expectedb, err := ioutil.ReadFile(path.Join(path.Dir(filename), "..", "..", "tests", "expected_output", rawname))
	if err != nil {
		t.Fatalf("cannot read expected: %v", err)
	}
	expected := make([]float32, len(expectedb)/4)
	buf := bytes.NewReader(expectedb)
	err = binary.Read(buf, binary.LittleEndian, &expected)
	if err != nil {
		t.Fatalf("error converting expected buffer: %v", err)
	}
	if len(expected) != len(buffer) {
		t.Fatalf("buffer length mismatch, got %v, expected %v", len(buffer), len(expected))
	}
	for i, v := range expected {
		if math.IsNaN(float64(buffer[i])) || math.Abs(float64(v-buffer[i])) > 1e-6 {
			t.Fatalf("error bigger than 1e-6 detected, at sample position %v", i)
		}
	}
}

func compareToRawInt16(t *testing.T, buffer []int16, rawname string) {
	_, filename, _, _ := runtime.Caller(0)
	expectedb, err := ioutil.ReadFile(path.Join(path.Dir(filename), "..", "..", "tests", "expected_output", rawname))
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
