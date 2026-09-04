package version

import (
	"testing"
)

func TestVersionDefaults(t *testing.T) {
	if Version == "" {
		t.Error("expected Version to have default value")
	}
	if Commit == "" {
		t.Error("expected Commit to have default value")
	}
	if Date == "" {
		t.Error("expected Date to have default value")
	}
}
