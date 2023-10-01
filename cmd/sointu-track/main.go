package main

import (
	"flag"
	"fmt"
	"os"

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

func main() {
	flag.Parse()
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
		os.Exit(0)
	}()
	app.Main()
}
