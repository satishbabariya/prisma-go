package main

import (
	"context"
	"fmt"
	"log"

	"github.com/yourproject/examples/basic/generated"
)

func main() {
	// Create a new Prisma client
	client, err := generated.NewPrismaClient("postgresql://localhost:5432/myapp")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Connect to database
	if err := client.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	fmt.Println("✅ Connected to database!")

	// Example operations (will be implemented when query execution is ready)
	
	// Find all users
	users, err := client.User.FindMany(ctx)
	if err != nil {
		log.Printf("Error finding users: %v", err)
	}
	fmt.Printf("Found %d users\n", len(users))

	// Create a user
	newUser := generated.User{
		Email: "user@example.com",
		Name:  stringPtr("John Doe"),
	}
	created, err := client.User.Create(ctx, newUser)
	if err != nil {
		log.Printf("Error creating user: %v", err)
	} else {
		fmt.Printf("Created user: %+v\n", created)
	}
}

func stringPtr(s string) *string {
	return &s
}

