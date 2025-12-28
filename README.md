# Prisma Go

**A fully native Go implementation of Prisma ORM**

[![Go Version](https://img.shields.io/badge/go-1.24.1-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

---

## Overview

Prisma Go is a complete reimplementation of Prisma in pure Go. It provides Go developers with a native Prisma-like experience **without** any external dependencies like Rust engines, Node.js runtime, or sidecar processes.

**Everything compiles into a single binary with zero runtime overhead.**

## Features

| Category | Capabilities |
|----------|-------------|
| **Schema** | Parse, validate, and format Prisma schemas with comprehensive diagnostics |
| **Migrations** | Database introspection, schema diffing, migration planning and execution |
| **Query Compiler** | Type-safe queries with JOINs, aggregations, nested writes, and transactions |
| **Code Generator** | Generate Go client code from schemas |
| **Runtime Client** | Connection pooling, middleware support, and full CRUD operations |

## Supported Databases

| Database | Status |
|----------|--------|
| PostgreSQL | ✅ Full support |
| MySQL | ✅ Full support |
| SQLite | ✅ Full support |
| MongoDB | ⚙️ Schema validation |
| MSSQL | ⚙️ Schema validation |
| CockroachDB | ⚙️ Schema validation |

## Installation

```bash
go install github.com/satishbabariya/prisma-go/cmd/prisma@latest
```

## Quick Start

### 1. Create Schema

```prisma
// schema.prisma
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

generator client {
  provider = "prisma-client-go"
  output   = "./generated"
}

model User {
  id    Int     @id @default(autoincrement())
  email String  @unique
  name  String?
  posts Post[]
}

model Post {
  id        Int     @id @default(autoincrement())
  title     String
  content   String?
  published Boolean @default(false)
  author    User    @relation(fields: [authorId], references: [id])
  authorId  Int
}
```

### 2. Validate and Generate

```bash
prisma-go validate
prisma-go generate
```

### 3. Create Migration

```bash
prisma-go migrate dev --name init
```

### 4. Use in Code

```go
package main

import (
    "context"
    "os"
    "github.com/satishbabariya/prisma-go/pkg/client"
)

func main() {
    ctx := context.Background()
    
    prisma, err := client.NewPrismaClient("postgresql", os.Getenv("DATABASE_URL"))
    if err != nil {
        panic(err)
    }
    defer prisma.Disconnect(ctx)
    
    if err := prisma.Connect(ctx); err != nil {
        panic(err)
    }
    
    // Use the generated type-safe query builders
}
```

## CLI Reference

### Schema Management

```bash
prisma-go format [--schema <path>]     # Format schema file
prisma-go validate [--schema <path>]   # Validate schema
prisma-go generate [--schema <path>]   # Generate Go client
```

### Database Migrations

```bash
prisma-go migrate dev --name <name>    # Create and apply migration
prisma-go migrate deploy               # Apply pending migrations
prisma-go migrate status               # Check migration status
prisma-go migrate reset                # Reset database
```

### Database Operations

```bash
prisma-go db push                      # Push schema to database
prisma-go db pull                      # Pull schema from database
prisma-go db seed                      # Seed database
```

### Utility

```bash
prisma-go init                         # Initialize new project
prisma-go -v                           # Show version
prisma-go --help                       # Show help
```

## Architecture

```
prisma-go/
├── cmd/prisma/              # CLI commands
├── internal/
│   ├── psl/                 # Prisma Schema Language parser
│   ├── core/
│   │   ├── schema/          # Schema validation & formatting
│   │   ├── migration/       # Migration engine
│   │   ├── query/           # Query compiler & executor
│   │   ├── generator/       # Code generator
│   │   └── introspection/   # Database introspection
│   ├── adapters/            # Database adapters
│   └── service/             # Application services
├── runtime/                 # ORM runtime client
├── pkg/client/              # Public client library
└── tests/                   # Test suites
```

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

---

Built with ❤️ for the Go community
