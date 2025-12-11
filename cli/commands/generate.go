package commands

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/satishbabariya/prisma-go/cli/internal/ui"
	"github.com/satishbabariya/prisma-go/cli/internal/watch"
	"github.com/satishbabariya/prisma-go/generator"
	psl "github.com/satishbabariya/prisma-go/psl"
)

var generateCmd = &cobra.Command{
	Use:   "generate [schema-path]",
	Short: "Generate Prisma Client for Go",
	Long: `Generate Prisma Client code from your Prisma schema.

This command will:
- Parse and validate your schema.prisma file
- Generate type-safe Go client code
- Create model structs and query builders`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGenerate,
}

var (
	generateSchemaPath string
	generateWatch      bool
	generateWatchOnly  bool
)

func init() {
	generateCmd.Flags().StringVarP(&generateSchemaPath, "schema", "s", "schema.prisma", "Path to schema file")
	generateCmd.Flags().BoolVarP(&generateWatch, "watch", "w", false, "Watch schema file for changes")
	generateCmd.Flags().BoolVar(&generateWatchOnly, "watch-only", false, "Only watch, don't generate initially")

	rootCmd.AddCommand(generateCmd)
}

func runGenerate(cmd *cobra.Command, args []string) error {
	schemaPath := getSchemaPath(generateSchemaPath, args)

	// Watch mode
	if generateWatch || generateWatchOnly {
		return runGenerateWatch(schemaPath, !generateWatchOnly)
	}

	ui.PrintHeader("Prisma-Go", "Generate Client")

	spinner, _ := ui.PrintSpinner("Generating Prisma Client...")
	defer spinner.Stop()

	// Check if schema file exists
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		spinner.Stop()
		return fmt.Errorf("schema file not found: %s", schemaPath)
	}

	// Read schema
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		spinner.Stop()
		return fmt.Errorf("failed to read schema: %w", err)
	}

	// Parse schema
	sourceFile := psl.NewSourceFile(schemaPath, string(content))
	ast, diags := psl.ParseSchemaFromFile(sourceFile)

	if diags.HasErrors() {
		spinner.Stop()
		ui.PrintError("Schema parsing failed:")
		fmt.Fprintf(os.Stderr, "\n%s\n", diags.ToPrettyString(schemaPath, string(content)))
		return fmt.Errorf("cannot generate from invalid schema")
	}

	// Determine output directory from generator config or use default
	outputDir := "./generated"
	provider := "postgresql" // Default provider

	// Get schema file directory for resolving relative paths
	schemaDir := filepath.Dir(schemaPath)

	// Extract provider from datasource
	for _, top := range ast.Tops {
		if datasource := top.AsSource(); datasource != nil {
			for _, prop := range datasource.Properties {
				if prop.Name.Name == "provider" {
					if value, _ := prop.Value.AsStringValue(); value != nil {
						provider = value.Value
					}
				}
			}
		}
		if gen := top.AsGenerator(); gen != nil {
			for _, prop := range gen.Properties {
				if prop.Name.Name == "output" {
					if value, _ := prop.Value.AsStringValue(); value != nil {
						outputDir = value.Value
						// Resolve relative paths relative to schema file directory
						if !filepath.IsAbs(outputDir) {
							outputDir = filepath.Join(schemaDir, outputDir)
						}
					}
				}
			}
		}
	}

	spinner.UpdateText("Parsing schema...")
	spinner.Stop()

	// Show generation info
	info := pterm.Info.WithPrefix(pterm.Prefix{
		Text:  "INFO",
		Style: pterm.NewStyle(pterm.FgBlue),
	})

	info.Println(fmt.Sprintf("Schema: %s", schemaPath))
	info.Println(fmt.Sprintf("Output: %s", outputDir))
	info.Println(fmt.Sprintf("Provider: %s", provider))
	fmt.Println()

	// Create generator
	spinner, _ = ui.PrintSpinner("Generating code...")
	gen := generator.NewGenerator(ast, provider)

	// Generate client code
	if err := gen.GenerateClient(outputDir); err != nil {
		spinner.Stop()
		return fmt.Errorf("code generation failed: %w", err)
	}

	spinner.Stop()

	absPath, _ := filepath.Abs(outputDir)
	ui.PrintSuccess("Generated Prisma Client at %s", absPath)
	fmt.Println()

	// Show generated files
	ui.PrintSection("Generated Files")
	files := []string{
		"models.go  - Model structs",
		"client.go  - Prisma client",
	}
	ui.PrintList(files)

	fmt.Println()
	ui.PrintSection("Next Steps")
	nextSteps := []string{
		"Import the generated package in your code",
		"Create a client: client, _ := generated.NewPrismaClient(\"connection-string\")",
		"Use the client: users, _ := client.User.FindMany(ctx)",
	}
	ui.PrintList(nextSteps)

	return nil
}

