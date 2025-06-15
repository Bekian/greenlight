package vcs

import (
	"runtime/debug"
)

func Version() string {
	// get the version from build info
	bi, ok := debug.ReadBuildInfo()
	if ok {
		return bi.Main.Version
	}

	return ""
}
