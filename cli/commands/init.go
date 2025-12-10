package commands

import (
	"fmt"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/satishbabariya/prisma-go/cli/internal/config"
	"github.com/satishbabariya/prisma-go/cli/internal/ui"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new Prisma-Go project",
	Long: `Initialize a new Prisma-Go project with a schema file and configuration.

This command will:
- Create a schema.prisma file
- Create a .env.example file
- Create a .gitignore file
- Set up the project structure`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

var (
	initProjectName   string
	initProvider      string
	initDatabaseURL   string
	initSkipEnv       bool
	initInteractive   bool
)

func init() {
	initCmd.Flags().StringVarP(&initProjectName, "name", "n", "", "Project name (directory name)")
	initCmd.Flags().StringVarP(&initProvider, "provider", "p", "postgresql", "Database provider (postgresql, mysql, sqlite)")
	initCmd.Flags().StringVar(&initDatabaseURL, "database-url", "", "Database connection URL")
	initCmd.Flags().BoolVar(&initSkipEnv, "skip-env", false, "Skip creating .env.example file")
	initCmd.Flags().BoolVarP(&initInteractive, "interactive", "i", true, "Run in interactive mode")

	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	projectName := "."
	if len(args) > 0 {
		projectName = args[0]
	} else if initProjectName != "" {
		projectName = initProjectName
	}

	ui.PrintHeader("Prisma-Go", "Initialize a new project")

	// Interactive mode
	if initInteractive && !cmd.Flag("name").Changed && !cmd.Flag("provider").Changed {
		if err := runInitInteractive(&projectName, &initProvider, &initDatabaseURL); err != nil {
			return err
		}
	}

	// Validate provider
	validProviders := map[string]bool{
		"postgresql": true,
		"postgres":   true,
		"mysql":      true,
		"sqlite":     true,
	}
	if !validProviders[initProvider] {
		return fmt.Errorf("invalid provider: %s (supported: postgresql, mysql, sqlite)", initProvider)
	}

	// Normalize provider name
	if initProvider == "postgres" {
		initProvider = "postgresql"
	}

	fs := config.AppFs

	// Create project directory if needed
	if projectName != "." {
		ui.PrintStep(1, 5, fmt.Sprintf("Creating project directory: %s", projectName))
		if err := fs.MkdirAll(projectName, 0755); err != nil {
			ui.PrintError("Failed to create project directory: %v", err)
			return err
		}
		ui.PrintSuccess("Created project directory: %s", projectName)
	}

	// Create schema.prisma file
	schemaPath := filepath.Join(projectName, "schema.prisma")
	ui.PrintStep(2, 5, "Creating schema.prisma file")
	if exists, _ := afero.Exists(fs, schemaPath); exists {
		ui.PrintWarning("Schema file already exists: %s", schemaPath)
		overwrite := false
		if initInteractive {
			prompt := &survey.Confirm{
				Message: "Overwrite existing schema.prisma?",
				Default: false,
			}
			if err := survey.AskOne(prompt, &overwrite); err != nil {
				// User cancelled or error occurred - treat as "don't overwrite"
				ui.PrintInfo("Skipping schema overwrite")
			}
		}
		if !overwrite {
			ui.PrintInfo("Skipping schema creation...")
		} else {
			if err := createSchemaFile(fs, schemaPath, initProvider); err != nil {
				return err
			}
		}
	} else {
		if err := createSchemaFile(fs, schemaPath, initProvider); err != nil {
			return err
		}
	}

	// Create .env.example file and optionally .env file
	if !initSkipEnv {
		ui.PrintStep(3, 5, "Creating .env.example file")
		envExamplePath := filepath.Join(projectName, ".env.example")
		if exists, _ := afero.Exists(fs, envExamplePath); !exists {
			envContent := getEnvExampleContent(initProvider)
			if err := afero.WriteFile(fs, envExamplePath, []byte(envContent), 0644); err != nil {
				ui.PrintWarning("Failed to create .env.example: %v", err)
			} else {
				ui.PrintSuccess("Created .env.example file")
			}
		} else {
			ui.PrintInfo(".env.example already exists, skipping...")
		}

		// If database URL was provided, create .env file with it
		if initDatabaseURL != "" {
			envPath := filepath.Join(projectName, ".env")
			if exists, _ := afero.Exists(fs, envPath); !exists {
				envContent := fmt.Sprintf(`# Database connection string
DATABASE_URL="%s"
`, initDatabaseURL)
				if err := afero.WriteFile(fs, envPath, []byte(envContent), 0644); err != nil {
					ui.PrintWarning("Failed to create .env file: %v", err)
				} else {
					ui.PrintSuccess("Created .env file with provided DATABASE_URL")
				}
			} else {
				ui.PrintInfo(".env already exists, skipping...")
			}
		}
	}

	// Create .gitignore
	ui.PrintStep(4, 5, "Creating .gitignore file")
	gitignorePath := filepath.Join(projectName, ".gitignore")
	if exists, _ := afero.Exists(fs, gitignorePath); !exists {
		gitignoreContent := getGitignoreContent()
		if err := afero.WriteFile(fs, gitignorePath, []byte(gitignoreContent), 0644); err != nil {
			ui.PrintWarning("Failed to create .gitignore: %v", err)
		} else {
			ui.PrintSuccess("Created .gitignore file")
		}
	} else {
		ui.PrintInfo(".gitignore already exists, skipping...")
	}

	// Final step
	ui.PrintStep(5, 5, "Project initialization complete")

	ui.PrintSuccess("Prisma-Go project initialized successfully!")
	fmt.Println()

	// Show next steps
	ui.PrintSection("Next Steps")
	nextSteps := []string{
		"Set up your database and update DATABASE_URL in .env",
		"Edit schema.prisma to define your models",
		"Run: prisma-go generate",
		"Run: prisma-go migrate dev --name init",
		"Start building your application!",
	}
	ui.PrintList(nextSteps)

	return nil
}

func runInitInteractive(projectName *string, provider *string, databaseURL *string) error {
	qs := []*survey.Question{
		{
			Name: "projectName",
			Prompt: &survey.Input{
				Message: "Project name:",
				Default: ".",
				Help:    "Enter the project directory name (use '.' for current directory)",
			},
		},
		{
			Name: "provider",
			Prompt: &survey.Select{
				Message: "Database provider:",
				Options: []string{"postgresql", "mysql", "sqlite"},
				Default: "postgresql",
			},
		},
		{
			Name: "databaseURL",
			Prompt: &survey.Input{
				Message: "Database URL (optional):",
				Help:    "Leave empty to use DATABASE_URL from environment",
			},
		},
	}

	answers := struct {
		ProjectName string
		Provider    string
		DatabaseURL string
	}{}

	if err := survey.Ask(qs, &answers); err != nil {
		return err
	}

	*projectName = answers.ProjectName
	*provider = answers.Provider
	*databaseURL = answers.DatabaseURL

	return nil
}

func createSchemaFile(fs afero.Fs, schemaPath string, provider string) error {
	schemaContent := getSchemaTemplate(provider)
	if err := afero.WriteFile(fs, schemaPath, []byte(schemaContent), 0644); err != nil {
		ui.PrintError("Failed to create schema file: %v", err)
		return err
	}
	absPath, _ := filepath.Abs(schemaPath)
	ui.PrintSuccess("Created schema file: %s", absPath)
	return nil
}

func getSchemaTemplate(provider string) string {
	return fmt.Sprintf(`datasource db {
  provider = "%s"
  url      = env("DATABASE_URL")
}

generator client {
  provider = "prisma-client-go"
  output   = "./generated"
}

model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String?
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt
}
`, provider)
}

func getEnvExampleContent(provider string) string {
	var url string
	switch provider {
	case "postgresql", "postgres":
		url = `postgresql://user:password@localhost:5432/mydb?sslmode=disable`
	case "mysql":
		url = `mysql://user:password@localhost:3306/mydb`
	case "sqlite":
		url = `file:./dev.db`
	default:
		url = `postgresql://user:password@localhost:5432/mydb?sslmode=disable`
	}

	return fmt.Sprintf(`# Database connection string
DATABASE_URL="%s"
`, url)
}

func getGitignoreContent() string {
	return `# Generated files
generated/
*.generated.go

# Environment variables
.env
.env.local

# Migrations (optional - uncomment if you want to ignore migrations)
# migrations/

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db
`
}

