package compiler

import (
	"bytes"
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/vsariola/sointu/go4k"
)

//go:generate go run generate.go

type Compiler struct {
	Template        *template.Template
	Amd64           bool
	OS              string
	DisableSections bool
}

// New returns a new compiler using the default .asm templates
func New() (*Compiler, error) {
	_, myname, _, _ := runtime.Caller(0)
	templateDir := filepath.Join(path.Dir(myname), "..", "..", "templates")
	compiler, err := NewFromTemplates(templateDir)
	return compiler, err
}

func NewFromTemplates(directory string) (*Compiler, error) {
	globPtrn := filepath.Join(directory, "*.*")
	tmpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).ParseGlob(globPtrn)
	if err != nil {
		return nil, fmt.Errorf(`could not create template based on directory "%v": %v`, directory, err)
	}
	return &Compiler{Template: tmpl, Amd64: runtime.GOARCH == "amd64", OS: runtime.GOOS}, nil
}

func (com *Compiler) compile(templateName string, data interface{}) (string, error) {
	result := bytes.NewBufferString("")
	err := com.Template.ExecuteTemplate(result, templateName, data)
	return result.String(), err
}

func (com *Compiler) Library() (map[string]string, error) {
	features := AllFeatures{}
	m := NewMacros(*com, features)
	m.Library = true
	asmCode, err := com.compile("library.asm", m)
	if err != nil {
		return nil, fmt.Errorf(`could not execute template "library.asm": %v`, err)
	}

	m = NewMacros(*com, features)
	m.Library = true
	header, err := com.compile("library.h", &m)
	if err != nil {
		return nil, fmt.Errorf(`could not execute template "library.h": %v`, err)
	}
	return map[string]string{"asm": asmCode, "h": header}, nil
}

func (com *Compiler) Player(song *go4k.Song, maxSamples int) (map[string]string, error) {
	features := NecessaryFeaturesFor(song.Patch)
	encodedPatch, err := Encode(&song.Patch, features)
	if err != nil {
		return nil, fmt.Errorf(`could not encode patch: %v`, err)
	}
	asmCode, err := com.compile("player.asm", NewPlayerMacros(*com, features, song, encodedPatch, maxSamples))
	if err != nil {
		return nil, fmt.Errorf(`could not execute template "player.asm": %v`, err)
	}

	header, err := com.compile("player.h", NewPlayerMacros(*com, features, song, encodedPatch, maxSamples))
	if err != nil {
		return nil, fmt.Errorf(`could not execute template "player.h": %v`, err)
	}
	return map[string]string{"asm": asmCode, "h": header}, nil
}
