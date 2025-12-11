package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/satishbabariya/prisma-go/migrate/diff"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
	"github.com/satishbabariya/prisma-go/migrate/sqlgen"
	"github.com/satishbabariya/prisma-go/psl/parsing"
	"github.com/stretchr/testify/require"
)

// TestMigrationExecution tests migration execution functionality
func (suite *TestSuite) TestMigrationExecution() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create a simple schema for testing
	schemaTemplate := `
	datasource db {
		provider = "%s"
		url      = "test"
	}

	generator client {
		provider = "prisma-client-go"
		output   = "./generated"
	}

	model User {
		id    Int    @id @default(autoincrement())
		email String @unique
		name  String?
		posts Post[]
	}

	model Post {
		id       Int    @id @default(autoincrement())
		title    String
		content  String?
		published Boolean @default(false)
		author   User   @relation(fields: [authorId], references: [id])
		authorId Int
	}
	`

	schemaContent := fmt.Sprintf(schemaTemplate, suite.config.Provider)

	// Parse the test schema
	_, _ = parsing.ParseSchema(schemaContent)

	// Create a simple schema for testing
	testSchema := &introspect.DatabaseSchema{
		Tables: []introspect.Table{
			{
				Name: "users",
				Columns: []introspect.Column{
					{Name: "id", Type: "SERIAL", Nullable: false},
					{Name: "email", Type: "VARCHAR(255)", Nullable: false},
					{Name: "name", Type: "VARCHAR(255)", Nullable: true},
					{Name: "age", Type: "INTEGER", Nullable: true},
				},
				PrimaryKey: &introspect.PrimaryKey{
					Columns: []string{"id"},
				},
			},
			{
				Name: "posts",
				Columns: []introspect.Column{
					{Name: "id", Type: "SERIAL", Nullable: false},
					{Name: "title", Type: "VARCHAR(255)", Nullable: false},
					{Name: "content", Type: "TEXT", Nullable: true},
					{Name: "author_id", Type: "INTEGER", Nullable: false},
				},
				PrimaryKey: &introspect.PrimaryKey{
					Columns: []string{"id"},
				},
			},
		},
	}

	// Create a diff result for testing
	diffResult := &diff.DiffResult{
		TablesToCreate: []diff.TableChange{
			{
				Name: "users",
				Changes: []diff.Change{
					{
						Type:   "AddColumn",
						Column: "id",
						ColumnMetadata: &diff.ColumnMetadata{
							Type:     "SERIAL",
							Nullable: false,
						},
					},
					{
						Type:   "AddColumn",
						Column: "email",
						ColumnMetadata: &diff.ColumnMetadata{
							Type:     "VARCHAR(255)",
							Nullable: false,
						},
					},
					{
						Type:   "AddColumn",
						Column: "name",
						ColumnMetadata: &diff.ColumnMetadata{
							Type:     "VARCHAR(255)",
							Nullable: true,
						},
					},
					{
						Type:   "AddColumn",
						Column: "age",
						ColumnMetadata: &diff.ColumnMetadata{
							Type:     "INTEGER",
							Nullable: true,
						},
					},
				},
			},
			{
				Name: "posts",
				Changes: []diff.Change{
					{
						Type:   "AddColumn",
						Column: "id",
						ColumnMetadata: &diff.ColumnMetadata{
							Type:     "SERIAL",
							Nullable: false,
						},
					},
					{
						Type:   "AddColumn",
						Column: "title",
						ColumnMetadata: &diff.ColumnMetadata{
							Type:     "VARCHAR(255)",
							Nullable: false,
						},
					},
					{
						Type:   "AddColumn",
						Column: "content",
						ColumnMetadata: &diff.ColumnMetadata{
							Type:     "TEXT",
							Nullable: true,
						},
					},
					{
						Type:   "AddColumn",
						Column: "author_id",
						ColumnMetadata: &diff.ColumnMetadata{
							Type:     "INTEGER",
							Nullable: false,
						},
					},
				},
			},
		},
	}

	// Debug: Print diff result
	suite.T().Logf("TablesToCreate: %d", len(diffResult.TablesToCreate))
	for _, change := range diffResult.TablesToCreate {
		suite.T().Logf("Table to create: %s", change.Name)
	}

	// Debug: Test SQL generation directly
	sqlGen, err := sqlgen.NewMigrationGenerator(suite.config.Provider)
	require.NoError(suite.T(), err)

	generatedSQL, err := sqlGen.GenerateMigrationSQL(diffResult, testSchema)
	require.NoError(suite.T(), err)
	suite.T().Logf("Generated SQL: %s", generatedSQL)

	// Execute SQL directly
	// MySQL doesn't support multiple statements in one ExecContext call
	if suite.config.Provider == "mysql" {
		// Split by semicolon and execute each statement separately
		statements := strings.Split(generatedSQL, ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt != "" {
				_, err := suite.db.ExecContext(ctx, stmt)
				require.NoError(suite.T(), err)
			}
		}
	} else {
		_, err = suite.db.ExecContext(ctx, generatedSQL)
		require.NoError(suite.T(), err)
	}

	// Verify tables were created
	var userTableExists bool

	switch suite.config.Provider {
	case "postgresql":
		err = suite.db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' AND table_name = 'users'
			)`).Scan(&userTableExists)
		require.NoError(suite.T(), err)

	case "mysql":
		err = suite.db.QueryRowContext(ctx, `
			SELECT COUNT(*) > 0 
			FROM information_schema.tables 
			WHERE table_schema = DATABASE() AND table_name = 'users'
		`).Scan(&userTableExists)
		require.NoError(suite.T(), err)

	case "sqlite":
		err = suite.db.QueryRowContext(ctx, `
			SELECT COUNT(*) > 0 
			FROM sqlite_master 
			WHERE type='table' AND name='users'
		`).Scan(&userTableExists)
		require.NoError(suite.T(), err)
	}

	require.True(suite.T(), userTableExists, "Users table should exist")
}
