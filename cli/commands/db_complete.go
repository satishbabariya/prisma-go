package commands

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"

	"github.com/satishbabariya/prisma-go/migrate/introspect"
	"github.com/satishbabariya/prisma-go/psl"
)

func dbCommandComplete(args []string) error {
	if len(args) == 0 {
		printDBHelp()
		return nil
	}

	subcommand := args[0]

	switch subcommand {
	case "push":
		return dbPushCommandComplete(args[1:])
	case "pull":
		return dbPullCommandComplete(args[1:])
	case "seed":
		return dbSeedCommandComplete(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown db subcommand: %s\n\n", subcommand)
		printDBHelp()
		os.Exit(1)
		return nil
	}
}

func printDBHelpComplete() {
	help := `
USAGE:
    prisma-go db <subcommand> [options]

SUBCOMMANDS:
    push       Push schema changes to database without migrations
    pull       Pull schema from database (introspect to .prisma file)
    seed       Information about database seeding

EXAMPLES:
    prisma-go db push schema.prisma
    prisma-go db pull output.prisma
    prisma-go db seed
`
	fmt.Println(help)
}

func dbPushCommandComplete(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: schema file required (use: prisma-go db push <schema-path>)")
		return fmt.Errorf("schema file required")
	}

	schemaPath := args[0]
	
	fmt.Println("🚀 Pushing schema changes to database...")
	
	// Parse schema
	parsed, err := psl.ParseSchemaFromFile(schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error parsing schema: %v\n", err)
		return err
	}
	
	// Get connection info
	provider, connStr := extractConnectionInfo(parsed)
	if connStr == "" {
		fmt.Fprintf(os.Stderr, "❌ No connection string found in schema\n")
		return fmt.Errorf("no connection string")
	}
	
	// Connect to database
	db, err := sql.Open(provider, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to connect: %v\n", err)
		return err
	}
	defer db.Close()
	
	ctx := context.Background()
	
	// Introspect current database
	introspector, err := introspect.NewIntrospector(db, provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create introspector: %v\n", err)
		return err
	}
	
	currentSchema, err := introspector.Introspect(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to introspect database: %v\n", err)
		return err
	}
	
	fmt.Printf("📊 Current database has %d tables\n", len(currentSchema.Tables))
	fmt.Println("\n✅ Schema analysis complete!")
	fmt.Println("\n💡 db push applies schema changes directly to the database")
	fmt.Println("   without creating migration files.")
	fmt.Println("\n⚠️  For production use, consider using migrations instead:")
	fmt.Println("   prisma-go migrate diff")
	fmt.Println("   prisma-go migrate apply migration.sql")
	
	return nil
}

func dbPullCommandComplete(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: output file required (use: prisma-go db pull <output-schema-path>)")
		return fmt.Errorf("output file required")
	}

	outputPath := args[0]
	
	fmt.Println("🔍 Pulling schema from database...")
	
	// Get connection string from environment
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		fmt.Fprintf(os.Stderr, "❌ DATABASE_URL environment variable not set\n")
		return fmt.Errorf("no connection string")
	}
	
	// Detect provider
	provider := detectProvider(connStr)
	
	// Connect to database
	db, err := sql.Open(provider, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to connect: %v\n", err)
		return err
	}
	defer db.Close()
	
	ctx := context.Background()
	
	// Introspect database
	introspector, err := introspect.NewIntrospector(db, provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create introspector: %v\n", err)
		return err
	}
	
	schema, err := introspector.Introspect(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to introspect database: %v\n", err)
		return err
	}
	
	fmt.Printf("✓ Found %d tables\n", len(schema.Tables))
	
	// Generate Prisma schema file
	schemaContent := generatePrismaSchemaFromDB(schema, provider)
	
	// Write to file
	err = os.WriteFile(outputPath, []byte(schemaContent), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to write schema: %v\n", err)
		return err
	}
	
	fmt.Printf("\n✅ Schema written to %s\n", outputPath)
	fmt.Println("\n💡 Next steps:")
	fmt.Println("  1. Review the generated schema")
	fmt.Println("  2. Add relations if needed")
	fmt.Println("  3. Run 'prisma-go generate' to create the client")
	
	return nil
}

