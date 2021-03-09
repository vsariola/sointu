package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vsariola/sointu/oto"
	"github.com/vsariola/sointu/rpc"
	"github.com/vsariola/sointu/tracker/gioui"
	"github.com/vsariola/sointu/vm"
)

func main() {
	syncAddress := flag.String("address", "", "remote RPC server where to send sync data")
	flag.Parse()
	audioContext, err := oto.NewContext()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer audioContext.Close()
	var syncChannel chan<- []float32
	if *syncAddress != "" {
		syncChannel, err = rpc.Sender(*syncAddress)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	synthService := vm.SynthService{}
	gioui.Main(audioContext, synthService, syncChannel)
}
