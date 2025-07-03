package tracker

import (
	"fmt"

	"github.com/vsariola/sointu"
)

type (
	SignalRail struct {
		signals [][]Signal
		scratch []signalScratch

		error SignalError
	}

	SignalError struct {
		InstrIndex, UnitIndex int
		Err                   error
	}

	Signal struct {
		PassThrough int
		Send        bool
		StackUse    sointu.StackUse
	}

	signalScratch struct {
		instr, unit int
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

func (s *SignalRailType) Error() SignalError {
	i := s.d.InstrIndex
	if i < 0 || i >= len(s.derived.rail.signals) {
		return SignalError{}
	}
	if i == s.derived.rail.error.InstrIndex {
		return s.derived.rail.error
	}
	return SignalError{}
}

func (s *SignalRailType) MaxWidth() int {
	i := s.d.InstrIndex
	if i < 0 || i >= len(s.derived.rail.signals) {
		return 0
	}
	ret := 0
	for _, signal := range s.derived.rail.signals[i] {
		ret = max(ret, signal.PassThrough+max(len(signal.StackUse.Inputs), signal.StackUse.NumOutputs))
	}
	return ret
}

func (st *SignalRailType) update() {
	s := &st.derived.rail
	patch := st.d.Song.Patch
	s.scratch = s.scratch[:0]
	s.error = SignalError{}
	for i, instr := range patch {
		for len(s.signals) <= i {
			s.signals = append(s.signals, make([]Signal, len(instr.Units)))
		}
		start := len(s.scratch)
		for u, unit := range instr.Units {
			for len(s.signals[i]) <= u {
				s.signals[i] = append(s.signals[i], Signal{})
			}
			stackUse := unit.StackUse()
			numInputs := len(stackUse.Inputs)
			if len(s.scratch) < numInputs {
				if s.error.Err == nil {
					s.error.Err = fmt.Errorf("%s unit in instrument %d / %s needs %d inputs, but got only %d", unit.Type, i, instr.Name, numInputs, len(s.scratch))
					s.error.InstrIndex = i
					s.error.UnitIndex = u
				}
				s.scratch = s.scratch[:0]
			} else {
				s.scratch = s.scratch[:len(s.scratch)-numInputs]
			}
			s.signals[i][u] = Signal{
				PassThrough: len(s.scratch),
				StackUse:    stackUse,
				Send:        unit.Type == "send",
			}
			for _ = range stackUse.NumOutputs {
				s.scratch = append(s.scratch, signalScratch{instr: i, unit: u})
			}
		}
		diff := len(s.scratch) - start
		if instr.NumVoices > 1 && diff != 0 {
			if diff < 0 {
				morepop := (instr.NumVoices - 1) * diff
				if morepop > len(s.scratch) {
					if s.error.Err == nil {
						s.error.Err = fmt.Errorf("each voice of instrument %d / %s consumes %d signals, but there was not enough signals available", i, instr.Name, -diff)
						s.error.InstrIndex = i
						s.error.UnitIndex = -1
					}
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
	if len(s.scratch) > 0 && s.error.Err == nil {
		s.error.Err = fmt.Errorf("instrument %d / %s unit %d / %s leave a signal on stack ", s.scratch[0].instr, patch[s.scratch[0].instr].Name, s.scratch[0].unit, patch[s.scratch[0].instr].Units[s.scratch[0].unit].Type)
		s.error.InstrIndex = s.scratch[0].instr
		s.error.UnitIndex = s.scratch[0].unit
	}
	if s.error.Err != nil {
		(*Model)(st).Alerts().AddNamed("SignalError", s.error.Error(), Error)
	}
}

func (e *SignalError) Error() string {
	return e.Err.Error()
}
