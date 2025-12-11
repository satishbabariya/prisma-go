package e2e

import (
	"context"
	"strings"
	"time"

	"github.com/stretchr/testify/require"
)

// TestAggregationQueries tests aggregation operations (count, sum, avg, min, max)
func (suite *TestSuite) TestAggregationQueries() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test tables with sample data
	suite.createAggregationTables(ctx)
	defer suite.cleanupAggregationTables(ctx)

	// Insert test data
	suite.insertAggregationTestData(ctx)

	// Test count operations
	suite.testCountOperations(ctx)

	// Test sum operations
	suite.testSumOperations(ctx)

	// Test average operations
	suite.testAverageOperations(ctx)

	// Test min/max operations
	suite.testMinMaxOperations(ctx)

	suite.T().Logf("Aggregation queries test passed for provider: %s", suite.config.Provider)
}

// createAggregationTables creates tables for aggregation tests
func (suite *TestSuite) createAggregationTables(ctx context.Context) {
	var createSQL string

	switch suite.config.Provider {
	case "postgresql":
		createSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255),
			age INTEGER,
			salary DECIMAL(10,2),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS posts (
			id SERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			published BOOLEAN DEFAULT FALSE,
			view_count INTEGER DEFAULT 0,
			author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
	case "mysql":
		createSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255),
			age INT,
			salary DECIMAL(10,2),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS posts (
			id INT AUTO_INCREMENT PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			published BOOLEAN DEFAULT FALSE,
			view_count INT DEFAULT 0,
			author_id INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
		);`
	case "sqlite":
		createSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			name TEXT,
			age INTEGER,
			salary REAL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT,
			published BOOLEAN DEFAULT FALSE,
			view_count INTEGER DEFAULT 0,
			author_id INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
		);`
	}

	// MySQL doesn't support multiple CREATE TABLE statements in one Exec
	if suite.config.Provider == "mysql" {
		statements := strings.Split(createSQL, ";")
		for _, stmt := range statements {
			trimmedStmt := strings.TrimSpace(stmt)
			if trimmedStmt != "" {
				_, err := suite.db.ExecContext(ctx, trimmedStmt)
				if err != nil {
					suite.T().Logf("Table creation error for statement '%s': %v", trimmedStmt, err)
					require.NoError(suite.T(), err)
				}
			}
		}
	} else {
		_, err := suite.db.ExecContext(ctx, createSQL)
		require.NoError(suite.T(), err)
	}
}

// cleanupAggregationTables removes aggregation test tables
func (suite *TestSuite) cleanupAggregationTables(ctx context.Context) {
	// Clean up data first, then drop tables
	// SQLite and MySQL don't support multiple tables in one DROP statement
	if suite.config.Provider == "sqlite" || suite.config.Provider == "mysql" {
		_, _ = suite.db.ExecContext(ctx, "DELETE FROM posts")
		_, _ = suite.db.ExecContext(ctx, "DELETE FROM users")
		_, err := suite.db.ExecContext(ctx, "DROP TABLE IF EXISTS posts")
		if err != nil {
			suite.T().Logf("Error dropping posts table: %v", err)
		}
		_, err = suite.db.ExecContext(ctx, "DROP TABLE IF EXISTS users")
		if err != nil {
			suite.T().Logf("Error dropping users table: %v", err)
		}
	} else {
		_, err := suite.db.ExecContext(ctx, "DROP TABLE IF EXISTS posts, users")
		if err != nil {
			suite.T().Logf("Error dropping tables: %v", err)
		}
	}
}

// insertAggregationTestData inserts sample data for aggregation tests
func (suite *TestSuite) insertAggregationTestData(ctx context.Context) {
	// Insert users with different ages and salaries
	users := []struct {
		Email  string
		Name   string
		Age    int
		Salary float64
	}{
		{"user1@example.com", "User One", 25, 50000.00},
		{"user2@example.com", "User Two", 30, 60000.00},
		{"user3@example.com", "User Three", 35, 75000.00},
		{"user4@example.com", "User Four", 28, 55000.00},
		{"user5@example.com", "User Five", 32, 65000.00},
	}

	for _, user := range users {
		var userID int
		var err error
		if suite.config.Provider == "postgresql" || suite.config.Provider == "postgres" {
			err = suite.db.QueryRowContext(ctx,
				suite.convertPlaceholders("INSERT INTO users (email, name, age, salary) VALUES (?, ?, ?, ?) RETURNING id"),
				user.Email, user.Name, user.Age, user.Salary).Scan(&userID)
			require.NoError(suite.T(), err)
		} else {
			// MySQL and SQLite don't support RETURNING
			result, err := suite.db.ExecContext(ctx,
				suite.convertPlaceholders("INSERT INTO users (email, name, age, salary) VALUES (?, ?, ?, ?)"),
				user.Email, user.Name, user.Age, user.Salary)
			require.NoError(suite.T(), err)
			lastID, err := result.LastInsertId()
			require.NoError(suite.T(), err)
			userID = int(lastID)
		}

		// Insert posts for each user
		posts := []struct {
			Title     string
			Published bool
			ViewCount int
		}{
			{"First Post", true, 100},
			{"Second Post", false, 50},
			{"Third Post", true, 200},
		}

		for _, post := range posts {
			_, err := suite.db.ExecContext(ctx,
				suite.convertPlaceholders("INSERT INTO posts (title, published, view_count, author_id) VALUES (?, ?, ?, ?)"),
				post.Title, post.Published, post.ViewCount, userID)
			require.NoError(suite.T(), err)
		}
	}
}

