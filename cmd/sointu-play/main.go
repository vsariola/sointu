package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/bridge"
	"github.com/vsariola/sointu/oto"
)

func main() {
	stdout := flag.Bool("s", false, "Do not write files; write to standard output instead.")
	help := flag.Bool("h", false, "Show help.")
	directory := flag.String("o", "", "Directory where to output all files. The directory and its parents are created if needed. By default, everything is placed in the same directory where the original song file is.")
	play := flag.Bool("p", false, "Play the input songs (default behaviour when no other output is defined).")
	//start := flag.Float64("start", 0, "Start playing from part; given in the units defined by parameter `unit`.")
	//stop := flag.Float64("stop", -1, "Stop playing at part; given in the units defined by parameter `unit`. Negative values indicate render until end.")
	//units := flag.String("unit", "pattern", "Units for parameters start and stop. Possible values: second, sample, pattern, beat. Warning: beat and pattern do not take SPEED modulations into account.")
	rawOut := flag.Bool("r", false, "Output the rendered song as .raw file. By default, saves stereo float32 buffer to disk.")
	wavOut := flag.Bool("w", false, "Output the rendered song as .wav file. By default, saves stereo float32 buffer to disk.")
	pcm := flag.Bool("c", false, "Convert audio to 16-bit signed PCM when outputting.")
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() == 0 || *help {
		flag.Usage()
		os.Exit(0)
	}
	if !*rawOut && !*wavOut {
		*play = true // if the user gives nothing to output, then the default behaviour is just to play the file
	}
	var audioContext sointu.AudioContext
	if *play {
		audioContext, err := oto.NewContext()
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not acquire oto AudioContext: %v\n", err)
			os.Exit(1)
		}
		defer audioContext.Close()
	}
	process := func(filename string) error {
		output := func(extension string, contents []byte) error {
			if *stdout {
				fmt.Print(contents)
				return nil
			}
			dir, name := filepath.Split(filename)
			if *directory != "" {
				dir = *directory
			}
			name = strings.TrimSuffix(name, filepath.Ext(name)) + extension
			f := filepath.Join(dir, name)
			if dir != "" {
				if err := os.MkdirAll(dir, os.ModePerm); err != nil {
					return fmt.Errorf("could not create output directory %v: %v", dir, err)
				}
			}
			err := ioutil.WriteFile(f, contents, 0644)
			if err != nil {
				return fmt.Errorf("could not write file %v: %v", f, err)
			}
			return nil
		}
		inputBytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("could not read file %v: %v", filename, err)
		}
		var song sointu.Song
		if errJSON := json.Unmarshal(inputBytes, &song); errJSON != nil {
			if errYaml := yaml.Unmarshal(inputBytes, &song); errYaml != nil {
				return fmt.Errorf("the song could not be parsed as .json (%v) or .yml (%v)", errJSON, errYaml)
			}
		}
		synth, err := bridge.Synth(song.Patch)
		if err != nil {
			return fmt.Errorf("could not create synth based on the patch: %v", err)
		}
		buffer, err := sointu.Play(synth, song) // render the song to calculate its length
		if err != nil {
			return fmt.Errorf("sointu.Play failed: %v", err)
		}
		if *play {
			output := audioContext.Output()
			defer output.Close()
			if err := output.WriteAudio(buffer); err != nil {
				return fmt.Errorf("error playing: %v", err)
			}
		}
		var data interface{}
		data = buffer
		if *pcm {
			int16buffer := make([]int16, len(buffer))
			for i, v := range buffer {
				int16buffer[i] = int16(clamp(int(v*math.MaxInt16), math.MinInt16, math.MaxInt16))
			}
			data = int16buffer
		}
		if *rawOut {
			buf := new(bytes.Buffer)
			err := binary.Write(buf, binary.LittleEndian, data)
			if err != nil {
				return fmt.Errorf("could not binary write data to binary buffer: %v", err)
			}
			if err := output(".raw", buf.Bytes()); err != nil {
				return fmt.Errorf("error outputting raw audio file: %v", err)
			}
		}
		if *wavOut {
			buf := new(bytes.Buffer)
			header := createWavHeader(len(buffer), *pcm)
			err := binary.Write(buf, binary.LittleEndian, header)
			if err != nil {
				return fmt.Errorf("could not binary write header to binary buffer: %v", err)
			}
			err = binary.Write(buf, binary.LittleEndian, data)
			if err != nil {
				return fmt.Errorf("could not binary write data to binary buffer: %v", err)
			}
			if err := output(".wav", buf.Bytes()); err != nil {
				return fmt.Errorf("error outputting wav audio file: %v", err)
			}
		}
		return nil
	}
	retval := 0
	for _, param := range flag.Args() {
		if info, err := os.Stat(param); err == nil && info.IsDir() {
			jsonfiles, err := filepath.Glob(filepath.Join(param, "*.json"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not glob the path %v for json files: %v\n", param, err)
				retval = 1
				continue
			}
			ymlfiles, err := filepath.Glob(filepath.Join(param, "*.yml"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not glob the path %v for yml files: %v\n", param, err)
				retval = 1
				continue
			}
			files := append(ymlfiles, jsonfiles...)
			for _, file := range files {
				err := process(file)
				if err != nil {
					fmt.Fprintf(os.Stderr, "could not process file %v: %v\n", file, err)
					retval = 1
				}
			}
		} else {
			err := process(param)
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not process file %v: %v\n", param, err)
				retval = 1
			}
		}
	}
	os.Exit(retval)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Sointu command line utility for playing .asm/.json song files.\nUsage: %s [flags] [path ...]\n", os.Args[0])
	flag.PrintDefaults()
}

