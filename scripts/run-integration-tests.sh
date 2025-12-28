#!/bin/bash

# Integration Test Runner for Prisma-Go v3
set -e

echo "🚀 Starting Prisma-Go v3 Integration Tests"
echo "=========================================="

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker first.${NC}"
    exit 1
fi

# Start PostgreSQL container
echo -e "${YELLOW}📦 Starting PostgreSQL test database...${NC}"
docker-compose -f docker-compose.test.yml up -d

# Wait for PostgreSQL to be ready
echo -e "${YELLOW}⏳ Waiting for database to be ready...${NC}"
sleep 5

# Check if database is ready
max_attempts=30
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if docker-compose -f docker-compose.test.yml exec -T postgres pg_isready -U prisma > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Database is ready!${NC}"
        break
    fi
    attempt=$((attempt + 1))
    echo "Waiting... ($attempt/$max_attempts)"
    sleep 1
done

if [ $attempt -eq $max_attempts ]; then
    echo -e "${RED}❌ Database failed to start${NC}"
    docker-compose -f docker-compose.test.yml down
    exit 1
fi

# Set environment variable
export DATABASE_URL="postgresql://prisma:prisma@localhost:5433/prisma_test?sslmode=disable"
export RUN_INTEGRATION_TESTS=true

# Run tests
echo -e "${YELLOW}🧪 Running integration tests...${NC}"
go test ./test/integration/... -v -count=1

# Capture exit code
TEST_EXIT_CODE=$?

# Cleanup
echo -e "${YELLOW}🧹 Cleaning up...${NC}"
docker-compose -f docker-compose.test.yml down -v

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ All integration tests passed!${NC}"
else
    echo -e "${RED}❌ Some tests failed${NC}"
fi

exit $TEST_EXIT_CODE