func runGenerateWatch(schemaPath string, generateInitially bool) error {
	ui.PrintHeader("Prisma-Go", "Watch Mode")

	// Check if schema file exists
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		return fmt.Errorf("schema file not found: %s", schemaPath)
	}

	// Generate callback function
	generateCallback := func() error {
		ui.PrintInfo("Schema changed, regenerating...")

		// Read schema
		content, err := os.ReadFile(schemaPath)
		if err != nil {
			return fmt.Errorf("failed to read schema: %w", err)
		}

		// Parse schema
		sourceFile := psl.NewSourceFile(schemaPath, string(content))
		ast, diags := psl.ParseSchemaFromFile(sourceFile)

		if diags.HasErrors() {
			ui.PrintError("Schema parsing failed:")
			fmt.Fprintf(os.Stderr, "\n%s\n", diags.ToPrettyString(schemaPath, string(content)))
			return fmt.Errorf("cannot generate from invalid schema")
		}

		// Determine output directory from generator config or use default
		outputDir := "./generated"
		provider := "postgresql" // Default provider

		// Get schema file directory for resolving relative paths
		schemaDir := filepath.Dir(schemaPath)

		// Extract provider from datasource
		for _, top := range ast.Tops {
			if datasource := top.AsSource(); datasource != nil {
				for _, prop := range datasource.Properties {
					if prop.Name.Name == "provider" {
						if value, _ := prop.Value.AsStringValue(); value != nil {
							provider = value.Value
						}
					}
				}
			}
			if gen := top.AsGenerator(); gen != nil {
				for _, prop := range gen.Properties {
					if prop.Name.Name == "output" {
						if value, _ := prop.Value.AsStringValue(); value != nil {
							outputDir = value.Value
							// Resolve relative paths relative to schema file directory
							if !filepath.IsAbs(outputDir) {
								outputDir = filepath.Join(schemaDir, outputDir)
							}
						}
					}
				}
			}
		}

		// Create generator
		gen := generator.NewGenerator(ast, provider)

		// Generate client code
		if err := gen.GenerateClient(outputDir); err != nil {
			return fmt.Errorf("code generation failed: %w", err)
		}

		absPath, _ := filepath.Abs(outputDir)
		ui.PrintSuccess("Generated Prisma Client at %s", absPath)
		return nil
	}

	// Generate initially if requested
	if generateInitially {
		if err := generateCallback(); err != nil {
			return err
		}
	}

	// Create watcher
	watcher, err := watch.NewWatcher(schemaPath, generateCallback)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Stop()

	// Start watching
	if err := watcher.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	ui.PrintSuccess("Watching %s for changes... (Press Ctrl+C to stop)", schemaPath)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	ui.PrintInfo("\nStopping watch mode...")
	return nil
}

// generateCommand is a helper function for backward compatibility
// It can be called from other commands that need to trigger generation
func generateCommand(args []string) error {
	return runGenerate(nil, args)
}
