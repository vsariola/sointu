package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"gioui.org/app"
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/cmd"
	"github.com/vsariola/sointu/oto"
	"github.com/vsariola/sointu/tracker"
	"github.com/vsariola/sointu/tracker/gioui"
)

type NullContext struct {
}

func (NullContext) NextEvent() (event tracker.MIDINoteEvent, ok bool) {
	return tracker.MIDINoteEvent{}, false
}

func (NullContext) BPM() (bpm float64, ok bool) {
	return 0, false
}

func (NullContext) Params() (ret tracker.ExtValueArray, ok bool) {
	return tracker.ExtValueArray{}, false
}

func (NullContext) SetParams(params tracker.ExtParamArray) bool {
	return false
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

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
	defer audioContext.Close()
	recoveryFile := ""
	if configDir, err := os.UserConfigDir(); err == nil {
		recoveryFile = filepath.Join(configDir, "Sointu", "sointu-track-recovery")
	}
	model, player := tracker.NewModelPlayer(cmd.MainSynther, recoveryFile)
	if a := flag.Args(); len(a) > 0 {
		f, err := os.Open(a[0])
		if err == nil {
			model.ReadSong(f)
		}
		f.Close()
	}
	tracker := gioui.NewTracker(model)
	output := audioContext.Output()
	defer output.Close()
	go func() {
		buf := make(sointu.AudioBuffer, 1024)
		ctx := NullContext{}
		for {
			player.Process(buf, ctx)
			output.WriteAudio(buf)
		}
	}()
	go func() {
		tracker.Main()
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
