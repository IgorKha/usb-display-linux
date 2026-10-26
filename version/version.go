package version

import "runtime"

var (
	// Version is an application version set via ldflags at build time.
	Version = "dev"
	// GitCommit is a short commit hash set via ldflags at build time.
	GitCommit = "none"
	// BuildTime is an RFC3339 UTC timestamp set via ldflags at build time.
	BuildTime = "unknown"
)

// Info returns build and runtime version metadata.
func Info() map[string]string {
	return map[string]string{
		"version":    Version,
		"git_commit": GitCommit,
		"build_time": BuildTime,
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
	}
}
