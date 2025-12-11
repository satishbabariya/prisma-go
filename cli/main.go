package main

import (
	"os"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/satishbabariya/prisma-go/cli/commands"
	"github.com/satishbabariya/prisma-go/cli/internal/version"
	"github.com/satishbabariya/prisma-go/telemetry"
)

func main() {
	// Initialize telemetry (opt-in, respects --no-telemetry flag)
	telemetry.InitTelemetry(version.Version, true)
	defer telemetry.Shutdown()

	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
