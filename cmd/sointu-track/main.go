package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"gioui.org/app"
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/cmd"
	"github.com/vsariola/sointu/oto"
	"github.com/vsariola/sointu/tracker"
	"github.com/vsariola/sointu/tracker/gioui"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var defaultMidiInput = flag.String("midi-input", "", "connect MIDI input to matching device name")
var firstMidiInput = flag.Bool("first-midi-input", false, "connect MIDI input to first device found")

func main() {
	flag.Parse()
	var f *os.File
	if *cpuprofile != "" {
		var err error
		f, err = os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}
	audioContext, err := oto.NewContext()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	recoveryFile := ""
	if configDir, err := os.UserConfigDir(); err == nil {
		recoveryFile = filepath.Join(configDir, "Sointu", "sointu-track-recovery")
	}
	broker := tracker.NewBroker()
	midiContext := cmd.NewMidiContext(broker)
	defer midiContext.Close()
	midiContext.TryToOpenBy(*defaultMidiInput, *firstMidiInput)
	model := tracker.NewModel(broker, cmd.Synthers, midiContext, recoveryFile)
	player := tracker.NewPlayer(broker, cmd.Synthers[0])
	detector := tracker.NewDetector(broker)
	go detector.Run()

	if a := flag.Args(); len(a) > 0 {
		f, err := os.Open(a[0])
		if err == nil {
			model.ReadSong(f)
		}
		f.Close()
	}

	trackerUi := gioui.NewTracker(model)
	audioCloser := audioContext.Play(func(buf sointu.AudioBuffer) error {
		player.Process(buf, tracker.NullPlayerProcessContext{})
		return nil
	})

	go func() {
		trackerUi.Main()
		audioCloser.Close()
		tracker.TrySend(broker.CloseDetector, struct{}{})
		tracker.TimeoutReceive(broker.FinishedDetector, 3*time.Second)
		if *cpuprofile != "" {
			pprof.StopCPUProfile()
			f.Close()
		}
		if *memprofile != "" {
			f, err := os.Create(*memprofile)
			if err != nil {
				log.Fatal("could not create memory profile: ", err)
			}
			defer f.Close() // error handling omitted for example
			runtime.GC()    // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}
		}
		os.Exit(0)
	}()
	app.Main()
}