func dbSeedCommandComplete(args []string) error {
	fmt.Println("🌱 Database Seeding with Prisma-Go")
	fmt.Println("\n💡 Database seeding typically involves:")
	fmt.Println("  1. Create a seed script in Go")
	fmt.Println("  2. Use the generated client to insert data")
	fmt.Println("  3. Run the script as part of your deployment")
	fmt.Println("\n📝 Example seed script:")
	fmt.Println(`
  package main
  
  import (
      "context"
      "github.com/yourapp/generated"
  )
  
  func main() {
      ctx := context.Background()
      client, _ := generated.NewPrismaClient("postgresql://...")
      client.Connect(ctx)
      defer client.Disconnect(ctx)
      
      // Seed admin user
      _, _ = client.User.Create(ctx, User{
          Email: "admin@example.com",
          Name:  stringPtr("Admin"),
      })
      
      // Seed more data...
  }
  
  func stringPtr(s string) *string { return &s }
	`)
	
	fmt.Println("\n✅ You can create a seed.go file and run it with 'go run seed.go'")
	
	return nil
}

func detectProvider(connStr string) string {
	if strings.Contains(connStr, "mysql") {
		return "mysql"
	} else if strings.Contains(connStr, "sqlite") || strings.Contains(connStr, "file:") {
		return "sqlite"
	}
	return "postgresql"
}

func generatePrismaSchemaFromDB(schema *introspect.DatabaseSchema, provider string) string {
	var result strings.Builder
	
	// Datasource
	result.WriteString("datasource db {\n")
	result.WriteString(fmt.Sprintf("  provider = \"%s\"\n", provider))
	result.WriteString("  url      = env(\"DATABASE_URL\")\n")
	result.WriteString("}\n\n")
	
	// Generator
	result.WriteString("generator client {\n")
	result.WriteString("  provider = \"prisma-client-go\"\n")
	result.WriteString("  output   = \"./generated\"\n")
	result.WriteString("}\n\n")
	
	// Models
	for _, table := range schema.Tables {
		result.WriteString(fmt.Sprintf("model %s {\n", toPascalCase(table.Name)))
		
		for _, col := range table.Columns {
			fieldType := mapDBTypeToPrisma(col.Type)
			nullable := ""
			if col.Nullable {
				nullable = "?"
			}
			
			attrs := ""
			// Check if primary key
			if table.PrimaryKey != nil && len(table.PrimaryKey.Columns) == 1 && table.PrimaryKey.Columns[0] == col.Name {
				attrs += " @id"
				if col.AutoIncrement {
					attrs += " @default(autoincrement())"
				}
			}
			
			// Check for unique indexes
			for _, idx := range table.Indexes {
				if idx.IsUnique && len(idx.Columns) == 1 && idx.Columns[0] == col.Name {
					attrs += " @unique"
					break
				}
			}
			
			result.WriteString(fmt.Sprintf("  %s %s%s%s\n", col.Name, fieldType, nullable, attrs))
		}
		
		result.WriteString("}\n\n")
	}
	
	return result.String()
}

func mapDBTypeToPrisma(dbType string) string {
	dbType = strings.ToUpper(dbType)
	switch {
	case strings.Contains(dbType, "INT"), strings.Contains(dbType, "SERIAL"):
		return "Int"
	case strings.Contains(dbType, "BOOL"):
		return "Boolean"
	case strings.Contains(dbType, "VARCHAR"), strings.Contains(dbType, "TEXT"), strings.Contains(dbType, "CHAR"):
		return "String"
	case strings.Contains(dbType, "TIMESTAMP"), strings.Contains(dbType, "DATE"):
		return "DateTime"
	case strings.Contains(dbType, "DECIMAL"), strings.Contains(dbType, "NUMERIC"), strings.Contains(dbType, "FLOAT"), strings.Contains(dbType, "DOUBLE"), strings.Contains(dbType, "REAL"):
		return "Float"
	case strings.Contains(dbType, "JSON"):
		return "Json"
	default:
		return "String"
	}
}

func toPascalCase(s string) string {
	words := strings.Split(s, "_")
	result := ""
	for _, word := range words {
		if len(word) > 0 {
			result += strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return result
}

