package gioui

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"gioui.org/unit"
)

type (
	Preferences struct {
		Window WindowPreferences
	}

	WindowPreferences struct {
		Width     int
		Height    int
		Maximized bool `yaml:",omitempty"`
	}
)

//go:embed preferences.yml
var defaultPreferences []byte

// ReadCustomConfig modifies the target argument, i.e. needs a pointer. Just
// fails silently if the file cannot be found/read, but will warn about
// malformed files.
func ReadCustomConfig(filename string, target any) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil
	}
	path := filepath.Join(configDir, "sointu", filename)
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	if err := yaml.Unmarshal(bytes, target); err != nil {
		return fmt.Errorf("ReadCustomConfig %v: %w", filename, err)
	}
	return nil
}

// ReadConfig first unmarshals the defaultConfig which should be the embedded
// default config, and then tries to read the custom config with
// ReadCustomConfig. It panics right away if the embedded defaultConfig could
// not be parsed as yaml as this should never happen except during development.
// The returned error should be treated as a warning: this function will always
// return at least the default config, and the warning will just tell if there
// was a problem parsing the custom config.
func ReadConfig(defaultConfig []byte, path string, target any) (warn error) {
	dec := yaml.NewDecoder(bytes.NewReader(defaultConfig))
	dec.KnownFields(true)
	if err := dec.Decode(target); err != nil {
		panic(fmt.Errorf("ReadConfig %v failed to unmarshal the embedded default config: %w", path, err))
	}
	return ReadCustomConfig(path, target)
}

func (p Preferences) WindowSize() (unit.Dp, unit.Dp) {
	return unit.Dp(p.Window.Width), unit.Dp(p.Window.Height)
}
