package main

import (
	"fmt"

	psl "github.com/satishbabariya/prisma-go/psl"
	ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

func main() {
	// Create a sample Prisma schema
	schema := `datasource db {
  provider = "postgresql"
  url = "postgresql://localhost:5432/mydb"
}

generator client {
  provider = "prisma-client-js"
}

model User {
  id    Int     @id
  email String  @unique
  name  String?
}

model Post {
  id        Int     @id
  title     String
  content   String?
  published Boolean
}
`

	fmt.Println("🚀 PSL-Go Example - Domain-Driven Structure")
	fmt.Println("=" + "===========================================")
	fmt.Println()

	// Parse the schema
	fmt.Println("📝 Parsing schema...")
	source := psl.NewSourceFile("schema.prisma", schema)
	schemaAst, diags := psl.ParseSchemaFromFile(source)

	if diags.HasErrors() {
		fmt.Println("❌ Parse errors:")
		fmt.Println(diags.ToPrettyString(source.Path, source.Data))
		return
	}

	fmt.Printf("✅ Successfully parsed! Found %d top-level declarations:\n", len(schemaAst.Tops))
	for i, top := range schemaAst.Tops {
		switch t := top.(type) {
		case *ast.Model:
			fmt.Printf("   %d. Model: %s\n", i+1, t.Name.Name)
		case *ast.Enum:
			fmt.Printf("   %d. Enum: %s\n", i+1, t.Name.Name)
		case *ast.SourceConfig:
			fmt.Printf("   %d. Datasource: %s\n", i+1, t.Name.Name)
		case *ast.GeneratorConfig:
			fmt.Printf("   %d. Generator: %s\n", i+1, t.Name.Name)
		}
	}

	fmt.Println()

	// Reformat the schema
	fmt.Println("✨ Reformatting schema with 2-space indentation...")
	formatted, err := psl.Reformat(schema, 2)
	if err != nil {
		fmt.Printf("❌ Format error: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("📄 Formatted schema:")
	fmt.Println("-------------------")
	fmt.Println(formatted)

	fmt.Println()
	fmt.Println("✅ Example completed successfully!")
	fmt.Println()
	fmt.Println("📦 Package Structure:")
	fmt.Println("   - core/        : Core types and interfaces")
	fmt.Println("   - parsing/     : Lexer, Parser, AST")
	fmt.Println("   - formatting/  : Schema formatting")
	fmt.Println("   - diagnostics/ : Error handling")
}
