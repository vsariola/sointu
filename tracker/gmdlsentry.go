package tracker

import "github.com/vsariola/sointu/vm"

type GmDlsEntry struct {
	Start              int
	LoopStart          int
	LoopLength         int
	SuggestedTranspose int
	Name               string
}

var GmDlsEntryMap = make(map[vm.SampleOffset]int)

func init() {
	for i, e := range GmDlsEntries {
		key := vm.SampleOffset{Start: uint32(e.Start), LoopStart: uint16(e.LoopStart), LoopLength: uint16(e.LoopLength)}
		GmDlsEntryMap[key] = i
	}
}

//go:generate go run generate/main.go
