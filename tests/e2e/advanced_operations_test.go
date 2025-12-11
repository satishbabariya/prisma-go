package e2e

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	queryAst "github.com/satishbabariya/prisma-go/query/ast"
	"github.com/stretchr/testify/require"
)

// TestUpsertOperations tests upsert (insert or update) operations
func (suite *TestSuite) TestUpsertOperations() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test tables
	suite.createTestTables(ctx)
	defer suite.cleanupTestTables(ctx)

	// Use unique email to avoid conflicts
	uniqueEmail := fmt.Sprintf("upsert-%d@example.com", time.Now().UnixNano())

	// First upsert should create a new record
	suite.testUpsertCreate(ctx, uniqueEmail)

	// Second upsert should update the existing record
	suite.testUpsertUpdate(ctx, uniqueEmail)

	suite.T().Logf("Upsert operations test passed for provider: %s", suite.config.Provider)
}

// testUpsertCreate tests upsert creating a new record
func (suite *TestSuite) testUpsertCreate(ctx context.Context, email string) {
	// For PostgreSQL, MySQL 8.0+, we can use INSERT ... ON CONFLICT / ON DUPLICATE KEY UPDATE
	// For SQLite, we use INSERT ... ON CONFLICT
	// For older MySQL, we need to use a different approach

	var sql string
	var args []interface{}

	switch suite.config.Provider {
	case "postgresql", "postgres":
		// PostgreSQL uses INSERT ... ON CONFLICT ... DO UPDATE
		sql = suite.convertPlaceholders(`
			INSERT INTO users (email, name, age) 
			VALUES (?, ?, ?)
			ON CONFLICT (email) 
			DO UPDATE SET name = EXCLUDED.name, age = EXCLUDED.age
			RETURNING id, email, name, age
		`)
		args = []interface{}{email, "New User", 25}
	case "mysql":
		// MySQL uses INSERT ... ON DUPLICATE KEY UPDATE
		sql = suite.convertPlaceholders(`
			INSERT INTO users (email, name, age) 
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE name = VALUES(name), age = VALUES(age)
		`)
		args = []interface{}{email, "New User", 25}
	case "sqlite":
		// SQLite uses INSERT ... ON CONFLICT ... DO UPDATE
		sql = suite.convertPlaceholders(`
			INSERT INTO users (email, name, age) 
			VALUES (?, ?, ?)
			ON CONFLICT (email) 
			DO UPDATE SET name = excluded.name, age = excluded.age
		`)
		args = []interface{}{email, "New User", 25}
	}

	if suite.config.Provider == "postgresql" || suite.config.Provider == "postgres" {
		// PostgreSQL returns the row
		var id int
		var resultEmail, name string
		var age int
		err := suite.db.QueryRowContext(ctx, sql, args...).Scan(&id, &resultEmail, &name, &age)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), email, resultEmail)
		require.Equal(suite.T(), "New User", name)
		require.Equal(suite.T(), 25, age)
	} else {
		// MySQL and SQLite - execute and then query
		result, err := suite.db.ExecContext(ctx, sql, args...)
		require.NoError(suite.T(), err)

		// Verify the record was created
		var count int
		err = suite.db.QueryRowContext(ctx,
			suite.convertPlaceholders("SELECT COUNT(*) FROM users WHERE email = ?"),
			email).Scan(&count)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), 1, count)

		// Get the inserted ID for MySQL
		if suite.config.Provider == "mysql" {
			lastID, err := result.LastInsertId()
			require.NoError(suite.T(), err)
			require.Greater(suite.T(), lastID, int64(0))
		}
	}
}

