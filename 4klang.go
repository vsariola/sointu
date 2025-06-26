package sointu

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Read4klangPatch reads a 4klang patch (a file usually with .4kp extension)
// from r and returns a Patch, making best attempt to convert 4klang file to a
// sointu Patch. It returns an error if the file is malformed or if the 4kp file
// version is not supported.
func Read4klangPatch(r io.Reader) (patch Patch, err error) {
	var versionTag uint32
	var version int
	var polyphonyUint32 uint32
	var instrumentNames [_4KLANG_MAX_INSTRS]string
	patch = make(Patch, 0)
	if err := binary.Read(r, binary.LittleEndian, &versionTag); err != nil {
		return nil, fmt.Errorf("binary.Read: %w", err)
	}
	var ok bool
	if version, ok = _4klangVersionTags[versionTag]; !ok {
		return nil, fmt.Errorf("unknown 4klang version tag: %d", versionTag)
	}
	if err := binary.Read(r, binary.LittleEndian, &polyphonyUint32); err != nil {
		return nil, fmt.Errorf("binary.Read: %w", err)
	}
	for i := range instrumentNames {
		instrumentNames[i], err = read4klangName(r)
		if err != nil {
			return nil, fmt.Errorf("read4klangName: %w", err)
		}
	}
	m := make(_4klangTargetMap)
	id := 1
	for instrIndex := 0; instrIndex < _4KLANG_MAX_INSTRS; instrIndex++ {
		var units []Unit
		if units, err = read4klangUnits(r, version, instrIndex, m, &id); err != nil {
			return nil, fmt.Errorf("read4klangUnits: %w", err)
		}
		if len(units) > 0 {
			patch = append(patch, Instrument{Name: instrumentNames[instrIndex], NumVoices: 1, Units: units})
		}
	}
	var units []Unit
	if units, err = read4klangUnits(r, version, _4KLANG_MAX_INSTRS, m, &id); err != nil {
		return nil, fmt.Errorf("read4klangUnits: %w", err)
	}
	if len(units) > 0 {
		patch = append(patch, Instrument{Name: "Global", NumVoices: 1, Units: units})
	}
	for i, instr := range patch {
		fix4klangTargets(i, instr, m)
	}
	return
}

// Read4klangInstrument reads a 4klang instrument (a file usually with .4ki
// extension) from r and returns an Instrument, making best attempt to convert
// 4ki file to a sointu Instrument. It returns an error if the file is malformed
// or if the 4ki file version is not supported.
func Read4klangInstrument(r io.Reader) (instr Instrument, err error) {
	var versionTag uint32
	var version int
	var name string
	if err := binary.Read(r, binary.LittleEndian, &versionTag); err != nil {
		return Instrument{}, fmt.Errorf("binary.Read: %w", err)
	}
	var ok bool
	if version, ok = _4klangVersionTags[versionTag]; !ok {
		return Instrument{}, fmt.Errorf("unknown 4klang version tag: %d", versionTag)
	}
	if name, err = read4klangName(r); err != nil {
		return Instrument{}, fmt.Errorf("read4klangName: %w", err)
	}
	var units []Unit
	id := 1
	m := make(_4klangTargetMap)
	if units, err = read4klangUnits(r, version, 0, m, &id); err != nil {
		return Instrument{}, fmt.Errorf("read4klangUnits: %w", err)
	}
	ret := Instrument{Name: name, NumVoices: 1, Units: units}
	fix4klangTargets(0, ret, m)
	return ret, nil
}

type (
	_4klangStackUnit struct {
		stack, unit int
	}

	_4klangTargetMap map[_4klangStackUnit]int

	_4klangPorts struct {
		UnitType string
		PortName [8]string
	}
)

const (
	_4KLANG_MAX_INSTRS   = 16
	_4KLANG_MAX_UNITS    = 64
	_4KLANG_MAX_SLOTS    = 16
	_4KLANG_MAX_NAME_LEN = 64
)

