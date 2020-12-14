package compiler

import (
	"fmt"
	"math"
	"strings"

	"github.com/vsariola/sointu/go4k"
)

type OplistEntry struct {
	Type      string
	NumParams int
}

type Macros struct {
	Stacklocs      []string
	Output16Bit    bool
	Clip           bool
	Library        bool
	Sine           int // TODO: how can we elegantly access global constants in template, without wrapping each one by one
	Trisaw         int
	Pulse          int
	Gate           int
	Sample         int
	usesFloatConst map[float32]bool
	usesIntConst   map[int]bool
	floatConsts    []float32
	intConsts      []int
	calls          map[string]bool
	stackframes    map[string][]string
	FeatureSet
	Compiler
}

func NewMacros(c Compiler, f FeatureSet) *Macros {
	return &Macros{
		calls:          map[string]bool{},
		usesFloatConst: map[float32]bool{},
		usesIntConst:   map[int]bool{},
		stackframes:    map[string][]string{},
		Sine:           go4k.Sine,
		Trisaw:         go4k.Trisaw,
		Pulse:          go4k.Pulse,
		Gate:           go4k.Gate,
		Sample:         go4k.Sample,
		Compiler:       c,
		FeatureSet:     f,
	}
}

func (p *Macros) HasOp(instruction string) bool {
	_, ok := p.Opcode(instruction)
	return ok
}

func (p *Macros) Stereo(unitType string) bool {
	return p.SupportsParamValue(unitType, "stereo", 1)
}

func (p *Macros) Mono(unitType string) bool {
	return p.SupportsParamValue(unitType, "stereo", 0)
}

func (p *Macros) StereoAndMono(unitType string) bool {
	return p.Stereo(unitType) && p.Mono(unitType)
}

// Macros and functions to accumulate constants automagically

func (p *Macros) Float(value float32) string {
	if _, ok := p.usesFloatConst[value]; !ok {
		p.usesFloatConst[value] = true
		p.floatConsts = append(p.floatConsts, value)
	}
	return nameForFloat(value)
}

func (p *Macros) Int(value int) string {
	if _, ok := p.usesIntConst[value]; !ok {
		p.usesIntConst[value] = true
		p.intConsts = append(p.intConsts, value)
	}
	return nameForInt(value)
}

func (p *Macros) Constants() string {
	var b strings.Builder
	for _, v := range p.floatConsts {
		fmt.Fprintf(&b, "%-23s dd 0x%x\n", nameForFloat(v), math.Float32bits(v))
	}
	for _, v := range p.intConsts {
		fmt.Fprintf(&b, "%-23s dd 0x%x\n", nameForInt(v), v)
	}
	return b.String()
}

func nameForFloat(value float32) string {
	s := fmt.Sprintf("%#g", value)
	s = strings.Replace(s, ".", "_", 1)
	s = strings.Replace(s, "-", "m", 1)
	s = strings.Replace(s, "+", "p", 1)
	return "FCONST_" + s
}

func nameForInt(value int) string {
	return "ICONST_" + fmt.Sprintf("%d", value)
}

func (p *Macros) PTRSIZE() int {
	if p.Amd64 {
		return 8
	}
	return 4
}

func (p *Macros) DPTR() string {
	if p.Amd64 {
		return "dq"
	}
	return "dd"
}

func (p *Macros) PTRWORD() string {
	if p.Amd64 {
		return "qword"
	}
	return "dword"
}

func (p *Macros) AX() string {
	if p.Amd64 {
		return "rax"
	}
	return "eax"
}

func (p *Macros) BX() string {
	if p.Amd64 {
		return "rbx"
	}
	return "ebx"
}

func (p *Macros) CX() string {
	if p.Amd64 {
		return "rcx"
	}
	return "ecx"
}

func (p *Macros) DX() string {
	if p.Amd64 {
		return "rdx"
	}
	return "edx"
}

func (p *Macros) SI() string {
	if p.Amd64 {
		return "rsi"
	}
	return "esi"
}

func (p *Macros) DI() string {
	if p.Amd64 {
		return "rdi"
	}
	return "edi"
}

func (p *Macros) SP() string {
	if p.Amd64 {
		return "rsp"
	}
	return "esp"
}

func (p *Macros) BP() string {
	if p.Amd64 {
		return "rbp"
	}
	return "ebp"
}

func (p *Macros) WRK() string {
	return p.BP()
}

func (p *Macros) VAL() string {
	return p.SI()
}

func (p *Macros) COM() string {
	return p.BX()
}

func (p *Macros) INP() string {
	return p.DX()
}

func (p *Macros) SaveStack(scope string) string {
	p.stackframes[scope] = p.Stacklocs
	return ""
}

