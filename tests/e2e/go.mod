module github.com/satishbabariya/prisma-go/tests/e2e

go 1.24.1

require (
	github.com/go-sql-driver/mysql v1.9.3
	github.com/lib/pq v1.10.9
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/satishbabariya/prisma-go v0.0.0-20251211044635-96fc42de5d9a
	github.com/stretchr/testify v1.11.1
)

replace github.com/satishbabariya/prisma-go => ../..

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
