package tracker

import "github.com/vsariola/sointu/compiler"

type GmDlsEntry struct {
	Start              int
	LoopStart          int
	LoopLength         int
	SuggestedTranspose int
	Name               string
}

var GmDlsEntryMap = make(map[compiler.SampleOffset]int)

func init() {
	for i, e := range GmDlsEntries {
		key := compiler.SampleOffset{Start: uint32(e.Start), LoopStart: uint16(e.LoopStart), LoopLength: uint16(e.LoopLength)}
		GmDlsEntryMap[key] = i
	}
}

//go:generate go run ../cmd/sointu-generate/main.go