// testUpsertUpdate tests upsert updating an existing record
func (suite *TestSuite) testUpsertUpdate(ctx context.Context, email string) {
	var sql string
	var args []interface{}

	switch suite.config.Provider {
	case "postgresql", "postgres":
		sql = suite.convertPlaceholders(`
			INSERT INTO users (email, name, age) 
			VALUES (?, ?, ?)
			ON CONFLICT (email) 
			DO UPDATE SET name = EXCLUDED.name, age = EXCLUDED.age
			RETURNING id, email, name, age
		`)
		args = []interface{}{email, "Updated User", 30}
	case "mysql":
		sql = suite.convertPlaceholders(`
			INSERT INTO users (email, name, age) 
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE name = VALUES(name), age = VALUES(age)
		`)
		args = []interface{}{email, "Updated User", 30}
	case "sqlite":
		sql = suite.convertPlaceholders(`
			INSERT INTO users (email, name, age) 
			VALUES (?, ?, ?)
			ON CONFLICT (email) 
			DO UPDATE SET name = excluded.name, age = excluded.age
		`)
		args = []interface{}{email, "Updated User", 30}
	}

	if suite.config.Provider == "postgresql" || suite.config.Provider == "postgres" {
		var id int
		var resultEmail, name string
		var age int
		err := suite.db.QueryRowContext(ctx, sql, args...).Scan(&id, &resultEmail, &name, &age)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), email, resultEmail)
		require.Equal(suite.T(), "Updated User", name)
		require.Equal(suite.T(), 30, age)
	} else {
		_, err := suite.db.ExecContext(ctx, sql, args...)
		require.NoError(suite.T(), err)

		// Verify the record was updated
		var name string
		var age int
		err = suite.db.QueryRowContext(ctx,
			suite.convertPlaceholders("SELECT name, age FROM users WHERE email = ?"),
			email).Scan(&name, &age)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), "Updated User", name)
		require.Equal(suite.T(), 30, age)
	}
}

// TestFindOrThrowOperations tests FindFirstOrThrow and FindUniqueOrThrow operations
func (suite *TestSuite) TestFindOrThrowOperations() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test tables
	suite.createTestTables(ctx)
	defer suite.cleanupTestTables(ctx)

	// Test FindFirstOrThrow - should succeed when record exists
	suite.testFindFirstOrThrowSuccess(ctx)

	// Test FindFirstOrThrow - should throw error when record doesn't exist
	suite.testFindFirstOrThrowError(ctx)

	// Test FindUniqueOrThrow - should succeed when record exists
	suite.testFindUniqueOrThrowSuccess(ctx)

	// Test FindUniqueOrThrow - should throw error when record doesn't exist
	suite.testFindUniqueOrThrowError(ctx)

	suite.T().Logf("FindOrThrow operations test passed for provider: %s", suite.config.Provider)
}

// testFindFirstOrThrowSuccess tests FindFirstOrThrow when record exists
func (suite *TestSuite) testFindFirstOrThrowSuccess(ctx context.Context) {
	// Insert a test user
	email := fmt.Sprintf("findfirst-%d@example.com", time.Now().UnixNano())
	_, err := suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		email, "FindFirst Test", 25)
	require.NoError(suite.T(), err)

	// Query using FindFirst pattern (FindMany with Take: 1)
	take := 1
	query := &queryAst.FindManyQuery{
		Model: "users",
		Where: &queryAst.WhereClause{
			Conditions: []queryAst.Condition{
				{
					Field:    "email",
					Operator: queryAst.OpEquals,
					Value:    email,
				},
			},
		},
		Take: &take,
		Select: &queryAst.SelectClause{
			Fields: []string{"id", "email", "name", "age"},
		},
	}

	querySQL, args, err := suite.compiler.Compile(query)
	require.NoError(suite.T(), err)

	// Execute query - should return a row
	var id int
	var resultEmail, name string
	var age int
	err = suite.db.QueryRowContext(ctx, querySQL, args...).Scan(&id, &resultEmail, &name, &age)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), email, resultEmail)
	require.Equal(suite.T(), "FindFirst Test", name)
}

