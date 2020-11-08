// +build windows

package bridge

// #cgo CFLAGS: -I"${SRCDIR}/../../include/sointu"
// #cgo LDFLAGS: "${SRCDIR}/../../build/libsointu.a"
// #include <sointu.h>
import "C"

func Init() {
	C.su_load_gmdls() // GM.DLS is an windows specific sound bank so samples work currently only on windows
}
