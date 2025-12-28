// Package validation provides comprehensive test harness for comparing
// Prisma-go v3 behavior against reference implementations.
package validation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestResult captures the outcome of a comparison test
type TestResult struct {
	Feature   string        `json:"feature"`
	Category  string        `json:"category"`
	Passed    bool          `json:"passed"`
	SQLQuery  string        `json:"sql_query,omitempty"`
	Result    interface{}   `json:"result,omitempty"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration_ns"`
	Timestamp time.Time     `json:"timestamp"`
}

// ComparisonResult holds side-by-side comparison data
type ComparisonResult struct {
	TestName    string     `json:"test_name"`
	PrismaGo    TestResult `json:"prisma_go"`
	Match       bool       `json:"match"`
	Discrepancy string     `json:"discrepancy,omitempty"`
}

// ValidationReport aggregates all test results
type ValidationReport struct {
	GeneratedAt    time.Time          `json:"generated_at"`
	TotalTests     int                `json:"total_tests"`
	PassedTests    int                `json:"passed_tests"`
	FailedTests    int                `json:"failed_tests"`
	PassRate       float64            `json:"pass_rate"`
	Categories     map[string]int     `json:"categories"`
	Results        []ComparisonResult `json:"results"`
	FailedFeatures []string           `json:"failed_features"`
}

// TestHarness provides utilities for validation testing
type TestHarness struct {
	db          *sql.DB
	dbURL       string
	sqlLog      []string
	results     []ComparisonResult
	currentTest string
}

// NewTestHarness creates a new test harness
func NewTestHarness(dbURL string) (*TestHarness, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &TestHarness{
		db:      db,
		dbURL:   dbURL,
		sqlLog:  make([]string, 0),
		results: make([]ComparisonResult, 0),
	}, nil
}

// Close closes the database connection
func (h *TestHarness) Close() error {
	return h.db.Close()
}

// DB returns the database connection
func (h *TestHarness) DB() *sql.DB {
	return h.db
}

// LogSQL logs a SQL query for comparison
func (h *TestHarness) LogSQL(query string, args ...interface{}) {
	formatted := fmt.Sprintf("%s | args: %v", query, args)
	h.sqlLog = append(h.sqlLog, formatted)
}

// GetSQLLog returns all logged SQL queries
func (h *TestHarness) GetSQLLog() []string {
	return h.sqlLog
}

// ClearSQLLog clears the SQL log
func (h *TestHarness) ClearSQLLog() {
	h.sqlLog = make([]string, 0)
}

// StartTest marks the start of a test
func (h *TestHarness) StartTest(name string) {
	h.currentTest = name
	h.ClearSQLLog()
}

// RecordResult records a test result
func (h *TestHarness) RecordResult(category string, passed bool, result interface{}, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	sqlQuery := ""
	if len(h.sqlLog) > 0 {
		sqlQuery = strings.Join(h.sqlLog, "\n")
	}

	h.results = append(h.results, ComparisonResult{
		TestName: h.currentTest,
		PrismaGo: TestResult{
			Feature:   h.currentTest,
			Category:  category,
			Passed:    passed,
			SQLQuery:  sqlQuery,
			Result:    result,
			Error:     errStr,
			Timestamp: time.Now(),
		},
		Match: passed,
	})
}

// GenerateReport generates a validation report
func (h *TestHarness) GenerateReport() *ValidationReport {
	report := &ValidationReport{
		GeneratedAt:    time.Now(),
		TotalTests:     len(h.results),
		Categories:     make(map[string]int),
		Results:        h.results,
		FailedFeatures: make([]string, 0),
	}

	for _, r := range h.results {
		if r.Match {
			report.PassedTests++
		} else {
			report.FailedTests++
			report.FailedFeatures = append(report.FailedFeatures, r.TestName)
		}
		report.Categories[r.PrismaGo.Category]++
	}

	if report.TotalTests > 0 {
		report.PassRate = float64(report.PassedTests) / float64(report.TotalTests) * 100
	}

	return report
}

// SaveReport saves the report to a JSON file
func (h *TestHarness) SaveReport(path string) error {
	report := h.GenerateReport()
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// SetupTestDB helper for test setup
func SetupTestDB(t *testing.T) (*TestHarness, func()) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://prisma:prisma@localhost:5433/prisma_test?sslmode=disable"
	}

	harness, err := NewTestHarness(dbURL)
	if err != nil {
		t.Fatalf("Failed to create test harness: %v", err)
	}

	cleanup := func() {
		harness.Close()
	}

	return harness, cleanup
}

// RunValidation helper to run validation and check result
func RunValidation(t *testing.T, h *TestHarness, category, testName string, fn func() (interface{}, error)) {
	t.Run(testName, func(t *testing.T) {
		h.StartTest(testName)
		start := time.Now()
		result, err := fn()
		duration := time.Since(start)

		passed := err == nil
		h.RecordResult(category, passed, result, err)

		if !passed {
			t.Errorf("%s failed: %v (duration: %v)", testName, err, duration)
		}
	})
}

// AssertEqual compares two values and records the result
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) bool {
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)

	if string(expectedJSON) != string(actualJSON) {
		t.Errorf("%s: expected %s, got %s", msg, expectedJSON, actualJSON)
		return false
	}
	return true
}

// WaitForDB waits for database to be ready
func WaitForDB(ctx context.Context, dbURL string, timeout time.Duration) error {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return err
	}
	defer db.Close()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := db.PingContext(ctx); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("database not ready after %v", timeout)
}
