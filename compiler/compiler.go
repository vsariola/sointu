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
)

type Compiler struct {
	Template *template.Template
	OS       string
	Arch     string
}

// New returns a new compiler using the default .asm templates
func New(os string, arch string) (*Compiler, error) {
	_, myname, _, _ := runtime.Caller(0)
	templateDir := filepath.Join(path.Dir(myname), "..", "templates")
	compiler, err := NewFromTemplates(os, arch, templateDir)
	return compiler, err
}

func NewFromTemplates(os string, arch string, templateDirectory string) (*Compiler, error) {
	globPtrn := filepath.Join(templateDirectory, "*.*")
	tmpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).ParseGlob(globPtrn)
	if err != nil {
		return nil, fmt.Errorf(`could not create template based on directory "%v": %v`, templateDirectory, err)
	}
	return &Compiler{Template: tmpl, OS: os, Arch: arch}, nil
}

func (com *Compiler) Library() (map[string]string, error) {
	if com.Arch != "386" && com.Arch != "amd64" {
		return nil, fmt.Errorf(`compiling as a library is supported only on 386 and amd64 architectures (targeted architecture was %v)`, com.Arch)
	}
	templates := []string{"library.asm", "library.h"}
	features := AllFeatures{}
	retmap := map[string]string{}
	for _, templateName := range templates {
		macros := NewMacros(*com, features)
		macros.Library = true
		populatedTemplate, extension, err := com.compile(templateName, macros)
		if err != nil {
			return nil, fmt.Errorf(`could not execute template "%v": %v`, templateName, err)
		}
		retmap[extension] = populatedTemplate
	}
	return retmap, nil
}

func (com *Compiler) Song(song *sointu.Song) (map[string]string, error) {
	if com.Arch != "386" && com.Arch != "amd64" {
		return nil, fmt.Errorf(`compiling a song player is supported only on 386 and amd64 architectures (targeted architecture was %v)`, com.Arch)
	}
	templates := []string{"player.asm", "player.h"}
	features := NecessaryFeaturesFor(song.Patch)
	retmap := map[string]string{}
	encodedPatch, err := Encode(&song.Patch, features)
	if err != nil {
		return nil, fmt.Errorf(`could not encode patch: %v`, err)
	}
	for _, templateName := range templates {
		macros := NewPlayerMacros(*com, features, song, encodedPatch)
		populatedTemplate, extension, err := com.compile(templateName, macros)
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
