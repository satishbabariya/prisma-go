package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/satishbabariya/prisma-go/cli/internal/ui"
	"github.com/satishbabariya/prisma-go/cli/internal/version"
)

var (
	cfgFile      string
	verbose      bool
	noColor      bool
	skipEnvCheck bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "prisma-go",
	Short: "Prisma-Go - Native Go Prisma ORM & Schema Engine",
	Long: `Prisma-Go is a native Go implementation of Prisma, providing:
- Schema management and validation
- Database migrations
- Type-safe query builder
- Code generation

For more information, visit: https://github.com/satishbabariya/prisma-go`,
	Version: version.Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize UI settings
		if noColor {
			// Disable colors
			os.Setenv("NO_COLOR", "1")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Show help if no subcommand
		if err := cmd.Help(); err != nil {
			ui.PrintError("Failed to show help: %v", err)
			os.Exit(1)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/prisma-go/.prisma-go.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().BoolVar(&skipEnvCheck, "skip-env-check", false, "skip environment variable checks")
	rootCmd.PersistentFlags().Bool("no-telemetry", false, "disable telemetry collection")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("no_color", rootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("skip_env_check", rootCmd.PersistentFlags().Lookup("skip-env-check"))

	// Add version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print the version number and build information for prisma-go",
		Run: func(cmd *cobra.Command, args []string) {
			info := version.Get()
			if verbose {
				fmt.Println(info.FullString())
			} else {
				fmt.Println(info.String())
			}
		},
	}

	rootCmd.AddCommand(versionCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in home directory with name ".prisma-go" (without extension).
		home, err := os.UserHomeDir()
		if err != nil {
			ui.PrintError("Failed to get home directory: %v", err)
			os.Exit(1)
		}

		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
		viper.AddConfigPath(fmt.Sprintf("%s/.config/prisma-go", home))
		viper.SetConfigType("yaml")
		viper.SetConfigName(".prisma-go")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			ui.PrintInfo("Using config file: %s", viper.ConfigFileUsed())
		}
	}
}