var (
	_4klangVersionTags map[uint32]int = map[uint32]int{
		0x31316b34: 11, // 4k11
		0x32316b34: 12, // 4k12
		0x33316b34: 13, // 4k13
		0x34316b34: 14, // 4k14
	}

	_4klangDelays []int = []int{ // these are the numerators, if denominator is 48, fraction of beat time
		4,   // 0 = 4.0f * (1.0f/32.0f) * (2.0f/3.0f)
		6,   // 1 = 4.0f * (1.0f/32.0f),
		9,   // 2 = 4.0f * (1.0f/32.0f) * (3.0f/2.0f),
		8,   // 3 = 4.0f * (1.0f/16.0f) * (2.0f/3.0f),
		12,  // 4 = 4.0f * (1.0f/16.0f),
		18,  // 5 = 4.0f * (1.0f/16.0f) * (3.0f/2.0f),
		16,  // 6 = 4.0f * (1.0f/8.0f) * (2.0f/3.0f),
		24,  // 7 = 4.0f * (1.0f/8.0f),
		36,  // 8 = 4.0f * (1.0f/8.0f) * (3.0f/2.0f),
		32,  // 9 = 4.0f * (1.0f/4.0f) * (2.0f/3.0f),
		48,  // 10 = 4.0f * (1.0f/4.0f),
		72,  // 11 = 4.0f * (1.0f/4.0f) * (3.0f/2.0f),
		64,  // 12 = 4.0f * (1.0f/2.0f) * (2.0f/3.0f),
		96,  // 13 = 4.0f * (1.0f/2.0f),
		144, // 14 = 4.0f * (1.0f/2.0f) * (3.0f/2.0f),
		128, // 15 = 4.0f * (1.0f) * (2.0f/3.0f),
		192, // 16 = 4.0f * (1.0f),
		288, // 17 = 4.0f * (1.0f) * (3.0f/2.0f),
		256, // 18 = 4.0f * (2.0f) * (2.0f/3.0f),
		384, // 19 = 4.0f * (2.0f),
		576, // 20 = 4.0f * (2.0f) * (3.0f/2.0f),
		72,  // 21 = 4.0f * (3.0f/8.0f),
		120, // 22 = 4.0f * (5.0f/8.0f),
		168, // 23 = 4.0f * (7.0f/8.0f),
		216, // 24 = 4.0f * (9.0f/8.0f),
		264, // 25 = 4.0f * (11.0f/8.0f),
		312, // 26 = 4.0f * (13.0f/8.0f),
		360, // 27 = 4.0f * (15.0f/8.0f),
		144, // 28 = 4.0f * (3.0f/4.0f),
		240, // 29 = 4.0f * (5.0f/4.0f),
		336, // 30 = 4.0f * (7.0f/4.0f),
		288, // 31 = 4.0f * (3.0f/2.0f),
		288, // 32 = 4.0f * (3.0f/2.0f),
	}

	_4klangUnitPorts []_4klangPorts = []_4klangPorts{
		{"", [8]string{"", "", "", "", "", "", "", ""}},
		{"envelope", [8]string{"", "", "gain", "attack", "decay", "", "release", ""}},
		{"oscillator", [8]string{"", "transpose", "detune", "", "phase", "color", "shape", "gain"}},
		{"filter", [8]string{"", "", "", "", "frequency", "resonance", "", ""}},
		{"envelope", [8]string{"", "", "drive", "frequency", "", "", "", ""}},
		{"delay", [8]string{"pregain", "feedback", "dry", "damp", "", "", "", ""}},
		{"", [8]string{"", "", "", "", "", "", "", ""}},
		{"", [8]string{"", "", "", "", "", "", "", ""}},
		{"pan", [8]string{"panning", "", "", "", "", "", "", ""}},
		{"outaux", [8]string{"auxgain", "outgain", "", "", "", "", "", ""}},
		{"", [8]string{"", "", "", "", "", "", "", ""}},
		{"load", [8]string{"value", "", "", "", "", "", "", ""}},
	}
)

func read4klangName(r io.Reader) (string, error) {
	var name [_4KLANG_MAX_NAME_LEN]byte
	if err := binary.Read(r, binary.LittleEndian, &name); err != nil {
		return "", fmt.Errorf("binary.Read: %w", err)
	}
	n := bytes.IndexByte(name[:], 0)
	if n == -1 {
		n = _4KLANG_MAX_NAME_LEN
	}
	return string(name[:n]), nil
}

func read4klangUnits(r io.Reader, version, instrIndex int, m _4klangTargetMap, id *int) (units []Unit, err error) {
	numUnits := _4KLANG_MAX_UNITS
	if version <= 13 {
		numUnits = 32
	}
	units = make([]Unit, 0, numUnits)
	for unitIndex := 0; unitIndex < numUnits; unitIndex++ {
		var u []Unit
		if u, err = read4klangUnit(r, version); err != nil {
			return nil, fmt.Errorf("read4klangUnit: %w", err)
		}
		if u == nil {
			continue
		}
		m[_4klangStackUnit{instrIndex, unitIndex}] = *id
		for i := range u {
			u[i].ID = *id
			*id++
		}
		units = append(units, u...)
	}
	return
}

