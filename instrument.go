package sointu

// Instrument includes a list of units consisting of the instrument, and the number of polyphonic voices for this instrument
type Instrument struct {
	Name      string `yaml:",omitempty"`
	NumVoices int
	Units     []Unit
}

func (instr *Instrument) Copy() Instrument {
	units := make([]Unit, len(instr.Units))
	for i, u := range instr.Units {
		units[i] = u.Copy()
	}
	return Instrument{Name: instr.Name, NumVoices: instr.NumVoices, Units: units}
}
