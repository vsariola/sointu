package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/vsariola/sointu/go4k"
	"github.com/vsariola/sointu/go4k/audio"
	"github.com/vsariola/sointu/go4k/audio/oto"
	"github.com/vsariola/sointu/go4k/bridge"
)

func main() {
	// parse flags
	quiet := flag.Bool("quiet", false, "no sound output")
	out := flag.String("out", "", "write output to file")
	help := flag.Bool("h", false, "show help")
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() == 0 || *help {
		flag.Usage()
		os.Exit(0)
	}

	// read input song
	var song go4k.Song
	if bytes, err := ioutil.ReadFile(flag.Arg(0)); err != nil {
		fmt.Printf("Cannot read song file: %v", err)
		os.Exit(1)
	} else if err := json.Unmarshal(bytes, &song); err != nil {
		song2, err2 := go4k.DeserializeAsm(string(bytes))
		if err2 != nil {
			fmt.Printf("Cannot unmarshal / parse song file: %v / %v", err, err2)
			os.Exit(1)
		}
		song = *song2
	}

	bridge.Init()

	// set up synth
	synth, err := bridge.Synth(song.Patch)
	if err != nil {
		fmt.Printf("Cannot create synth: %v", err)
		os.Exit(1)
	}

	// render the actual data for the entire song
	fmt.Print("Rendering.. ")
	buff, err := go4k.Play(synth, song)
	if err != nil {
		fmt.Printf("Error rendering with go4k: %v\n", err.Error())
		os.Exit(1)
	} else {
		fmt.Printf("Rendered %v samples.\n", len(buff))
	}

	// play output if not in quiet mode
	if !*quiet {
		fmt.Print("Playing.. ")
		player, err := oto.NewPlayer()
		if err != nil {
			fmt.Printf("Error creating oto player: %v\n", err.Error())
			os.Exit(1)
		}
		defer player.Close()
		if err := player.Play(buff); err != nil {
			fmt.Printf("Error playing: %v\n", err.Error())
			os.Exit(1)
		}
		fmt.Println("Played.")
	}

	// write output to file if output given
	if out != nil && *out != "" {
		fmt.Printf("Writing output to %v.. ", *out)
		if bbuffer, err := audio.FloatBufferTo16BitLE(buff); err != nil {
			fmt.Printf("Error converting buffer: %v\n", err.Error())
			os.Exit(1)
		} else if err := ioutil.WriteFile(*out, bbuffer, os.ModePerm); err != nil {
			fmt.Printf("Error writing: %v\n", err.Error())
			os.Exit(1)
		} else {
			fmt.Printf("Wrote %v bytes.\n", len(bbuffer))
		}
	}

	fmt.Println("All done.")
	os.Exit(0)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] [SONG FILE] [OUTPUT FILE]\n", os.Args[0])
	flag.PrintDefaults()
}