func read4klangUnit(r io.Reader, version int) ([]Unit, error) {
	var unitType byte
	if err := binary.Read(r, binary.LittleEndian, &unitType); err != nil {
		return nil, fmt.Errorf("binary.Read: %w", err)
	}
	var vals [15]byte
	if err := binary.Read(r, binary.LittleEndian, &vals); err != nil {
		return nil, fmt.Errorf("binary.Read: %w", err)
	}
	if version <= 13 {
		// versions <= 13 had 16 unused slots for each unit
		if written, err := io.CopyN(io.Discard, r, 16); err != nil || written < 16 {
			return nil, fmt.Errorf("io.CopyN: %w", err)
		}
	}
	switch unitType {
	case 1:
		return read4klangENV(vals, version), nil
	case 2:
		return read4klangVCO(vals, version), nil
	case 3:
		return read4klangVCF(vals, version), nil
	case 4:
		return read4klangDST(vals, version), nil
	case 5:
		return read4klangDLL(vals, version), nil
	case 6:
		return read4klangFOP(vals, version), nil
	case 7:
		return read4klangFST(vals, version), nil
	case 8:
		return read4klangPAN(vals, version), nil
	case 9:
		return read4klangOUT(vals, version), nil
	case 10:
		return read4klangACC(vals, version), nil
	case 11:
		return read4klangFLD(vals, version), nil
	default:
		return nil, nil
	}
}

func read4klangENV(vals [15]byte, _ int) []Unit {
	return []Unit{{
		Type: "envelope",
		Parameters: map[string]int{
			"stereo":  0,
			"attack":  int(vals[0]),
			"decay":   int(vals[1]),
			"sustain": int(vals[2]),
			"release": int(vals[3]),
			"gain":    int(vals[4]),
		},
	}}
}

func read4klangVCO(vals [15]byte, version int) []Unit {
	v := vals[:8]
	var transpose, detune, phase, color, gate, shape, gain, flags, stereo, typ, lfo int
	transpose, v = int(v[0]), v[1:]
	detune, v = int(v[0]), v[1:]
	phase, v = int(v[0]), v[1:]
	if version <= 11 {
		gate = 0x55
	} else {
		gate, v = int(v[0]), v[1:]
	}
	color, v = int(v[0]), v[1:]
	shape, v = int(v[0]), v[1:]
	gain, v = int(v[0]), v[1:]
	flags, _ = int(v[0]), v[1:]
	if flags&0x10 == 0x10 {
		lfo = 1
	}
	if flags&0x40 == 0x40 {
		stereo = 1
	}
	switch {
	case flags&0x01 == 0x01: // Sine
		typ = Sine
		if version <= 13 {
			color = 128
		}
	case flags&0x02 == 0x02: // Trisaw
		typ = Trisaw
	case flags&0x04 == 0x04: // Pulse
		typ = Pulse
	case flags&0x08 == 0x08: // Noise is handled differently in sointu
		return []Unit{{
			Type: "noise",
			Parameters: map[string]int{
				"stereo": stereo,
				"shape":  shape,
				"gain":   gain,
			},
		}}
	case flags&0x20 == 0x20: // Gate
		color = gate
	}
	return []Unit{{
		Type: "oscillator",
		Parameters: map[string]int{
			"stereo":    stereo,
			"transpose": transpose,
			"detune":    detune,
			"phase":     phase,
			"color":     color,
			"shape":     shape,
			"gain":      gain,
			"type":      typ,
			"lfo":       lfo,
		},
	}}
}

func read4klangVCF(vals [15]byte, _ int) []Unit {
	flags := vals[2]
	var stereo, lowpass, bandpass, highpass int
	if flags&0x01 == 0x01 {
		lowpass = 1
	}
	if flags&0x02 == 0x02 {
		highpass = 1
	}
	if flags&0x04 == 0x04 {
		bandpass = 1
	}
	if flags&0x08 == 0x08 {
		lowpass = 1
		highpass = -1
	}
	if flags&0x10 == 0x10 {
		stereo = 1
	}
	return []Unit{{
		Type: "filter",
		Parameters: map[string]int{
			"stereo":    stereo,
			"frequency": int(vals[0]),
			"resonance": int(vals[1]),
			"lowpass":   lowpass,
			"bandpass":  bandpass,
			"highpass":  highpass,
		}},
	}
}

func read4klangDST(vals [15]byte, _ int) []Unit {
	return []Unit{
		{Type: "distort", Parameters: map[string]int{"drive": int(vals[0]), "stereo": int(vals[2])}},
		{Type: "hold", Parameters: map[string]int{"holdfreq": int(vals[1]), "stereo": int(vals[2])}},
	}
}

