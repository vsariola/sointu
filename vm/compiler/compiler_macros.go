package compiler

import (
	"github.com/vsariola/sointu"
)

type CompilerMacros struct {
	Clip    bool
	Library bool

	Sine   int // TODO: how can we elegantly access global constants in template, without wrapping each one by one
	Trisaw int
	Pulse  int
	Gate   int
	Sample int
	Compiler
}

func NewCompilerMacros(c Compiler) *CompilerMacros {
	return &CompilerMacros{
		Sine:     sointu.Sine,
		Trisaw:   sointu.Trisaw,
		Pulse:    sointu.Pulse,
		Gate:     sointu.Gate,
		Sample:   sointu.Sample,
		Compiler: c,
	}
}
