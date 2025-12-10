package main

import (
	"context"
	"fmt"
	"log"

	"github.com/satishbabariya/prisma-go/examples/basic/generated"
)

func main() {
	// This example shows how to use WHERE clauses
	// Note: This requires a database connection

	client, err := generated.NewPrismaClient("postgresql://user:pass@localhost:5432/mydb")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Connect
	if err := client.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	// Example 1: Find all users
	allUsers, err := client.User.FindMany(ctx)
	if err != nil {
		log.Printf("Error: %v", err)
	}
	fmt.Printf("Found %d users\n", len(allUsers))

	// Example 2: Find users with WHERE clause - fluent API
	users, err := client.User.Where().
		EmailEquals("user@example.com").
		Execute(ctx)
	if err != nil {
		log.Printf("Error: %v", err)
	}
	fmt.Printf("Found %d users with email\n", len(users))

	// Example 3: Find first user with WHERE clause
	user, err := client.User.Where().
		EmailContains("@example.com").
		ExecuteFirst(ctx)
	if err != nil {
		log.Printf("Error: %v", err)
	}
	if user != nil {
		fmt.Printf("Found user: %+v\n", user)
	}

	// Example 4: Complex WHERE clause
	users2, err := client.User.Where().
		EmailEquals("user@example.com").
		NameIsNotNull().
		Execute(ctx)
	if err != nil {
		log.Printf("Error: %v", err)
	}
	fmt.Printf("Found %d users with name\n", len(users2))

	// Example 5: Numeric comparisons
	users3, err := client.User.Where().
		IdGreaterThan(10).
		IdLessThan(100).
		Execute(ctx)
	if err != nil {
		log.Printf("Error: %v", err)
	}
	fmt.Printf("Found %d users with ID between 10 and 100\n", len(users3))
}

