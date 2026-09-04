package cli

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/marko-durasic/CloudAI-CLI/internal/version"
)

func TestVersionCommand(t *testing.T) {
	// Catch deleted init wiring before we override package vars.
	if rootCmd.Version != version.Version {
		t.Fatalf("rootCmd.Version=%q must equal version.Version=%q (init must set it)", rootCmd.Version, version.Version)
	}

	origVersion, origCommit, origDate := version.Version, version.Commit, version.Date
	origOut := rootCmd.OutOrStdout()

	t.Cleanup(func() {
		version.Version, version.Commit, version.Date = origVersion, origCommit, origDate
		rootCmd.SetOut(origOut)
		rootCmd.SetArgs(nil)
	})

	version.Version = "9.9.9-test"
	version.Commit = "deadbeef"
	version.Date = "2026-09-05T00:00:00Z"

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected version command to succeed, got: %v", err)
	}

	got := buf.String()
	want := fmt.Sprintf(
		"cloudai version %s (commit: %s, built at: %s)\n",
		version.Version,
		version.Commit,
		version.Date,
	)
	if got != want {
		t.Errorf("version output must equal package vars;\n got: %q\nwant: %q", got, want)
	}
}
