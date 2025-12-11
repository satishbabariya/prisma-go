package e2e

import (
	"context"
	"fmt"
	"time"

	queryAst "github.com/satishbabariya/prisma-go/query/ast"
	"github.com/stretchr/testify/require"
)

// TestComplexQueries tests complex query scenarios
func (suite *TestSuite) TestComplexQueries() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create test tables with relationships
	suite.createComplexTestTables(ctx)
	defer suite.cleanupComplexTestTables(ctx)

	// Test complex WHERE clauses
	suite.testComplexWhereClauses(ctx)

	// Test JOIN queries
	suite.testJoinQueries(ctx)

	// Test aggregations
	suite.testAggregations(ctx)

	// Test subqueries
	suite.testSubqueries(ctx)

	// Test pagination
	suite.testPagination(ctx)

	suite.T().Logf("Complex queries test passed for provider: %s", suite.config.Provider)
}

// createComplexTestTables creates tables for complex query testing
func (suite *TestSuite) createComplexTestTables(ctx context.Context) {
	var createSQL string

	switch suite.config.Provider {
	case "postgresql":
		createSQL = `CREATE TABLE IF NOT EXISTS departments (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			budget DECIMAL(12,2)
		);
		CREATE TABLE IF NOT EXISTS employees (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE,
			salary DECIMAL(10,2),
			department_id INTEGER REFERENCES departments(id) ON DELETE SET NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS projects (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			budget DECIMAL(12,2),
			start_date DATE,
			end_date DATE,
			department_id INTEGER REFERENCES departments(id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS employee_projects (
			employee_id INTEGER REFERENCES employees(id) ON DELETE CASCADE,
			project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE,
			role VARCHAR(50),
			hours_worked INTEGER DEFAULT 0,
			PRIMARY KEY (employee_id, project_id)
		);`
	case "mysql":
		createSQL = `CREATE TABLE IF NOT EXISTS departments (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			budget DECIMAL(10,2)
		);
		CREATE TABLE IF NOT EXISTS employees (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE,
			salary DECIMAL(10,2),
			department_id INT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE SET NULL
		);
		CREATE TABLE IF NOT EXISTS projects (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			budget DECIMAL(12,2),
			start_date DATE,
			end_date DATE,
			department_id INT,
			FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS employee_projects (
			employee_id INT,
			project_id INT,
			role VARCHAR(50),
			hours_worked INT DEFAULT 0,
			PRIMARY KEY (employee_id, project_id),
			FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`
	case "sqlite":
		createSQL = `CREATE TABLE IF NOT EXISTS departments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			budget REAL
		);
		CREATE TABLE IF NOT EXISTS employees (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT UNIQUE,
			salary REAL,
			department_id INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE SET NULL
		);
		CREATE TABLE IF NOT EXISTS projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			budget REAL,
			start_date TEXT,
			end_date TEXT,
			department_id INTEGER,
			FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS employee_projects (
			employee_id INTEGER,
			project_id INTEGER,
			role TEXT,
			hours_worked INTEGER DEFAULT 0,
			PRIMARY KEY (employee_id, project_id),
			FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`
	}

	// Drop tables first to ensure clean state
	var err error
	if suite.config.Provider == "sqlite" {
		// SQLite doesn't support dropping multiple tables in one statement
		tables := []string{"employee_projects", "projects", "employees", "departments"}
		for _, table := range tables {
			_, err = suite.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
			if err != nil {
				suite.T().Logf("Drop table %s error (may be expected): %v", table, err)
			}
		}
	} else {
		_, err = suite.db.ExecContext(ctx, "DROP TABLE IF EXISTS employee_projects, projects, employees, departments")
		if err != nil {
			suite.T().Logf("Drop tables error (may be expected): %v", err)
		}
	}

	suite.T().Logf("Creating tables with SQL: %s", createSQL)
	_, err = suite.db.ExecContext(ctx, createSQL)
	suite.T().Logf("Table creation error: %v", err)
	require.NoError(suite.T(), err)

	// Insert test data
	suite.insertComplexTestData(ctx)
}

