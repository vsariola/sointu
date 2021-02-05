package main

import (
	"fmt"
	"os"

	"gioui.org/app"
	"gioui.org/unit"
	"github.com/vsariola/sointu/bridge"
	"github.com/vsariola/sointu/oto"
	"github.com/vsariola/sointu/tracker"
)

func main() {
	audioContext, err := oto.NewContext()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer audioContext.Close()
	synthService := bridge.BridgeService{}
	go func() {
		w := app.NewWindow(
			app.Size(unit.Dp(800), unit.Dp(600)),
			app.Title("Sointu Tracker"),
		)
		t := tracker.New(audioContext, synthService)
		defer t.Close()
		if err := t.Run(w); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}
