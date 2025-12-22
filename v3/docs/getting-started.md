# Prisma-Go v3 - Getting Started

## Installation

```bash
go get github.com/satishbabariya/prisma-go/v3
```

## Quick Start

### 1. Initialize Your Project

```bash
mkdir myproject && cd myproject
go mod init myproject
prisma init
```

This creates:
- `prisma/schema.prisma` - Your database schema
- `.env` - Environment variables (with DATABASE_URL)

### 2. Define Your Schema

Edit `prisma/schema.prisma`:

```prisma
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

generator client {
  provider = "prisma-client-go"
}

model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String
  posts     Post[]
  createdAt DateTime @default(now())
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

### 3. Create Database & Generate Client

```bash
# Push schema to database
prisma db push

# Generate type-safe client
prisma generate
```

### 4. Use the Client

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "myproject/generated/db"
)

func main() {
    client, err := db.NewClient()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    ctx := context.Background()
    
    // Create a user
    user, err := client.User.Create(db.User{
        Email: "alice@example.com",
        Name:  "Alice",
    }).Exec(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Created user: %s\n", user.Name)
    
    // Query users
    users, err := client.User.FindMany(
        db.User.Email.Contains("alice"),
    ).Exec(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, u := range users {
        fmt.Printf("User: %s (%s)\n", u.Name, u.Email)
    }
}
```

## Next Steps

- [API Reference](./reference/api/client.md) - Complete API documentation
- [Schema Guide](./reference/schema/models.md) - Advanced schema features
- [Tutorials](./tutorials/) - Real-world examples
- [Best Practices](./best-practices/) - Production tips

## Key Features

✅ **Type-Safe**: Compile-time query validation  
✅ **Multi-Database**: PostgreSQL, MySQL, SQLite  
✅ **Migrations**: Built-in migration system  
✅ **Introspection**: Generate schema from existing DB  
✅ **Relations**: Type-safe relation loading  
✅ **Transactions**: Full ACID transaction support  
✅ **Connection Pooling**: Production-ready pooling  

## Common Patterns

### Create with Relations

```go
post, err := client.Post.Create(db.Post{
    Title:   "Hello World",
    Content: "My first post",
    Author: db.User.Link(db.User.ID.Equals(userID)),
}).Exec(ctx)
```

### Query with Relations

```go
posts, err := client.Post.FindMany().
    Include(db.Post.Author.Fetch()).
    Where(db.Post.Published.Equals(true)).
    OrderBy(db.Post.CreatedAt.Desc()).
    Exec(ctx)

for _, post := range posts {
    fmt.Printf("%s by %s\n", post.Title, post.Author.Name)
}
```

### Transactions

```go
err := client.Transaction(func(tx *db.Client) error {
    user, err := tx.User.Create(/* ... */).Exec(ctx)
    if err != nil {
        return err
    }
    
    _, err = tx.Post.Create(db.Post{
        AuthorID: user.ID,
        Title: "First post",
    }).Exec(ctx)
    
    return err
})
```

### Aggregations

```go
result, err := client.Post.Aggregate().
    GroupBy(db.Post.AuthorID).
    Count().
    Sum(db.Post.Views).
    Exec(ctx)
```

## Support

- **Documentation**: https://prisma-go.dev
- **GitHub**: https://github.com/satishbabariya/prisma-go
- **Discord**: https://discord.gg/prisma