// insertComplexTestData inserts test data for complex queries
func (suite *TestSuite) insertComplexTestData(ctx context.Context) {
	// Insert departments
	_, err := suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO departments (name, budget) VALUES (?, ?), (?, ?), (?, ?)"),
		"Engineering", 1000000.50,
		"Marketing", 500000.75,
		"Sales", 750000.25)
	require.NoError(suite.T(), err)

	// Insert employees
	_, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO employees (name, email, salary, department_id, created_at) VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?), (?, ?, ?, ?, ?), (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)"),
		"John Doe", "john@example.com", 80000.00, 1, time.Now().AddDate(-2, 0, 0),
		"Jane Smith", "jane@example.com", 75000.00, 1, time.Now().AddDate(-1, -6, 0),
		"Bob Johnson", "bob@example.com", 60000.00, 2, time.Now().AddDate(-1, 0, 0),
		"Alice Brown", "alice@example.com", 70000.00, 2, time.Now().AddDate(-6, -6, 0),
		"Charlie Wilson", "charlie@example.com", 90000.00, 3, time.Now().AddDate(-3, 0, 0))
	require.NoError(suite.T(), err)

	// Insert projects
	_, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO projects (name, budget, start_date, end_date, department_id) VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)"),
		"Project Alpha", 200000.00, "2023-01-01", "2023-06-30", 1,
		"Project Beta", 150000.00, "2023-03-01", "2023-09-30", 1,
		"Project Gamma", 100000.00, "2023-02-01", "2023-08-31", 2)
	require.NoError(suite.T(), err)

	// Insert employee-project relationships
	_, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO employee_projects (employee_id, project_id, role, hours_worked) VALUES (?, ?, ?, ?), (?, ?, ?, ?), (?, ?, ?, ?), (?, ?, ?, ?), (?, ?, ?, ?), (?, ?, ?, ?)"),
		1, 1, "Lead", 160,
		2, 1, "Developer", 180,
		3, 2, "Manager", 120,
		4, 2, "Designer", 140,
		5, 3, "Consultant", 80,
		1, 3, "Advisor", 40)
	require.NoError(suite.T(), err)
}

// cleanupComplexTestTables removes complex test tables
func (suite *TestSuite) cleanupComplexTestTables(ctx context.Context) {
	if suite.config.Provider == "sqlite" {
		// SQLite doesn't support dropping multiple tables in one statement
		tables := []string{"employee_projects", "projects", "employees", "departments"}
		for _, table := range tables {
			_, err := suite.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
			if err != nil {
				suite.T().Logf("Error dropping table %s: %v", table, err)
			}
		}
	} else {
		_, err := suite.db.ExecContext(ctx, "DROP TABLE IF EXISTS employee_projects, projects, employees, departments")
		require.NoError(suite.T(), err)
	}
}

// testComplexWhereClauses tests complex WHERE conditions
func (suite *TestSuite) testComplexWhereClauses(ctx context.Context) {
	// Test multiple conditions with AND/OR
	rows, err := suite.db.QueryContext(ctx, `
		SELECT name, salary, department_id 
		FROM employees 
		WHERE (salary > 70000 AND department_id = 1) OR (salary < 65000 AND department_id = 2)
		ORDER BY salary DESC`)
	require.NoError(suite.T(), err)
	defer rows.Close()

	var employees []struct {
		Name         string
		Salary       float64
		DepartmentID int
	}

	for rows.Next() {
		var name string
		var salary float64
		var deptID int
		err = rows.Scan(&name, &salary, &deptID)
		require.NoError(suite.T(), err)

		employees = append(employees, struct {
			Name         string
			Salary       float64
			DepartmentID int
		}{
			Name:         name,
			Salary:       salary,
			DepartmentID: deptID,
		})
	}

	require.Greater(suite.T(), len(employees), 0)

	// Verify results match expected conditions
	for _, emp := range employees {
		if emp.DepartmentID == 1 {
			require.Greater(suite.T(), emp.Salary, 70000.0)
		} else if emp.DepartmentID == 2 {
			require.Less(suite.T(), emp.Salary, 65000.0)
		}
	}

	// Test IN clause with subquery
	var count int
	err = suite.db.QueryRowContext(ctx, `
		SELECT COUNT(*) 
		FROM employees 
		WHERE department_id IN (SELECT id FROM departments WHERE budget > 600000)`).Scan(&count)
	require.NoError(suite.T(), err)
	require.Greater(suite.T(), count, 0)
}

