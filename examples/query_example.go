package main

import (
	"context"
	"fmt"
	"log"

	"github.com/satishbabariya/prisma-go/examples/basic/generated"
)

func main() {
	// This example shows the full query builder API
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

	// Example 1: Simple WHERE clause
	users, err := client.User.Where().
		EmailEquals("user@example.com").
		Execute(ctx)
	if err != nil {
		log.Printf("Error: %v", err)
	}
	fmt.Printf("Found %d users\n", len(users))

	// Example 2: Full query builder with WHERE, ORDER BY, LIMIT, OFFSET
	users2, err := client.User.Query().
		Where().
		EmailContains("@example.com").
		NameIsNotNull().
		OrderByEmailAsc().
		Limit(10).
		Offset(0).
		Execute(ctx)
	if err != nil {
		log.Printf("Error: %v", err)
	}
	fmt.Printf("Found %d users (paginated)\n", len(users2))

	// Example 3: ORDER BY with WHERE
	users3, err := client.User.Query().
		Where().
		IdGreaterThan(10).
		OrderByIdDesc().
		Limit(5).
		Execute(ctx)
	if err != nil {
		log.Printf("Error: %v", err)
	}
	fmt.Printf("Found %d users (ordered)\n", len(users3))

	// Example 4: Find first with ordering
	user, err := client.User.Query().
		Where().
		EmailContains("@example.com").
		OrderByEmailAsc().
		ExecuteFirst(ctx)
	if err != nil {
		log.Printf("Error: %v", err)
	}
	if user != nil {
		fmt.Printf("First user: %+v\n", user)
	}

	// Example 5: Pagination
	page := 0
	pageSize := 10
	for {
		users, err := client.User.Query().
			OrderByIdAsc().
			Limit(pageSize).
			Offset(page * pageSize).
			Execute(ctx)
		if err != nil {
			log.Printf("Error: %v", err)
			break
		}
		if len(users) == 0 {
			break
		}
		fmt.Printf("Page %d: %d users\n", page, len(users))
		page++
	}
}

