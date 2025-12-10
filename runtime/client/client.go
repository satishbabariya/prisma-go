// Package client provides the runtime client for Prisma Go.
package client

import (
	"context"
	"database/sql"
)

// PrismaClient is the main database client
type PrismaClient struct {
	db       *sql.DB
	provider string
}

// NewPrismaClient creates a new Prisma client
func NewPrismaClient(provider string, connectionString string) (*PrismaClient, error) {
	db, err := sql.Open(provider, connectionString)
	if err != nil {
		return nil, err
	}

	return &PrismaClient{
		db:       db,
		provider: provider,
	}, nil
}

// Connect establishes the database connection
func (c *PrismaClient) Connect(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// Disconnect closes the database connection
func (c *PrismaClient) Disconnect(ctx context.Context) error {
	return c.db.Close()
}

// DB returns the underlying database connection
func (c *PrismaClient) DB() *sql.DB {
	return c.db
}

