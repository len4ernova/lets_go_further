package vcs

import (
	"fmt"
	"runtime/debug"
)

func Version() string {
	// получим структуру  debug.BuildInfo,
	// если доступна, то достанем номер версии
	bi, ok := debug.ReadBuildInfo()
	if ok {
		return bi.Main.Version
	}
	return ""
}

// пользовательский формат версии 2025-02-21T10:16:24Z-1c9b6ff48ea8
func VersionCustom() string {
	var (
		time     string
		revision string
		modified bool
	)
	bi, ok := debug.ReadBuildInfo()
	fmt.Println(bi.Settings)
	if ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.time":
				time = s.Value
			case "vcs.revision":
				revision = s.Value
			case "vcs.modified":
				if s.Value == "true" {
					modified = true
				}
			}
		}
	}
	if modified {
		return fmt.Sprintf("%s-%s+dirty", time, revision)
	}
	return fmt.Sprintf("%s-%s", time, revision)
}
