# Prisma-Go v3 - Integration Tests

## Overview

Comprehensive end-to-end integration tests with real PostgreSQL database to validate all implemented features.

## Prerequisites

- Docker and Docker Compose installed
- Go 1.21 or later
- Port 5433 available (or modify docker-compose.test.yml)

## Quick Start

### Method 1: Using the Script (Recommended)

```bash
# Make script executable (first time only)
chmod +x scripts/run-integration-tests.sh

# Run all integration tests
./scripts/run-integration-tests.sh
```

### Method 2: Manual Steps

```bash
# 1. Start PostgreSQL
docker-compose -f docker-compose.test.yml up -d

# 2. Wait for database (check logs)
docker-compose -f docker-compose.test.yml logs -f postgres

# 3. Run tests
export DATABASE_URL="postgresql://prisma:prisma@localhost:5433/prisma_test?sslmode=disable"
export RUN_INTEGRATION_TESTS=true
go test ./test/integration/... -v

# 4. Cleanup
docker-compose -f docker-compose.test.yml down -v
```

## Test Coverage

### Features Tested

- ✅ **OrThrow Methods**
  - FindFirstOrThrow error handling
  - FindUniqueOrThrow with conditions
  - Successful retrieval

- ✅ **DISTINCT Queries**
  - Unique value selection
  - Multiple field DISTINCT

- ✅ **Cursor Pagination**
  - Page navigation
  - Cursor-based filtering
  - Ordering integration

- ✅ **GroupBy & Having**
  - Grouping by fields
  - Aggregations (COUNT, SUM, AVG)
  - HAVING clause filters

- ✅ **Nested Writes**
  - NestedCreate with foreign keys
  - Transaction handling
  - Cascading operations

- ✅ **Raw SQL**
  - Parameter binding security
  - QueryRaw results
  - ExecuteRaw mutations
  - Type-safe mapping

- ✅ **Complex Queries**
  - Combined filters
  - Ordering + pagination
  - Multiple conditions

## Database Schema

The tests use a realistic schema:

```sql
users
├── id (PK)
├── email (unique)
├── name
├── status
└── timestamps

profiles
├── id (PK)
├── bio
├── user_id (FK → users)

posts
├── id (PK)
├── title
├── content
├── author_id (FK → users)
├── category
├── tags (array)
└── timestamps

comments
├── id (PK)
├── text
├── post_id (FK → posts)
└── created_at
```

## Test Structure

```
test/integration/
├── setup_test.go        # Database setup/teardown
├── features_test.go     # Feature integration tests
```

## Environment Variables

- `RUN_INTEGRATION_TESTS` - Set to "true" to run integration tests
- `DATABASE_URL` - PostgreSQL connection string (auto-set by script)

## Database Container

The test database runs in Docker with:
- **Image**: postgres:15-alpine
- **Port**: 5433 (to avoid conflicts with local PostgreSQL)
- **User**: prisma
- **Password**: prisma
- **Database**: prisma_test

## Running Specific Tests

```bash
# Run only OrThrow tests
go test ./test/integration/... -v -run TestOrThrow

# Run only pagination tests
go test ./test/integration/... -v -run TestCursorPagination

# Run with coverage
go test ./test/integration/... -v -cover
```

## Troubleshooting

### Database not ready
```bash
# Check database logs
docker-compose -f docker-compose.test.yml logs postgres

# Restart database
docker-compose -f docker-compose.test.yml restart
```

### Port already in use
Edit `docker-compose.test.yml` and change port mapping:
```yaml
ports:
  - "5434:5432"  # Use different port
```

### Tests failing
```bash
# Clean start
docker-compose -f docker-compose.test.yml down -v
./scripts/run-integration-tests.sh
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run Integration Tests
        run: ./scripts/run-integration-tests.sh
```

## Next Steps

1. **Add more test cases**
   - Transaction rollback scenarios
   - Concurrent operations
   - Error edge cases

2. **Performance testing**
   - Benchmark cursor vs offset pagination
   - Large dataset handling
   - Connection pooling

3. **Multi-database support**
   - MySQL integration tests
   - SQLite integration tests

## Results

All integration tests validate:
- ✅ Database connectivity
- ✅ Query compilation
- ✅ Parameter binding
- ✅ Result mapping
- ✅ Transaction handling
- ✅ Error propagation
- ✅ Foreign key constraints
- ✅ Cascade operations

**Production Ready:** All features tested with real database operations! 🚀
