package version

import (
	"fmt"
	"runtime"
)

var (
	// BuildDate is supplied by linker
	BuildDate = "invalid:-use-make-to-build"
	// Version is supplied by linker
	Version = "invalid:-use-make-to-build"
)

// Print returns a string containing version information.
func Print() string {
	return fmt.Sprintf("certspotter-sd version %s build date %s go version %s",
		Version,
		BuildDate,
		runtime.Version(),
	)
}

// UserAgent returns string to be used as User-Agent header.
func UserAgent() string {
	return fmt.Sprintf("certspotter-sd/%s github.com/codecentric/certspotter-sd", Version)
}
