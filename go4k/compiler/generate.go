// The following directive is necessary to make the package coherent:

// +build ignore

// This program generates the library headers and assembly files for the library
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/vsariola/sointu/go4k/compiler"
)

func main() {
	targetArch := flag.String("arch", runtime.GOARCH, "Target architecture. Defaults to Go architecture. Possible values: amd64, 386 (anything else is assumed 386)")
	targetOs := flag.String("os", runtime.GOOS, "Target OS. Defaults to current Go OS. Possible values: windows, darwin, linux (anything else is assumed linux)")
	flag.Usage = printUsage
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(0)
	}

	comp, err := compiler.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, `error creating compiler: %v`, err)
		os.Exit(1)
	}

	comp.Amd64 = *targetArch == "amd64"
	comp.OS = *targetOs

	library, err := comp.Library()
	if err != nil {
		fmt.Fprintf(os.Stderr, `error compiling library: %v`, err)
		os.Exit(1)
	}

	filenames := map[string]string{"h": "sointu.h", "asm": "sointu.asm"}

	for t, contents := range library {
		filename := filenames[t]
		err := ioutil.WriteFile(filepath.Join(flag.Args()[0], filename), []byte(contents), os.ModePerm)
		if err != nil {
			fmt.Fprintf(os.Stderr, `could not write to file "%v": %v`, filename, err)
			os.Exit(1)
		}
	}
	os.Exit(0)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Sointu command line utility for generating the library .asm and .h files.\nUsage: %s [flags] outputDirectory\n", os.Args[0])
	flag.PrintDefaults()
}
