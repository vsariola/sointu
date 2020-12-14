package go4k

import (
	"bytes"
	"fmt"
	"math"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
)

type OplistEntry struct {
	Type      string
	NumParams int
}

type Macros struct {
	Opcodes          []OplistEntry
	Polyphony        bool
	MultivoiceTracks bool
	PolyphonyBitmask int
	Stacklocs        []string
	Output16Bit      bool
	Clip             bool
	Amd64            bool
	OS               string
	DisableSections  bool
	Sine             int // TODO: how can we elegantly access global constants in template, without wrapping each one by one
	Trisaw           int
	Pulse            int
	Gate             int
	Sample           int
	usesFloatConst   map[float32]bool
	usesIntConst     map[int]bool
	floatConsts      []float32
	intConsts        []int
	calls            map[string]bool
	stereo           map[string]bool
	mono             map[string]bool
	ops              map[string]bool
	stackframes      map[string][]string
	unitInputMap     map[string](map[string]int)
}

type PlayerMacros struct {
	Song              *Song
	VoiceTrackBitmask int
	JumpTable         []string
	Code              []byte
	Values            []byte
	Macros
}

func NewPlayerMacros(song *Song, targetArch string, targetOS string) *PlayerMacros {
	unitInputMap := map[string](map[string]int){}
	for k, v := range UnitTypes {
		inputMap := map[string]int{}
		inputCount := 0
		for _, t := range v {
			if t.CanModulate {
				inputMap[t.Name] = inputCount
				inputCount++
			}
		}
		unitInputMap[k] = inputMap
	}
	jumpTable, code, values := song.Patch.Encode()
	amd64 := targetArch == "amd64"
	p := &PlayerMacros{
		Song:      song,
		JumpTable: jumpTable,
		Code:      code,
		Values:    values,
		Macros: Macros{
			mono:           map[string]bool{},
			stereo:         map[string]bool{},
			calls:          map[string]bool{},
			ops:            map[string]bool{},
			usesFloatConst: map[float32]bool{},
			usesIntConst:   map[int]bool{},
			stackframes:    map[string][]string{},
			unitInputMap:   unitInputMap,
			Amd64:          amd64,
			OS:             targetOS,
			Sine:           Sine,
			Trisaw:         Trisaw,
			Pulse:          Pulse,
			Gate:           Gate,
			Sample:         Sample,
		}}
	for _, track := range song.Tracks {
		if track.NumVoices > 1 {
			p.MultivoiceTracks = true
		}
	}
	trackVoiceNumber := 0
	for _, t := range song.Tracks {
		for b := 0; b < t.NumVoices-1; b++ {
			p.VoiceTrackBitmask += 1 << trackVoiceNumber
			trackVoiceNumber++
		}
		trackVoiceNumber++ // set all bits except last one
	}
	totalVoices := 0
	for _, instr := range song.Patch.Instruments {
		if instr.NumVoices > 1 {
			p.Polyphony = true
		}
		for _, unit := range instr.Units {
			if !p.ops[unit.Type] {
				p.ops[unit.Type] = true
				numParams := 0
				for _, v := range UnitTypes[unit.Type] {
					if v.CanSet && v.CanModulate {
						numParams++
					}
				}
				p.Opcodes = append(p.Opcodes, OplistEntry{
					Type:      unit.Type,
					NumParams: numParams,
				})
			}
			if unit.Parameters["stereo"] == 1 {
				p.stereo[unit.Type] = true
			} else {
				p.mono[unit.Type] = true
			}
		}
		totalVoices += instr.NumVoices
		for k := 0; k < instr.NumVoices-1; k++ {
			p.PolyphonyBitmask = (p.PolyphonyBitmask << 1) + 1
		}
		p.PolyphonyBitmask <<= 1
	}
	p.Output16Bit = song.Output16Bit
	return p
}

func (p *Macros) Opcode(t string) bool {
	return p.ops[t]
}

func (p *Macros) Stereo(t string) bool {
	return p.stereo[t]
}

func (p *Macros) Mono(t string) bool {
	return p.mono[t]
}

