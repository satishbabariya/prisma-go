package main

import (
	"fmt"
	"github.com/satishbabariya/prisma-go/psl/parsing/v2"
)

func main() {
	input := `
model Test {
	model String @db.VarChar
}
`

	schema, err := schema.ParseSchemaString("test.prisma", input)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Parsed successfully!")
	models := schema.Models()
	if len(models) > 0 {
		fmt.Printf("Model: %s\n", models[0].GetName())
		for _, field := range models[0].Fields {
			fmt.Printf("  Field: %s (%s)\n", field.GetName(), field.GetTypeName())
		}
	}
}
