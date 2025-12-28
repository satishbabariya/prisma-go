# Prisma Go

**A fully native Go implementation of Prisma ORM**

[![Go Version](https://img.shields.io/badge/go-1.24.1-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## 🎯 Vision

Prisma Go is a complete rewrite of Prisma in Go, providing Go developers a native Prisma-like experience **without** running Rust engines, Node.js, RPC servers, or separate client engine processes.

This is essentially: **a native Go-based Prisma**.

## ✅ What We Provide

- ✅ **Native Go PSL (Prisma Schema Language)** - Complete schema parsing, validation, and formatting
- ✅ **Go Migration Engine** - Database introspection, diffing, migration planning, and execution
- ✅ **Go Query Compiler** - Type-safe query generation with relation support, aggregations, and transactions
- ✅ **Go Code Generator** - Generate Go client code from Prisma schemas with watch mode
- ✅ **Pure Go CLI** - All operations in a single binary with beautiful UI
- ✅ **Runtime Client** - Full-featured database client with connection pooling and middleware support

## 🗄️ Supported Databases

- ✅ **PostgreSQL** - Full support (introspection, migrations, queries)
- ✅ **MySQL** - Full support (introspection, migrations, queries)
- ✅ **SQLite** - Full support (introspection, migrations, queries)
- ✅ **MongoDB** - Schema validation support
- ✅ **MSSQL** - Schema validation support
- ✅ **CockroachDB** - Schema validation support

## 🔧 Features

### Schema Management
- Parse and validate Prisma schemas
- Format schemas automatically
- Comprehensive error diagnostics
- Support for all Prisma schema features (models, enums, relations, indexes, etc.)

### Migrations
- Database introspection (Postgres, MySQL, SQLite)
- Schema diffing with detailed change detection
- Migration file generation
- Migration history tracking
- Safe migration planning

### Query Compiler
- Type-safe query building
- Optimized JOIN queries for relations
- Complex WHERE clauses
- Aggregations (Count, Sum, Avg, Min, Max)
- Pagination support
- Nested writes (create, update, delete, connect, disconnect, upsert)
- Transaction support

### Code Generation
- Generate Go structs from Prisma models
- Type-safe query builders
- Watch mode for development
- Proper Go struct tags

## 🚀 NO Runtime Overhead

- ❌ No gRPC
- ❌ No Rust engines
- ❌ No Node.js runtime
- ❌ No sidecar processes
- ✅ Everything compiles into **one binary**
- ✅ Zero runtime overhead
- ✅ Full Go-native developer experience

## 📦 Architecture

```
prisma-go/
│
├── cmd/prisma/           # CLI tool
│   └── commands/         # CLI commands (format, validate, generate, migrate, db)
│
├── internal/             # Internal packages
│   ├── psl/              # PSL Parser (Layer 1)
│   │   ├── ast/          # AST definitions
│   │   ├── lexer.go      # Lexer
│   │   └── parser.go     # Schema parser
│   │
│   ├── core/             # Core functionality
│   │   ├── schema/       # Schema domain, validation, formatting
│   │   ├── migration/    # Migration Engine (Layer 2)
│   │   │   ├── diff/     # Schema comparison
│   │   │   ├── executor/ # Migration execution
│   │   │   └── history/  # Migration history
│   │   │
│   │   ├── query/        # Query Compiler (Layer 3)
│   │   │   ├── builder/  # Query builder
│   │   │   ├── compiler/ # Query compiler
│   │   │   └── executor/ # Query executor
│   │   │
│   │   ├── generator/    # Code Generator (Layer 4)
│   │   │   ├── analyzer/ # Schema analyzer
│   │   │   ├── template/ # Code templates
│   │   │   └── service/  # Generator service
│   │   │
│   │   └── introspection/ # Database introspection
│   │
│   ├── adapters/         # Database adapters
│   └── service/          # Application services
│
├── runtime/              # ORM runtime
│   ├── client/           # Client runtime
│   └── types/            # Runtime types
│
├── pkg/                  # Public packages
│   └── client/           # Client library
│
└── tests/                # Test suites
    ├── integration/      # Integration tests
    └── validation/       # Validation tests
```

## 🎮 CLI Commands

```bash
# Schema management
prisma-go format [schema-path]          # Format Prisma schema
prisma-go validate [schema-path]         # Validate Prisma schema

# Code generation
prisma-go generate [schema-path]         # Generate Go client
prisma-go generate --watch               # Watch mode for auto-regeneration

# Database migrations
prisma-go migrate dev [schema-path]      # Create and apply migration
prisma-go migrate dev --name <name>     # Create named migration
prisma-go migrate dev --apply            # Auto-apply migration
prisma-go migrate deploy                 # Deploy pending migrations
prisma-go migrate diff [schema-path]     # Compare schema to database
prisma-go migrate apply <file>           # Apply specific migration
prisma-go migrate status                 # Check migration status
prisma-go migrate reset                  # Reset database

# Database operations
prisma-go db push [schema-path]          # Push schema changes to database
prisma-go db pull                        # Pull schema from database
prisma-go db execute <sql>               # Execute raw SQL
prisma-go db seed                        # Seed database

# Utility
prisma-go version                        # Show version information
prisma-go init                           # Initialize new Prisma project
```

## 📖 Getting Started

### Installation

```bash
go install github.com/satishbabariya/prisma-go/cli@latest
```

### Quick Start

1. Create a `schema.prisma` file:

```prisma
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
  id        Int      @id @default(autoincrement())
  title     String
  content   String?
  published Boolean  @default(false)
  author    User     @relation(fields: [authorId], references: [id])
  authorId  Int
}
```

2. Format and validate your schema:

```bash
prisma-go format
prisma-go validate
```

3. Generate the Go client:

```bash
prisma-go generate
```

4. Use the generated client in your Go code:

```go
package main

import (
    "context"
    "os"
    "github.com/yourproject/generated"
    "github.com/satishbabariya/prisma-go/runtime/client"
)

func main() {
    ctx := context.Background()
    
    // Create Prisma client
    dbURL := os.Getenv("DATABASE_URL")
    prismaClient, err := client.NewPrismaClient("postgresql", dbURL)
    if err != nil {
        panic(err)
    }
    defer prismaClient.Disconnect(ctx)
    
    // Connect to database
    if err := prismaClient.Connect(ctx); err != nil {
        panic(err)
    }
    
    // Use the generated client with query executor
    // The generated client provides type-safe query builders
    // Example usage would depend on your generated code structure
}
```

### Database Migrations

1. Create your first migration:

```bash
prisma-go migrate dev --name init
```

2. Apply migrations to production:

```bash
prisma-go migrate deploy
```

3. Compare schema with database:

```bash
prisma-go migrate diff
```

### Database Introspection

Pull your existing database schema:

```bash
prisma-go db pull
```

This will generate a `schema.prisma` file from your existing database.

## 🏗️ Current Status

### ✅ Completed (Layer 1 - PSL)
- [x] Schema parser with lexer
- [x] AST generation
- [x] Schema validation (49+ validators)
- [x] Attribute validation
- [x] Relation validation
- [x] Connector support (Postgres, MySQL, SQLite, MongoDB, MSSQL, CockroachDB)
- [x] Native types validation
- [x] Schema formatting
- [x] Diagnostics with pretty printing
- [x] CLI format & validate commands

### ✅ Completed (Layer 2 - Migration Engine)
- [x] Foundation & structure
- [x] PostgreSQL introspection
- [x] MySQL introspection
- [x] SQLite introspection
- [x] Schema diffing (tables, columns, indexes, foreign keys)
- [x] Migration planning
- [x] Migration execution
- [x] Migration history tracking
- [x] SQL generation for migrations (Postgres, MySQL, SQLite)
- [x] CLI commands: `migrate dev`, `deploy`, `diff`, `apply`, `status`, `reset`

### ✅ Completed (Layer 3 - Query Compiler)
- [x] Query AST
- [x] Query compilation
- [x] SQL generation (Postgres, MySQL, SQLite)
- [x] Relation resolution with optimized JOINs
- [x] Complex WHERE clause handling
- [x] Pagination (limit/offset)
- [x] Aggregations (Count, Sum, Avg, Min, Max)
- [x] Nested writes (create, update, delete, connect, disconnect, upsert)
- [x] Transaction support
- [x] Prepared statement caching
- [x] Query executor with result mapping

### ✅ Completed (Layer 4 - Code Generator)
- [x] Generator foundation
- [x] Model generation from schema
- [x] Client generation with type-safe methods
- [x] Type mapping (Prisma → Go)
- [x] CLI generate command
- [x] Watch mode for auto-regeneration
- [x] Generated code with proper struct tags

### ✅ Completed (Runtime Client)
- [x] Database connection management
- [x] Connection pooling configuration
- [x] Middleware support
- [x] Raw SQL query execution
- [x] CRUD operations (FindMany, FindFirst, Create, Update, Delete)
- [x] Batch operations (CreateMany, UpdateMany, DeleteMany)
- [x] Upsert operations
- [x] Transaction support

## 🤝 Contributing

Contributions are welcome! This is an ambitious project and we'd love your help.

## 📄 License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

This project is inspired by [Prisma](https://github.com/prisma/prisma) and aims to bring the same excellent developer experience to the Go ecosystem.

---

**Status:** Active Development 🚀

Built with ❤️ for the Go community.

