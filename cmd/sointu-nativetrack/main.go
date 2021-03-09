package main

import (
	"fmt"
	"os"

	"github.com/vsariola/sointu/oto"
	"github.com/vsariola/sointu/tracker/gioui"
	"github.com/vsariola/sointu/vm/compiler/bridge"
)

func main() {
	audioContext, err := oto.NewContext()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer audioContext.Close()
	synthService := bridge.BridgeService{}
	// TODO: native track does not support syncing at the moment (which is why
	// we pass nil), as the native bridge does not support sync data
	gioui.Main(audioContext, synthService, nil)
}
