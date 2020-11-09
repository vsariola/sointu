package go4k

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

func ParseAsm(reader io.Reader) (*Song, error) {
	var bpm int
	scanner := bufio.NewScanner(reader)
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
	unitNameMap := map[string]string{
		"SU_ADD":      "add",
		"SU_ADDP":     "addp",
		"SU_POP":      "pop",
		"SU_LOADNOTE": "loadnote",
		"SU_MUL":      "mul",
		"SU_MULP":     "mulp",
		"SU_PUSH":     "push",
		"SU_XCH":      "xch",
		"SU_DISTORT":  "distort",
		"SU_HOLD":     "hold",
		"SU_CRUSH":    "crush",
		"SU_GAIN":     "gain",
		"SU_INVGAIN":  "invgain",
		"SU_FILTER":   "filter",
		"SU_CLIP":     "clip",
		"SU_PAN":      "pan",
		"SU_DELAY":    "delay",
		"SU_COMPRES":  "compressor",
		"SU_SPEED":    "speed",
		"SU_OUT":      "out",
		"SU_OUTAUX":   "outaux",
		"SU_AUX":      "aux",
		"SU_SEND":     "send",
		"SU_ENVELOPE": "envelope",
		"SU_NOISE":    "noise",
		"SU_OSCILLAT": "oscillator",
		"SU_LOADVAL":  "loadval",
		"SU_RECEIVE":  "receive",
		"SU_IN":       "in",
	}
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
			case "END_INSTRUMENT":
				patch = append(patch, instr)
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
			if unittype, ok := unitNameMap[word]; ok {
				instrMatch := wordReg.FindStringSubmatch(rest)
				if instrMatch != nil {
					stereoMono, instrRest := instrMatch[1], instrMatch[2]
					stereo := stereoMono == "STEREO"
					parameters, err := parseParams(instrRest)
					if err != nil {
						return nil, fmt.Errorf("Error parsing parameters: %v", err)
					}
					if unittype == "oscillator" {
						match := typeReg.FindStringSubmatch(instrRest)
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
					} else if unittype == "send" {
						if _, ok := parameters["voice"]; !ok {
							parameters["voice"] = -1
						}
					}
					unit := Unit{Type: unittype, Stereo: stereo, Parameters: parameters}
					instr.Units = append(instr.Units, unit)
				}
			}
		}
	}
	for i := range patch {
		for u := range patch[i].Units {
			if patch[i].Units[u].Type == "delay" {
				s := patch[i].Units[u].Parameters["delay"]
				e := patch[i].Units[u].Parameters["count"]
				if patch[i].Units[u].Stereo {
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
