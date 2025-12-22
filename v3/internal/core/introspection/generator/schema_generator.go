// Package generator converts introspected database to Prisma schema.
package generator

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/introspection/domain"
	schemadomain "github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
)

// SchemaGenerator generates Prisma schemas from introspected databases.
type SchemaGenerator struct {
	typeMapper       domain.TypeMapper
	relationInferrer domain.RelationInferrer
}

// NewSchemaGenerator creates a new schema generator.
func NewSchemaGenerator(typeMapper domain.TypeMapper, relationInferrer domain.RelationInferrer) *SchemaGenerator {
	return &SchemaGenerator{
		typeMapper:       typeMapper,
		relationInferrer: relationInferrer,
	}
}

// Generate converts an introspected database to a Prisma schema.
func (g *SchemaGenerator) Generate(db *domain.IntrospectedDatabase, datasource schemadomain.Datasource) (*schemadomain.Schema, error) {
	schema := &schemadomain.Schema{
		Datasources: []schemadomain.Datasource{datasource},
		Models:      []schemadomain.Model{},
		Enums:       []schemadomain.Enum{},
	}

	// Generate models from tables
	for _, table := range db.Tables {
		model, err := g.generateModel(table)
		if err != nil {
			return nil, fmt.Errorf("failed to generate model for table %s: %w", table.Name, err)
		}
		schema.Models = append(schema.Models, model)
	}

	// Generate enums
	for _, enum := range db.Enums {
		schema.Enums = append(schema.Enums, schemadomain.Enum{
			Name:   toPascalCase(enum.Name),
			Values: enum.Values,
		})
	}

	// Infer and add relations
	if err := g.addRelations(schema, db); err != nil {
		return nil, fmt.Errorf("failed to add relations: %w", err)
	}

	return schema, nil
}

// generateModel generates a Prisma model from an introspected table.
func (g *SchemaGenerator) generateModel(table domain.IntrospectedTable) (schemadomain.Model, error) {
	model := schemadomain.Model{
		Name:       toPascalCase(table.Name),
		Fields:     []schemadomain.Field{},
		Indexes:    []schemadomain.Index{},
		Attributes: []schemadomain.Attribute{},
	}

	// Add @@map if table name differs from model name
	if table.Name != model.Name {
		model.Attributes = append(model.Attributes, schemadomain.Attribute{
			Name:      "map",
			Arguments: []interface{}{table.Name},
		})
	}

	// Generate fields from columns
	for _, col := range table.Columns {
		field, err := g.generateField(col, table)
		if err != nil {
			return model, fmt.Errorf("failed to generate field %s: %w", col.Name, err)
		}
		model.Fields = append(model.Fields, field)
	}

	// Generate indexes
	for _, idx := range table.Indexes {
		index := g.generateIndex(idx)
		model.Indexes = append(model.Indexes, index)
	}

	return model, nil
}

// generateField generates a Prisma field from an introspected column.
func (g *SchemaGenerator) generateField(col domain.IntrospectedColumn, table domain.IntrospectedTable) (schemadomain.Field, error) {
	// Map database type to Prisma type
	prismaType := g.typeMapper.MapToPrismaType(col.Type, col.IsNullable)

	field := schemadomain.Field{
		Name: col.Name,
		Type: schemadomain.FieldType{
			Name: strings.TrimSuffix(prismaType, "?"),
		},
		IsRequired:   !col.IsNullable,
		DefaultValue: nil,
		Attributes:   []schemadomain.Attribute{},
	}

	// Add @id if primary key
	if col.IsPrimaryKey || g.isPrimaryKeyColumn(col.Name, table.PrimaryKey) {
		field.Attributes = append(field.Attributes, schemadomain.Attribute{
			Name: "id",
		})

		// Add @default(autoincrement()) if auto-increment
		if col.IsAutoIncrement {
			field.Attributes = append(field.Attributes, schemadomain.Attribute{
				Name:      "default",
				Arguments: []interface{}{"autoincrement()"},
			})
		}
	}

	// Add @unique if unique
	if col.IsUnique {
		field.Attributes = append(field.Attributes, schemadomain.Attribute{
			Name: "unique",
		})
	}

	// Add @default if has default value
	if col.DefaultValue != nil && !col.IsAutoIncrement {
		field.Attributes = append(field.Attributes, schemadomain.Attribute{
			Name:      "default",
			Arguments: []interface{}{*col.DefaultValue},
		})
	}

	// Add @map if column name differs from field name
	if col.Name != field.Name {
		field.Attributes = append(field.Attributes, schemadomain.Attribute{
			Name:      "map",
			Arguments: []interface{}{col.Name},
		})
	}

	return field, nil
}

// generateIndex generates a Prisma index from an introspected index.
func (g *SchemaGenerator) generateIndex(idx domain.IntrospectedIndex) schemadomain.Index {
	return schemadomain.Index{
		Name:   idx.Name,
		Fields: idx.Columns,
		Unique: idx.IsUnique,
		Type:   schemadomain.IndexType(idx.Type),
	}
}

// addRelations infers relations and adds them to the schema.
func (g *SchemaGenerator) addRelations(schema *schemadomain.Schema, db *domain.IntrospectedDatabase) error {
	// Infer relations from foreign keys
	inferredRelations, err := g.relationInferrer.InferRelations(db)
	if err != nil {
		return err
	}

	// Build a map of models by name
	modelMap := make(map[string]*schemadomain.Model)
	for i := range schema.Models {
		modelMap[schema.Models[i].Name] = &schema.Models[i]
	}

	// Add relation fields to models
	for _, rel := range inferredRelations {
		fromModel := modelMap[toPascalCase(rel.FromTable)]
		toModel := modelMap[toPascalCase(rel.ToTable)]

		if fromModel == nil || toModel == nil {
			continue // Skip if models not found
		}

		// Create relation field
		relationField := schemadomain.Field{
			Name: rel.RelationName,
			Type: schemadomain.FieldType{
				Name:    toModel.Name,
				IsModel: true,
			},
			IsList: rel.RelationType == domain.OneToMany,
			Attributes: []schemadomain.Attribute{
				{
					Name: "relation",
					Arguments: []interface{}{
						fmt.Sprintf("fields: [%s]", strings.Join(rel.FromFields, ", ")),
						fmt.Sprintf("references: [%s]", strings.Join(rel.ToFields, ", ")),
					},
				},
			},
		}

		fromModel.Fields = append(fromModel.Fields, relationField)
	}

	return nil
}

// Helper functions

func (g *SchemaGenerator) isPrimaryKeyColumn(colName string, pk *domain.IntrospectedPrimaryKey) bool {
	if pk == nil {
		return false
	}
	for _, pkCol := range pk.Columns {
		if pkCol == colName {
			return true
		}
	}
	return false
}

func toPascalCase(s string) string {
	// Split by underscores and capitalize each word
	words := strings.Split(s, "_")
	for i := range words {
		if len(words[i]) > 0 {
			words[i] = strings.ToUpper(words[i][:1]) + strings.ToLower(words[i][1:])
		}
	}
	return strings.Join(words, "")
}

func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if len(pascal) > 0 {
		return strings.ToLower(pascal[:1]) + pascal[1:]
	}
	return pascal
}
