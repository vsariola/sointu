package bridge

// #include "sointu.h"
import "C"

func init() {
	C.su_load_gmdls() // GM.DLS is an windows specific sound bank so samples work currently only on windows
}