func (p *Macros) Call(funcname string) (string, error) {
	p.calls[funcname] = true
	var s = make([]string, len(p.Stacklocs))
	copy(s, p.Stacklocs)
	p.stackframes[funcname] = s
	return "call    " + funcname, nil
}

func (p *Macros) TailCall(funcname string) (string, error) {
	p.calls[funcname] = true
	p.stackframes[funcname] = p.Stacklocs
	return "jmp     " + funcname, nil
}

func (p *Macros) SectText(name string) string {
	if p.OS == "windows" {
		if p.DisableSections {
			return "section .code align=1"
		}
		return fmt.Sprintf("section .%v code align=1", name)
	} else if p.OS == "darwin" {
		return "section .text align=1"
	} else {
		if p.DisableSections {
			return "section .text. progbits alloc exec nowrite align=1"
		}
		return fmt.Sprintf("section .text.%v progbits alloc exec nowrite align=1", name)
	}
}

func (p *Macros) SectData(name string) string {
	if p.OS == "windows" || p.OS == "darwin" {
		if p.OS == "windows" && !p.DisableSections {
			return fmt.Sprintf("section .%v data align=1", name)
		}
		return "section .data align=1"
	} else {
		if !p.DisableSections {
			return fmt.Sprintf("section .data.%v progbits alloc noexec write align=1", name)
		}
		return "section .data. progbits alloc exec nowrite align=1"
	}
}

func (p *Macros) SectBss(name string) string {
	if p.OS == "windows" || p.OS == "darwin" {
		if p.OS == "windows" && !p.DisableSections {
			return fmt.Sprintf("section .%v bss align=256", name)
		}
		return "section .bss align=256"
	} else {
		if !p.DisableSections {
			return fmt.Sprintf("section .bss.%v progbits alloc noexec write align=256", name)
		}
		return "section .bss. progbits alloc exec nowrite align=256"
	}
}

func (p *Macros) Data(label string) string {
	return fmt.Sprintf("%v\n%v:", p.SectData(label), label)
}

func (p *Macros) Func(funcname string, scope ...string) (string, error) {
	scopeName := funcname
	if len(scope) > 1 {
		return "", fmt.Errorf(`Func macro "%v" can take only one additional scope parameter, "%v" were given`, funcname, scope)
	} else if len(scope) > 0 {
		scopeName = scope[0]
	}
	p.Stacklocs = append(p.stackframes[scopeName], "retaddr_"+funcname)
	return fmt.Sprintf("%v\n%v:", p.SectText(funcname), funcname), nil
}

func (p *Macros) HasCall(funcname string) bool {
	return p.calls[funcname]
}

func (p *Macros) Push(value string, name string) string {
	p.Stacklocs = append(p.Stacklocs, name)
	return fmt.Sprintf("push    %v		; Stack: %v ", value, p.FmtStack())
}

func (p *Macros) PushRegs(params ...string) string {
	if p.Amd64 {
		var b strings.Builder
		for i := 0; i < len(params); i = i + 2 {
			b.WriteRune('\n')
			b.WriteString(p.Push(params[i], params[i+1]))
		}
		return b.String()
	} else {
		var pushadOrder = [...]string{"eax", "ecx", "edx", "ebx", "esp", "ebp", "esi", "edi"}
		for _, name := range pushadOrder {
			for j := 0; j < len(params); j = j + 2 {
				if params[j] == name {
					name = params[j+1]
				}
			}
			p.Stacklocs = append(p.Stacklocs, name)
		}
		return fmt.Sprintf("\npushad  ; Stack: %v", p.FmtStack())
	}
}

func (p *Macros) PopRegs(params ...string) string {
	if p.Amd64 {
		var b strings.Builder
		for i := len(params) - 1; i >= 0; i-- {
			b.WriteRune('\n')
			b.WriteString(p.Pop(params[i]))
		}
		return b.String()
	} else {
		var regs = [...]string{"eax", "ecx", "edx", "ebx", "esp", "ebp", "esi", "edi"}
		var b strings.Builder
		for i, name := range p.Stacklocs[len(p.Stacklocs)-8:] {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(regs[i])
			if regs[i] != name {
				b.WriteString(" = ")
				b.WriteString(name)
			}
		}
		p.Stacklocs = p.Stacklocs[:len(p.Stacklocs)-8]
		return fmt.Sprintf("\npopad  ; Popped: %v. Stack: %v", b.String(), p.FmtStack())
	}
}

func (p *Macros) Pop(register string) string {
	last := p.Stacklocs[len(p.Stacklocs)-1]
	p.Stacklocs = p.Stacklocs[:len(p.Stacklocs)-1]
	return fmt.Sprintf("pop     %v      ; %v = %v, Stack: %v ", register, register, last, p.FmtStack())
}

