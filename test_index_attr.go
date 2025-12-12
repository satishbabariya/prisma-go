package main

import (
	"fmt"
	"github.com/satishbabariya/prisma-go/psl/parsing/v2"
)

func main() {
	input := `
model Test {
	id Int
	name String
	
	@@index([name], map: "test_index")
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
		fmt.Printf("Block attributes: %d\n", len(models[0].BlockAttributes))
		for _, attr := range models[0].BlockAttributes {
			fmt.Printf("  Attribute: %s\n", attr.String())
		}
	}
}
