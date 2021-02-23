package sointu

// Unit is e.g. a filter, oscillator, envelope and its parameters
type Unit struct {
	Type       string         `yaml:",omitempty"`
	ID         int            `yaml:",omitempty"`
	Parameters map[string]int `yaml:",flow"`
	VarArgs    []int          `yaml:",flow,omitempty"`
}

const (
	Sine   = iota
	Trisaw = iota
	Pulse  = iota
	Gate   = iota
	Sample = iota
)

func (u *Unit) Copy() Unit {
	parameters := make(map[string]int)
	for k, v := range u.Parameters {
		parameters[k] = v
	}
	varArgs := make([]int, len(u.VarArgs))
	copy(varArgs, u.VarArgs)
	return Unit{Type: u.Type, Parameters: parameters, VarArgs: varArgs, ID: u.ID}
}

func (u *Unit) StackChange() int {
	switch u.Type {
	case "addp", "mulp", "pop", "out", "outaux", "aux":
		return -1 - u.Parameters["stereo"]
	case "envelope", "oscillator", "push", "noise", "receive", "loadnote", "loadval", "in", "compressor":
		return 1 + u.Parameters["stereo"]
	case "pan":
		return 1 - u.Parameters["stereo"]
	case "speed":
		return -1
	case "send":
		return (-1 - u.Parameters["stereo"]) * u.Parameters["sendpop"]
	}
	return 0
}

func (u *Unit) StackNeed() int {
	switch u.Type {
	case "", "envelope", "oscillator", "noise", "receive", "loadnote", "loadval", "in":
		return 0
	case "mulp", "mul", "add", "addp", "xch":
		return 2 * (1 + u.Parameters["stereo"])
	case "speed":
		return 1
	}
	return 1 + u.Parameters["stereo"]
}