func (p *Macros) SaveFPUState() string {
	i := 0
	for ; i < 108; i += p.PTRSIZE() {
		p.Stacklocs = append(p.Stacklocs, fmt.Sprintf("F%v", i))
	}
	return fmt.Sprintf("sub     %[1]v, %[2]v\nfsave   [%[1]v]", p.SP(), i)
}

func (p *Macros) LoadFPUState() string {
	i := 0
	for ; i < 108; i += p.PTRSIZE() {
		p.Stacklocs = p.Stacklocs[:len(p.Stacklocs)-1]
	}
	return fmt.Sprintf("frstor   [%[1]v]\nadd     %[1]v, %[2]v", p.SP(), i)
}

func (p *Macros) Stack(name string) (string, error) {
	for i, k := range p.Stacklocs {
		if k == name {
			pos := len(p.Stacklocs) - i - 1
			if p.Amd64 {
				pos = pos * 8
			} else {
				pos = pos * 4
			}
			if pos != 0 {
				return fmt.Sprintf("%v + %v", p.SP(), pos), nil
			}
			return p.SP(), nil
		}
	}
	return "", fmt.Errorf("unknown symbol %v", name)
}

func (p *Macros) FmtStack() string {
	var b strings.Builder
	last := len(p.Stacklocs) - 1
	for i := range p.Stacklocs {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(p.Stacklocs[last-i])
	}
	return b.String()
}

func (p *Macros) ExportFunc(name string, params ...string) string {
	if !p.Amd64 {
		reverseParams := make([]string, len(params))
		for i, param := range params {
			reverseParams[len(params)-1-i] = param
		}
		p.Stacklocs = append(reverseParams, "retaddr_"+name) // in 32-bit, we use stdcall and parameters are in the stack
		if p.OS == "windows" {
			return fmt.Sprintf("%[1]v\nglobal _%[2]v@%[3]v\n_%[2]v@%[3]v:", p.SectText(name), name, len(params)*4)
		}
	}
	if p.OS == "darwin" {
		return fmt.Sprintf("%[1]v\nglobal _%[2]v\n_%[2]v:", p.SectText(name), name)
	}
	return fmt.Sprintf("%[1]v\nglobal %[2]v\n%[2]v:", p.SectText(name), name)
}

func (p *Macros) Input(unit string, port string) (string, error) {
	i := p.InputNumber(unit, port)
	if i != 0 {
		return fmt.Sprintf("%v + %v", p.INP(), i*4), nil
	}
	return p.INP(), nil
}

func (p *Macros) Modulation(unit string, port string) (string, error) {
	i := p.InputNumber(unit, port)
	return fmt.Sprintf("%v + %v", p.WRK(), i*4+32), nil
}

func (p *Macros) Prepare(value string, regs ...string) (string, error) {
	if p.Amd64 {
		if len(regs) > 1 {
			return "", fmt.Errorf("macro Prepare cannot accept more than one register parameter")
		} else if len(regs) > 0 {
			return fmt.Sprintf("\nmov     r9, qword %v\nlea		r9, [r9 + %v]", value, regs[0]), nil
		}
		return fmt.Sprintf("\nmov     r9, qword %v", value), nil
	}
	return "", nil
}

func (p *Macros) Use(value string, regs ...string) (string, error) {
	if p.Amd64 {
		return "r9", nil
	}
	if len(regs) > 1 {
		return "", fmt.Errorf("macro Use cannot accept more than one register parameter")
	} else if len(regs) > 0 {
		return value + " + " + regs[0], nil
	}
	return value, nil
}

type PlayerMacros struct {
	Song              *go4k.Song
	VoiceTrackBitmask int
	MaxSamples        int
	Macros
	EncodedPatch
}

func NewPlayerMacros(c Compiler, f FeatureSet, s *go4k.Song, e *EncodedPatch, maxSamples int) *PlayerMacros {
	if maxSamples == 0 {
		maxSamples = s.SamplesPerRow() * s.TotalRows()
	}
	macros := *NewMacros(c, f)
	macros.Output16Bit = s.Output16Bit // TODO: should we actually store output16bit in Songs or not?
	p := PlayerMacros{Song: s, Macros: macros, MaxSamples: maxSamples, EncodedPatch: *e}
	trackVoiceNumber := 0
	for _, t := range s.Tracks {
		for b := 0; b < t.NumVoices-1; b++ {
			p.VoiceTrackBitmask += 1 << trackVoiceNumber
			trackVoiceNumber++
		}
		trackVoiceNumber++ // set all bits except last one
	}
	return &p
}

func (p *PlayerMacros) NumDelayLines() string {
	total := 0
	for _, instr := range p.Song.Patch.Instruments {
		for _, unit := range instr.Units {
			if unit.Type == "delay" {
				total += unit.Parameters["count"] * (1 + unit.Parameters["stereo"])
			}
		}
	}
	return fmt.Sprintf("%v", total)
}
