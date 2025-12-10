package commands

import (
	"fmt"
	"os"
)

// Execute is the main entry point for the CLI
func Execute() error {
	if len(os.Args) < 2 {
		printHelp()
		return nil
	}

	command := os.Args[1]

	switch command {
	case "init":
		return initCommand(os.Args[2:])
	case "format", "fmt":
		return formatCommand(os.Args[2:])
	case "validate":
		return validateCommand(os.Args[2:])
	case "generate":
		return generateCommand(os.Args[2:])
	case "migrate":
		return migrateCommand(os.Args[2:])
	case "db":
		return dbCommand(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Println("prisma-go version 0.1.0")
		return nil
	case "help", "-h", "--help":
		printHelp()
		return nil
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printHelp()
		os.Exit(1)
		return nil
	}
}

func printHelp() {
	help := `
╔═══════════════════════════════════════════════════════════╗
║                      PRISMA-GO                            ║
║          Native Go Prisma ORM & Schema Engine             ║
╚═══════════════════════════════════════════════════════════╝

USAGE:
    prisma-go <command> [options]

COMMANDS:
    init             Initialize a new Prisma-Go project
    format, fmt      Format a Prisma schema file
    validate         Validate a Prisma schema file
    generate         Generate Prisma Client for Go
    migrate          Manage database migrations
    db               Manage your database schema
    version          Print version information
    help             Print this help message

MIGRATE COMMANDS:
    migrate dev      Create and apply migrations in development
    migrate deploy   Apply pending migrations to production
    migrate diff     Compare schema to database (use --create-only to generate SQL)
    migrate apply    Apply a migration SQL file
    migrate status   Check migration status
    migrate reset    Reset the database

DB COMMANDS:
    db push          Push schema changes to database
    db pull          Pull schema from database (introspect)
    db seed          Seed the database

OPTIONS:
    -h, --help       Print help
    -v, --version    Print version

EXAMPLES:
    prisma-go init
    prisma-go format ./schema.prisma
    prisma-go validate ./schema.prisma
    prisma-go generate
    prisma-go migrate dev
    prisma-go db push

For more information, visit: https://github.com/satishbabariya/prisma-go
`
	fmt.Println(help)
}
