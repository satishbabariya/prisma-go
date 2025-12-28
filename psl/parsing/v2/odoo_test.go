package schema

import (
	"testing"

	"github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

func TestParseOdooSchema(t *testing.T) {
	input := `
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

generator client {
  provider = "prisma-client-js"
}

model ResPartner {
  id            Int      @id @default(autoincrement())
  name          String?  @map("name")
  displayName   String?  @map("display_name")
  isCompany     Boolean  @default(false) @map("is_company")
  parentId      Int?     @map("parent_id")
  email         String?
  active        Boolean  @default(true)
  createUid     Int      @map("create_uid")
  writeUid      Int      @map("write_uid")
  createDate    DateTime @default(now()) @map("create_date")
  writeDate     DateTime @updatedAt @map("write_date")

  parent        ResPartner?  @relation("PartnerHierarchy", fields: [parentId], references: [id])
  children      ResPartner[] @relation("PartnerHierarchy")

  @@map("res_partner")
  @@index([parentId])
  @@index([email])
}

enum SaleOrderState {
  DRAFT
  SENT
  SALE
  DONE
  CANCEL
}
`
	schema, err := ParseSchemaString("test_odoo.prisma", input)
	if err != nil {
		t.Fatalf("Failed to parse Odoo schema: %v", err)
	}

	// Validate parsed schema
	models := schema.Models()
	if len(models) != 1 {
		t.Fatalf("Expected 1 model, got %d", len(models))
	}

	model := models[0]
	if model.GetName() != "ResPartner" {
		t.Errorf("Expected model name 'ResPartner', got '%s'", model.GetName())
	}

	// Check field count
	if len(model.Fields) != 13 {
		t.Errorf("Expected 13 fields, got %d", len(model.Fields))
	}

	// Check enums
	enums := schema.Enums()
	if len(enums) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(enums))
	}

	enum := enums[0]
	if enum.GetName() != "SaleOrderState" {
		t.Errorf("Expected enum name 'SaleOrderState', got '%s'", enum.GetName())
	}

	if len(enum.Values) != 5 {
		t.Errorf("Expected 5 enum values, got %d", len(enum.Values))
	}
}

func TestParseComplexOdooSchema(t *testing.T) {
	input := `
model SaleOrder {
  id            Int      @id @default(autoincrement())
  name          String   @unique
  clientId      Int      @map("client_id")
  state         SaleOrderState @default(DRAFT)
  amountTotal   Float    @default(0) @map("amount_total")
  dateOrder     DateTime @default(now()) @map("date_order")
  createUid     Int      @map("create_uid")
  writeUid      Int      @map("write_uid")
  createDate    DateTime @default(now()) @map("create_date")
  writeDate     DateTime @updatedAt @map("write_date")

  client        ResPartner @relation("SaleOrderClient", fields: [clientId], references: [id])
  lines         SaleOrderLine[]

  @@map("sale_order")
  @@index([clientId])
  @@index([state])
}

model SaleOrderLine {
  id              Int      @id @default(autoincrement())
  orderId         Int      @map("order_id")
  name            String
  priceUnit       Float    @default(0) @map("price_unit")
  discount        Float    @default(0)
  createUid       Int      @map("create_uid")
  writeUid        Int      @map("write_uid")
  createDate      DateTime @default(now()) @map("create_date")
  writeDate       DateTime @updatedAt @map("write_date")

  order           SaleOrder @relation(fields: [orderId], references: [id])

  @@map("sale_order_line")
  @@index([orderId])
}

enum SaleOrderState {
  DRAFT
  SENT
  SALE
  DONE
  CANCEL
}
`
	schema, err := ParseSchemaString("test_complex_odoo.prisma", input)
	if err != nil {
		t.Fatalf("Failed to parse complex Odoo schema: %v", err)
	}

	// Validate models
	models := schema.Models()
	if len(models) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(models))
	}

	// Check SaleOrder model
	var saleOrder *ast.Model
	for _, model := range models {
		if model.GetName() == "SaleOrder" {
			saleOrder = model
			break
		}
	}

	if saleOrder == nil {
		t.Fatal("SaleOrder model not found")
	}

	// Check for relationship fields
	foundRelation := false
	foundIndex := false
	for _, field := range saleOrder.Fields {
		if field.GetName() == "client" && field.GetTypeName() == "ResPartner" {
			foundRelation = true
		}
	}

	// Check block attributes (indexes)
	for _, attr := range saleOrder.BlockAttributes {
		if attr.GetName() == "index" {
			foundIndex = true
		}
	}

	if !foundRelation {
		t.Error("Expected to find client relation field")
	}

	if !foundIndex {
		t.Error("Expected to find @@index block attribute")
	}
}

func TestParseWithAttributesAndArguments(t *testing.T) {
	input := `
model Product {
  id          Int      @id @default(autoincrement())
  code        String   @unique @map("product_code")
  active      Boolean  @default(true)
  price       Float    @default(0.0)
  created     DateTime @default(now())

  @@map("product_table")
  @@index([code])
  @@unique([price, active])
}
`
	schema, err := ParseSchemaString("test_attributes.prisma", input)
	if err != nil {
		t.Fatalf("Failed to parse schema with attributes: %v", err)
	}

	models := schema.Models()
	if len(models) != 1 {
		t.Fatalf("Expected 1 model, got %d", len(models))
	}

	model := models[0]

	// Check field attributes
	var idField *ast.Field
	for _, field := range model.Fields {
		if field.GetName() == "id" {
			idField = field
			break
		}
	}

	if idField == nil {
		t.Fatal("id field not found")
	}

	// Should have @id and @default attributes
	hasIdAttr := false
	hasDefaultAttr := false
	for _, attr := range idField.Attributes {
		if attr.GetName() == "id" {
			hasIdAttr = true
		}
		if attr.GetName() == "default" {
			hasDefaultAttr = true
		}
	}

	if !hasIdAttr {
		t.Error("Expected @id attribute on id field")
	}

	if !hasDefaultAttr {
		t.Error("Expected @default attribute on id field")
	}

	// Check block attributes
	if len(model.BlockAttributes) != 3 {
		t.Errorf("Expected 3 block attributes, got %d", len(model.BlockAttributes))
	}

	hasMapAttr := false
	hasIndexAttr := false
	hasUniqueAttr := false
	for _, attr := range model.BlockAttributes {
		if attr.GetName() == "map" {
			hasMapAttr = true
		}
		if attr.GetName() == "index" {
			hasIndexAttr = true
		}
		if attr.GetName() == "unique" {
			hasUniqueAttr = true
		}
	}

	if !hasMapAttr {
		t.Error("Expected @@map block attribute")
	}

	if !hasIndexAttr {
		t.Error("Expected @@index block attribute")
	}

	if !hasUniqueAttr {
		t.Error("Expected @@unique block attribute")
	}
}
