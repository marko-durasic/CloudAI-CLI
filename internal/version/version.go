package version

var (
	// Version is the semantic version of the binary (set via ldflags).
	Version = "dev"

	// Commit is the git commit SHA (set via ldflags).
	Commit = "none"

	// Date is the build timestamp (set via ldflags).
	Date = "unknown"
)
