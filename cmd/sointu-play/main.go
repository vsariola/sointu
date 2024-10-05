package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/oto"
	"github.com/vsariola/sointu/version"
	"github.com/vsariola/sointu/vm/compiler/bridge"
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
	versionFlag := flag.Bool("v", false, "Print version.")
	flag.Usage = printUsage
	flag.Parse()
	if *versionFlag {
		fmt.Println(version.VersionOrHash)
		os.Exit(0)
	}
	if flag.NArg() == 0 || *help {
		flag.Usage()
		os.Exit(0)
	}
	if !*rawOut && !*wavOut {
		*play = true // if the user gives nothing to output, then the default behaviour is just to play the file
	}
	var audioContext sointu.AudioContext
	var playWaiter sointu.CloserWaiter
	if *play {
		var err error
		audioContext, err = oto.NewContext()
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not acquire oto AudioContext: %v\n", err)
			os.Exit(1)
		}
	}
	process := func(filename string) error {
		output := func(extension string, contents []byte) error {
			if *stdout {
				fmt.Print(contents)
				return nil
			}
			_, name := filepath.Split(filename)
			var dir string
			if *directory != "" {
				dir = *directory
			}
			if dir == "" {
				var err error
				dir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("could not get working directory, specify the output directory explicitly: %v", err)
				}
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
		buffer, err := sointu.Play(bridge.NativeSynther{}, song, nil) // render the song to calculate its length
		if err != nil {
			return fmt.Errorf("sointu.Play failed: %v", err)
		}
		if *play {
			playWaiter = audioContext.Play(buffer.Source())
		}
		if *rawOut {
			raw, err := buffer.Raw(*pcm)
			if err != nil {
				return fmt.Errorf("could not generate .raw file: %v", err)
			}
			if err := output(".raw", raw); err != nil {
				return fmt.Errorf("error outputting .raw file: %v", err)
			}
		}
		if *wavOut {
			wav, err := buffer.Wav(*pcm)
			if err != nil {
				return fmt.Errorf("could not generate .wav file: %v", err)
			}
			if err := output(".wav", wav); err != nil {
				return fmt.Errorf("error outputting .wav file: %v", err)
			}
		}
		if *play {
			playWaiter.Wait()
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