// testJoinQueries tests various JOIN operations
func (suite *TestSuite) testJoinQueries(ctx context.Context) {
	// Test INNER JOIN with multiple tables
	rows, err := suite.db.QueryContext(ctx, `
		SELECT e.name AS employee_name, d.name AS department_name, p.name AS project_name, ep.role
		FROM employees e
		INNER JOIN departments d ON e.department_id = d.id
		INNER JOIN employee_projects ep ON e.id = ep.employee_id
		INNER JOIN projects p ON ep.project_id = p.id
		ORDER BY e.name, p.name`)
	require.NoError(suite.T(), err)
	defer rows.Close()

	var results []struct {
		EmployeeName   string
		DepartmentName string
		ProjectName    string
		Role           string
	}

	for rows.Next() {
		var empName, deptName, projName, role string
		err = rows.Scan(&empName, &deptName, &projName, &role)
		require.NoError(suite.T(), err)

		results = append(results, struct {
			EmployeeName   string
			DepartmentName string
			ProjectName    string
			Role           string
		}{
			EmployeeName:   empName,
			DepartmentName: deptName,
			ProjectName:    projName,
			Role:           role,
		})
	}

	require.Greater(suite.T(), len(results), 0)

	// Verify join integrity
	for _, result := range results {
		require.NotEmpty(suite.T(), result.EmployeeName)
		require.NotEmpty(suite.T(), result.DepartmentName)
		require.NotEmpty(suite.T(), result.ProjectName)
		require.NotEmpty(suite.T(), result.Role)
	}

	// Test LEFT JOIN
	var leftJoinCount int
	err = suite.db.QueryRowContext(ctx, `
		SELECT COUNT(*) 
		FROM departments d
		LEFT JOIN employees e ON d.id = e.department_id
		WHERE e.id IS NULL`).Scan(&leftJoinCount)
	require.NoError(suite.T(), err)
	// Should be 0 since all departments have employees in our test data
	require.Equal(suite.T(), 0, leftJoinCount)
}

// testAggregations tests aggregate functions
func (suite *TestSuite) testAggregations(ctx context.Context) {
	// Test COUNT with GROUP BY
	rows, err := suite.db.QueryContext(ctx, `
		SELECT d.name, COUNT(e.id) as employee_count, AVG(e.salary) as avg_salary
		FROM departments d
		LEFT JOIN employees e ON d.id = e.department_id
		GROUP BY d.id, d.name
		ORDER BY d.name`)
	require.NoError(suite.T(), err)
	defer rows.Close()

	var deptStats []struct {
		Name          string
		EmployeeCount int
		AvgSalary     float64
	}

	for rows.Next() {
		var name string
		var empCount int
		var avgSalary float64
		err = rows.Scan(&name, &empCount, &avgSalary)
		require.NoError(suite.T(), err)

		deptStats = append(deptStats, struct {
			Name          string
			EmployeeCount int
			AvgSalary     float64
		}{
			Name:          name,
			EmployeeCount: empCount,
			AvgSalary:     avgSalary,
		})
	}

	require.Equal(suite.T(), 3, len(deptStats)) // 3 departments

	// Verify aggregation results
	for _, stat := range deptStats {
		switch stat.Name {
		case "Engineering":
			require.Equal(suite.T(), 2, stat.EmployeeCount)
			require.Equal(suite.T(), 77500.0, stat.AvgSalary) // (80000 + 75000) / 2
		case "Marketing":
			require.Equal(suite.T(), 2, stat.EmployeeCount)
			require.Equal(suite.T(), 65000.0, stat.AvgSalary) // (60000 + 70000) / 2
		case "Sales":
			require.Equal(suite.T(), 1, stat.EmployeeCount)
			require.Equal(suite.T(), 90000.0, stat.AvgSalary)
		}
	}

	// Test SUM and MAX
	var totalBudget, maxSalary float64
	err = suite.db.QueryRowContext(ctx, `
		SELECT SUM(budget), MAX(salary) 
		FROM departments d
		LEFT JOIN employees e ON d.id = e.department_id`).Scan(&totalBudget, &maxSalary)
	require.NoError(suite.T(), err)
	require.Greater(suite.T(), totalBudget, 0.0)
	require.Equal(suite.T(), 90000.0, maxSalary)
}

// testSubqueries tests subquery functionality
func (suite *TestSuite) testSubqueries(ctx context.Context) {
	// Test EXISTS subquery
	rows, err := suite.db.QueryContext(ctx, `
		SELECT name, salary 
		FROM employees e
		WHERE EXISTS (
			SELECT 1 FROM employee_projects ep 
			JOIN projects p ON ep.project_id = p.id 
			WHERE ep.employee_id = e.id AND p.budget > 150000
		)
		ORDER BY name`)
	require.NoError(suite.T(), err)
	defer rows.Close()

	var highBudgetEmployees []struct {
		Name   string
		Salary float64
	}

	for rows.Next() {
		var name string
		var salary float64
		err = rows.Scan(&name, &salary)
		require.NoError(suite.T(), err)

		highBudgetEmployees = append(highBudgetEmployees, struct {
			Name   string
			Salary float64
		}{
			Name:   name,
			Salary: salary,
		})
	}

	require.Greater(suite.T(), len(highBudgetEmployees), 0)

	// Test scalar subquery
	var avgProjectBudget float64
	err = suite.db.QueryRowContext(ctx, `
		SELECT AVG(budget) 
		FROM projects 
		WHERE department_id = (SELECT id FROM departments WHERE name = 'Engineering')`).Scan(&avgProjectBudget)
	require.NoError(suite.T(), err)
	require.Greater(suite.T(), avgProjectBudget, 0.0)
}

