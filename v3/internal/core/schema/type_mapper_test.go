package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeMapper_PrismaToGo(t *testing.T) {
	tm := NewTypeMapper()

	tests := []struct {
		name       string
		prismaType string
		isOptional bool
		isList     bool
		want       string
	}{
		// Scalar types - required
		{"String required", "String", false, false, "string"},
		{"Boolean required", "Boolean", false, false, "bool"},
		{"Int required", "Int", false, false, "int64"},
		{"BigInt required", "BigInt", false, false, "int64"},
		{"Float required", "Float", false, false, "float64"},
		{"Decimal required", "Decimal", false, false, "decimal.Decimal"},
		{"DateTime required", "DateTime", false, false, "time.Time"},
		{"Json required", "Json", false, false, "json.RawMessage"},
		{"Bytes required", "Bytes", false, false, "[]byte"},

		// Scalar types - optional
		{"String optional", "String", true, false, "*string"},
		{"Boolean optional", "Boolean", true, false, "*bool"},
		{"Int optional", "Int", true, false, "*int64"},
		{"Float optional", "Float", true, false, "*float64"},
		{"DateTime optional", "DateTime", true, false, "*time.Time"},

		// List types
		{"String list", "String", false, true, "[]string"},
		{"Int list", "Int", false, true, "[]int64"},
		{"Model list", "User", false, true, "[]User"},

		// Optional lists (lists are already nullable in Go)
		{"String optional list", "String", true, true, "[]string"},

		// Model references
		{"Model reference required", "User", false, false, "User"},
		{"Model reference optional", "User", true, false, "*User"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tm.PrismaToGo(tt.prismaType, tt.isOptional, tt.isList)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTypeMapper_GoToPrisma(t *testing.T) {
	tm := NewTypeMapper()

	tests := []struct {
		name         string
		goType       string
		wantPrisma   string
		wantOptional bool
		wantList     bool
		wantErr      bool
	}{
		// Basic types
		{"string", "string", "String", false, false, false},
		{"bool", "bool", "Boolean", false, false, false},
		{"int64", "int64", "Int", false, false, false},
		{"float64", "float64", "Float", false, false, false},
		{"time.Time", "time.Time", "DateTime", false, false, false},

		// Optional types
		{"*string", "*string", "String", true, false, false},
		{"*bool", "*bool", "Boolean", true, false, false},
		{"*int64", "*int64", "Int", true, false, false},

		// List types
		{"[]string", "[]string", "String", false, true, false},
		{"[]int64", "[]int64", "Int", false, true, false},
		{"[]User", "[]User", "User", false, true, false},

		// Special case: []byte -> Bytes
		{"[]byte", "[]byte", "Bytes", false, false, false},

		// JSON
		{"json.RawMessage", "json.RawMessage", "Json", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prisma, optional, list, err := tm.GoToPrisma(tt.goType)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantPrisma, prisma)
			assert.Equal(t, tt.wantOptional, optional)
			assert.Equal(t, tt.wantList, list)
		})
	}
}

func TestTypeMapper_Enums(t *testing.T) {
	tm := NewTypeMapper()

	// Register enum types
	tm.RegisterEnum("Role")
	tm.RegisterEnum("Status")

	t.Run("IsEnum returns true for registered enums", func(t *testing.T) {
		assert.True(t, tm.IsEnum("Role"))
		assert.True(t, tm.IsEnum("Status"))
		assert.False(t, tm.IsEnum("User"))
	})

	t.Run("PrismaToGo handles enum types", func(t *testing.T) {
		goType := tm.PrismaToGo("Role", false, false)
		assert.Equal(t, "Role", goType)

		goTypeOptional := tm.PrismaToGo("Role", true, false)
		assert.Equal(t, "*Role", goTypeOptional)

		goTypeList := tm.PrismaToGo("Role", false, true)
		assert.Equal(t, "[]Role", goTypeList)
	})

	t.Run("GoToPrisma recognizes enum types", func(t *testing.T) {
		prisma, optional, list, err := tm.GoToPrisma("Role")
		require.NoError(t, err)
		assert.Equal(t, "Role", prisma)
		assert.False(t, optional)
		assert.False(t, list)
	})
}

func TestTypeMapper_CustomMappings(t *testing.T) {
	tm := NewTypeMapper()

	// Register a custom mapping
	tm.RegisterCustomMapping("UUID", "uuid.UUID")

	t.Run("Custom mapping is used", func(t *testing.T) {
		goType := tm.PrismaToGo("UUID", false, false)
		assert.Equal(t, "uuid.UUID", goType)

		goTypeOptional := tm.PrismaToGo("UUID", true, false)
		assert.Equal(t, "*uuid.UUID", goTypeOptional)
	})
}

func TestTypeMapper_GetDefaultValue(t *testing.T) {
	tm := NewTypeMapper()

	tests := []struct {
		goType      string
		wantDefault string
	}{
		{"string", `""`},
		{"bool", "false"},
		{"int64", "0"},
		{"float64", "0"},
		{"time.Time", "time.Time{}"},
		{"json.RawMessage", "nil"},
		{"decimal.Decimal", "decimal.Zero"},
		{"*string", "nil"},
		{"[]string", "nil"},
		{"User", "nil"},
	}

	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			got := tm.GetDefaultValue(tt.goType)
			assert.Equal(t, tt.wantDefault, got)
		})
	}
}

