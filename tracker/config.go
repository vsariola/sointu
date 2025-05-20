package tracker

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

func ReadConfig(embedFS embed.FS, path string, out any) (warning error) {
	bytes := must(fs.ReadFile(embedFS, path)) // the file _must_ exist in the embedded fs; panic on purpose if not
	if err := yaml.UnmarshalStrict(bytes, out); err != nil {
		panic(err) // also, the embedded default config file _must_ be strictly valid yaml; panic if not
	}
	// now, try to read the user config file, overlapping the default one.
	// Notice that the return values are just warnings - there's always the
	// default config there so no harm done if the user config is missing or
	// invalid.
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("ReadConfig %v: %v", path, err)
	}
	configPath := filepath.Join(configDir, "sointu", path)
	bytesUser, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("ReadConfig %v: %v", path, err)
	}
	if err := yaml.Unmarshal(bytesUser, out); err != nil {
		return fmt.Errorf("ReadConfig %v: %v", path, err)
	}
	return nil
}

func must[T any](ic T, err error) T {
	if err != nil {
		panic(err)
	}
	return ic
}