func createWavHeader(bufferLength int, pcm bool) []byte {
	// Refer to: http://www-mmsp.ece.mcgill.ca/Documents/AudioFormats/WAVE/WAVE.html
	numChannels := 2
	sampleRate := 44100
	var bytesPerSample, chunkSize, fmtChunkSize, waveFormat int
	var factChunk bool
	if pcm {
		bytesPerSample = 2
		chunkSize = 36 + bytesPerSample*bufferLength
		fmtChunkSize = 16
		waveFormat = 1 // PCM
		factChunk = false
	} else {
		bytesPerSample = 4
		chunkSize = 50 + bytesPerSample*bufferLength
		fmtChunkSize = 18
		waveFormat = 3 // IEEE float
		factChunk = true
	}
	buf := new(bytes.Buffer)
	buf.Write([]byte("RIFF"))
	binary.Write(buf, binary.LittleEndian, uint32(chunkSize))
	buf.Write([]byte("WAVE"))
	buf.Write([]byte("fmt "))
	binary.Write(buf, binary.LittleEndian, uint32(fmtChunkSize))
	binary.Write(buf, binary.LittleEndian, uint16(waveFormat))
	binary.Write(buf, binary.LittleEndian, uint16(numChannels))
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate*numChannels*bytesPerSample)) // avgBytesPerSec
	binary.Write(buf, binary.LittleEndian, uint16(numChannels*bytesPerSample))            // blockAlign
	binary.Write(buf, binary.LittleEndian, uint16(8*bytesPerSample))                      // bits per sample
	if fmtChunkSize > 16 {
		binary.Write(buf, binary.LittleEndian, uint16(0)) // size of extension
	}
	if factChunk {
		buf.Write([]byte("fact"))
		binary.Write(buf, binary.LittleEndian, uint32(4))            // fact chunk size
		binary.Write(buf, binary.LittleEndian, uint32(bufferLength)) // sample length
	}
	buf.Write([]byte("data"))
	binary.Write(buf, binary.LittleEndian, uint32(bytesPerSample*bufferLength))
	return buf.Bytes()
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
