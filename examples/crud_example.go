package main

import (
	"context"
	"fmt"
	"log"

	"github.com/satishbabariya/prisma-go/examples/basic/generated"
)

func main() {
	// This example demonstrates full CRUD operations
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

	// CREATE - Create a new user
	newUser := generated.User{
		Email: "newuser@example.com",
		Name:  stringPtr("New User"),
	}
	created, err := client.User.Create(ctx, newUser)
	if err != nil {
		log.Printf("Error creating user: %v", err)
	} else {
		fmt.Printf("Created user: %+v\n", created)
	}

	// READ - Find users
	users, err := client.User.FindMany(ctx)
	if err != nil {
		log.Printf("Error finding users: %v", err)
	} else {
		fmt.Printf("Found %d users\n", len(users))
	}

	// READ - Find user by email
	user, err := client.User.Where().
		EmailEquals("newuser@example.com").
		ExecuteFirst(ctx)
	if err != nil {
		log.Printf("Error finding user: %v", err)
	} else if user != nil {
		fmt.Printf("Found user: %+v\n", user)
	}

	// UPDATE - Update user
	if user != nil {
		updated, err := client.User.Update().
			SetEmail("updated@example.com").
			SetName(stringPtr("Updated User")).
			Where().
			IdEquals(user.Id).
			Execute(ctx)
		if err != nil {
			log.Printf("Error updating user: %v", err)
		} else {
			fmt.Printf("Updated user: %+v\n", updated)
		}
	}

	// DELETE - Delete user
	if user != nil {
		err := client.User.Delete().
			IdEquals(user.Id).
			Execute(ctx)
		if err != nil {
			log.Printf("Error deleting user: %v", err)
		} else {
			fmt.Println("User deleted successfully")
		}
	}

	// Complex UPDATE with WHERE
	_, err = client.User.Update().
		SetName(stringPtr("Bulk Updated")).
		Where().
		EmailContains("@example.com").
		Execute(ctx)
	if err != nil {
		log.Printf("Error bulk updating: %v", err)
	}

	// Complex DELETE with WHERE
	err = client.User.Delete().
		EmailContains("@old.com").
		Execute(ctx)
	if err != nil {
		log.Printf("Error bulk deleting: %v", err)
	}
}

func stringPtr(s string) *string {
	return &s
}

