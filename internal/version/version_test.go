package version

import (
	"testing"
)

// The defaults are what every `go install` user sees, because go install builds
// carry no ldflags. Asserting the literal values means a default that looks like
// a real version (and would silently misreport) fails here.
func TestVersionDefaults(t *testing.T) {
	if Version != "dev" {
		t.Errorf("Version default = %q, want %q", Version, "dev")
	}
	if Commit != "none" {
		t.Errorf("Commit default = %q, want %q", Commit, "none")
	}
	if Date != "unknown" {
		t.Errorf("Date default = %q, want %q", Date, "unknown")
	}
}
