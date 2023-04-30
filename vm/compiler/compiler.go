package compiler

import (
	"bytes"
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
)

type Compiler struct {
	Template    *template.Template
	OS          string
	Arch        string
	Output16Bit bool
	RowSync     bool
}

// New returns a new compiler using the default .asm templates
func New(os string, arch string, output16Bit bool, rowsync bool) (*Compiler, error) {
	_, myname, _, _ := runtime.Caller(0)
	var subdir string
	if arch == "386" || arch == "amd64" {
		subdir = "amd64-386"
	} else if arch == "wasm" {
		subdir = "wasm"
	} else {
		return nil, fmt.Errorf("compiler.New failed, because only amd64, 386 and wasm archs are supported (targeted architecture was %v)", arch)
	}
	templateDir := filepath.Join(path.Dir(myname), "..", "..", "templates", subdir)
	compiler, err := NewFromTemplates(os, arch, output16Bit, rowsync, templateDir)
	return compiler, err
}

func NewFromTemplates(os string, arch string, output16Bit bool, rowsync bool, templateDirectory string) (*Compiler, error) {
	globPtrn := filepath.Join(templateDirectory, "*.*")
	tmpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).ParseGlob(globPtrn)
	if err != nil {
		return nil, fmt.Errorf(`could not create template based on directory "%v": %v`, templateDirectory, err)
	}
	return &Compiler{Template: tmpl, OS: os, Arch: arch, RowSync: rowsync, Output16Bit: output16Bit}, nil
}

func (com *Compiler) Library() (map[string]string, error) {
	if com.Arch != "386" && com.Arch != "amd64" {
		return nil, fmt.Errorf(`compiling as a library is supported only on 386 and amd64 architectures (targeted architecture was %v)`, com.Arch)
	}
	templates := []string{"library.asm", "library.h"}
	features := vm.AllFeatures{}
	retmap := map[string]string{}
	for _, templateName := range templates {
		compilerMacros := *NewCompilerMacros(*com)
		compilerMacros.Library = true
		featureSetMacros := FeatureSetMacros{features}
		x86Macros := *NewX86Macros(com.OS, com.Arch == "amd64", features, false)
		data := struct {
			CompilerMacros
			FeatureSetMacros
			X86Macros
		}{compilerMacros, featureSetMacros, x86Macros}
		populatedTemplate, extension, err := com.compile(templateName, &data)
		if err != nil {
			return nil, fmt.Errorf(`could not execute template "%v": %v`, templateName, err)
		}
		retmap[extension] = populatedTemplate
	}
	return retmap, nil
}

func (com *Compiler) Song(song *sointu.Song) (map[string]string, error) {
	if com.Arch != "386" && com.Arch != "amd64" && com.Arch != "wasm" {
		return nil, fmt.Errorf(`compiling a song player is supported only on 386, amd64 and wasm architectures (targeted architecture was %v)`, com.Arch)
	}
	var templates []string
	if com.Arch == "386" || com.Arch == "amd64" {
		templates = []string{"player.asm", "player.h"}
	} else if com.Arch == "wasm" {
		templates = []string{"player.wat"}
	}
	features := vm.NecessaryFeaturesFor(song.Patch)
	retmap := map[string]string{}
	encodedPatch, err := vm.Encode(song.Patch, features)
	if err != nil {
		return nil, fmt.Errorf(`could not encode patch: %v`, err)
	}
	patterns, sequences, err := vm.ConstructPatterns(song)
	if err != nil {
		return nil, fmt.Errorf(`could not encode song: %v`, err)
	}
	for _, templateName := range templates {
		compilerMacros := *NewCompilerMacros(*com)
		featureSetMacros := FeatureSetMacros{features}
		songMacros := *NewSongMacros(song)
		var populatedTemplate, extension string
		var err error
		if com.Arch == "386" || com.Arch == "amd64" {
			x86Macros := *NewX86Macros(com.OS, com.Arch == "amd64", features, false)
			data := struct {
				CompilerMacros
				FeatureSetMacros
				X86Macros
				SongMacros
				*vm.BytePatch
				Patterns       [][]byte
				Sequences      [][]byte
				PatternLength  int
				SequenceLength int
				Hold           int
			}{compilerMacros, featureSetMacros, x86Macros, songMacros, encodedPatch, patterns, sequences, len(patterns[0]), len(sequences[0]), 1}
			populatedTemplate, extension, err = com.compile(templateName, &data)
		} else if com.Arch == "wasm" {
			wasmMacros := *NewWasmMacros()
			data := struct {
				CompilerMacros
				FeatureSetMacros
				WasmMacros
				SongMacros
				*vm.BytePatch
				Patterns       [][]byte
				Sequences      [][]byte
				PatternLength  int
				SequenceLength int
				Hold           int
				RenderOnStart  bool
			}{compilerMacros, featureSetMacros, wasmMacros, songMacros, encodedPatch, patterns, sequences, len(patterns[0]), len(sequences[0]), 1, !song.WasmDisableRenderOnStart}
			populatedTemplate, extension, err = com.compile(templateName, &data)
		}
		if err != nil {
			return nil, fmt.Errorf(`could not execute template "%v": %v`, templateName, err)
		}
		retmap[extension] = populatedTemplate
	}
	return retmap, nil
}

func (com *Compiler) compile(templateName string, data interface{}) (string, string, error) {
	result := bytes.NewBufferString("")
	err := com.Template.ExecuteTemplate(result, templateName, data)
	extension := filepath.Ext(templateName)
	return result.String(), extension, err
}
