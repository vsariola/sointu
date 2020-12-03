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
	output16Bit := false
	scanner := bufio.NewScanner(strings.NewReader(asmcode))
	patterns := make([][]byte, 0)
	tracks := make([]Track, 0)
	var patch Patch
	var instr Instrument
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
					} else if defineName == "OUTPUT_16BIT" {
						output16Bit = true
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
				patch.Instruments = append(patch.Instruments, instr)
				inInstrument = false
			case "DELTIME":
				ints, err := parseNumbers(rest)
				if err != nil {
					return nil, err
				}
				for _, v := range ints {
					patch.DelayTimes = append(patch.DelayTimes, v)
				}
			case "SAMPLE_OFFSET":
				ints, err := parseNumbers(rest)
				if err != nil {
					return nil, err
				}
				patch.SampleOffsets = append(patch.SampleOffsets, SampleOffset{
					Start:      ints[0],
					LoopStart:  ints[1],
					LoopLength: ints[2]})
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
	s := Song{BPM: bpm, Patterns: patterns, Tracks: tracks, Patch: patch, Output16Bit: output16Bit}
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
	// The actual printing starts here
	println("%%define BPM %d", song.BPM)
	if song.Output16Bit {
		println("%%define OUTPUT_16BIT")
	}
	// delay modulation is pretty much the only %define that the asm preprocessor cannot figure out
	// as the preprocessor has no clue if a SEND modulates a delay unit. So, unfortunately, for the
	// time being, we need to figure during export if INCLUDE_DELAY_MODULATION needs to be defined.
	delaymod := false
	for i, instrument := range song.Patch.Instruments {
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
				if unit.Parameters["unit"] < 0 || unit.Parameters["unit"] >= len(song.Patch.Instruments[targetInstrument].Units) {
					return "", fmt.Errorf("INSTRUMENT #%v / SEND #%v target unit %v out of range", i, j, unit.Parameters["unit"])
				}
				if song.Patch.Instruments[targetInstrument].Units[unit.Parameters["unit"]].Type == "delay" && unit.Parameters["port"] == 5 {
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
	for _, instrument := range song.Patch.Instruments {
		var instrTable [][]string
		for _, unit := range instrument.Units {
			row := []string{fmt.Sprintf("SU_%v", strings.ToUpper(unit.Type))}
			for _, parname := range paramorder[unit.Type] {
				if unit.Type == "oscillator" && parname == "type" {
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
	if len(song.Patch.DelayTimes) > 0 {
		var delStrTable [][]string
		for _, v := range song.Patch.DelayTimes {
			row := []string{"DELTIME", strconv.Itoa(int(v))}
			delStrTable = append(delStrTable, row)
		}
		println("BEGIN_DELTIMES")
		printTable(align(delStrTable, "lr"))
		println("END_DELTIMES\n")
	}
	if len(song.Patch.SampleOffsets) > 0 {
		var samStrTable [][]string
		for _, v := range song.Patch.SampleOffsets {
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

func ExportCHeader(song *Song, maxSamples int) string {
	template :=
		`// auto-generated by Sointu, editing not recommended
#ifndef SU_RENDER_H
#define SU_RENDER_H

#define SU_MAX_SAMPLES     %v
#define SU_BUFFER_LENGTH   (SU_MAX_SAMPLES*2)

#define SU_SAMPLE_RATE     44100
#define SU_BPM             %v
#define SU_PATTERN_SIZE    %v
#define SU_MAX_PATTERNS    %v
#define SU_TOTAL_ROWS      (SU_MAX_PATTERNS*SU_PATTERN_SIZE)
#define SU_SAMPLES_PER_ROW (SU_SAMPLE_RATE*4*60/(SU_BPM*16))

#include <stdint.h>
#if UINTPTR_MAX == 0xffffffff
    #if defined(__clang__) || defined(__GNUC__)
        #define SU_CALLCONV __attribute__ ((stdcall))
    #elif defined(_WIN32)
        #define SU_CALLCONV __stdcall
    #endif
#else
    #define SU_CALLCONV
#endif

typedef %v SUsample;
#define SU_SAMPLE_RANGE %v

#ifdef __cplusplus
extern "C" {
#endif

void SU_CALLCONV su_render_song(SUsample *buffer);
%v
#ifdef __cplusplus
}
#endif

#endif
`
	maxSamplesText := "SU_TOTAL_ROWS*SU_SAMPLES_PER_ROW"
	if maxSamples > 0 {
		maxSamplesText = fmt.Sprintf("%v", maxSamples)
	}
	sampleType := "float"
	sampleRange := "1.0f"
	if song.Output16Bit {
		sampleType = "short"
		sampleRange = "32768"
	}
	defineGmdls := ""
	for _, instr := range song.Patch.Instruments {
		for _, unit := range instr.Units {
			if unit.Type == "oscillator" && unit.Parameters["type"] == Sample {
				defineGmdls = "\n#define SU_LOAD_GMDLS\nvoid SU_CALLCONV su_load_gmdls(void);"
				break
			}
		}
	}
	header := fmt.Sprintf(template, maxSamplesText, song.BPM, song.PatternRows(), song.SequenceLength(), sampleType, sampleRange, defineGmdls)
	return header
}
