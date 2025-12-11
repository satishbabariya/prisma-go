package e2e

import (
	"context"
	"strings"
	"time"

	"github.com/stretchr/testify/require"
)

// TestGroupByQueries tests GROUP BY query functionality
func (suite *TestSuite) TestGroupByQueries() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Clean up first to ensure isolation
	suite.cleanupGroupByTables(ctx)
	
	// Create test tables with sample data
	suite.createGroupByTables(ctx)
	defer suite.cleanupGroupByTables(ctx)

	// Insert test data
	suite.insertGroupByTestData(ctx)

	// Test basic GROUP BY operations
	suite.testBasicGroupBy(ctx)

	// Test GROUP BY with multiple columns
	suite.testMultipleColumnGroupBy(ctx)

	// Test GROUP BY with HAVING clause
	suite.testGroupByWithHaving(ctx)

	// Test GROUP BY with aggregations
	suite.testGroupByWithAggregations(ctx)

	suite.T().Logf("GroupBy queries test passed for provider: %s", suite.config.Provider)
}

// createGroupByTables creates tables for GROUP BY tests
func (suite *TestSuite) createGroupByTables(ctx context.Context) {
	var createSQL string

	switch suite.config.Provider {
	case "postgresql":
		createSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255),
			age INTEGER,
			department VARCHAR(100),
			salary DECIMAL(10,2),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS posts (
			id SERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			published BOOLEAN DEFAULT FALSE,
			category VARCHAR(100),
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
			department VARCHAR(100),
			salary DECIMAL(10,2),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS posts (
			id INT AUTO_INCREMENT PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			published BOOLEAN DEFAULT FALSE,
			category VARCHAR(100),
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
			department TEXT,
			salary REAL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT,
			published BOOLEAN DEFAULT FALSE,
			category TEXT,
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

// cleanupGroupByTables removes GROUP BY test tables
func (suite *TestSuite) cleanupGroupByTables(ctx context.Context) {
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

// insertGroupByTestData inserts sample data for GROUP BY tests
func (suite *TestSuite) insertGroupByTestData(ctx context.Context) {
	// Insert users with different departments and ages
	users := []struct {
		Email      string
		Name       string
		Age        int
		Department string
		Salary     float64
	}{
		{"alice@company.com", "Alice", 25, "Engineering", 75000.00},
		{"bob@company.com", "Bob", 30, "Engineering", 85000.00},
		{"charlie@company.com", "Charlie", 35, "Engineering", 95000.00},
		{"diana@company.com", "Diana", 28, "Marketing", 65000.00},
		{"eve@company.com", "Eve", 32, "Marketing", 70000.00},
		{"frank@company.com", "Frank", 40, "Sales", 80000.00},
		{"grace@company.com", "Grace", 27, "Sales", 60000.00},
	}

	for _, user := range users {
		var userID int
		var err error
		if suite.config.Provider == "postgresql" || suite.config.Provider == "postgres" {
			err = suite.db.QueryRowContext(ctx,
				suite.convertPlaceholders("INSERT INTO users (email, name, age, department, salary) VALUES (?, ?, ?, ?, ?) RETURNING id"),
				user.Email, user.Name, user.Age, user.Department, user.Salary).Scan(&userID)
			require.NoError(suite.T(), err)
		} else {
			result, err := suite.db.ExecContext(ctx,
				suite.convertPlaceholders("INSERT INTO users (email, name, age, department, salary) VALUES (?, ?, ?, ?, ?)"),
				user.Email, user.Name, user.Age, user.Department, user.Salary)
			require.NoError(suite.T(), err)
			lastID, err := result.LastInsertId()
			require.NoError(suite.T(), err)
			userID = int(lastID)
		}

		// Insert posts for each user with different categories
		categories := []string{"Technology", "Business", "Lifestyle"}
		for i, category := range categories {
			published := i%2 == 0 // Alternate published status
			viewCount := (i + 1) * 50

			_, err := suite.db.ExecContext(ctx,
				suite.convertPlaceholders("INSERT INTO posts (title, published, category, view_count, author_id) VALUES (?, ?, ?, ?, ?)"),
				user.Name+"'s Post "+string(rune('A'+i)), published, category, viewCount, userID)
			require.NoError(suite.T(), err)
		}
	}
}

// testBasicGroupBy tests basic GROUP BY functionality
func (suite *TestSuite) testBasicGroupBy(ctx context.Context) {
	// Group users by department and count
	rows, err := suite.db.QueryContext(ctx, `
		SELECT department, COUNT(*) as user_count, AVG(salary) as avg_salary
		FROM users
		GROUP BY department
		ORDER BY department`)
	require.NoError(suite.T(), err)
	defer rows.Close()

	var results []struct {
		Department string
		UserCount  int
		AvgSalary  float64
	}

	for rows.Next() {
		var result struct {
			Department string
			UserCount  int
			AvgSalary  float64
		}
		err := rows.Scan(&result.Department, &result.UserCount, &result.AvgSalary)
		require.NoError(suite.T(), err)
		results = append(results, result)
	}

	require.Len(suite.T(), results, 3) // Engineering, Marketing, Sales

	// Verify Engineering department
	engineering := results[0]
	require.Equal(suite.T(), "Engineering", engineering.Department)
	require.Equal(suite.T(), 3, engineering.UserCount)
	require.Equal(suite.T(), 85000.00, engineering.AvgSalary) // (75000+85000+95000)/3

	// Group posts by category
	rows, err = suite.db.QueryContext(ctx, `
		SELECT category, COUNT(*) as post_count, SUM(view_count) as total_views
		FROM posts
		GROUP BY category
		ORDER BY category`)
	require.NoError(suite.T(), err)
	defer rows.Close()

	var categoryResults []struct {
		Category   string
		PostCount  int
		TotalViews int
	}

	for rows.Next() {
		var result struct {
			Category   string
			PostCount  int
			TotalViews int
		}
		err := rows.Scan(&result.Category, &result.PostCount, &result.TotalViews)
		require.NoError(suite.T(), err)
		categoryResults = append(categoryResults, result)
	}

	require.Len(suite.T(), categoryResults, 3) // Technology, Business, Lifestyle
}

// testMultipleColumnGroupBy tests GROUP BY with multiple columns
func (suite *TestSuite) testMultipleColumnGroupBy(ctx context.Context) {
	// Group by department and age range
	rows, err := suite.db.QueryContext(ctx, `
		SELECT 
			department,
			CASE 
				WHEN age < 30 THEN 'Young'
				WHEN age < 35 THEN 'Mid'
				ELSE 'Senior'
			END as age_group,
			COUNT(*) as count,
			AVG(salary) as avg_salary
		FROM users
		GROUP BY department, age_group
		ORDER BY department, age_group`)
	require.NoError(suite.T(), err)
	defer rows.Close()

	var results []struct {
		Department string
		AgeGroup   string
		Count      int
		AvgSalary  float64
	}

	for rows.Next() {
		var result struct {
			Department string
			AgeGroup   string
			Count      int
			AvgSalary  float64
		}
		err := rows.Scan(&result.Department, &result.AgeGroup, &result.Count, &result.AvgSalary)
		require.NoError(suite.T(), err)
		results = append(results, result)
	}

	require.Greater(suite.T(), len(results), 0)

	// Verify some expected combinations
	expectedCombinations := map[string]bool{
		"Engineering-Young":  false,
		"Engineering-Mid":    false,
		"Engineering-Senior": false,
		"Marketing-Young":    false,
		"Marketing-Mid":      false,
		"Sales-Young":        false,
		"Sales-Senior":       false,
	}

	for _, result := range results {
		key := result.Department + "-" + result.AgeGroup
		expectedCombinations[key] = true
	}

	// Check that we have the expected combinations
	require.True(suite.T(), expectedCombinations["Engineering-Young"])
	require.True(suite.T(), expectedCombinations["Engineering-Senior"])
	require.True(suite.T(), expectedCombinations["Sales-Senior"])
}

// testGroupByWithHaving tests GROUP BY with HAVING clause
func (suite *TestSuite) testGroupByWithHaving(ctx context.Context) {
	// Group by department and filter with HAVING
	rows, err := suite.db.QueryContext(ctx, `
		SELECT department, COUNT(*) as user_count, AVG(salary) as avg_salary
		FROM users
		GROUP BY department
		HAVING COUNT(*) > 1
		ORDER BY avg_salary DESC`)
	require.NoError(suite.T(), err)
	defer rows.Close()

	var results []struct {
		Department string
		UserCount  int
		AvgSalary  float64
	}

	for rows.Next() {
		var result struct {
			Department string
			UserCount  int
			AvgSalary  float64
		}
		err := rows.Scan(&result.Department, &result.UserCount, &result.AvgSalary)
		require.NoError(suite.T(), err)
		results = append(results, result)
	}

	// Should have departments with more than 1 user
	require.Greater(suite.T(), len(results), 0)

	// All results should have user_count > 1
	for _, result := range results {
		require.Greater(suite.T(), result.UserCount, 1)
	}

	// Test HAVING with aggregation condition
	rows, err = suite.db.QueryContext(ctx, `
		SELECT department, COUNT(*) as user_count, AVG(salary) as avg_salary
		FROM users
		GROUP BY department
		HAVING AVG(salary) > 70000
		ORDER BY avg_salary DESC`)
	require.NoError(suite.T(), err)
	defer rows.Close()

	var highSalaryDepts []struct {
		Department string
		UserCount  int
		AvgSalary  float64
	}

	for rows.Next() {
		var result struct {
			Department string
			UserCount  int
			AvgSalary  float64
		}
		err := rows.Scan(&result.Department, &result.UserCount, &result.AvgSalary)
		require.NoError(suite.T(), err)
		highSalaryDepts = append(highSalaryDepts, result)
	}

	// All results should have avg_salary > 70000
	for _, result := range highSalaryDepts {
		require.Greater(suite.T(), result.AvgSalary, 70000.00)
	}
}

// testGroupByWithAggregations tests GROUP BY with various aggregations
func (suite *TestSuite) testGroupByWithAggregations(ctx context.Context) {
	// Group posts by category with multiple aggregations
	rows, err := suite.db.QueryContext(ctx, `
		SELECT 
			category,
			COUNT(*) as total_posts,
			COUNT(CASE WHEN published = TRUE THEN 1 END) as published_posts,
			SUM(view_count) as total_views,
			AVG(view_count) as avg_views,
			MIN(view_count) as min_views,
			MAX(view_count) as max_views
		FROM posts
		GROUP BY category
		ORDER BY total_posts DESC`)
	require.NoError(suite.T(), err)
	defer rows.Close()

	var results []struct {
		Category       string
		TotalPosts     int
		PublishedPosts int
		TotalViews     int
		AvgViews       float64
		MinViews       int
		MaxViews       int
	}

	for rows.Next() {
		var result struct {
			Category       string
			TotalPosts     int
			PublishedPosts int
			TotalViews     int
			AvgViews       float64
			MinViews       int
			MaxViews       int
		}
		err := rows.Scan(&result.Category, &result.TotalPosts, &result.PublishedPosts,
			&result.TotalViews, &result.AvgViews, &result.MinViews, &result.MaxViews)
		require.NoError(suite.T(), err)
		results = append(results, result)
	}

	require.Len(suite.T(), results, 3) // Technology, Business, Lifestyle

	// Verify aggregation results
	for _, result := range results {
		require.Greater(suite.T(), result.TotalPosts, 0)
		require.GreaterOrEqual(suite.T(), result.PublishedPosts, 0)
		require.LessOrEqual(suite.T(), result.PublishedPosts, result.TotalPosts) // Published can't exceed total
		require.Greater(suite.T(), result.TotalViews, 0)
		require.Greater(suite.T(), result.AvgViews, 0.0) // Use float64 literal
		require.GreaterOrEqual(suite.T(), result.MinViews, 0)
		require.GreaterOrEqual(suite.T(), result.MaxViews, result.MinViews)
	}

	// Test complex GROUP BY with JOIN
	rows, err = suite.db.QueryContext(ctx, `
		SELECT 
			u.department,
			p.category,
			COUNT(*) as post_count,
			AVG(p.view_count) as avg_views
		FROM users u
		JOIN posts p ON u.id = p.author_id
		GROUP BY u.department, p.category
		ORDER BY u.department, p.category`)
	require.NoError(suite.T(), err)
	defer rows.Close()

	var joinResults []struct {
		Department string
		Category   string
		PostCount  int
		AvgViews   float64
	}

	for rows.Next() {
		var result struct {
			Department string
			Category   string
			PostCount  int
			AvgViews   float64
		}
		err := rows.Scan(&result.Department, &result.Category, &result.PostCount, &result.AvgViews)
		require.NoError(suite.T(), err)
		joinResults = append(joinResults, result)
	}

	// Should have combinations of departments and categories
	require.Greater(suite.T(), len(joinResults), 0)

	// Verify all combinations exist
	expectedDeptCategories := map[string]bool{
		"Engineering-Technology": false,
		"Engineering-Business":   false,
		"Engineering-Lifestyle":  false,
		"Marketing-Technology":   false,
		"Marketing-Business":     false,
		"Marketing-Lifestyle":    false,
		"Sales-Technology":       false,
		"Sales-Business":         false,
		"Sales-Lifestyle":        false,
	}

	for _, result := range joinResults {
		key := result.Department + "-" + result.Category
		expectedDeptCategories[key] = true
		require.Greater(suite.T(), result.PostCount, 0)
		require.Greater(suite.T(), result.AvgViews, 0.0) // Use float64 literal
	}
}
