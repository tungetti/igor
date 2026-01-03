package main

// Build information. These variables are set via ldflags during build.
// Example:
//
//	go build -ldflags "-X main.Version=1.0.0 -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.GitCommit=$(git rev-parse HEAD)"
var (
	// Version is the semantic version of the application.
	Version = "1.1.0"

	// BuildTime is the UTC timestamp when the binary was built.
	BuildTime = "unknown"

	// GitCommit is the git commit hash the binary was built from.
	GitCommit = "unknown"
)
