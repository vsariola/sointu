package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/gorilla/mux"
	"github.com/vsariola/sointu"
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

func output(filename string, extension string, contents []byte) error {
	_, name := filepath.Split(filename)
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get working directory, specify the output directory explicitly: %v", err)
	}

	name = strings.TrimSuffix(name, filepath.Ext(name)) + extension
	f := filepath.Join(dir, name)
	err = ioutil.WriteFile(f, contents, 0644)
	if err != nil {
		return fmt.Errorf("could not write file %v: %v", f, err)
	}
	return nil
}

func process(filename string) error {
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

	comp, err := compiler.New(runtime.GOOS, "wasm", false, false)
	if err != nil {
		return fmt.Errorf("error creating compiler: %v", err)
	}

	compiledPlayer, err := comp.Song(&song)
	if err != nil {
		return fmt.Errorf("compiling player failed: %v", err)
	}

	for extension, code := range compiledPlayer {
		if err := output(filename, extension, []byte(code)); err != nil {
			return fmt.Errorf("error outputting %v file: %v", extension, err)
		}
	}

	return nil
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	// Handle pre-flight OPTIONS request
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var input map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}

	content, ok := input["content"].(string)
	if !ok {
		http.Error(w, "Invalid JSON input: content is missing or not a string", http.StatusBadRequest)
		return
	}

	filename := "temp_song_file.yml"
	if err := ioutil.WriteFile(filename, []byte(content), 0644); err != nil {
		http.Error(w, fmt.Sprintf("Error writing temporary file: %v", err), http.StatusInternalServerError)
		return
	}

	err = process(filename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error processing file: %v", err), http.StatusInternalServerError)
		return
	}

	watFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".wat"
	watContent, err := ioutil.ReadFile(watFilename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading WAT file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	w.WriteHeader(http.StatusOK)
	w.Write(watContent)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	// Your root handling logic goes here
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Sointu server. Find the WebAssembly Music app to use it."))
}

func main() {
	// Set up router and server
	router := mux.NewRouter()
	router.HandleFunc("/process", handleRequest).Methods("POST", "OPTIONS")
	router.HandleFunc("/", handleRoot).Methods("GET")
	http.Handle("/", router)

	port := "10000"
	fmt.Printf("Starting server on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
}
