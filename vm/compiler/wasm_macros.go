package compiler

import (
	"bytes"
	"encoding/binary"
)

type WasmMacros struct {
	data   *bytes.Buffer
	Labels map[string]int
}

func NewWasmMacros() *WasmMacros {
	return &WasmMacros{
		data:   new(bytes.Buffer),
		Labels: map[string]int{},
	}
}

func (wm *WasmMacros) SetLabel(label string) string {
	wm.Labels[label] = wm.data.Len()
	return ""
}

func (wm *WasmMacros) GetLabel(label string) int {
	return wm.Labels[label]
}

func (wm *WasmMacros) DataB(value byte) string {
	binary.Write(wm.data, binary.LittleEndian, value)
	return ""
}

func (wm *WasmMacros) DataW(value uint16) string {
	binary.Write(wm.data, binary.LittleEndian, value)
	return ""
}

func (wm *WasmMacros) DataD(value uint32) string {
	binary.Write(wm.data, binary.LittleEndian, value)
	return ""
}

func (wm *WasmMacros) ToByte(value int) byte {
	return byte(value)
}

func (wm *WasmMacros) Data() []byte {
	return wm.data.Bytes()
}
