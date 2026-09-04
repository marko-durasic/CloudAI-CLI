package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected version command to succeed, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "cloudai version") {
		t.Errorf("expected output to contain 'cloudai version', got: %s", output)
	}
}
