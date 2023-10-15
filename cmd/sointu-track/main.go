package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"gioui.org/app"
	"github.com/vsariola/sointu/cmd"
	"github.com/vsariola/sointu/oto"
	"github.com/vsariola/sointu/tracker"
	"github.com/vsariola/sointu/tracker/gioui"
)

type NullContext struct {
}

func (NullContext) NextEvent() (event tracker.PlayerProcessEvent, ok bool) {
	return tracker.PlayerProcessEvent{}, false
}

func (NullContext) BPM() (bpm float64, ok bool) {
	return 0, false
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	audioContext, err := oto.NewContext()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer audioContext.Close()
	modelMessages := make(chan interface{}, 1024)
	playerMessages := make(chan tracker.PlayerMessage, 1024)
	model, err := tracker.LoadRecovery(modelMessages, playerMessages)
	if err != nil {
		model = tracker.NewModel(modelMessages, playerMessages)
	}
	player := tracker.NewPlayer(cmd.DefaultService, playerMessages, modelMessages)
	tracker := gioui.NewTracker(model, cmd.DefaultService)
	output := audioContext.Output()
	defer output.Close()
	go func() {
		buf := make([]float32, 2048)
		ctx := NullContext{}
		for {
			player.Process(buf, ctx)
			output.WriteAudio(buf)
		}
	}()
	go func() {
		tracker.Main()
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