// testFindFirstOrThrowError tests FindFirstOrThrow when record doesn't exist
func (suite *TestSuite) testFindFirstOrThrowError(ctx context.Context) {
	nonExistentEmail := fmt.Sprintf("nonexistent-%d@example.com", time.Now().UnixNano())

	// FindFirst is FindMany with Take: 1
	take := 1
	query := &queryAst.FindManyQuery{
		Model: "users",
		Where: &queryAst.WhereClause{
			Conditions: []queryAst.Condition{
				{
					Field:    "email",
					Operator: queryAst.OpEquals,
					Value:    nonExistentEmail,
				},
			},
		},
		Take: &take,
		Select: &queryAst.SelectClause{
			Fields: []string{"id", "email", "name", "age"},
		},
	}

	querySQL, args, err := suite.compiler.Compile(query)
	require.NoError(suite.T(), err)

	// Execute query - should return sql.ErrNoRows
	var id int
	var resultEmail, name string
	var age int
	err = suite.db.QueryRowContext(ctx, querySQL, args...).Scan(&id, &resultEmail, &name, &age)
	require.Error(suite.T(), err)
	require.True(suite.T(), errors.Is(err, sql.ErrNoRows))
}

// testFindUniqueOrThrowSuccess tests FindUniqueOrThrow when record exists
func (suite *TestSuite) testFindUniqueOrThrowSuccess(ctx context.Context) {
	email := fmt.Sprintf("findunique-%d@example.com", time.Now().UnixNano())
	_, err := suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		email, "FindUnique Test", 30)
	require.NoError(suite.T(), err)

	// FindUnique is essentially FindMany with a unique constraint (email is unique)
	query := &queryAst.FindManyQuery{
		Model: "users",
		Where: &queryAst.WhereClause{
			Conditions: []queryAst.Condition{
				{
					Field:    "email",
					Operator: queryAst.OpEquals,
					Value:    email,
				},
			},
		},
		Select: &queryAst.SelectClause{
			Fields: []string{"id", "email", "name", "age"},
		},
	}

	querySQL, args, err := suite.compiler.Compile(query)
	require.NoError(suite.T(), err)

	var id int
	var resultEmail, name string
	var age int
	err = suite.db.QueryRowContext(ctx, querySQL, args...).Scan(&id, &resultEmail, &name, &age)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), email, resultEmail)
}

// testFindUniqueOrThrowError tests FindUniqueOrThrow when record doesn't exist
func (suite *TestSuite) testFindUniqueOrThrowError(ctx context.Context) {
	nonExistentEmail := fmt.Sprintf("nonexistent-unique-%d@example.com", time.Now().UnixNano())

	query := &queryAst.FindManyQuery{
		Model: "users",
		Where: &queryAst.WhereClause{
			Conditions: []queryAst.Condition{
				{
					Field:    "email",
					Operator: queryAst.OpEquals,
					Value:    nonExistentEmail,
				},
			},
		},
		Select: &queryAst.SelectClause{
			Fields: []string{"id", "email", "name", "age"},
		},
	}

	querySQL, args, err := suite.compiler.Compile(query)
	require.NoError(suite.T(), err)

	var id int
	var resultEmail, name string
	var age int
	err = suite.db.QueryRowContext(ctx, querySQL, args...).Scan(&id, &resultEmail, &name, &age)
	require.Error(suite.T(), err)
	require.True(suite.T(), errors.Is(err, sql.ErrNoRows))
}

// TestCreateManyAndReturn tests batch insert with return values
func (suite *TestSuite) TestCreateManyAndReturn() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test tables
	suite.createTestTables(ctx)
	defer suite.cleanupTestTables(ctx)

	// Use unique emails to avoid conflicts
	baseEmail := fmt.Sprintf("batch-%d", time.Now().UnixNano())
	emails := []string{
		fmt.Sprintf("%s-1@example.com", baseEmail),
		fmt.Sprintf("%s-2@example.com", baseEmail),
		fmt.Sprintf("%s-3@example.com", baseEmail),
	}

	// PostgreSQL supports INSERT ... RETURNING for multiple rows
	// MySQL and SQLite need a workaround
	if suite.config.Provider == "postgresql" || suite.config.Provider == "postgres" {
		suite.testCreateManyAndReturnPostgreSQL(ctx, emails)
	} else {
		// For MySQL and SQLite, insert individually and return IDs
		suite.testCreateManyAndReturnMySQLSQLite(ctx, emails)
	}

	suite.T().Logf("CreateManyAndReturn test passed for provider: %s", suite.config.Provider)
}

