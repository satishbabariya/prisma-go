package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	
	"github.com/satishbabariya/prisma-go/generated"
	"github.com/satishbabariya/prisma-go/query/executor"
)

func main() {
	// Example demonstrating transaction support
	
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
	
	// Example 1: Simple transaction
	fmt.Println("=== Example 1: Simple Transaction ===")
	err = client.Transaction(ctx, func(tx *sql.Tx) error {
		// Create transaction executor
		txExec := executor.NewTxExecutor(tx, "postgresql")
		
		// Create a user within the transaction
		newUser := generated.User{
			Email: "transaction@example.com",
			Name:  strPtr("Transaction User"),
		}
		
		_, err := txExec.Create(ctx, "user", newUser)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		
		fmt.Println("User created in transaction")
		
		// If we return an error here, the transaction will be rolled back
		// return fmt.Errorf("simulated error - will rollback")
		
		return nil // Commit transaction
	})
	
	if err != nil {
		fmt.Printf("Transaction failed: %v\n", err)
	} else {
		fmt.Println("Transaction committed successfully")
	}
	
	// Example 2: Transaction with rollback
	fmt.Println("\n=== Example 2: Transaction with Rollback ===")
	err = client.Transaction(ctx, func(tx *sql.Tx) error {
		txExec := executor.NewTxExecutor(tx, "postgresql")
		
		// Create first user
		user1 := generated.User{
			Email: "user1@example.com",
			Name:  strPtr("User 1"),
		}
		_, err := txExec.Create(ctx, "user", user1)
		if err != nil {
			return err
		}
		fmt.Println("Created user 1")
		
		// Create second user
		user2 := generated.User{
			Email: "user2@example.com",
			Name:  strPtr("User 2"),
		}
		_, err = txExec.Create(ctx, "user", user2)
		if err != nil {
			return err
		}
		fmt.Println("Created user 2")
		
		// Simulate an error - this will rollback both inserts
		return fmt.Errorf("simulated error - rolling back both users")
	})
	
	if err != nil {
		fmt.Printf("Transaction rolled back: %v\n", err)
	}
	
	// Example 3: Read-only transaction
	fmt.Println("\n=== Example 3: Read-Only Transaction ===")
	err = client.ReadOnlyTransaction(ctx, func(tx *sql.Tx) error {
		txExec := executor.NewTxExecutor(tx, "postgresql")
		
		// Count users
		count, err := txExec.Count(ctx, "user", nil)
		if err != nil {
			return err
		}
		fmt.Printf("Total users: %d\n", count)
		
		// Any write operations would fail in a read-only transaction
		// user := generated.User{Email: "test@example.com"}
		// _, err = txExec.Create(ctx, "user", user) // This would fail
		
		return nil
	})
	
	if err != nil {
		fmt.Printf("Read-only transaction failed: %v\n", err)
	}
	
	// Example 4: Complex transaction with multiple operations
	fmt.Println("\n=== Example 4: Complex Transaction ===")
	err = client.Transaction(ctx, func(tx *sql.Tx) error {
		txExec := executor.NewTxExecutor(tx, "postgresql")
		
		// Create a user
		newUser := generated.User{
			Email: "complex@example.com",
			Name:  strPtr("Complex User"),
		}
		createdUser, err := txExec.Create(ctx, "user", newUser)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		fmt.Println("Created user")
		
		// Update the user
		updates := map[string]interface{}{
			"name": "Updated Complex User",
		}
		err = txExec.Update(ctx, "user", updates, nil, createdUser)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
		fmt.Println("Updated user")
		
		// Count users
		count, err := txExec.Count(ctx, "user", nil)
		if err != nil {
			return fmt.Errorf("failed to count users: %w", err)
		}
		fmt.Printf("Total users after operations: %d\n", count)
		
		return nil
	})
	
	if err != nil {
		fmt.Printf("Complex transaction failed: %v\n", err)
	} else {
		fmt.Println("Complex transaction completed successfully")
	}
	
	fmt.Println("\n✅ Transaction support implemented!")
	fmt.Println("Features:")
	fmt.Println("  • Transaction() - Execute with automatic commit/rollback")
	fmt.Println("  • ReadOnlyTransaction() - Read-only transactions")
	fmt.Println("  • TransactionWithOptions() - Custom transaction options")
	fmt.Println("  • Automatic rollback on error or panic")
	fmt.Println("  • All CRUD operations work within transactions")
}

func strPtr(s string) *string {
	return &s
}

