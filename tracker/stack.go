package tracker

import (
	"fmt"

	"github.com/vsariola/sointu"
)

type (
	SignalRail struct {
		signals [][]Signal
		scratch []signalScratch
	}

	signalScratch struct {
		instr, unit int
	}

	Signal struct {
		PassThrough int
		StackUse    sointu.StackUse
	}

	SignalRailType Model
)

func (m *Model) SignalRail() *SignalRailType {
	return (*SignalRailType)(m)
}

func (s *SignalRailType) Item(u int) Signal {
	i := s.d.InstrIndex
	if i < 0 || u < 0 || i >= len(s.derived.rail.signals) || u >= len(s.derived.rail.signals[i]) {
		return Signal{}
	}
	return s.derived.rail.signals[i][u]
}

func (s *SignalRail) update(patch sointu.Patch) (err error) {
	s.scratch = s.scratch[:0]
	for i, instr := range patch {
		for len(s.signals) <= i {
			s.signals = append(s.signals, make([]Signal, len(instr.Units)))
		}
		start := len(s.scratch)
		for u, unit := range instr.Units {
			for len(s.signals[i]) <= i {
				s.signals[i] = append(s.signals[i], Signal{})
			}
			stackUse := unit.StackUse()
			numInputs := len(stackUse.Inputs)
			if len(s.scratch) < numInputs && err != nil {
				err = fmt.Errorf("%s unit in instrument %d / %s needs %d inputs, but got only %d", unit.Type, i, instr.Name, numInputs, len(s.scratch))
				s.scratch = s.scratch[:0]
			} else {
				s.scratch = s.scratch[:len(s.scratch)-numInputs]
			}
			s.signals[i][u] = Signal{
				PassThrough: len(s.scratch),
				StackUse:    stackUse,
			}
			for _ = range stackUse.NumOutputs {
				s.scratch = append(s.scratch, signalScratch{instr: i, unit: u})
			}
		}
		diff := len(s.scratch) - start
		if instr.NumVoices > 1 && diff != 0 {
			if diff < 0 {
				morepop := (instr.NumVoices - 1) * diff
				if morepop > len(s.scratch) && err != nil {
					err = fmt.Errorf("each voice of instrument %d / %s consumes %d signals, but there was not enough signals available", i, instr.Name, -diff)
					s.scratch = s.scratch[:0]
				} else {
					s.scratch = s.scratch[:len(s.scratch)-morepop]
				}
			} else {
				for range (instr.NumVoices - 1) * diff {
					s.scratch = append(s.scratch, s.scratch[len(s.scratch)-diff])
				}
			}
		}
	}
	return err
}
