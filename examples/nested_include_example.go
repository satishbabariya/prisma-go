package main

import (
	"context"
	"fmt"
	"log"
	
	"github.com/satishbabariya/prisma-go/generated"
)

func main() {
	// Example demonstrating nested includes
	// This shows how to load deep relations in a single query
	
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
	
	// Example 1: Simple include - load users with their posts
	fmt.Println("=== Example 1: Simple Include ===")
	users, err := client.User.
		Query().
		Include().Posts().Done().
		Execute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	
	for _, user := range users {
		fmt.Printf("User: %s\n", user.Email)
		fmt.Printf("  Posts: %d\n", len(user.Posts))
	}
	
	// Example 2: Nested include - load posts with their authors
	// (This would require a Comment model with author relation)
	// posts, err := client.Post.
	// 	Query().
	// 	Include().Author().Done().
	// 	Execute(ctx)
	
	// Example 3: Deep nested include (3 levels)
	// If we had: User -> Posts -> Comments -> Author
	// users, err := client.User.
	// 	Query().
	// 	Include().Posts().Comments().Author().Done().
	// 	Execute(ctx)
	
	// Example 4: Multiple includes at same level
	// users, err := client.User.
	// 	Query().
	// 	Include().Posts().Done().
	// 	Include().Profile().Done().
	// 	Execute(ctx)
	
	// Example 5: Combining includes with filters
	fmt.Println("\n=== Example 5: Include with Filters ===")
	filteredUsers, err := client.User.
		Query().
		Where().EmailContains("example.com").Query().
		Include().Posts().Done().
		Limit(10).
		Execute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	
	for _, user := range filteredUsers {
		fmt.Printf("User: %s (Posts: %d)\n", user.Email, len(user.Posts))
	}
	
	// Example 6: Include with ordering
	fmt.Println("\n=== Example 6: Include with Ordering ===")
	orderedUsers, err := client.User.
		Query().
		Include().Posts().Done().
		OrderByEmailAsc().
		Execute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	
	for _, user := range orderedUsers {
		fmt.Printf("User: %s\n", user.Email)
	}
}

