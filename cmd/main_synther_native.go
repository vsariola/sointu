//go:build native

package cmd

import "github.com/vsariola/sointu/vm/compiler/bridge"

var MainSynther = bridge.NativeSynther{}
