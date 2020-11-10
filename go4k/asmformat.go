package go4k

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func DeserializeAsm(asmcode string) (*Song, error) {
	var bpm int
	scanner := bufio.NewScanner(strings.NewReader(asmcode))
	patterns := make([][]byte, 0)
	tracks := make([]Track, 0)
	var patch Patch
	var instr Instrument
	var delayTimes []int
	var sampleOffsets [][]int
	paramReg, err := regexp.Compile(`([a-zA-Z]\w*)\s*\(\s*([0-9]+)\s*\)`) // matches FOO(42), groups "FOO" and "42"
	if err != nil {
		return nil, err
	}
	parseParams := func(s string) (map[string]int, error) {
		matches := paramReg.FindAllStringSubmatch(s, 256)
		ret := map[string]int{}
		for _, match := range matches {
			val, err := strconv.Atoi(match[2])
			if err != nil {
				return nil, fmt.Errorf("Error converting %v to integer, which is unexpected as regexp matches only numbers", match[2])
			}
			ret[strings.ToLower(match[1])] = val
		}
		return ret, nil
	}
	typeReg, err := regexp.Compile(`TYPE\s*\(\s*(SINE|TRISAW|PULSE|GATE|SAMPLE)\s*\)`) // matches TYPE(TRISAW), groups "TRISAW"
	if err != nil {
		return nil, err
	}
	wordReg, err := regexp.Compile(`\s*([a-zA-Z_][a-zA-Z0-9_]*)([^;\n]*)`) // matches a word and "the rest", until newline or a comment
	if err != nil {
		return nil, err
	}
	numberReg, err := regexp.Compile(`-?[0-9]+|HLD`) // finds integer numbers, possibly with a sign in front. HLD is the magic value used by sointu, will be interpreted as 1
	if err != nil {
		return nil, err
	}
	parseNumbers := func(s string) ([]int, error) {
		matches := numberReg.FindAllString(s, 256)
		ret := []int{}
		for _, str := range matches {
			var i int
			var err error
			if str == "HLD" {
				i = 1
			} else {
				i, err = strconv.Atoi(str)
				if err != nil {
					return nil, err
				}
			}
			ret = append(ret, i)
		}
		return ret, nil
	}
	toBytes := func(ints []int) []byte {
		ret := []byte{}
		for _, v := range ints {
			ret = append(ret, byte(v))
		}
		return ret
	}
	inInstrument := false
	for scanner.Scan() {
		line := scanner.Text()
		macroMatch := wordReg.FindStringSubmatch(line)
		if macroMatch != nil {
			word, rest := macroMatch[1], macroMatch[2]
			switch word {
			case "define":
				defineMatch := wordReg.FindStringSubmatch(rest)
				if defineMatch != nil {
					defineName, defineRest := defineMatch[1], defineMatch[2]
					if defineName == "BPM" {
						ints, err := parseNumbers(defineRest)
						if err != nil {
							return nil, err
						}
						bpm = ints[0]
					}
				}
			case "PATTERN":
				ints, err := parseNumbers(rest)
				if err != nil {
					return nil, err
				}
				patterns = append(patterns, toBytes(ints))
			case "TRACK":
				ints, err := parseNumbers(rest)
				if err != nil {
					return nil, err
				}
				track := Track{ints[0], toBytes(ints[1:])}
				tracks = append(tracks, track)
			case "BEGIN_INSTRUMENT":
				ints, err := parseNumbers(rest)
				if err != nil {
					return nil, err
				}
				instr = Instrument{NumVoices: ints[0], Units: []Unit{}}
				inInstrument = true
			case "END_INSTRUMENT":
				patch = append(patch, instr)
				inInstrument = false
			case "DELTIME":
				ints, err := parseNumbers(rest)
				if err != nil {
					return nil, err
				}
				for _, v := range ints {
					delayTimes = append(delayTimes, v)
				}
			case "SAMPLE_OFFSET":
				ints, err := parseNumbers(rest)
				if err != nil {
					return nil, err
				}
				sampleOffsets = append(sampleOffsets, ints)
			}
			if inInstrument && strings.HasPrefix(word, "SU_") {
				unittype := strings.ToLower(word[3:])
				parameters, err := parseParams(rest)
				if err != nil {
					return nil, fmt.Errorf("Error parsing parameters: %v", err)
				}
				if unittype == "oscillator" {
					match := typeReg.FindStringSubmatch(rest)
					if match == nil {
						return nil, errors.New("Oscillator should define a type")
					}
					switch match[1] {
					case "SINE":
						parameters["type"] = Sine
					case "TRISAW":
						parameters["type"] = Trisaw
					case "PULSE":
						parameters["type"] = Pulse
					case "GATE":
						parameters["type"] = Gate
					case "SAMPLE":
						parameters["type"] = Sample
					}
				}
				unit := Unit{Type: unittype, Parameters: parameters}
				instr.Units = append(instr.Units, unit)
			}
		}
	}
	for i := range patch {
		for u := range patch[i].Units {
			if patch[i].Units[u].Type == "delay" {
				s := patch[i].Units[u].Parameters["delay"]
				e := patch[i].Units[u].Parameters["count"]
				if patch[i].Units[u].Parameters["stereo"] == 1 {
					e *= 2 // stereo delays use 'count' number of delaytimes, but for both channels
				}
				patch[i].Units[u].DelayTimes = append(patch[i].Units[u].DelayTimes, delayTimes[s:e]...)
				delete(patch[i].Units[u].Parameters, "delay")
				delete(patch[i].Units[u].Parameters, "count")
			} else if patch[i].Units[u].Type == "oscillator" && patch[i].Units[u].Parameters["type"] == Sample {
				sampleno := patch[i].Units[u].Parameters["color"]
				patch[i].Units[u].Parameters["start"] = sampleOffsets[sampleno][0]
				patch[i].Units[u].Parameters["loopstart"] = sampleOffsets[sampleno][1]
				patch[i].Units[u].Parameters["looplength"] = sampleOffsets[sampleno][2]
				delete(patch[i].Units[u].Parameters, "color")
			}
		}
	}
	s := Song{BPM: bpm, Patterns: patterns, Tracks: tracks, Patch: patch, SongLength: -1}
	return &s, nil
}

