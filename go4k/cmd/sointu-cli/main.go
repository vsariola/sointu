package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/vsariola/sointu/go4k"
	"github.com/vsariola/sointu/go4k/audio/oto"
	"github.com/vsariola/sointu/go4k/bridge"
)

func main() {
	write := flag.Bool("w", false, "Do not output to standard output; (over)write files on disk instead.")
	list := flag.Bool("l", false, "Do not output to standard output; list files that change if -w is applied.")
	help := flag.Bool("h", false, "Show help.")
	play := flag.Bool("p", false, "Play the input songs.")
	asmOut := flag.Bool("a", false, "Output the song as .asm file, to standard output unless otherwise specified.")
	jsonOut := flag.Bool("j", false, "Output the song as .json file, to standard output unless otherwise specified.")
	headerOut := flag.Bool("c", false, "Output .h C header file, to standard output unless otherwise specified.")
	exactLength := flag.Bool("e", false, "When outputting the C header file, calculate the exact length of song by rendering it once. Only useful when using SPEED opcodes.")
	rawOut := flag.Bool("r", false, "Output the rendered song as .raw stereo float32 buffer, to standard output unless otherwise specified.")
	directory := flag.String("d", "", "Directory where to output all files. The directory and its parents are created if needed. By default, everything is placed in the same directory where the original song file is.")
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() == 0 || *help {
		flag.Usage()
		os.Exit(0)
	}
	if !*asmOut && !*jsonOut && !*rawOut && !*headerOut && !*play {
		*play = true // if the user gives nothing to output, then the default behaviour is just to play the file
	}
	needsRendering := *play || *exactLength || *rawOut
	if needsRendering {
		bridge.Init()
	}
	process := func(filename string) error {
		output := func(extension string, contents []byte) error {
			if !*write && !*list {
				fmt.Print(string(contents))
				return nil
			}
			dir, name := filepath.Split(filename)
			if *directory != "" {
				dir = *directory
			}
			name = strings.TrimSuffix(name, filepath.Ext(name)) + extension
			f := filepath.Join(dir, name)
			original, err := ioutil.ReadFile(f)
			if err == nil {
				if bytes.Compare(original, contents) == 0 {
					return nil // no need to update
				}
			}
			if *list {
				fmt.Println(f)
			}
			if *write {
				if err := os.MkdirAll(dir, os.ModePerm); err != nil {
					return fmt.Errorf("Could not create output directory %v: %v", dir, err)
				}
				err := ioutil.WriteFile(f, contents, 0644)
				if err != nil {
					return fmt.Errorf("Could not write file %v: %v", f, err)
				}
			}
			return nil
		}
		inputBytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("Could not read file %v: %v", filename, err)
		}
		var song go4k.Song
		if err := json.Unmarshal(inputBytes, &song); err != nil {
			song2, err2 := go4k.DeserializeAsm(string(inputBytes))
			if err2 != nil {
				return fmt.Errorf("The song could not be parsed as .json (%v) nor .asm (%v)", err, err2)
			}
			song = *song2
		}
		var buffer []float32
		if needsRendering {
			synth, err := bridge.Synth(song.Patch)
			if err != nil {
				return fmt.Errorf("Could not create synth based on the patch: %v", err)
			}
			buffer, err = go4k.Play(synth, song) // render the song to calculate its length
			if err != nil {
				return fmt.Errorf("go4k.Play failed: %v", err)
			}
		}
		if *play {
			player, err := oto.NewPlayer()
			if err != nil {
				return fmt.Errorf("Error creating oto player: %v", err)
			}
			defer player.Close()
			if err := player.Play(buffer); err != nil {
				return fmt.Errorf("Error playing: %v", err)
			}
		}
		if *headerOut {
			maxSamples := 0 // 0 means it is calculated automatically
			if *exactLength {

				maxSamples = len(buffer) / 2
			}
			header := go4k.ExportCHeader(&song, maxSamples)
			if err := output(".h", []byte(header)); err != nil {
				return fmt.Errorf("Error outputting header file: %v", err)
			}
		}
		if *asmOut {
			asmCode, err := go4k.SerializeAsm(&song)
			if err != nil {
				return fmt.Errorf("Could not format the song as asm file: %v", err)
			}
			if err := output(".asm", []byte(asmCode)); err != nil {
				return fmt.Errorf("Error outputting asm file: %v", err)
			}
		}
		if *jsonOut {
			jsonSong, err := json.Marshal(song)
			if err != nil {
				return fmt.Errorf("Could not marshal the song as json file: %v", err)
			}
			if err := output(".json", jsonSong); err != nil {
				return fmt.Errorf("Error outputting JSON file: %v", err)
			}
		}
		if *rawOut {
			buf := new(bytes.Buffer)
			err := binary.Write(buf, binary.LittleEndian, buffer)
			if err != nil {
				return fmt.Errorf("Could not binary write the float32 buffer to a byte buffer: %v", err)
			}
			if err := output(".raw", buf.Bytes()); err != nil {
				return fmt.Errorf("Error outputting raw audio file: %v", err)
			}
		}
		return nil
	}
	retval := 0
	for _, param := range flag.Args() {
		if info, err := os.Stat(param); err == nil && info.IsDir() {
			asmfiles, err := filepath.Glob(filepath.Join(param, "*.asm"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not glob the path %v for asm files: %v\n", param, err)
				retval = 1
				continue
			}
			jsonfiles, err := filepath.Glob(filepath.Join(param, "*.json"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not glob the path %v for json files: %v\n", param, err)
				retval = 1
				continue
			}
			files := append(asmfiles, jsonfiles...)
			for _, file := range files {
				err := process(file)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Could not process file %v: %v\n", file, err)
					retval = 1
				}
			}
		} else {
			err := process(param)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not process file %v: %v\n", param, err)
				retval = 1
			}
		}
	}
	os.Exit(retval)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Sointu command line utility for processing .asm/.json song files.\nUsage: %s [flags] [path ...]\n", os.Args[0])
	flag.PrintDefaults()
}
