package version

import "runtime/debug"

// You can set the version at build time using something like:
// go build -ldflags "-X github.com/vsariola/sointu/version.Version=$(git describe --dirty)"

var Version string

var Hash = func() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		modified := false
		for _, setting := range info.Settings {
			if setting.Key == "vcs.modified" && setting.Value == "true" {
				modified = true
				break
			}
		}
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				shortHash := setting.Value[:7]
				if modified {
					return shortHash + "-dirty"
				}
				return shortHash
			}
		}
	}
	return ""
}()

var VersionOrHash = func() string {
	if Version != "" {
		return Version
	}
	return Hash
}()
