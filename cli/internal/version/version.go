package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the version of the CLI
	Version = "0.1.0"
	// BuildDate is the build date
	BuildDate = "unknown"
	// GitCommit is the git commit hash
	GitCommit = "unknown"
)

// Info holds version information
type Info struct {
	Version   string
	BuildDate string
	GitCommit string
	GoVersion string
	Platform  string
}

// Get returns version information
func Get() Info {
	return Info{
		Version:   Version,
		BuildDate: BuildDate,
		GitCommit: GitCommit,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string
func (i Info) String() string {
	return fmt.Sprintf("prisma-go version %s (%s %s)", i.Version, i.Platform, i.GoVersion)
}

// FullString returns a detailed version string
func (i Info) FullString() string {
	return fmt.Sprintf(`prisma-go version %s
Build Date: %s
Git Commit: %s
Platform: %s
Go Version: %s`, i.Version, i.BuildDate, i.GitCommit, i.Platform, i.GoVersion)
}
