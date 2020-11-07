package main

import (
	"fmt"
	"gioui.org/app"
	"gioui.org/unit"
	"github.com/vsariola/sointu/go4k/tracker"
	"os"
)

func main() {
	go func() {
		w := app.NewWindow(
			app.Size(unit.Dp(800), unit.Dp(600)),
			app.Title("Sointu Tracker"),
		)
		if err := tracker.New().Run(w); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}