func SerializeAsm(song *Song) (string, error) {
	paramorder := map[string][]string{
		"add":        []string{"stereo"},
		"addp":       []string{"stereo"},
		"pop":        []string{"stereo"},
		"loadnote":   []string{"stereo"},
		"mul":        []string{"stereo"},
		"mulp":       []string{"stereo"},
		"push":       []string{"stereo"},
		"xch":        []string{"stereo"},
		"distort":    []string{"stereo", "drive"},
		"hold":       []string{"stereo", "holdfreq"},
		"crush":      []string{"stereo", "resolution"},
		"gain":       []string{"stereo", "gain"},
		"invgain":    []string{"stereo", "invgain"},
		"filter":     []string{"stereo", "frequency", "resonance", "lowpass", "bandpass", "highpass", "negbandpass", "neghighpass"},
		"clip":       []string{"stereo"},
		"pan":        []string{"stereo", "panning"},
		"delay":      []string{"stereo", "pregain", "dry", "feedback", "damp", "delay", "count", "notetracking"},
		"compressor": []string{"stereo", "attack", "release", "invgain", "threshold", "ratio"},
		"speed":      []string{},
		"out":        []string{"stereo", "gain"},
		"outaux":     []string{"stereo", "outgain", "auxgain"},
		"aux":        []string{"stereo", "gain", "channel"},
		"send":       []string{"stereo", "amount", "voice", "unit", "port", "sendpop"},
		"envelope":   []string{"stereo", "attack", "decay", "sustain", "release", "gain"},
		"noise":      []string{"stereo", "shape", "gain"},
		"oscillator": []string{"stereo", "transpose", "detune", "phase", "color", "shape", "gain", "type", "lfo", "unison"},
		"loadval":    []string{"stereo", "value"},
		"receive":    []string{"stereo"},
		"in":         []string{"stereo", "channel"},
	}
	indentation := 0
	indent := func() string {
		return strings.Repeat(" ", indentation*4)
	}
	var b strings.Builder
	println := func(format string, params ...interface{}) {
		if len(format) > 0 {
			fmt.Fprintf(&b, "%v", indent())
			fmt.Fprintf(&b, format, params...)
		}
		fmt.Fprintf(&b, "\n")
	}
	align := func(table [][]string, format string) [][]string {
		var maxwidth []int
		// find the maximum width of each column
		for _, row := range table {
			for k, elem := range row {
				l := len(elem)
				if len(maxwidth) <= k {
					maxwidth = append(maxwidth, l)
				} else {
					if maxwidth[k] < l {
						maxwidth[k] = l
					}
				}
			}
		}
		// align each column, depending on the specified formatting
		for _, row := range table {
			for k, elem := range row {
				l := len(elem)
				var f byte
				if k >= len(format) {
					f = format[len(format)-1] // repeat the last format specifier for all remaining columns
				} else {
					f = format[k]
				}
				switch f {
				case 'n': // no alignment
					row[k] = elem
				case 'l': // left align
					row[k] = elem + strings.Repeat(" ", maxwidth[k]-l)
				case 'r': // right align
					row[k] = strings.Repeat(" ", maxwidth[k]-l) + elem
				}
			}
		}
		return table
	}
	printTable := func(table [][]string) {
		indentation++
		for _, row := range table {
			println("%v %v", row[0], strings.Join(row[1:], ","))
		}
		indentation--
	}
	delayTable, delayIndices := ConstructDelayTimeTable(song.Patch)
	sampleTable, sampleIndices := ConstructSampleOffsetTable(song.Patch)
	// The actual printing starts here
	println("%%define BPM %d", song.BPM)
	// delay modulation is pretty much the only %define that the asm preprocessor cannot figure out
	// as the preprocessor has no clue if a SEND modulates a delay unit. So, unfortunately, for the
	// time being, we need to figure during export if INCLUDE_DELAY_MODULATION needs to be defined.
	delaymod := false
	for i, instrument := range song.Patch {
		for j, unit := range instrument.Units {
			if unit.Type == "send" {
				targetInstrument := i
				if unit.Parameters["voice"] > 0 {
					v, err := song.Patch.InstrumentForVoice(unit.Parameters["voice"] - 1)
					if err != nil {
						return "", fmt.Errorf("INSTRUMENT #%v / SEND #%v targets voice %v, which does not exist", i, j, unit.Parameters["voice"])
					}
					targetInstrument = v
				}
				if unit.Parameters["unit"] < 0 || unit.Parameters["unit"] >= len(song.Patch[targetInstrument].Units) {
					return "", fmt.Errorf("INSTRUMENT #%v / SEND #%v target unit %v out of range", i, j, unit.Parameters["unit"])
				}
				if song.Patch[targetInstrument].Units[unit.Parameters["unit"]].Type == "delay" && unit.Parameters["port"] == 5 {
					delaymod = true
				}
			}
		}
	}
	if delaymod {
		println("%%define INCLUDE_DELAY_MODULATION")
	}
	println("")
	println("%%include \"sointu/header.inc\"\n")
	var patternTable [][]string
	for _, pattern := range song.Patterns {
		row := []string{"PATTERN"}
		for _, v := range pattern {
			if v == 1 {
				row = append(row, "HLD")
			} else {
				row = append(row, strconv.Itoa(int(v)))
			}
		}
		patternTable = append(patternTable, row)
	}
	println("BEGIN_PATTERNS")
	printTable(align(patternTable, "lr"))
	println("END_PATTERNS\n")
	var trackTable [][]string
	for _, track := range song.Tracks {
		row := []string{"TRACK", fmt.Sprintf("VOICES(%d)", track.NumVoices)}
		for _, v := range track.Sequence {
			row = append(row, strconv.Itoa(int(v)))
		}
		trackTable = append(trackTable, row)
	}
	println("BEGIN_TRACKS")
	printTable(align(trackTable, "lr"))
	println("END_TRACKS\n")
	println("BEGIN_PATCH")
	indentation++
	for i, instrument := range song.Patch {
		var instrTable [][]string
		for j, unit := range instrument.Units {
			row := []string{fmt.Sprintf("SU_%v", strings.ToUpper(unit.Type))}
			for _, parname := range paramorder[unit.Type] {
				if unit.Type == "oscillator" && unit.Parameters["type"] == Sample && parname == "color" {
					row = append(row, fmt.Sprintf("COLOR(%v)", strconv.Itoa(sampleIndices[i][j])))
				} else if unit.Type == "delay" && parname == "count" {
					count := len(unit.DelayTimes)
					if unit.Parameters["stereo"] == 1 {
						count /= 2
					}
					row = append(row, fmt.Sprintf("COUNT(%v)", strconv.Itoa(count)))
				} else if unit.Type == "delay" && parname == "delay" {
					row = append(row, fmt.Sprintf("DELAY(%v)", strconv.Itoa(delayIndices[i][j])))
				} else if unit.Type == "oscillator" && parname == "type" {
					switch unit.Parameters["type"] {
					case Sine:
						row = append(row, "TYPE(SINE)")
					case Trisaw:
						row = append(row, "TYPE(TRISAW)")
					case Pulse:
						row = append(row, "TYPE(PULSE)")
					case Gate:
						row = append(row, "TYPE(GATE)")
					case Sample:
						row = append(row, "TYPE(SAMPLE)")
					}
				} else if v, ok := unit.Parameters[parname]; ok {
					row = append(row, fmt.Sprintf("%v(%v)", strings.ToUpper(parname), strconv.Itoa(int(v))))
				} else {
					return "", fmt.Errorf("The parameter map for unit %v does not contain %v, even though it should", unit.Type, parname)
				}
			}
			instrTable = append(instrTable, row)
		}
		println("BEGIN_INSTRUMENT VOICES(%d)", instrument.NumVoices)
		printTable(align(instrTable, "ln"))
		println("END_INSTRUMENT")
	}
	indentation--
	println("END_PATCH\n")
	if len(delayTable) > 0 {
		var delStrTable [][]string
		for _, v := range delayTable {
			row := []string{"DELTIME", strconv.Itoa(int(v))}
			delStrTable = append(delStrTable, row)
		}
		println("BEGIN_DELTIMES")
		printTable(align(delStrTable, "lr"))
		println("END_DELTIMES\n")
	}
	if len(sampleTable) > 0 {
		var samStrTable [][]string
		for _, v := range sampleTable {
			samStrTable = append(samStrTable, []string{
				"SAMPLE_OFFSET",
				fmt.Sprintf("START(%d)", v.Start),
				fmt.Sprintf("LOOPSTART(%d)", v.LoopStart),
				fmt.Sprintf("LOOPLENGTH(%d)", v.LoopLength),
			})
		}
		println("BEGIN_SAMPLE_OFFSETS")
		printTable(align(samStrTable, "r"))
		println("END_SAMPLE_OFFSETS\n")
	}
	println("%%include \"sointu/footer.inc\"")
	ret := b.String()
	return ret, nil
}
