package commands

import (
	"fmt"
	"os"
)

func dbCommand(args []string) error {
	if len(args) == 0 {
		printDbHelp()
		return nil
	}

	subcommand := args[0]

	switch subcommand {
	case "push":
		return dbPushCommand(args[1:])
	case "pull":
		return dbPullCommand(args[1:])
	case "seed":
		return dbSeedCommand(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown db subcommand: %s\n\n", subcommand)
		printDbHelp()
		os.Exit(1)
		return nil
	}
}

func printDbHelp() {
	help := `
USAGE:
    prisma-go db <subcommand> [options]

SUBCOMMANDS:
    push       Push schema changes to database (no migrations)
    pull       Pull schema from database (introspection)
    seed       Seed the database

EXAMPLES:
    prisma-go db push
    prisma-go db pull
    prisma-go db seed
`
	fmt.Println(help)
}

func dbPushCommand(args []string) error {
	fmt.Println("🚧 DB push - coming soon!")
	return nil
}

func dbPullCommand(args []string) error {
	fmt.Println("🚧 DB pull (introspection) - coming soon!")
	return nil
}

func dbSeedCommand(args []string) error {
	fmt.Println("🚧 DB seed - coming soon!")
	return nil
}
