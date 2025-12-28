package schema

import (
	"testing"

	"github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

func TestUnsupportedType(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		wantErr bool
	}{
		{
			name: "Simple Unsupported type",
			schema: `
model Test {
  id       Int     @id
  location Unsupported("geography")
}
`,
			wantErr: false,
		},
		{
			name: "PostgreSQL tsvector",
			schema: `
model Article {
  id           Int     @id
  searchVector Unsupported("tsvector")?
}
`,
			wantErr: false,
		},
		{
			name: "PostGIS geography with params",
			schema: `
model Location {
  id       Int     @id
  point    Unsupported("geography(Point,4326)")
}
`,
			wantErr: false,
		},
		{
			name: "TimescaleDB hypertable",
			schema: `
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model SensorData {
  time     DateTime @id
  value    Float
  metadata Unsupported("jsonb")
}
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := ParseSchemaString("test.prisma", tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSchemaString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Verify we got models
			if len(schema.Tops) == 0 {
				t.Error("Expected at least one top-level item")
				return
			}

			// Check that Unsupported fields were parsed
			foundUnsupported := false
			for _, top := range schema.Tops {
				if model, ok := top.(*ast.Model); ok {
					for _, field := range model.Fields {
						if field.Type != nil && field.Type.IsUnsupported() {
							foundUnsupported = true
							t.Logf("Found Unsupported field: %s %s", field.GetName(), field.Type.String())
						}
					}
				}
			}

			if !foundUnsupported {
				t.Error("Expected to find at least one Unsupported field")
			}
		})
	}
}
