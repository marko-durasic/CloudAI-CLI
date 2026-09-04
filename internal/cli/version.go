package cli

import (
	"fmt"

	"github.com/marko-durasic/CloudAI-CLI/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version, commit, and build date of CloudAI-CLI",
	Long:  `Print detailed version information including the version tag, git commit SHA, and build timestamp.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "cloudai version %s (commit: %s, built at: %s)\n", version.Version, version.Commit, version.Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.Version = version.Version
}