// testPagination tests LIMIT and OFFSET functionality
func (suite *TestSuite) testPagination(ctx context.Context) {
	// Test basic pagination
	pageSize := 2
	page := 1

	rows, err := suite.db.QueryContext(ctx, suite.convertPlaceholders(`
		SELECT name, email, salary 
		FROM employees 
		ORDER BY salary DESC, name
		LIMIT ? OFFSET ?`), pageSize, (page-1)*pageSize)
	require.NoError(suite.T(), err)
	defer rows.Close()

	var pageEmployees []struct {
		Name   string
		Email  string
		Salary float64
	}

	for rows.Next() {
		var name, email string
		var salary float64
		err = rows.Scan(&name, &email, &salary)
		require.NoError(suite.T(), err)

		pageEmployees = append(pageEmployees, struct {
			Name   string
			Email  string
			Salary float64
		}{
			Name:   name,
			Email:  email,
			Salary: salary,
		})
	}

	require.LessOrEqual(suite.T(), len(pageEmployees), pageSize)
	require.Greater(suite.T(), len(pageEmployees), 0)

	// Verify ordering (highest salary first)
	if len(pageEmployees) > 1 {
		require.GreaterOrEqual(suite.T(), pageEmployees[0].Salary, pageEmployees[1].Salary)
	}

	// Test second page
	page = 2
	rows, err = suite.db.QueryContext(ctx, suite.convertPlaceholders(`
		SELECT name, email, salary 
		FROM employees 
		ORDER BY salary DESC, name
		LIMIT ? OFFSET ?`), pageSize, (page-1)*pageSize)
	require.NoError(suite.T(), err)
	defer rows.Close()

	pageEmployees = nil
	for rows.Next() {
		var name, email string
		var salary float64
		err = rows.Scan(&name, &email, &salary)
		require.NoError(suite.T(), err)

		pageEmployees = append(pageEmployees, struct {
			Name   string
			Email  string
			Salary float64
		}{
			Name:   name,
			Email:  email,
			Salary: salary,
		})
	}

	// Should have remaining employees
	require.Greater(suite.T(), len(pageEmployees), 0)
}

// TestQueryOptimization tests query compilation and optimization
func (suite *TestSuite) TestQueryOptimization() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a complex query with multiple joins and conditions
	complexQuery := &queryAst.FindManyQuery{
		Model: "employees",
		Where: &queryAst.WhereClause{
			Conditions: []queryAst.Condition{
				{
					Field:    "salary",
					Operator: queryAst.OpGreaterThan,
					Value:    50000,
				},
				{
					Field:    "department_id",
					Operator: queryAst.OpIn,
					Value:    []int{1, 2},
				},
			},
		},
		Select: &queryAst.SelectClause{
			Fields: []string{"name", "email", "salary"},
		},
		OrderBy: []queryAst.OrderByClause{
			{
				Field:     "salary",
				Direction: queryAst.SortDesc,
			},
		},
		Take: &[]int{10}[0],
		Skip: &[]int{0}[0],
	}

	// Compile the query
	sql, args, err := suite.compiler.Compile(complexQuery)
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), sql)
	require.NotNil(suite.T(), args)

	suite.T().Logf("Optimized SQL: %s, Args: %v", sql, args)

	// Verify the compiled SQL contains expected elements
	require.Contains(suite.T(), sql, "SELECT")
	require.Contains(suite.T(), sql, "FROM")
	require.Contains(suite.T(), sql, "WHERE")
	require.Contains(suite.T(), sql, "ORDER BY")
	require.Contains(suite.T(), sql, "LIMIT")

	// Test that the query can be executed (if tables exist)
	suite.createComplexTestTables(ctx)
	defer suite.cleanupComplexTestTables(ctx)

	rows, err := suite.db.QueryContext(ctx, sql, args...)
	require.NoError(suite.T(), err)
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
		var name, email string
		var salary float64
		err = rows.Scan(&name, &email, &salary)
		require.NoError(suite.T(), err)
		require.Greater(suite.T(), salary, 50000.0)
	}

	require.Greater(suite.T(), count, 0)
	require.LessOrEqual(suite.T(), count, 10)
}
