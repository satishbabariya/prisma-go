package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	
	_ "github.com/lib/pq"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

func main() {
	// Example demonstrating database introspection
	
	ctx := context.Background()
	
	// Connect to PostgreSQL
	db, err := sql.Open("postgres", "postgresql://localhost:5432/mydb?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	
	// Ping to verify connection
	if err := db.PingContext(ctx); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	
	fmt.Println("=== PostgreSQL Database Introspection ===\n")
	
	// Create introspector
	introspector, err := introspect.NewIntrospector(db, "postgresql")
	if err != nil {
		log.Fatal(err)
	}
	
	// Introspect database
	schema, err := introspector.Introspect(ctx)
	if err != nil {
		log.Fatal("Failed to introspect database:", err)
	}
	
	// Display results
	fmt.Printf("Found %d tables\n\n", len(schema.Tables))
	
	for _, table := range schema.Tables {
		fmt.Printf("Table: %s (schema: %s)\n", table.Name, table.Schema)
		fmt.Printf("  Columns: %d\n", len(table.Columns))
		
		for _, col := range table.Columns {
			nullable := ""
			if col.Nullable {
				nullable = " (nullable)"
			}
			
			autoInc := ""
			if col.AutoIncrement {
				autoInc = " [AUTO_INCREMENT]"
			}
			
			defaultVal := ""
			if col.DefaultValue != nil {
				defaultVal = fmt.Sprintf(" DEFAULT: %s", *col.DefaultValue)
			}
			
			fmt.Printf("    - %s: %s%s%s%s\n", col.Name, col.Type, nullable, autoInc, defaultVal)
		}
		
		if table.PrimaryKey != nil {
			fmt.Printf("  Primary Key: %s (%s)\n", table.PrimaryKey.Name, table.PrimaryKey.Columns)
		}
		
		if len(table.Indexes) > 0 {
			fmt.Printf("  Indexes: %d\n", len(table.Indexes))
			for _, idx := range table.Indexes {
				unique := ""
				if idx.IsUnique {
					unique = " [UNIQUE]"
				}
				fmt.Printf("    - %s%s on (%s)\n", idx.Name, unique, idx.Columns)
			}
		}
		
		if len(table.ForeignKeys) > 0 {
			fmt.Printf("  Foreign Keys: %d\n", len(table.ForeignKeys))
			for _, fk := range table.ForeignKeys {
				fmt.Printf("    - %s: %s -> %s(%s)\n", 
					fk.Name, 
					fk.Columns, 
					fk.ReferencedTable, 
					fk.ReferencedColumns)
				fmt.Printf("      ON UPDATE: %s, ON DELETE: %s\n", fk.OnUpdate, fk.OnDelete)
			}
		}
		
		fmt.Println()
	}
	
	if len(schema.Enums) > 0 {
		fmt.Printf("Enums: %d\n", len(schema.Enums))
		for _, enum := range schema.Enums {
			fmt.Printf("  - %s: %v\n", enum.Name, enum.Values)
		}
		fmt.Println()
	}
	
	if len(schema.Sequences) > 0 {
		fmt.Printf("Sequences: %d\n", len(schema.Sequences))
		for _, seq := range schema.Sequences {
			fmt.Printf("  - %s\n", seq.Name)
		}
		fmt.Println()
	}
	
	fmt.Println("✅ Introspection complete!")
	fmt.Println("\nThis information can be used to:")
	fmt.Println("  • Generate Prisma schema from existing database")
	fmt.Println("  • Compare schema with database (diffing)")
	fmt.Println("  • Generate migrations")
	fmt.Println("  • Validate schema consistency")
}