func (p *Macros) StereoAndMono(t string) bool {
	return p.stereo[t] && p.mono[t]
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
		p.Stacklocs = append(params, "retaddr_"+name) // in 32-bit, we use stdcall and parameters are in the stack
		if p.OS == "windows" {
			return fmt.Sprintf("%[1]v\nglobal _%[2]v@%[3]v\n_%[2]v@%[3]v:", p.SectText(name), name, len(params)*4)
		}
	}
	if p.OS == "darwin" {
		return fmt.Sprintf("%[1]v\nglobal _%[2]v\n_%[2]v:", p.SectText(name), name)
	}
	return fmt.Sprintf("%[1]v\nglobal %[2]v\n%[2]v:", p.SectText(name), name)
}

func (p *Macros) Count(count int) []int {
	s := make([]int, count)
	for i := range s {
		s[i] = i
	}
	return s
}

func (p *Macros) Sub(a int, b int) int {
	return a - b
}

func (p *Macros) Input(unit string, port string) (string, error) {
	umap, ok := p.unitInputMap[unit]
	if !ok {
		return "", fmt.Errorf(`trying to find input for unknown unit "%v"`, unit)
	}
	i, ok := umap[port]
	if !ok {
		return "", fmt.Errorf(`trying to find input for unknown input "%v" for unit "%v"`, port, unit)
	}
	if i != 0 {
		return fmt.Sprintf("%v + %v", p.INP(), i*4), nil
	}
	return p.INP(), nil
}

func (p *Macros) InputNumber(unit string, port string) (string, error) {
	umap, ok := p.unitInputMap[unit]
	if !ok {
		return "", fmt.Errorf(`trying to find InputNumber for unknown unit "%v"`, unit)
	}
	i, ok := umap[port]
	if !ok {
		return "", fmt.Errorf(`trying to find InputNumber for unknown input "%v" for unit "%v"`, port, unit)
	}
	return fmt.Sprintf("%v", i), nil
}

func (p *Macros) Modulation(unit string, port string) (string, error) {
	umap, ok := p.unitInputMap[unit]
	if !ok {
		return "", fmt.Errorf(`trying to find input for unknown unit "%v"`, unit)
	}
	i, ok := umap[port]
	if !ok {
		return "", fmt.Errorf(`trying to find input for unknown input "%v" for unit "%v"`, port, unit)
	}
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

func (p *PlayerMacros) UsesDelayModulation() (bool, error) {
	for i, instrument := range p.Song.Patch.Instruments {
		for j, unit := range instrument.Units {
			if unit.Type == "send" {
				targetInstrument := i
				if unit.Parameters["voice"] > 0 {
					v, err := p.Song.Patch.InstrumentForVoice(unit.Parameters["voice"] - 1)
					if err != nil {
						return false, fmt.Errorf("INSTRUMENT #%v / SEND #%v targets voice %v, which does not exist", i, j, unit.Parameters["voice"])
					}
					targetInstrument = v
				}
				if unit.Parameters["unit"] < 0 || unit.Parameters["unit"] >= len(p.Song.Patch.Instruments[targetInstrument].Units) {
					return false, fmt.Errorf("INSTRUMENT #%v / SEND #%v target unit %v out of range", i, j, unit.Parameters["unit"])
				}
				if p.Song.Patch.Instruments[targetInstrument].Units[unit.Parameters["unit"]].Type == "delay" && unit.Parameters["port"] == 4 {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (p *PlayerMacros) HasParamValue(unitType string, paramName string, value int) bool {
	for _, instr := range p.Song.Patch.Instruments {
		for _, unit := range instr.Units {
			if unit.Type == unitType {
				if unit.Parameters[paramName] == value {
					return true
				}
			}
		}
	}
	return false
}

func (p *PlayerMacros) HasParamValueOtherThan(unitType string, paramName string, value int) bool {
	for _, instr := range p.Song.Patch.Instruments {
		for _, unit := range instr.Units {
			if unit.Type == unitType {
				if unit.Parameters[paramName] != value {
					return true
				}
			}
		}
	}
	return false
}

func Compile(song *Song, targetArch string, targetOs string) (string, error) {
	_, myname, _, _ := runtime.Caller(0)
	templateDir := filepath.Join(path.Dir(myname), "..", "templates", "*.asm")
	tmpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).ParseGlob(templateDir)
	if err != nil {
		return "", fmt.Errorf(`could not create template based on dir "%v": %v`, templateDir, err)
	}
	b := bytes.NewBufferString("")
	err = tmpl.ExecuteTemplate(b, "player.asm", NewPlayerMacros(song, targetArch, targetOs))
	if err != nil {
		return "", fmt.Errorf(`could not execute template "player.asm": %v`, err)
	}
	return b.String(), nil
}
