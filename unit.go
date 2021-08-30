package sointu

// Unit is e.g. a filter, oscillator, envelope and its parameters
type Unit struct {
	// Type is the type of the unit, e.g. "add","oscillator" or "envelope".
	// Always in lowercase. "" type should be ignored, no invalid types should
	// be used.
	Type string `yaml:",omitempty"`

	// ID should be a unique ID for this unit, used by SEND units to target
	// specific units. ID = 0 means that no ID has been given to a unit and thus
	// cannot be targeted by SENDs. When possible, units that are not targeted
	// by any SENDs should be cleaned from having IDs, e.g. to keep the exported
	// data clean.
	ID int `yaml:",omitempty"`

	// Parameters is a map[string]int of parameters of a unit. For example, for
	// an oscillator, unit.Type == "oscillator" and unit.Parameters["attack"]
	// could be 64. Most parameters are either limites to 0 and 1 (e.g. stereo
	// parameters) or between 0 and 128, inclusive.
	Parameters map[string]int `yaml:",flow"`

	// VarArgs is a list containing the variable number arguments that some
	// units require, most notably the DELAY units. For example, for a DELAY
	// unit, VarArgs is the delaytimes, in samples, of the different delaylines
	// in the unit.
	VarArgs []int `yaml:",flow,omitempty"`
}

// When unit.Type = "oscillator", its unit.Parameter["Type"] tells the type of
// the oscillator. There is five different oscillator types, so these consts
// just enumerate them.
const (
	Sine   = iota
	Trisaw = iota
	Pulse  = iota
	Gate   = iota
	Sample = iota
)

// Copy makes a deep copy of a unit.
func (u *Unit) Copy() Unit {
	parameters := make(map[string]int)
	for k, v := range u.Parameters {
		parameters[k] = v
	}
	varArgs := make([]int, len(u.VarArgs))
	copy(varArgs, u.VarArgs)
	return Unit{Type: u.Type, Parameters: parameters, VarArgs: varArgs, ID: u.ID}
}

// StackChange returns how this unit will affect the signal stack. "pop" and
// "addp" and such will consume the topmost signal, and thus return -1 (or -2,
// if the unit is a stereo unit). On the other hand, "oscillator" and "envelope"
// will produce a signal, and thus return 1 (or 2, if the unit is a stereo
// unit). Effects that just change the topmost signal and will not change the
// number of signals on the stack and thus return 0.
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

// StackNeed returns the number of signals that should be on the stack before
// this unit is executed. Used to prevent stack underflow. Units producing
// signals do not care what is on the stack before and will return 0.
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
