package go4k_test

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
)

func TestAllAsmFiles(t *testing.T) {
	bridge.Init()
	_, myname, _, _ := runtime.Caller(0)
	files, err := filepath.Glob(path.Join(path.Dir(myname), "..", "tests", "*.asm"))
	if err != nil {
		t.Fatalf("cannot glob files in the test directory: %v", err)
	}
	for _, filename := range files {
		basename := filepath.Base(filename)
		testname := strings.TrimSuffix(basename, path.Ext(basename))
		t.Run(testname, func(t *testing.T) {
			asmcode, err := ioutil.ReadFile(filename)
			if err != nil {
				t.Fatalf("cannot read the .asm file: %v", filename)
			}
			song, err := go4k.DeserializeAsm(string(asmcode))
			if err != nil {
				t.Fatalf("could not parse the .asm file: %v", err)
			}
			synth, err := bridge.Synth(song.Patch)
			if err != nil {
				t.Fatalf("Compiling patch failed: %v", err)
			}
			buffer, err := go4k.Play(synth, *song)
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
			compareToRaw(t, buffer, testname+".raw")
		})
	}
}

func TestSerializingAllAsmFiles(t *testing.T) {
	bridge.Init()
	_, myname, _, _ := runtime.Caller(0)
	files, err := filepath.Glob(path.Join(path.Dir(myname), "..", "tests", "*.asm"))
	if err != nil {
		t.Fatalf("cannot glob files in the test directory: %v", err)
	}
	for _, filename := range files {
		basename := filepath.Base(filename)
		testname := strings.TrimSuffix(basename, path.Ext(basename))
		t.Run(testname, func(t *testing.T) {
			asmcode, err := ioutil.ReadFile(filename)
			if err != nil {
				t.Fatalf("cannot read the .asm file: %v", filename)
			}
			song, err := go4k.DeserializeAsm(string(asmcode)) // read the asm
			if err != nil {
				t.Fatalf("could not parse the .asm file: %v", err)
			}
			str, err := go4k.SerializeAsm(song) // serialize again
			if err != nil {
				t.Fatalf("Could not serialize asm file: %v", err)
			}
			song, err = go4k.DeserializeAsm(str) // deserialize again. The rendered song should still give same results.
			if err != nil {
				t.Fatalf("could not parse the serialized asm code: %v", err)
			}
			synth, err := bridge.Synth(song.Patch)
			if err != nil {
				t.Fatalf("Compiling patch failed: %v", err)
			}
			buffer, err := go4k.Play(synth, *song)
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
			compareToRaw(t, buffer, testname+".raw")
		})
	}
}

func compareToRaw(t *testing.T, buffer []float32, rawname string) {
	_, filename, _, _ := runtime.Caller(0)
	expectedb, err := ioutil.ReadFile(path.Join(path.Dir(filename), "..", "tests", "expected_output", rawname))
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
