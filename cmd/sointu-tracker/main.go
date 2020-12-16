package main

import (
	"fmt"
	"gioui.org/app"
	"gioui.org/unit"
	"github.com/vsariola/sointu/audio/oto"
	"github.com/vsariola/sointu/tracker"
	"os"
)

func main() {
	plr, err := oto.NewPlayer()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer plr.Close()
	go func() {
		w := app.NewWindow(
			app.Size(unit.Dp(800), unit.Dp(600)),
			app.Title("Sointu Tracker"),
		)
		t := tracker.New(plr)
		defer t.Close()
		if err := t.Run(w); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}