// testCountOperations tests various count operations
func (suite *TestSuite) testCountOperations(ctx context.Context) {
	// Count all users
	var userCount int
	err := suite.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 5, userCount)

	// Count users with specific condition
	var adultCount int
	err = suite.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE age >= 30").Scan(&adultCount)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 3, adultCount)

	// Count published posts
	var publishedCount int
	err = suite.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM posts WHERE published = TRUE").Scan(&publishedCount)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 10, publishedCount) // 5 users * 2 published posts each

	// Count distinct values
	var distinctAgeCount int
	err = suite.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT age) FROM users").Scan(&distinctAgeCount)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 5, distinctAgeCount) // All users have different ages
}

// testSumOperations tests sum aggregation
func (suite *TestSuite) testSumOperations(ctx context.Context) {
	// Sum of all ages
	var totalAge int
	err := suite.db.QueryRowContext(ctx, "SELECT SUM(age) FROM users").Scan(&totalAge)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 150, totalAge) // 25+30+35+28+32

	// Sum of salaries
	var totalSalary float64
	err = suite.db.QueryRowContext(ctx, "SELECT SUM(salary) FROM users").Scan(&totalSalary)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 305000.00, totalSalary) // 50000+60000+75000+55000+65000

	// Sum with condition
	var highEarnerTotal float64
	err = suite.db.QueryRowContext(ctx, "SELECT SUM(salary) FROM users WHERE salary >= 60000").Scan(&highEarnerTotal)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 200000.00, highEarnerTotal) // 60000+75000+65000

	// Sum of view counts
	// 5 users * 3 posts each = 15 posts total
	// Each user's posts: 100 + 50 + 200 = 350 views
	// Total: 5 * 350 = 1750 views
	var totalViews int
	err = suite.db.QueryRowContext(ctx, "SELECT SUM(view_count) FROM posts").Scan(&totalViews)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1750, totalViews) // 5 users * (100+50+200) = 1750
}

// testAverageOperations tests average aggregation
func (suite *TestSuite) testAverageOperations(ctx context.Context) {
	// Average age
	var avgAge float64
	err := suite.db.QueryRowContext(ctx, "SELECT AVG(age) FROM users").Scan(&avgAge)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 30.0, avgAge) // 150/5

	// Average salary
	var avgSalary float64
	err = suite.db.QueryRowContext(ctx, "SELECT AVG(salary) FROM users").Scan(&avgSalary)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 61000.00, avgSalary) // 305000/5

	// Average with condition
	var avgHighEarnerSalary float64
	err = suite.db.QueryRowContext(ctx, "SELECT AVG(salary) FROM users WHERE salary >= 60000").Scan(&avgHighEarnerSalary)
	require.NoError(suite.T(), err)
	// MySQL returns different precision, so use InDelta for floating point comparison
	require.InDelta(suite.T(), 66666.66666666667, avgHighEarnerSalary, 0.01) // 200000/3
}

// testMinMaxOperations tests min/max aggregation
func (suite *TestSuite) testMinMaxOperations(ctx context.Context) {
	// Min and max age
	var minAge, maxAge int
	err := suite.db.QueryRowContext(ctx, "SELECT MIN(age), MAX(age) FROM users").Scan(&minAge, &maxAge)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 25, minAge)
	require.Equal(suite.T(), 35, maxAge)

	// Min and max salary
	var minSalary, maxSalary float64
	err = suite.db.QueryRowContext(ctx, "SELECT MIN(salary), MAX(salary) FROM users").Scan(&minSalary, &maxSalary)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 50000.00, minSalary)
	require.Equal(suite.T(), 75000.00, maxSalary)

	// Min and max view count
	var minViews, maxViews int
	err = suite.db.QueryRowContext(ctx, "SELECT MIN(view_count), MAX(view_count) FROM posts").Scan(&minViews, &maxViews)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 50, minViews)
	require.Equal(suite.T(), 200, maxViews)

	// Combined aggregations
	var stats struct {
		UserCount  int
		AvgAge     float64
		MinSalary  float64
		MaxSalary  float64
		TotalPosts int
	}
	err = suite.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as user_count,
			AVG(age) as avg_age,
			MIN(salary) as min_salary,
			MAX(salary) as max_salary,
			(SELECT COUNT(*) FROM posts) as total_posts
		FROM users`).Scan(&stats.UserCount, &stats.AvgAge, &stats.MinSalary, &stats.MaxSalary, &stats.TotalPosts)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 5, stats.UserCount)
	require.Equal(suite.T(), 30.0, stats.AvgAge)
	require.Equal(suite.T(), 50000.00, stats.MinSalary)
	require.Equal(suite.T(), 75000.00, stats.MaxSalary)
	require.Equal(suite.T(), 15, stats.TotalPosts) // 5 users * 3 posts each
}
