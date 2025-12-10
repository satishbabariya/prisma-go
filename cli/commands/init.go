package commands

import (
	"fmt"
	"os"
	"path/filepath"
)

func initCommand(args []string) error {
	projectName := "."
	if len(args) > 0 {
		projectName = args[0]
	}
	
	fmt.Println("🚀 Initializing Prisma-Go project...")
	
	// Create project directory if needed
	if projectName != "." {
		if err := os.MkdirAll(projectName, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to create project directory: %v\n", err)
			return err
		}
		fmt.Printf("📁 Created project directory: %s\n", projectName)
	}
	
	// Create schema.prisma file
	schemaPath := filepath.Join(projectName, "schema.prisma")
	if _, err := os.Stat(schemaPath); err == nil {
		fmt.Printf("⚠️  Schema file already exists: %s\n", schemaPath)
		fmt.Println("   Skipping schema creation...")
	} else {
		schemaContent := `datasource db {
  provider = "postgresql"
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
`
		if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to create schema file: %v\n", err)
			return err
		}
		fmt.Printf("✅ Created schema file: %s\n", schemaPath)
	}
	
	// Create .env.example file
	envExamplePath := filepath.Join(projectName, ".env.example")
	if _, err := os.Stat(envExamplePath); err != nil {
		envContent := `# Database connection string
DATABASE_URL="postgresql://user:password@localhost:5432/mydb?sslmode=disable"
`
		if err := os.WriteFile(envExamplePath, []byte(envContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to create .env.example: %v\n", err)
		} else {
			fmt.Printf("✅ Created .env.example file\n")
		}
	}
	
	// Create .gitignore if it doesn't exist
	gitignorePath := filepath.Join(projectName, ".gitignore")
	if _, err := os.Stat(gitignorePath); err != nil {
		gitignoreContent := `# Generated files
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
		if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to create .gitignore: %v\n", err)
		} else {
			fmt.Printf("✅ Created .gitignore file\n")
		}
	}
	
	fmt.Println("\n✅ Prisma-Go project initialized successfully!")
	fmt.Println("\n📝 Next steps:")
	fmt.Println("  1. Set up your database and update DATABASE_URL in .env")
	fmt.Println("  2. Edit schema.prisma to define your models")
	fmt.Println("  3. Run: prisma-go generate")
	fmt.Println("  4. Run: prisma-go migrate dev schema.prisma --name init")
	fmt.Println("  5. Start building your application!")
	
	return nil
}