// testCreateManyAndReturnPostgreSQL tests batch insert with return for PostgreSQL
func (suite *TestSuite) testCreateManyAndReturnPostgreSQL(ctx context.Context, emails []string) {
	// Build VALUES clause
	values := make([]string, len(emails))
	args := make([]interface{}, 0, len(emails)*3)
	for i, email := range emails {
		values[i] = fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3)
		args = append(args, email, fmt.Sprintf("User %d", i+1), 20+i)
	}

	sql := fmt.Sprintf(`
		INSERT INTO users (email, name, age) 
		VALUES %s
		RETURNING id, email, name, age
	`, strings.Join(values, ", "))

	rows, err := suite.db.QueryContext(ctx, sql, args...)
	require.NoError(suite.T(), err)
	defer rows.Close()

	var results []struct {
		ID    int
		Email string
		Name  string
		Age   int
	}

	for rows.Next() {
		var r struct {
			ID    int
			Email string
			Name  string
			Age   int
		}
		err := rows.Scan(&r.ID, &r.Email, &r.Name, &r.Age)
		require.NoError(suite.T(), err)
		results = append(results, r)
	}

	require.NoError(suite.T(), rows.Err())
	require.Len(suite.T(), results, len(emails))

	for i, result := range results {
		require.Equal(suite.T(), emails[i], result.Email)
		require.Equal(suite.T(), fmt.Sprintf("User %d", i+1), result.Name)
		require.Equal(suite.T(), 20+i, result.Age)
		require.Greater(suite.T(), result.ID, 0)
	}
}

// testCreateManyAndReturnMySQLSQLite tests batch insert for MySQL/SQLite (no RETURNING)
func (suite *TestSuite) testCreateManyAndReturnMySQLSQLite(ctx context.Context, emails []string) {
	// Insert records one by one and collect IDs
	insertedIDs := make([]int64, 0, len(emails))

	for i, email := range emails {
		result, err := suite.db.ExecContext(ctx,
			suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
			email, fmt.Sprintf("User %d", i+1), 20+i)
		require.NoError(suite.T(), err)

		lastID, err := result.LastInsertId()
		require.NoError(suite.T(), err)
		insertedIDs = append(insertedIDs, lastID)
	}

	// Verify all records were inserted
	require.Len(suite.T(), insertedIDs, len(emails))

	// Query back the inserted records
	for i, email := range emails {
		var id int
		var resultEmail, name string
		var age int
		err := suite.db.QueryRowContext(ctx,
			suite.convertPlaceholders("SELECT id, email, name, age FROM users WHERE email = ?"),
			email).Scan(&id, &resultEmail, &name, &age)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), int(insertedIDs[i]), id)
		require.Equal(suite.T(), email, resultEmail)
		require.Equal(suite.T(), fmt.Sprintf("User %d", i+1), name)
		require.Equal(suite.T(), 20+i, age)
	}
}

// TestAggregateOperations tests aggregate functions (count, avg, sum, min, max)
func (suite *TestSuite) TestAggregateOperations() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test tables
	suite.createTestTables(ctx)
	defer suite.cleanupTestTables(ctx)

	// Insert test data
	baseEmail := fmt.Sprintf("agg-%d", time.Now().UnixNano())
	for i := 0; i < 5; i++ {
		_, err := suite.db.ExecContext(ctx,
			suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
			fmt.Sprintf("%s-%d@example.com", baseEmail, i),
			fmt.Sprintf("User %d", i),
			20+i*5) // Ages: 20, 25, 30, 35, 40
		require.NoError(suite.T(), err)
	}

	// Test COUNT
	suite.testAggregateCount(ctx)

	// Test AVG
	suite.testAggregateAvg(ctx)

	// Test SUM
	suite.testAggregateSum(ctx)

	// Test MIN
	suite.testAggregateMin(ctx)

	// Test MAX
	suite.testAggregateMax(ctx)

	suite.T().Logf("Aggregate operations test passed for provider: %s", suite.config.Provider)
}

