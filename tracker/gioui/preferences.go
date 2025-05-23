package gioui

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"gioui.org/unit"
)

type (
	Preferences struct {
		Window   WindowPreferences
		YmlError error
	}

	WindowPreferences struct {
		Width     int
		Height    int
		Maximized bool `yaml:",omitempty"`
	}
)

//go:embed preferences.yml
var defaultPreferencesYaml []byte

func loadDefaultPreferences() Preferences {
	var preferences Preferences
	err := yaml.UnmarshalStrict(defaultPreferencesYaml, &preferences)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal preferences: %w", err))
	}
	return preferences
}

// ReadCustomConfigYml modifies the target argument, i.e. needs a pointer
func ReadCustomConfigYml(filename string, target interface{}) (exists bool, err error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return false, err
	}
	path := filepath.Join(configDir, "sointu", filename)
	bytes, err2 := os.ReadFile(path)
	if err2 != nil {
		return false, err2
	}
	err = yaml.Unmarshal(bytes, target)
	return true, err
}

func MakePreferences() Preferences {
	preferences := loadDefaultPreferences()
	exists, err := ReadCustomConfigYml("preferences.yml", &preferences)
	if exists {
		preferences.YmlError = err
	}
	return preferences
}

func (p Preferences) WindowSize() (unit.Dp, unit.Dp) {
	return unit.Dp(p.Window.Width), unit.Dp(p.Window.Height)
}