func read4klangDLL(vals [15]byte, _ int) []Unit {
	var delaytimes []int
	var notetracking int
	if vals[11] > 0 {
		if vals[10] > 0 { // left reverb
			delaytimes = []int{1116, 1188, 1276, 1356, 1422, 1492, 1556, 1618}
		} else { // right reverb
			delaytimes = []int{1140, 1212, 1300, 1380, 1446, 1516, 1580, 1642}
		}
	} else {
		synctype := vals[9]
		switch synctype {
		case 0:
			delaytimes = []int{int(vals[8]) * 16}
		case 1: // relative to BPM
			notetracking = 2
			index := vals[8] >> 2
			delaytime := 48
			if int(index) < len(_4klangDelays) {
				delaytime = _4klangDelays[index]
			}
			delaytimes = []int{delaytime}
		case 2: // notetracking
			notetracking = 1
			delaytimes = []int{10787}
		}
	}
	return []Unit{{
		Type: "delay",
		Parameters: map[string]int{
			"stereo":       0,
			"pregain":      int(vals[0]),
			"dry":          int(vals[1]),
			"feedback":     int(vals[2]),
			"damp":         int(vals[3]),
			"notetracking": notetracking,
		},
		VarArgs: delaytimes,
	}}
}

func read4klangFOP(vals [15]byte, _ int) []Unit {
	var t string
	var stereo int
	switch vals[0] {
	case 1:
		t, stereo = "pop", 0
	case 2:
		t, stereo = "addp", 0
	case 3:
		t, stereo = "mulp", 0
	case 4:
		t, stereo = "push", 0
	case 5:
		t, stereo = "xch", 0
	case 6:
		t, stereo = "add", 0
	case 7:
		t, stereo = "mul", 0
	case 8:
		t, stereo = "addp", 1
	case 9:
		return []Unit{{Type: "loadnote", Parameters: map[string]int{"stereo": stereo}}, // 4klang loadnote gives 0..1, sointu gives -1..1
			{Type: "loadval", Parameters: map[string]int{"value": 128, "stereo": stereo}},
			{Type: "addp", Parameters: map[string]int{"stereo": stereo}},
			{Type: "gain", Parameters: map[string]int{"stereo": stereo, "gain": 64}}}
	default:
		t, stereo = "mulp", 1
	}
	return []Unit{{
		Type:       t,
		Parameters: map[string]int{"stereo": stereo},
	}}
}

func read4klangFST(vals [15]byte, _ int) []Unit {
	sendpop := 0
	if vals[1]&0x40 == 0x40 {
		sendpop = 1
	}
	return []Unit{{
		Type: "send",
		Parameters: map[string]int{
			"amount":     int(vals[0]),
			"sendpop":    sendpop,
			"dest_stack": int(vals[2]),
			"dest_unit":  int(vals[3]),
			"dest_slot":  int(vals[4]),
			"dest_id":    int(vals[5]),
		}}}
}

func fix4klangTargets(instrIndex int, instr Instrument, m _4klangTargetMap) {
	for _, u := range instr.Units {
		if u.Type == "send" {
			destStack := u.Parameters["dest_stack"]
			if destStack == 255 {
				destStack = instrIndex
			}
			fourKlangTarget := _4klangStackUnit{
				destStack,
				u.Parameters["dest_unit"]}
			u.Parameters["target"] = m[fourKlangTarget]
			if u.Parameters["dest_id"] < len(_4klangUnitPorts) && u.Parameters["dest_slot"] < 8 {
				if u.Parameters["dest_id"] == 4 && u.Parameters["dest_slot"] == 3 { // distortion is split into 2 units
					u.Parameters["target"]++
					u.Parameters["port"] = 0
				} else {
					modTarget := _4klangUnitPorts[u.Parameters["dest_id"]]
					for i, s := range Ports[modTarget.UnitType] {
						if s == modTarget.PortName[u.Parameters["dest_slot"]] {
							u.Parameters["port"] = i
							break
						}
					}
				}
			}
			delete(u.Parameters, "dest_stack")
			delete(u.Parameters, "dest_unit")
			delete(u.Parameters, "dest_slot")
			delete(u.Parameters, "dest_id")
		}
	}
}

func read4klangPAN(vals [15]byte, _ int) []Unit {
	return []Unit{{
		Type: "pan",
		Parameters: map[string]int{
			"stereo":  0,
			"panning": int(vals[0]),
		}}}
}

func read4klangOUT(vals [15]byte, _ int) []Unit {
	return []Unit{{
		Type: "outaux",
		Parameters: map[string]int{
			"stereo":  1,
			"outgain": int(vals[0]),
			"auxgain": int(vals[1])},
	}}
}

func read4klangACC(vals [15]byte, _ int) []Unit {
	c := 0
	if vals[0] != 0 {
		c = 2
	}
	return []Unit{{
		Type:       "in",
		Parameters: map[string]int{"stereo": 1, "channel": c},
	}}
}

func read4klangFLD(vals [15]byte, _ int) []Unit {
	return []Unit{{
		Type:       "loadval",
		Parameters: map[string]int{"stereo": 0, "value": int(vals[0])},
	}}
}
