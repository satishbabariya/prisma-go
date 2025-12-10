package main

import (
	"fmt"

	pls "github.com/satishbabariya/prisma-go/psl"
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
	source := pls.NewSourceFile("schema.prisma", schema)
	ast, diags := pls.ParseSchemaFromFile(source)

	if diags.HasErrors() {
		fmt.Println("❌ Parse errors:")
		fmt.Println(diags.ToPrettyString(source.Path, source.Data))
		return
	}

	fmt.Printf("✅ Successfully parsed! Found %d top-level declarations:\n", len(ast.Tops))
	for i, top := range ast.Tops {
		if model := top.AsModel(); model != nil {
			fmt.Printf("   %d. Model: %s\n", i+1, model.Name.Name)
		} else if enum := top.AsEnum(); enum != nil {
			fmt.Printf("   %d. Enum: %s\n", i+1, enum.Name.Name)
		} else if source := top.AsSource(); source != nil {
			fmt.Printf("   %d. Datasource: %s\n", i+1, source.Name.Name)
		} else if gen := top.AsGenerator(); gen != nil {
			fmt.Printf("   %d. Generator: %s\n", i+1, gen.Name.Name)
		}
	}

	fmt.Println()

	// Reformat the schema
	fmt.Println("✨ Reformatting schema with 2-space indentation...")
	formatted, err := pls.Reformat(schema, 2)
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
