package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	psl "github.com/satishbabariya/prisma-go/psl"
	ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

func main() {
	// Test directory containing Prisma schema files
	testDir := "sample/prisma-test-files"

	fmt.Println("🧪 Testing V2 Parser with Real-World Schemas")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println()

	// Get all .prisma files
	files, err := filepath.Glob(filepath.Join(testDir, "*.prisma"))
	if err != nil {
		fmt.Printf("Error reading test files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Printf("No .prisma files found in %s\n", testDir)
		os.Exit(1)
	}

	totalFiles := len(files)
	successCount := 0
	failCount := 0
	var failures []string

	for i, file := range files {
		baseName := filepath.Base(file)
		fmt.Printf("[%d/%d] Testing: %s\n", i+1, totalFiles, baseName)

		// Read schema file
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("  ❌ Error reading file: %v\n", err)
			failCount++
			failures = append(failures, fmt.Sprintf("%s: read error - %v", baseName, err))
			continue
		}

		// Parse schema
		sourceFile := psl.NewSourceFile(file, string(content))
		schema, diags := psl.ParseSchemaFromFile(sourceFile)

		// Check for errors
		if diags.HasErrors() {
			fmt.Printf("  ❌ Parse errors:\n")
			fmt.Printf("%s\n", diags.ToPrettyString(file, string(content)))
			failCount++
			failures = append(failures, fmt.Sprintf("%s: parse errors", baseName))
		} else {
			// Print summary
			modelCount := 0
			enumCount := 0
			datasourceCount := 0
			generatorCount := 0

			for _, top := range schema.Tops {
				switch top.(type) {
				case *ast.Model:
					modelCount++
				case *ast.Enum:
					enumCount++
				case *ast.SourceConfig:
					datasourceCount++
				case *ast.GeneratorConfig:
					generatorCount++
				}
			}

			fmt.Printf("  ✅ Success - %d models, %d enums, %d datasource(s), %d generator(s)\n",
				modelCount, enumCount, datasourceCount, generatorCount)
			successCount++
		}

		// Add warnings if present
		if len(diags.Warnings()) > 0 {
			fmt.Printf("  ⚠️  %d warning(s)\n", len(diags.Warnings()))
		}

		fmt.Println()
	}

	// Print summary
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("📊 Test Summary:\n")
	fmt.Printf("  Total: %d files\n", totalFiles)
	fmt.Printf("  ✅ Success: %d (%.1f%%)\n", successCount, float64(successCount)/float64(totalFiles)*100)
	fmt.Printf("  ❌ Failed: %d (%.1f%%)\n", failCount, float64(failCount)/float64(totalFiles)*100)
	fmt.Println()

	if failCount > 0 {
		fmt.Println("Failed files:")
		for _, failure := range failures {
			fmt.Printf("  - %s\n", failure)
		}
		os.Exit(1)
	} else {
		fmt.Println("🎉 All schemas parsed successfully!")
	}
}
