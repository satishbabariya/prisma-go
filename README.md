# Prisma Go

**A fully native Go implementation of Prisma ORM**

[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## 🎯 Vision

Prisma Go is a complete rewrite of Prisma in Go, providing Go developers a native Prisma-like experience **without** running Rust engines, Node.js, RPC servers, or separate client engine processes.

This is essentially: **a native Go-based Prisma**.

## ✅ What We Provide

- ✅ **Native Go PSL (Prisma Schema Language)** - Complete schema parsing, validation, and formatting
- ✅ **Go Migration Engine** - Database introspection, diffing, and migration management
- ✅ **Go Query Compiler** - Type-safe query generation for multiple databases
- ✅ **Go Code Generator** - Generate Go client code from Prisma schemas
- ✅ **Pure Go CLI** - All operations in a single binary

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
├── psl/                 # PSL (Layer 1) - Schema language
│   ├── parser           # Schema parser
│   ├── validator        # Schema validator
│   ├── database         # Database layer
│   ├── formatting       # Schema formatter
│   └── diagnostics      # Error reporting
│
├── migrate/             # Migration Engine (Layer 2)
│   ├── introspect       # Database introspection
│   ├── diff             # Schema comparison
│   ├── planner          # Migration planning
│   ├── executor         # Migration execution
│   └── history          # Migration history
│
├── query/               # Query Compiler (Layer 3)
│   ├── ast              # Query AST
│   ├── compiler         # Query compiler
│   ├── sqlgen           # SQL generation per provider
│   └── connectors       # Database connectors
│
├── generator/           # Code Generator (Layer 4)
│   ├── templates        # Code templates
│   └── codegen          # Code generation logic
│
├── runtime/             # ORM runtime
│   ├── client           # Client runtime
│   └── types            # Runtime types
│
└── cli/                 # CLI tool
    └── commands         # CLI commands
```

## 🎮 CLI Commands

```bash
# Format schema
prisma-go format ./schema.prisma

# Validate schema
prisma-go validate ./schema.prisma

# Generate Go client
prisma-go generate

# Database migrations
prisma-go migrate dev
prisma-go migrate deploy
prisma-go migrate diff
prisma-go migrate status
prisma-go migrate reset

# Database operations
prisma-go db push
prisma-go db pull
prisma-go db seed
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
    "github.com/yourproject/generated"
)

func main() {
    client := generated.NewPrismaClient()
    ctx := context.Background()
    
    // Connect to database
    client.Connect(ctx)
    defer client.Disconnect(ctx)
    
    // Query users
    users, _ := client.User.FindMany(
        user.Email.Contains("@example.com"),
    )
    
    // Create a user
    newUser, _ := client.User.Create(
        user.Email.Set("user@example.com"),
        user.Name.Set("John Doe"),
    )
}
```

## 🏗️ Current Status

### ✅ Completed (Layer 1 - PSL)
- [x] Schema parser with lexer
- [x] AST generation
- [x] Schema validation (49 validators!)
- [x] Attribute validation
- [x] Relation validation
- [x] Connector support (Postgres, MySQL, SQLite, MongoDB, etc.)
- [x] Native types validation
- [x] Schema formatting
- [x] Diagnostics with pretty printing
- [x] CLI format & validate commands

### ✅ Completed (Layer 4 - Code Generator)
- [x] Generator foundation
- [x] Model generation from schema
- [x] Client generation with type-safe methods
- [x] Type mapping (Prisma → Go)
- [x] CLI generate command
- [x] Generated code with proper tags

### 🚧 In Progress (Layer 2 - Migration Engine)
- [x] Foundation & structure
- [ ] PostgreSQL introspection
- [ ] MySQL introspection
- [ ] SQLite introspection
- [ ] Schema diffing
- [ ] Migration planning
- [ ] Migration execution
- [ ] Migration history tracking

### 📋 Planned (Layer 3 - Query Compiler)
- [x] Query AST
- [ ] Query compilation
- [ ] SQL generation (Postgres)
- [ ] SQL generation (MySQL)
- [ ] SQL generation (SQLite)
- [ ] Relation resolution
- [ ] Filter handling
- [ ] Pagination

## 🤝 Contributing

Contributions are welcome! This is an ambitious project and we'd love your help.

## 📄 License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

This project is inspired by [Prisma](https://github.com/prisma/prisma) and aims to bring the same excellent developer experience to the Go ecosystem.

---

**Status:** Early Development 🚧

Built with ❤️ for the Go community.

