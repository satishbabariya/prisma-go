package main

import (
	"context"
	"fmt"
	"log"
	
	"github.com/satishbabariya/prisma-go/generated"
	"github.com/satishbabariya/prisma-go/query/builder"
)

func main() {
	// Example demonstrating aggregation functions
	
	ctx := context.Background()
	
	// Create client
	client, err := generated.NewPrismaClient("postgresql://localhost:5432/mydb")
	if err != nil {
		log.Fatal(err)
	}
	
	// Connect
	if err := client.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	
	// Example 1: Count all users
	fmt.Println("=== Example 1: Count All ===")
	count, err := client.User.Count(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Total users: %d\n", count)
	
	// Example 2: Count with filter
	fmt.Println("\n=== Example 2: Count with Filter ===")
	where := builder.NewWhereBuilder().Like("email", "%@gmail.com")
	gmailCount, err := client.User.CountWhere(ctx, where)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Gmail users: %d\n", gmailCount)
	
	// Example 3: Sum of IDs
	fmt.Println("\n=== Example 3: Sum ===")
	sumIds, err := client.User.SumId(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Sum of all user IDs: %.2f\n", sumIds)
	
	// Example 4: Average ID
	fmt.Println("\n=== Example 4: Average ===")
	avgId, err := client.User.AvgId(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Average user ID: %.2f\n", avgId)
	
	// Example 5: Min and Max
	fmt.Println("\n=== Example 5: Min and Max ===")
	minId, err := client.User.MinId(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Minimum user ID: %.2f\n", minId)
	
	maxId, err := client.User.MaxId(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Maximum user ID: %.2f\n", maxId)
	
	// Example 6: Aggregations with filters
	fmt.Println("\n=== Example 6: Aggregations with Filters ===")
	whereActive := builder.NewWhereBuilder().GreaterThan("id", 100)
	
	activeCount, _ := client.User.CountWhere(ctx, whereActive)
	activeSumIds, _ := client.User.SumIdWhere(ctx, whereActive)
	activeAvgId, _ := client.User.AvgIdWhere(ctx, whereActive)
	
	fmt.Printf("Active users (ID > 100): %d\n", activeCount)
	fmt.Printf("Sum of active user IDs: %.2f\n", activeSumIds)
	fmt.Printf("Average active user ID: %.2f\n", activeAvgId)
	
	// Example 7: Complex filter with aggregation
	fmt.Println("\n=== Example 7: Complex Filter ===")
	complexWhere := builder.NewWhereBuilder()
	complexWhere.OR(
		builder.NewSubWhereBuilder().Like("email", "%@gmail.com"),
		builder.NewSubWhereBuilder().Like("email", "%@yahoo.com"),
	)
	
	emailCount, _ := client.User.CountWhere(ctx, complexWhere)
	fmt.Printf("Gmail or Yahoo users: %d\n", emailCount)
	
	// Example 8: Statistics summary
	fmt.Println("\n=== Example 8: Statistics Summary ===")
	fmt.Println("User Statistics:")
	fmt.Printf("  Total: %d\n", count)
	fmt.Printf("  Min ID: %.0f\n", minId)
	fmt.Printf("  Max ID: %.0f\n", maxId)
	fmt.Printf("  Avg ID: %.2f\n", avgId)
	fmt.Printf("  Sum IDs: %.0f\n", sumIds)
	
	fmt.Println("\n✅ Aggregations work perfectly!")
	fmt.Println("Available aggregations:")
	fmt.Println("  • Count() / CountWhere()")
	fmt.Println("  • Sum{Field}() / Sum{Field}Where()")
	fmt.Println("  • Avg{Field}() / Avg{Field}Where()")
	fmt.Println("  • Min{Field}() / Min{Field}Where()")
	fmt.Println("  • Max{Field}() / Max{Field}Where()")
}

