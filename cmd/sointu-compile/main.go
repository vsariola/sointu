package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/version"
	"github.com/vsariola/sointu/vm/compiler"
)

func filterExtensions(input map[string]string, extensions []string) map[string]string {
	ret := map[string]string{}
	for _, ext := range extensions {
		extWithDot := "." + ext
		if inputVal, ok := input[extWithDot]; ok {
			ret[extWithDot] = inputVal
		}
	}
	return ret
}

func main() {
	safe := flag.Bool("n", false, "Never overwrite files; if file already exists and would be overwritten, give an error.")
	list := flag.Bool("l", false, "Do not write files; just list files that would change instead.")
	stdout := flag.Bool("s", false, "Do not write files; write to standard output instead.")
	help := flag.Bool("h", false, "Show help.")
	rowsync := flag.Bool("r", false, "Write the current fractional row as sync #0")
	library := flag.Bool("a", false, "Compile Sointu into a library. Input files are not needed.")
	jsonOut := flag.Bool("j", false, "Output the song as .json file instead of compiling.")
	yamlOut := flag.Bool("y", false, "Output the song as .yml file instead of compiling.")
	tmplDir := flag.String("t", "", "When compiling, use the templates in this directory instead of the standard templates.")
	outPath := flag.String("o", "", "Directory or filename where to write compiled code. Extension is ignored. Directory and its parents are created if needed. By default, everything is placed in the same directory where the original song file is.")
	extensionsOut := flag.String("e", "", "Output only the compiled files with these comma separated extensions. For example: h,asm")
	targetArch := flag.String("arch", runtime.GOARCH, "Target architecture. Defaults to OS architecture. Possible values: 386, amd64, wasm")
	output16bit := flag.Bool("i", false, "Compiled song should output 16-bit integers, instead of floats.")
	targetOs := flag.String("os", runtime.GOOS, "Target OS. Defaults to current OS. Possible values: windows, darwin, linux. Anything else is assumed linuxy. Ignored when targeting wasm.")
	versionFlag := flag.Bool("v", false, "Print version.")
	forceSingleThread := flag.Bool("f", false, "Force single threaded rendering, even if patch if configured to use multiple threads.")
	flag.Usage = printUsage
	flag.Parse()
	if *versionFlag {
		fmt.Println(version.VersionOrHash)
		os.Exit(0)
	}
	if (flag.NArg() == 0 && !*library) || *help {
		flag.Usage()
		os.Exit(0)
	}
	compile := !*jsonOut && !*yamlOut // if the user gives nothing to output, then the default behaviour is to compile the file
	var comp *compiler.Compiler
	if compile || *library {
		var err error
		if *tmplDir != "" {
			comp, err = compiler.NewFromTemplates(*targetOs, *targetArch, *output16bit, *rowsync, *forceSingleThread, *tmplDir)
		} else {
			comp, err = compiler.New(*targetOs, *targetArch, *output16bit, *rowsync, *forceSingleThread)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, `error creating compiler: %v`, err)
			os.Exit(1)
		}
	}
	output := func(filename string, extension string, contents []byte) error {
		if *stdout {
			fmt.Print(string(contents))
			return nil
		}
		_, name := filepath.Split(filename)
		var dir string
		if *outPath != "" {
			// check if it's an already existing directory and the user just forgot trailing slash
			if info, err := os.Stat(*outPath); err == nil && info.IsDir() {
				dir = *outPath
			} else {
				outdir, outname := filepath.Split(*outPath)
				if outdir != "" {
					dir = outdir
				}
				if outname != "" {
					name = outname
				}
			}
		}
		if dir == "" {
			var err error
			dir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("could not get working directory, specify the output directory explicitly: %v", err)
			}
		}
		name = strings.TrimSuffix(name, filepath.Ext(name)) + extension
		f := filepath.Join(dir, name)
		original, err := ioutil.ReadFile(f)
		if err == nil {
			if bytes.Compare(original, contents) == 0 {
				return nil // no need to update
			}
			if !*list && *safe {
				return fmt.Errorf("file %v would be overwritten by compiler", f)
			}
		}
		if *list {
			fmt.Println(f)
		} else {
			if dir != "" {
				if err := os.MkdirAll(dir, os.ModePerm); err != nil {
					return fmt.Errorf("could not create output directory %v: %v", dir, err)
				}
			}
			err := ioutil.WriteFile(f, contents, 0644)
			if err != nil {
				return fmt.Errorf("could not write file %v: %v", f, err)
			}
		}
		return nil
	}
	process := func(filename string) error {
		inputBytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("could not read file %v: %v", filename, err)
		}
		var song sointu.Song
		if errJSON := json.Unmarshal(inputBytes, &song); errJSON != nil {
			if errYaml := yaml.Unmarshal(inputBytes, &song); errYaml != nil {
				return fmt.Errorf("song could not be unmarshaled as a .json (%v) or .yml (%v)", errJSON, errYaml)
			}
		}
		if song.RowsPerBeat == 0 {
			song.RowsPerBeat = 4
		}
		if song.Score.Length == 0 {
			song.Score.Length = len(song.Score.Tracks[0].Patterns)
		}
		var compiledPlayer map[string]string
		if compile {
			var err error
			compiledPlayer, err = comp.Song(song)
			if err != nil {
				return fmt.Errorf("compiling player failed: %v", err)
			}
			if len(*extensionsOut) > 0 {
				compiledPlayer = filterExtensions(compiledPlayer, strings.Split(*extensionsOut, ","))
			}
			for extension, code := range compiledPlayer {
				if err := output(filename, extension, []byte(code)); err != nil {
					return fmt.Errorf("error outputting %v file: %v", extension, err)
				}
			}
		}
		if *jsonOut {
			jsonSong, err := json.Marshal(song)
			if err != nil {
				return fmt.Errorf("could not marshal the song as json file: %v", err)
			}
			if err := output(filename, ".json", jsonSong); err != nil {
				return fmt.Errorf("error outputting json file: %v", err)
			}
		}
		if *yamlOut {
			yamlSong, err := yaml.Marshal(song)
			if err != nil {
				return fmt.Errorf("could not marshal the song as yaml file: %v", err)
			}
			if err := output(filename, ".yml", yamlSong); err != nil {
				return fmt.Errorf("error outputting yaml file: %v", err)
			}
		}
		return nil
	}
	retval := 0
	if *library {
		compiledLibrary, err := comp.Library()
		if err != nil {
			fmt.Fprintf(os.Stderr, "compiling library failed: %v\n", err)
			retval = 1
		} else {
			if len(*extensionsOut) > 0 {
				compiledLibrary = filterExtensions(compiledLibrary, strings.Split(*extensionsOut, ","))
			}
			for extension, code := range compiledLibrary {
				if err := output("sointu", extension, []byte(code)); err != nil {
					fmt.Fprintf(os.Stderr, "error outputting %v file: %v", extension, err)
					retval = 1
				}
			}
		}
	}
	for _, param := range flag.Args() {
		if info, err := os.Stat(param); err == nil && info.IsDir() {
			jsonfiles, err := filepath.Glob(filepath.Join(param, "*.json"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not glob the path %v for json files: %v\n", param, err)
				retval = 1
				continue
			}
			ymlfiles, err := filepath.Glob(filepath.Join(param, "*.yml"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not glob the path %v for yml files: %v\n", param, err)
				retval = 1
				continue
			}
			files := append(ymlfiles, jsonfiles...)
			for _, file := range files {
				err := process(file)
				if err != nil {
					fmt.Fprintf(os.Stderr, "could not process file %v: %v\n", file, err)
					retval = 1
				}
			}
		} else {
			err := process(param)
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not process file %v: %v\n", param, err)
				retval = 1
			}
		}
	}
	os.Exit(retval)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Sointu compiler. Input .yml or .json songs, outputs compiled songs (e.g. .asm and .h files).\nUsage: %s [flags] [path ...]\n", os.Args[0])
	flag.PrintDefaults()
}
