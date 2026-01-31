//go:build native

package cmd

import (
	"github.com/vsariola/sointu/vm/compiler/bridge"
)

func init() {
	Synthers = append(Synthers, bridge.NativeSynther{})
}