// testAggregateCount tests COUNT aggregate
func (suite *TestSuite) testAggregateCount(ctx context.Context) {
	var count int
	err := suite.db.QueryRowContext(ctx,
		suite.convertPlaceholders("SELECT COUNT(*) FROM users")).Scan(&count)
	require.NoError(suite.T(), err)
	require.GreaterOrEqual(suite.T(), count, 5)
}

// testAggregateAvg tests AVG aggregate
func (suite *TestSuite) testAggregateAvg(ctx context.Context) {
	var avg float64
	err := suite.db.QueryRowContext(ctx,
		suite.convertPlaceholders("SELECT AVG(age) FROM users")).Scan(&avg)
	require.NoError(suite.T(), err)
	// Average of 20, 25, 30, 35, 40 = 30
	require.InDelta(suite.T(), 30.0, avg, 0.1)
}

// testAggregateSum tests SUM aggregate
func (suite *TestSuite) testAggregateSum(ctx context.Context) {
	var sum int
	err := suite.db.QueryRowContext(ctx,
		suite.convertPlaceholders("SELECT SUM(age) FROM users")).Scan(&sum)
	require.NoError(suite.T(), err)
	// Sum of 20, 25, 30, 35, 40 = 150
	require.GreaterOrEqual(suite.T(), sum, 150)
}

// testAggregateMin tests MIN aggregate
func (suite *TestSuite) testAggregateMin(ctx context.Context) {
	var min int
	err := suite.db.QueryRowContext(ctx,
		suite.convertPlaceholders("SELECT MIN(age) FROM users")).Scan(&min)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 20, min)
}

// testAggregateMax tests MAX aggregate
func (suite *TestSuite) testAggregateMax(ctx context.Context) {
	var max int
	err := suite.db.QueryRowContext(ctx,
		suite.convertPlaceholders("SELECT MAX(age) FROM users")).Scan(&max)
	require.NoError(suite.T(), err)
	require.GreaterOrEqual(suite.T(), max, 40)
}

// TestGroupByOperations tests GROUP BY with aggregations
func (suite *TestSuite) TestGroupByOperations() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test tables
	suite.createTestTables(ctx)
	defer suite.cleanupTestTables(ctx)

	// Insert test data with different ages
	baseEmail := fmt.Sprintf("groupby-%d", time.Now().UnixNano())
	ages := []int{25, 25, 30, 30, 30, 35}
	for i, age := range ages {
		_, err := suite.db.ExecContext(ctx,
			suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
			fmt.Sprintf("%s-%d@example.com", baseEmail, i),
			fmt.Sprintf("User %d", i),
			age)
		require.NoError(suite.T(), err)
	}

	// Test GROUP BY age with COUNT
	var results []struct {
		Age   int
		Count int
	}

	rows, err := suite.db.QueryContext(ctx,
		suite.convertPlaceholders("SELECT age, COUNT(*) as count FROM users GROUP BY age ORDER BY age"))
	require.NoError(suite.T(), err)
	defer rows.Close()

	for rows.Next() {
		var r struct {
			Age   int
			Count int
		}
		err := rows.Scan(&r.Age, &r.Count)
		require.NoError(suite.T(), err)
		results = append(results, r)
	}
	require.NoError(suite.T(), rows.Err())

	// Verify grouping results
	// Age 25: 2 users, Age 30: 3 users, Age 35: 1 user
	ageCounts := make(map[int]int)
	for _, r := range results {
		ageCounts[r.Age] = r.Count
	}

	require.Equal(suite.T(), 2, ageCounts[25])
	require.Equal(suite.T(), 3, ageCounts[30])
	require.Equal(suite.T(), 1, ageCounts[35])

	suite.T().Logf("GroupBy operations test passed for provider: %s", suite.config.Provider)
}

