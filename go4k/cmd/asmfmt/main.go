package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/vsariola/sointu/go4k"
)

func main() {
	write := flag.Bool("w", false, "Do not print reformatted asm songs to standard output. If a file's formatting is different from asmfmt's, overwrite it with asmfmt's version.")
	list := flag.Bool("l", false, "Do not print reformatted asm songs to standard output, just list the filenames that reformatting changes.")
	help := flag.Bool("h", false, "show help")
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() == 0 || *help {
		flag.Usage()
		os.Exit(0)
	}
	process := func(filename string) error {
		origCodeBytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("could not read the file (%v)", err)
		}
		origCode := string(origCodeBytes)
		song, err := go4k.DeserializeAsm(origCode)
		if err != nil {
			return fmt.Errorf("could not parse the file (%v)", err)
		}
		formattedCode, err := go4k.SerializeAsm(song)
		if err != nil {
			return fmt.Errorf("could not reformat the file (%v)", err)
		}
		if *write {
			if formattedCode != origCode {
				err := ioutil.WriteFile(filename, []byte(formattedCode), 0644)
				if err != nil {
					return fmt.Errorf("could write to file (%v)", err)
				}
			}
		}
		if *list {
			if formattedCode != origCode {
				fmt.Println(filename)
			}
		} else if !*write {
			fmt.Print(formattedCode)
		}
		return nil
	}
	retval := 0
	for _, param := range flag.Args() {
		if info, err := os.Stat(param); err == nil && info.IsDir() {
			files, err := filepath.Glob(path.Join(param, "*.asm"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not glob the path %v\n", param)
				continue
			}
			for _, file := range files {
				err := process(file)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v: %v\n", file, err)
					retval = 1
				}
			}
		} else {
			err := process(param)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v: %v\n", param, err)
				retval = 1
			}
		}
	}
	os.Exit(retval)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [flags] [path ...]\n", os.Args[0])
	flag.PrintDefaults()
}
