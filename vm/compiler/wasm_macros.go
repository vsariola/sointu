package compiler

import (
	"bytes"
	"encoding/binary"
)

// WasmMacros are the macros called from .wat templates
//
// NOTE! Due to the single pass nature of the compilation and the way memory is
// organized, you should initialize all initialized data in the .wat files using
// DataB, DataW and DataD macros _before_ any calls to Block. Block allocates
// uninitialized data blocks from the memory.
type WasmMacros struct {
	data       *bytes.Buffer
	blockStart int
	blockAlign int
	Labels     map[string]int
}

func NewWasmMacros() *WasmMacros {
	return &WasmMacros{
		data:       new(bytes.Buffer),
		blockAlign: 128,
		Labels:     map[string]int{},
	}
}

func (wm *WasmMacros) SetDataLabel(label string) string {
	wm.Labels[label] = wm.data.Len()
	return ""
}

func (wm *WasmMacros) SetBlockLabel(label string) string {
	wm.Labels[label] = wm.blockStart
	return ""
}

func (wm *WasmMacros) Align() string {
	wm.blockStart += wm.blockAlign - 1 - ((wm.blockStart + wm.blockAlign - 1) % wm.blockAlign)
	return ""
}

func (wm *WasmMacros) MemoryPages() int {
	return (wm.blockStart + 65535) / 65536
}

func (wm *WasmMacros) GetLabel(label string) int {
	return wm.Labels[label]
}

func (wm *WasmMacros) DataB(value byte) string {
	binary.Write(wm.data, binary.LittleEndian, value)
	wm.blockStart++
	return ""
}

func (wm *WasmMacros) DataW(value uint16) string {
	binary.Write(wm.data, binary.LittleEndian, value)
	wm.blockStart += 2
	return ""
}

func (wm *WasmMacros) DataD(value uint32) string {
	binary.Write(wm.data, binary.LittleEndian, value)
	wm.blockStart += 4
	return ""
}

func (wm *WasmMacros) Block(value int) string {
	wm.blockStart += value
	return ""
}

func (wm *WasmMacros) ToByte(value int) byte {
	return byte(value)
}

func (wm *WasmMacros) Data() []byte {
	return wm.data.Bytes()
}
