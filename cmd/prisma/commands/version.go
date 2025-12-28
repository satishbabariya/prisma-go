// Package commands implements CLI commands.
package commands

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version information (set at build time).
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

// NewVersionCommand creates the version command.
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Display version information for prisma-go CLI",
		Run: func(cmd *cobra.Command, args []string) {
			printVersionInfo()
		},
	}
}

func printVersionInfo() {
	fmt.Printf("prisma-go version %s\n", Version)
	fmt.Printf("  Git Commit: %s\n", GitCommit)
	fmt.Printf("  Build Time: %s\n", BuildTime)
	fmt.Printf("  Go Version: %s\n", runtime.Version())
	fmt.Printf("  OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