func TestIsBuiltinType(t *testing.T) {
	tests := []struct {
		typeName string
		want     bool
	}{
		{"String", true},
		{"Int", true},
		{"Boolean", true},
		{"DateTime", true},
		{"Json", true},
		{"Bytes", true},
		{"Decimal", true},
		{"Float", true},
		{"BigInt", true},
		{"User", false},
		{"Post", false},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			got := IsBuiltinType(tt.typeName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetSQLType(t *testing.T) {
	tests := []struct {
		name       string
		prismaType string
		dialect    string
		want       string
	}{
		// PostgreSQL
		{"Postgres String", "String", "postgres", "TEXT"},
		{"Postgres Int", "Int", "postgres", "INTEGER"},
		{"Postgres DateTime", "DateTime", "postgres", "TIMESTAMP(3)"},
		{"Postgres Json", "Json", "postgres", "JSONB"},
		{"Postgres Bytes", "Bytes", "postgres", "BYTEA"},

		// MySQL
		{"MySQL String", "String", "mysql", "VARCHAR(191)"},
		{"MySQL Int", "Int", "mysql", "INT"},
		{"MySQL DateTime", "DateTime", "mysql", "DATETIME(3)"},
		{"MySQL Json", "Json", "mysql", "JSON"},
		{"MySQL Bytes", "Bytes", "mysql", "LONGBLOB"},

		// SQLite
		{"SQLite String", "String", "sqlite", "TEXT"},
		{"SQLite Int", "Int", "sqlite", "INTEGER"},
		{"SQLite Boolean", "Boolean", "sqlite", "INTEGER"},
		{"SQLite DateTime", "DateTime", "sqlite", "TEXT"},
		{"SQLite Json", "Json", "sqlite", "TEXT"},
		{"SQLite Bytes", "Bytes", "sqlite", "BLOB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSQLType(tt.prismaType, tt.dialect)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTypeMapper_RoundTrip(t *testing.T) {
	tm := NewTypeMapper()

	t.Run("Required types round-trip correctly", func(t *testing.T) {
		prismaTypes := []string{"String", "Int", "Boolean", "Float", "DateTime"}

		for _, original := range prismaTypes {
			// Prisma -> Go -> Prisma
			goType := tm.PrismaToGo(original, false, false)
			prisma, optional, list, err := tm.GoToPrisma(goType)

			require.NoError(t, err)
			assert.Equal(t, original, prisma, "Type should round-trip: %s", original)
			assert.False(t, optional)
			assert.False(t, list)
		}
	})

	t.Run("Optional types round-trip correctly", func(t *testing.T) {
		prismaTypes := []string{"String", "Int", "Boolean"}

		for _, original := range prismaTypes {
			// Prisma optional -> Go -> Prisma
			goType := tm.PrismaToGo(original, true, false)
			prisma, optional, list, err := tm.GoToPrisma(goType)

			require.NoError(t, err)
			assert.Equal(t, original, prisma)
			assert.True(t, optional, "Should preserve optional flag for: %s", original)
			assert.False(t, list)
		}
	})

	t.Run("List types round-trip correctly", func(t *testing.T) {
		prismaTypes := []string{"String", "Int", "User"}

		for _, original := range prismaTypes {
			// Prisma list -> Go -> Prisma
			goType := tm.PrismaToGo(original, false, true)
			prisma, optional, list, err := tm.GoToPrisma(goType)

			require.NoError(t, err)
			assert.Equal(t, original, prisma)
			assert.False(t, optional)
			assert.True(t, list, "Should preserve list flag for: %s", original)
		}
	})
}
