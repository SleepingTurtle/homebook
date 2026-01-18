package version

// These variables are set via ldflags at build time.
// Example: go build -ldflags "-X homebooks/internal/version.Version=1.0.0"
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)
