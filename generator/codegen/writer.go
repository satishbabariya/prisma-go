package codegen

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/satishbabariya/prisma-go/internal/debug"
)

// getColumnType returns the column type for a Go type
func getColumnType(goType string) string {
	baseType := strings.TrimPrefix(goType, "*")
	baseType = strings.TrimPrefix(baseType, "[]")

	switch baseType {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return "columns.IntColumn"
	case "string":
		if strings.HasPrefix(goType, "*") {
			return "columns.NullableStringColumn"
		}
		return "columns.StringColumn"
	case "bool":
		return "columns.BoolColumn"
	case "time.Time":
		return "columns.DateTimeColumn"
	case "float32", "float64":
		return "columns.IntColumn" // Use IntColumn for now, could add FloatColumn later
	default:
		return "columns.StringColumn" // Default fallback
	}
}

// getColumnConstructor returns the constructor function name for a column type
func getColumnConstructor(columnType string) string {
	switch columnType {
	case "columns.IntColumn":
		return "columns.NewIntColumn"
	case "columns.StringColumn":
		return "columns.NewStringColumn"
	case "columns.NullableStringColumn":
		return "columns.NewNullableStringColumn"
	case "columns.BoolColumn":
		return "columns.NewBoolColumn"
	case "columns.DateTimeColumn":
		return "columns.NewDateTimeColumn"
	default:
		return "columns.NewStringColumn"
	}
}

// AST helper functions for building Go AST nodes

// newFile creates a new AST file with package declaration
func newFile(packageName string) *ast.File {
	return &ast.File{
		Name:  ast.NewIdent(packageName),
		Decls: []ast.Decl{},
	}
}

// addComment adds a comment to the file
func addComment(file *ast.File, text string) {
	if file.Comments == nil {
		file.Comments = []*ast.CommentGroup{}
	}
	file.Comments = append(file.Comments, &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// " + text},
		},
	})
}

// addCommentGroup adds a comment group to the file
func addCommentGroup(file *ast.File, text string) {
	if file.Comments == nil {
		file.Comments = []*ast.CommentGroup{}
	}
	file.Comments = append(file.Comments, &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// " + text},
		},
	})
}

// parseType parses a Go type string into an AST expression
func parseType(typeStr string) ast.Expr {
	// Handle pointer types
	if strings.HasPrefix(typeStr, "*") {
		return &ast.StarExpr{
			X: parseType(typeStr[1:]),
		}
	}
	// Handle slice types
	if strings.HasPrefix(typeStr, "[]") {
		return &ast.ArrayType{
			Elt: parseType(typeStr[2:]),
		}
	}
	// Handle qualified types (e.g., "time.Time", "columns.IntColumn")
	if strings.Contains(typeStr, ".") {
		parts := strings.Split(typeStr, ".")
		return &ast.SelectorExpr{
			X:   ast.NewIdent(parts[0]),
			Sel: ast.NewIdent(parts[1]),
		}
	}
	// Simple identifier
	return ast.NewIdent(typeStr)
}

// parseTypeFromString parses a type string by parsing a minimal Go file
func parseTypeFromString(typeStr string) ast.Expr {
	// Create a minimal Go file to parse the type
	src := fmt.Sprintf("package p\nvar x %s", typeStr)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		// Fallback to simple parsing
		return parseType(typeStr)
	}
	if len(f.Decls) > 0 {
		if genDecl, ok := f.Decls[0].(*ast.GenDecl); ok && len(genDecl.Specs) > 0 {
			if valueSpec, ok := genDecl.Specs[0].(*ast.ValueSpec); ok && len(valueSpec.Names) > 0 {
				if valueSpec.Type != nil {
					return valueSpec.Type
				}
			}
		}
	}
	return parseType(typeStr)
}

// newImportSpec creates a new import spec
func newImportSpec(path string) *ast.ImportSpec {
	return &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf("%q", path),
		},
	}
}

// addImports adds import declarations to the file
func addImports(file *ast.File, imports []string) {
	if len(imports) == 0 {
		return
	}
	specs := make([]ast.Spec, len(imports))
	for i, imp := range imports {
		specs[i] = newImportSpec(imp)
	}
	file.Decls = append(file.Decls, &ast.GenDecl{
		Tok:   token.IMPORT,
		Specs: specs,
	})
}

// newStructType creates a new struct type
func newStructType(fields []*ast.Field) *ast.StructType {
	return &ast.StructType{
		Fields: &ast.FieldList{
			List: fields,
		},
	}
}

// newField creates a new struct field
func newField(name string, typeExpr ast.Expr, tag string) *ast.Field {
	field := &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(name)},
		Type:  typeExpr,
	}
	if tag != "" {
		field.Tag = &ast.BasicLit{
			Kind:  token.STRING,
			Value: tag,
		}
	}
	return field
}

// newTypeDecl creates a new type declaration
func newTypeDecl(name string, doc string, typeExpr ast.Expr) *ast.GenDecl {
	decl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(name),
				Type: typeExpr,
			},
		},
	}
	if doc != "" {
		decl.Doc = &ast.CommentGroup{
			List: []*ast.Comment{
				{Text: "// " + doc},
			},
		}
	}
	return decl
}

// newVarDecl creates a new variable declaration
func newVarDecl(name string, typeExpr ast.Expr, value ast.Expr) *ast.GenDecl {
	spec := &ast.ValueSpec{
		Names: []*ast.Ident{ast.NewIdent(name)},
	}
	if typeExpr != nil {
		spec.Type = typeExpr
	}
	if value != nil {
		spec.Values = []ast.Expr{value}
	}
	return &ast.GenDecl{
		Tok:   token.VAR,
		Specs: []ast.Spec{spec},
	}
}

// newFuncDecl creates a new function declaration
func newFuncDecl(name string, doc string, recv *ast.FieldList, params *ast.FieldList, results *ast.FieldList, body *ast.BlockStmt) *ast.FuncDecl {
	decl := &ast.FuncDecl{
		Name: ast.NewIdent(name),
		Type: &ast.FuncType{
			Params:  params,
			Results: results,
		},
		Body: body,
	}
	if recv != nil {
		decl.Recv = recv
	}
	if doc != "" {
		decl.Doc = &ast.CommentGroup{
			List: []*ast.Comment{
				{Text: "// " + doc},
			},
		}
	}
	return decl
}

// newMethod creates a new method declaration
func newMethod(recvType string, recvName string, name string, doc string, params *ast.FieldList, results *ast.FieldList, body *ast.BlockStmt) *ast.FuncDecl {
	recv := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{ast.NewIdent(recvName)},
				Type:  parseTypeFromString(recvType),
			},
		},
	}
	return newFuncDecl(name, doc, recv, params, results, body)
}

// newReturnStmt creates a new return statement
func newReturnStmt(exprs ...ast.Expr) *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Results: exprs,
	}
}

// newStringLit creates a string literal expression
func newStringLit(s string) *ast.BasicLit {
	return &ast.BasicLit{
		Kind:  token.STRING,
		Value: fmt.Sprintf("%q", s),
	}
}

// newBoolLit creates a boolean literal expression
func newBoolLit(b bool) *ast.Ident {
	if b {
		return ast.NewIdent("true")
	}
	return ast.NewIdent("false")
}

// newCallExpr creates a function call expression
func newCallExpr(fun ast.Expr, args ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun:  fun,
		Args: args,
	}
}

// newSelectorExpr creates a selector expression (e.g., a.B)
func newSelectorExpr(x ast.Expr, sel string) *ast.SelectorExpr {
	return &ast.SelectorExpr{
		X:   x,
		Sel: ast.NewIdent(sel),
	}
}

// newCompositeLit creates a composite literal expression
func newCompositeLit(typ ast.Expr, elts []ast.Expr) *ast.CompositeLit {
	return &ast.CompositeLit{
		Type: typ,
		Elts: elts,
	}
}

// newKeyValueExpr creates a key-value expression for struct literals (key is identifier)
func newKeyValueExpr(key string, value ast.Expr) *ast.KeyValueExpr {
	return &ast.KeyValueExpr{
		Key:   ast.NewIdent(key),
		Value: value,
	}
}

// newMapKeyValueExpr creates a key-value expression for map literals (key is expression)
func newMapKeyValueExpr(key ast.Expr, value ast.Expr) *ast.KeyValueExpr {
	return &ast.KeyValueExpr{
		Key:   key,
		Value: value,
	}
}

// newIfStmt creates an if statement
func newIfStmt(cond ast.Expr, body *ast.BlockStmt, elseStmt ast.Stmt) *ast.IfStmt {
	return &ast.IfStmt{
		Cond: cond,
		Body: body,
		Else: elseStmt,
	}
}

// newAssignStmt creates an assignment statement
func newAssignStmt(lhs []ast.Expr, tok token.Token, rhs []ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: lhs,
		Tok: tok,
		Rhs: rhs,
	}
}

// newBlockStmt creates a block statement
func newBlockStmt(stmts ...ast.Stmt) *ast.BlockStmt {
	return &ast.BlockStmt{
		List: stmts,
	}
}

// writeASTFile writes an AST file to disk with proper formatting
func writeASTFile(file *ast.File, filePath string) error {
	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create file
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Format and write the file
	debug.Debug("Formatting AST", "decl_count", len(file.Decls))
	formatStart := time.Now()
	fset := token.NewFileSet()
	if err := format.Node(f, fset, file); err != nil {
		return fmt.Errorf("failed to format file: %w", err)
	}
	debug.Debug("AST formatted successfully", "elapsed", time.Since(formatStart))

	return nil
}

// GenerateModelsFile generates the models.go file using AST
func GenerateModelsFile(models []ModelInfo, outputDir string) error {
	// Create AST file
	file := newFile("generated")

	// Add header comment
	file.Comments = []*ast.CommentGroup{
		{
			List: []*ast.Comment{
				{Text: "// Code generated by prisma-go. DO NOT EDIT."},
			},
		},
	}

	// Check for time.Time usage
	hasDateTime := false
	for _, model := range models {
		for _, field := range model.Fields {
			if strings.Contains(field.GoType, "time.Time") {
				hasDateTime = true
				break
			}
		}
		if hasDateTime {
			break
		}
	}

	// Add imports
	imports := []string{"github.com/satishbabariya/prisma-go/query/columns"}
	if hasDateTime {
		imports = append([]string{"time"}, imports...)
	}
	addImports(file, imports)

	// Generate model structs
	for _, model := range models {
		// Create struct fields
		fields := make([]*ast.Field, 0, len(model.Fields))
		for _, field := range model.Fields {
			fieldAST := newField(field.GoName, parseTypeFromString(field.GoType), field.Tags)
			fields = append(fields, fieldAST)
		}

		// Create struct type declaration
		structType := newStructType(fields)
		typeDecl := newTypeDecl(model.Name, fmt.Sprintf("%s represents the %s model", model.Name, model.Name), structType)
		file.Decls = append(file.Decls, typeDecl)

		// Add TableName method
		recv := &ast.FieldList{
			List: []*ast.Field{
				{
					Type: ast.NewIdent(model.Name),
				},
			},
		}
		params := &ast.FieldList{}
		results := &ast.FieldList{
			List: []*ast.Field{
				{
					Type: ast.NewIdent("string"),
				},
			},
		}
		body := newBlockStmt(
			newReturnStmt(newStringLit(model.TableName)),
		)
		method := newFuncDecl("TableName", fmt.Sprintf("TableName returns the table name for %s", model.Name), recv, params, results, body)
		file.Decls = append(file.Decls, method)
	}

	// Generate column structs and instances
	for _, model := range models {
		modelName := model.Name
		tableName := model.TableName

		// Generate column struct
		columnFields := make([]*ast.Field, 0)
		for _, field := range model.Fields {
			if !field.IsRelation {
				columnType := getColumnType(field.GoType)
				fieldAST := newField(field.GoName, parseTypeFromString(columnType), "")
				columnFields = append(columnFields, fieldAST)
			}
		}

		if len(columnFields) > 0 {
			columnStructType := newStructType(columnFields)
			columnTypeName := modelName + "Columns"
			typeDecl := newTypeDecl(columnTypeName, fmt.Sprintf("%s provides type-safe column references for %s", columnTypeName, modelName), columnStructType)
			file.Decls = append(file.Decls, typeDecl)

			// Generate column instance
			instanceFields := make([]ast.Expr, 0)
			for _, field := range model.Fields {
				if !field.IsRelation {
					fieldName := field.GoName
					columnName := toSnakeCase(field.Name)
					columnType := getColumnType(field.GoType)
					constructor := getColumnConstructor(columnType)

					// Create constructor call: columns.NewIntColumn("table", "column")
					constructorExpr := parseTypeFromString(constructor)
					callExpr := newCallExpr(constructorExpr, newStringLit(tableName), newStringLit(columnName))
					instanceFields = append(instanceFields, newKeyValueExpr(fieldName, callExpr))
				}
			}

			if len(instanceFields) > 0 {
				varType := parseTypeFromString(columnTypeName)
				compositeLit := newCompositeLit(varType, instanceFields)
				varName := modelName + "ColumnsInstance"
				varDecl := newVarDecl(varName, nil, compositeLit)
				varDecl.Doc = &ast.CommentGroup{
					List: []*ast.Comment{
						{Text: fmt.Sprintf("// %s provides type-safe column references for %s", varName, modelName)},
					},
				}
				file.Decls = append(file.Decls, varDecl)
			}
		}
	}

	// Write AST file to disk
	filePath := filepath.Join(outputDir, "models.go")
	return writeASTFile(file, filePath)
}

// buildPrismaClientStruct builds the PrismaClient struct AST
func buildPrismaClientStruct(models []ModelInfo) *ast.GenDecl {
	fields := []*ast.Field{
		{
			Type: &ast.StarExpr{
				X: newSelectorExpr(ast.NewIdent("client"), "PrismaClient"),
			},
		},
	}
	for _, model := range models {
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(model.Name)},
			Type: &ast.StarExpr{
				X: ast.NewIdent(model.Name + "Client"),
			},
		})
	}
	return newTypeDecl("PrismaClient", "PrismaClient is the main client for database operations", newStructType(fields))
}

// buildNewPrismaClientFunc builds the NewPrismaClient function AST
func buildNewPrismaClientFunc(models []ModelInfo, provider string) *ast.FuncDecl {
	params := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{ast.NewIdent("connectionString")},
				Type:  ast.NewIdent("string"),
			},
		},
	}
	results := &ast.FieldList{
		List: []*ast.Field{
			{
				Type: &ast.StarExpr{
					X: ast.NewIdent("PrismaClient"),
				},
			},
			{
				Type: ast.NewIdent("error"),
			},
		},
	}

	// Build function body
	bodyStmts := []ast.Stmt{
		// baseClient, err := client.NewPrismaClient(provider, connectionString)
		newAssignStmt(
			[]ast.Expr{ast.NewIdent("baseClient"), ast.NewIdent("err")},
			token.DEFINE,
			[]ast.Expr{
				newCallExpr(
					newSelectorExpr(ast.NewIdent("client"), "NewPrismaClient"),
					newStringLit(provider),
					ast.NewIdent("connectionString"),
				),
			},
		),
		// if err != nil { return nil, err }
		newIfStmt(
			&ast.BinaryExpr{
				X:  ast.NewIdent("err"),
				Op: token.NEQ,
				Y:  ast.NewIdent("nil"),
			},
			newBlockStmt(
				newReturnStmt(ast.NewIdent("nil"), ast.NewIdent("err")),
			),
			nil,
		),
		// c := &PrismaClient{PrismaClient: baseClient}
		newAssignStmt(
			[]ast.Expr{ast.NewIdent("c")},
			token.DEFINE,
			[]ast.Expr{
				&ast.UnaryExpr{
					Op: token.AND,
					X: newCompositeLit(
						ast.NewIdent("PrismaClient"),
						[]ast.Expr{
							newKeyValueExpr("PrismaClient", ast.NewIdent("baseClient")),
						},
					),
				},
			},
		),
		// exec := executor.NewExecutor(baseClient.DB(), provider)
		newAssignStmt(
			[]ast.Expr{ast.NewIdent("exec")},
			token.DEFINE,
			[]ast.Expr{
				newCallExpr(
					newSelectorExpr(ast.NewIdent("executor"), "NewExecutor"),
					newCallExpr(
						newSelectorExpr(ast.NewIdent("baseClient"), "DB"),
					),
					newStringLit(provider),
				),
			},
		),
	}

	// Add model client initialization
	for _, model := range models {
		// Build relations map
		relationElts := []ast.Expr{}
		for _, rel := range model.Relations {
			if rel.ForeignKey != "" {
				relationElts = append(relationElts, newMapKeyValueExpr(
					newStringLit(rel.FieldName),
					newCompositeLit(
						newSelectorExpr(ast.NewIdent("executor"), "RelationMetadata"),
						[]ast.Expr{
							newKeyValueExpr("RelatedTable", newStringLit(rel.ForeignKeyTable)),
							newKeyValueExpr("ForeignKey", newStringLit(toSnakeCase(rel.ForeignKey))),
							newKeyValueExpr("LocalKey", newStringLit(toSnakeCase(rel.LocalKey))),
							newKeyValueExpr("IsList", newBoolLit(rel.IsList)),
						},
					),
				))
			}
		}

		// c.ModelName = &ModelNameClient{...}
		clientFields := []ast.Expr{
			newKeyValueExpr("client", ast.NewIdent("baseClient")),
			newKeyValueExpr("executor", ast.NewIdent("exec")),
			newKeyValueExpr("table", newStringLit(model.TableName)),
			newKeyValueExpr("relations", &ast.CompositeLit{
				Type: &ast.MapType{
					Key:   ast.NewIdent("string"),
					Value: newSelectorExpr(ast.NewIdent("executor"), "RelationMetadata"),
				},
				Elts: relationElts,
			}),
		}
		bodyStmts = append(bodyStmts, newAssignStmt(
			[]ast.Expr{newSelectorExpr(ast.NewIdent("c"), model.Name)},
			token.ASSIGN,
			[]ast.Expr{
				&ast.UnaryExpr{
					Op: token.AND,
					X: newCompositeLit(
						ast.NewIdent(model.Name+"Client"),
						clientFields,
					),
				},
			},
		))
	}

	// return c, nil
	bodyStmts = append(bodyStmts, newReturnStmt(ast.NewIdent("c"), ast.NewIdent("nil")))

	return newFuncDecl("NewPrismaClient", "NewPrismaClient creates a new Prisma client", nil, params, results, newBlockStmt(bodyStmts...))
}

// buildRawSQLMethods builds the raw SQL method AST nodes
func buildRawSQLMethods() []ast.Decl {
	methods := []ast.Decl{}

	// Raw method
	recv := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{ast.NewIdent("c")},
				Type: &ast.StarExpr{
					X: ast.NewIdent("PrismaClient"),
				},
			},
		},
	}
	params := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
			{Names: []*ast.Ident{ast.NewIdent("query")}, Type: ast.NewIdent("string")},
			{Names: []*ast.Ident{ast.NewIdent("args")}, Type: &ast.Ellipsis{Elt: ast.NewIdent("interface{}")}},
		},
	}
	results := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("sql"), "Rows")}},
			{Type: ast.NewIdent("error")},
		},
	}
	body := newBlockStmt(
		newReturnStmt(
			newCallExpr(
				newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "PrismaClient"), "Raw"),
				ast.NewIdent("ctx"),
				ast.NewIdent("query"),
				&ast.Ellipsis{Elt: ast.NewIdent("args")},
			),
		),
	)
	methods = append(methods, newFuncDecl("Raw", "Raw executes a raw SQL query and returns the result", recv, params, results, body))

	// RawScan method
	params = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
			{Names: []*ast.Ident{ast.NewIdent("dest")}, Type: ast.NewIdent("interface{}")},
			{Names: []*ast.Ident{ast.NewIdent("query")}, Type: ast.NewIdent("string")},
			{Names: []*ast.Ident{ast.NewIdent("args")}, Type: &ast.Ellipsis{Elt: ast.NewIdent("interface{}")}},
		},
	}
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: ast.NewIdent("error")},
		},
	}
	body = newBlockStmt(
		newReturnStmt(
			newCallExpr(
				newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "PrismaClient"), "RawScan"),
				ast.NewIdent("ctx"),
				ast.NewIdent("dest"),
				ast.NewIdent("query"),
				&ast.Ellipsis{Elt: ast.NewIdent("args")},
			),
		),
	)
	methods = append(methods, newFuncDecl("RawScan", "RawScan executes a raw SQL query and scans the results into the destination", recv, params, results, body))

	// RawExec method
	params = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
			{Names: []*ast.Ident{ast.NewIdent("query")}, Type: ast.NewIdent("string")},
			{Names: []*ast.Ident{ast.NewIdent("args")}, Type: &ast.Ellipsis{Elt: ast.NewIdent("interface{}")}},
		},
	}
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: newSelectorExpr(ast.NewIdent("sql"), "Result")},
			{Type: ast.NewIdent("error")},
		},
	}
	body = newBlockStmt(
		newReturnStmt(
			newCallExpr(
				newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "PrismaClient"), "RawExec"),
				ast.NewIdent("ctx"),
				ast.NewIdent("query"),
				&ast.Ellipsis{Elt: ast.NewIdent("args")},
			),
		),
	)
	methods = append(methods, newFuncDecl("RawExec", "RawExec executes a raw SQL statement (INSERT, UPDATE, DELETE) and returns the result", recv, params, results, body))

	// RawQuery method
	params = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
			{Names: []*ast.Ident{ast.NewIdent("query")}, Type: ast.NewIdent("string")},
			{Names: []*ast.Ident{ast.NewIdent("args")}, Type: &ast.Ellipsis{Elt: ast.NewIdent("interface{}")}},
		},
	}
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("sql"), "Rows")}},
			{Type: ast.NewIdent("error")},
		},
	}
	body = newBlockStmt(
		newReturnStmt(
			newCallExpr(
				newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "PrismaClient"), "RawQuery"),
				ast.NewIdent("ctx"),
				ast.NewIdent("query"),
				&ast.Ellipsis{Elt: ast.NewIdent("args")},
			),
		),
	)
	methods = append(methods, newFuncDecl("RawQuery", "RawQuery executes a raw SQL query and returns rows", recv, params, results, body))

	return methods
}

// Helper functions for building model client methods

// buildFindManyMethods builds FindMany and FindManyWhere methods
func buildFindManyMethods(model ModelInfo) []ast.Decl {
	var decls []ast.Decl
	modelName := model.Name

	recv := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("c")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "Client")}},
		},
	}

	// FindMany(ctx context.Context) ([]Model, error)
	params := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
		},
	}
	results := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.ArrayType{Elt: ast.NewIdent(modelName)}},
			{Type: ast.NewIdent("error")},
		},
	}
	body := newBlockStmt(
		newReturnStmt(
			newCallExpr(
				newSelectorExpr(ast.NewIdent("c"), "FindManyWhere"),
				ast.NewIdent("ctx"),
				ast.NewIdent("nil"),
			),
		),
	)
	decls = append(decls, newFuncDecl("FindMany", "FindMany retrieves multiple "+modelName+" records", recv, params, results, body))

	// FindManyWhere(ctx context.Context, where *builder.WhereBuilder) ([]Model, error)
	params = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
			{Names: []*ast.Ident{ast.NewIdent("where")}, Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "WhereBuilder")}},
		},
	}
	body = newBlockStmt(
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("results")},
						Type:  &ast.ArrayType{Elt: ast.NewIdent(modelName)},
					},
				},
			},
		},
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("whereClause")},
						Type:  &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("sqlgen"), "WhereClause")},
					},
				},
			},
		},
		newIfStmt(
			&ast.BinaryExpr{X: ast.NewIdent("where"), Op: token.NEQ, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{ast.NewIdent("whereClause")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("where"), "Build"))},
				),
			),
			nil,
		),
		newIfStmt(
			&ast.BinaryExpr{
				X: &ast.CallExpr{
					Fun: newSelectorExpr(
						newSelectorExpr(ast.NewIdent("c"), "executor"),
						"FindManyWithRelations",
					),
					Args: []ast.Expr{
						ast.NewIdent("ctx"),
						newSelectorExpr(ast.NewIdent("c"), "table"),
						ast.NewIdent("nil"),
						ast.NewIdent("whereClause"),
						ast.NewIdent("nil"),
						ast.NewIdent("nil"),
						ast.NewIdent("nil"),
						ast.NewIdent("nil"),
						newSelectorExpr(ast.NewIdent("c"), "relations"),
						&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("results")},
					},
				},
				Op: token.NEQ,
				Y:  ast.NewIdent("nil"),
			},
			newBlockStmt(
				newReturnStmt(ast.NewIdent("nil"), ast.NewIdent("err")),
			),
			nil,
		),
		newReturnStmt(ast.NewIdent("results"), ast.NewIdent("nil")),
	)
	decls = append(decls, newFuncDecl("FindManyWhere", "FindManyWhere retrieves multiple "+modelName+" records with WHERE clause", recv, params, results, body))

	return decls
}

// buildQueryMethod builds the Query() method
func buildQueryMethod(model ModelInfo) []ast.Decl {
	modelName := model.Name
	recv := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("c")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "Client")}},
		},
	}
	params := &ast.FieldList{}
	results := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
		},
	}
	body := newBlockStmt(
		newReturnStmt(
			&ast.UnaryExpr{
				Op: token.AND,
				X: newCompositeLit(
					ast.NewIdent(modelName+"QueryBuilder"),
					[]ast.Expr{
						newKeyValueExpr("client", ast.NewIdent("c")),
					},
				),
			},
		),
	)
	return []ast.Decl{newFuncDecl("Query", "Query starts building a query with WHERE, ORDER BY, LIMIT, OFFSET", recv, params, results, body)}
}

// buildFindFirstMethods builds FindFirst and FindFirstWhere methods
func buildFindFirstMethods(model ModelInfo) []ast.Decl {
	var decls []ast.Decl
	modelName := model.Name

	recv := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("c")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "Client")}},
		},
	}

	// FindFirst(ctx context.Context) (*Model, error)
	params := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
		},
	}
	results := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName)}},
			{Type: ast.NewIdent("error")},
		},
	}
	body := newBlockStmt(
		newReturnStmt(
			newCallExpr(
				newSelectorExpr(ast.NewIdent("c"), "FindFirstWhere"),
				ast.NewIdent("ctx"),
				ast.NewIdent("nil"),
			),
		),
	)
	decls = append(decls, newFuncDecl("FindFirst", fmt.Sprintf("FindFirst retrieves the first %s record", modelName), recv, params, results, body))

	// FindFirstWhere(ctx context.Context, where *builder.WhereBuilder) (*Model, error)
	params = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
			{Names: []*ast.Ident{ast.NewIdent("where")}, Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "WhereBuilder")}},
		},
	}
	body = newBlockStmt(
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("result")},
						Type:  ast.NewIdent(modelName),
					},
				},
			},
		},
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("whereClause")},
						Type:  &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("sqlgen"), "WhereClause")},
					},
				},
			},
		},
		newIfStmt(
			&ast.BinaryExpr{X: ast.NewIdent("where"), Op: token.NEQ, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{ast.NewIdent("whereClause")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("where"), "Build"))},
				),
			),
			nil,
		),
		newIfStmt(
			&ast.BinaryExpr{
				X: &ast.CallExpr{
					Fun: newSelectorExpr(
						newSelectorExpr(ast.NewIdent("c"), "executor"),
						"FindFirstWithRelations",
					),
					Args: []ast.Expr{
						ast.NewIdent("ctx"),
						newSelectorExpr(ast.NewIdent("c"), "table"),
						ast.NewIdent("nil"),
						ast.NewIdent("whereClause"),
						ast.NewIdent("nil"),
						ast.NewIdent("nil"),
						newSelectorExpr(ast.NewIdent("c"), "relations"),
						&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("result")},
					},
				},
				Op: token.NEQ,
				Y:  ast.NewIdent("nil"),
			},
			newBlockStmt(
				newReturnStmt(ast.NewIdent("nil"), ast.NewIdent("err")),
			),
			nil,
		),
		newReturnStmt(&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("result")}, ast.NewIdent("nil")),
	)
	decls = append(decls, newFuncDecl("FindFirstWhere", fmt.Sprintf("FindFirstWhere retrieves the first %s record with WHERE clause", modelName), recv, params, results, body))

	return decls
}

// buildWhereBuilderExecuteMethods builds Execute and ExecuteFirst methods for WhereBuilder
func buildWhereBuilderExecuteMethods(model ModelInfo) []ast.Decl {
	var decls []ast.Decl
	modelName := model.Name

	recv := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("w")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "WhereBuilder")}},
		},
	}

	// Execute(ctx context.Context) ([]Model, error)
	params := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
		},
	}
	results := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.ArrayType{Elt: ast.NewIdent(modelName)}},
			{Type: ast.NewIdent("error")},
		},
	}
	body := newBlockStmt(
		newReturnStmt(
			newCallExpr(
				newSelectorExpr(newSelectorExpr(ast.NewIdent("w"), "client"), "FindManyWhere"),
				ast.NewIdent("ctx"),
				newSelectorExpr(ast.NewIdent("w"), "WhereBuilder"),
			),
		),
	)
	decls = append(decls, newFuncDecl("Execute", "Execute executes the query with the WHERE clause", recv, params, results, body))

	// ExecuteFirst(ctx context.Context) (*Model, error)
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName)}},
			{Type: ast.NewIdent("error")},
		},
	}
	body = newBlockStmt(
		newReturnStmt(
			newCallExpr(
				newSelectorExpr(newSelectorExpr(ast.NewIdent("w"), "client"), "FindFirstWhere"),
				ast.NewIdent("ctx"),
				newSelectorExpr(ast.NewIdent("w"), "WhereBuilder"),
			),
		),
	)
	decls = append(decls, newFuncDecl("ExecuteFirst", "ExecuteFirst executes the query and returns the first result", recv, params, results, body))

	return decls
}

// buildQueryBuilderMethods builds basic QueryBuilder methods (Where, Limit, Offset, Execute, ExecuteFirst)
func buildQueryBuilderMethods(model ModelInfo) []ast.Decl {
	var decls []ast.Decl
	modelName := model.Name

	recv := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("q")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
		},
	}

	// Where() *QueryBuilder
	params := &ast.FieldList{}
	results := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
		},
	}
	body := newBlockStmt(
		newIfStmt(
			&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "where"), Op: token.EQL, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "where")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewWhereBuilder"))},
				),
			),
			nil,
		),
		newReturnStmt(ast.NewIdent("q")),
	)
	decls = append(decls, newFuncDecl("Where", "Where starts building a WHERE clause", recv, params, results, body))

	// WhereCondition(cond columns.Condition) *QueryBuilder
	params = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("cond")}, Type: newSelectorExpr(ast.NewIdent("columns"), "Condition")},
		},
	}
	body = newBlockStmt(
		newIfStmt(
			&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "where"), Op: token.EQL, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "where")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewWhereBuilder"))},
				),
			),
			nil,
		),
		&ast.ExprStmt{
			X: newCallExpr(
				newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "where"), "AddCondition"),
				ast.NewIdent("cond"),
			),
		},
		newReturnStmt(ast.NewIdent("q")),
	)
	decls = append(decls, newFuncDecl("WhereCondition", "WhereCondition adds a column-based condition (type-safe)", recv, params, results, body))

	// WhereConditions(conds []columns.Condition) *QueryBuilder
	params = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("conds")}, Type: &ast.ArrayType{Elt: newSelectorExpr(ast.NewIdent("columns"), "Condition")}},
		},
	}
	body = newBlockStmt(
		newIfStmt(
			&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "where"), Op: token.EQL, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "where")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewWhereBuilder"))},
				),
			),
			nil,
		),
		&ast.ExprStmt{
			X: newCallExpr(
				newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "where"), "AddConditions"),
				ast.NewIdent("conds"),
			),
		},
		newReturnStmt(ast.NewIdent("q")),
	)
	decls = append(decls, newFuncDecl("WhereConditions", "WhereConditions adds multiple column-based conditions (type-safe)", recv, params, results, body))

	// Limit(limit int) *QueryBuilder
	params = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("limit")}, Type: ast.NewIdent("int")},
		},
	}
	body = newBlockStmt(
		newAssignStmt(
			[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "limit")},
			token.ASSIGN,
			[]ast.Expr{&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("limit")}},
		),
		newReturnStmt(ast.NewIdent("q")),
	)
	decls = append(decls, newFuncDecl("Limit", "Limit limits the number of results", recv, params, results, body))

	// Offset(offset int) *QueryBuilder
	params = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("offset")}, Type: ast.NewIdent("int")},
		},
	}
	body = newBlockStmt(
		newAssignStmt(
			[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "offset")},
			token.ASSIGN,
			[]ast.Expr{&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("offset")}},
		),
		newReturnStmt(ast.NewIdent("q")),
	)
	decls = append(decls, newFuncDecl("Offset", "Offset skips the first N results", recv, params, results, body))

	// Execute(ctx context.Context) ([]Model, error)
	params = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
		},
	}
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.ArrayType{Elt: ast.NewIdent(modelName)}},
			{Type: ast.NewIdent("error")},
		},
	}
	body = buildQueryBuilderExecuteBody(model, false)
	decls = append(decls, newFuncDecl("Execute", "Execute executes the query and returns all results", recv, params, results, body))

	// ExecuteFirst(ctx context.Context) (*Model, error)
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName)}},
			{Type: ast.NewIdent("error")},
		},
	}
	body = buildQueryBuilderExecuteFirstBody(model)
	decls = append(decls, newFuncDecl("ExecuteFirst", "ExecuteFirst executes the query and returns the first result", recv, params, results, body))

	return decls
}

// buildQueryBuilderExecuteBody builds the body for QueryBuilder.Execute
func buildQueryBuilderExecuteBody(model ModelInfo, withJoins bool) *ast.BlockStmt {
	modelName := model.Name
	stmts := []ast.Stmt{
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("results")},
						Type:  &ast.ArrayType{Elt: ast.NewIdent(modelName)},
					},
				},
			},
		},
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("whereClause")},
						Type:  &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("sqlgen"), "WhereClause")},
					},
				},
			},
		},
		newIfStmt(
			&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "where"), Op: token.NEQ, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{ast.NewIdent("whereClause")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "where"), "Build"))},
				),
			),
			nil,
		),
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("orderBy")},
						Type:  &ast.ArrayType{Elt: newSelectorExpr(ast.NewIdent("sqlgen"), "OrderBy")},
					},
				},
			},
		},
		newIfStmt(
			&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "orderBy"), Op: token.NEQ, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{ast.NewIdent("orderBy")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "orderBy"), "Build"))},
				),
			),
			nil,
		),
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("include")},
						Type:  &ast.MapType{Key: ast.NewIdent("string"), Value: ast.NewIdent("bool")},
					},
				},
			},
		},
		newIfStmt(
			&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "include"), Op: token.NEQ, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{ast.NewIdent("include")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "include"), "GetIncludes"))},
				),
			),
			nil,
		),
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("selectFields")},
						Type:  &ast.MapType{Key: ast.NewIdent("string"), Value: ast.NewIdent("bool")},
					},
				},
			},
		},
		newIfStmt(
			&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "select_"), Op: token.NEQ, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{ast.NewIdent("selectFields")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "select_"), "GetFields"))},
				),
			),
			nil,
		),
		newIfStmt(
			&ast.BinaryExpr{
				X: &ast.CallExpr{
					Fun: newSelectorExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "client"), "executor"),
						"FindManyWithRelations",
					),
					Args: []ast.Expr{
						ast.NewIdent("ctx"),
						newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "client"), "table"),
						ast.NewIdent("selectFields"),
						ast.NewIdent("whereClause"),
						ast.NewIdent("orderBy"),
						newSelectorExpr(ast.NewIdent("q"), "limit"),
						newSelectorExpr(ast.NewIdent("q"), "offset"),
						ast.NewIdent("include"),
						newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "client"), "relations"),
						&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("results")},
					},
				},
				Op: token.NEQ,
				Y:  ast.NewIdent("nil"),
			},
			newBlockStmt(
				newReturnStmt(ast.NewIdent("nil"), ast.NewIdent("err")),
			),
			nil,
		),
		newReturnStmt(ast.NewIdent("results"), ast.NewIdent("nil")),
	}
	return newBlockStmt(stmts...)
}

// buildQueryBuilderExecuteFirstBody builds the body for QueryBuilder.ExecuteFirst
func buildQueryBuilderExecuteFirstBody(model ModelInfo) *ast.BlockStmt {
	modelName := model.Name
	stmts := []ast.Stmt{
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("result")},
						Type:  ast.NewIdent(modelName),
					},
				},
			},
		},
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("whereClause")},
						Type:  &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("sqlgen"), "WhereClause")},
					},
				},
			},
		},
		newIfStmt(
			&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "where"), Op: token.NEQ, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{ast.NewIdent("whereClause")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "where"), "Build"))},
				),
			),
			nil,
		),
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("joins")},
						Type:  &ast.ArrayType{Elt: newSelectorExpr(ast.NewIdent("sqlgen"), "Join")},
					},
				},
			},
		},
		newIfStmt(
			&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "joins"), Op: token.NEQ, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{ast.NewIdent("joins")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "joins"), "Build"))},
				),
			),
			nil,
		),
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("orderBy")},
						Type:  &ast.ArrayType{Elt: newSelectorExpr(ast.NewIdent("sqlgen"), "OrderBy")},
					},
				},
			},
		},
		newIfStmt(
			&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "orderBy"), Op: token.NEQ, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{ast.NewIdent("orderBy")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "orderBy"), "Build"))},
				),
			),
			nil,
		),
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("include")},
						Type:  &ast.MapType{Key: ast.NewIdent("string"), Value: ast.NewIdent("bool")},
					},
				},
			},
		},
		newIfStmt(
			&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "include"), Op: token.NEQ, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{ast.NewIdent("include")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "include"), "GetIncludes"))},
				),
			),
			nil,
		),
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("selectFields")},
						Type:  &ast.MapType{Key: ast.NewIdent("string"), Value: ast.NewIdent("bool")},
					},
				},
			},
		},
		newIfStmt(
			&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "select_"), Op: token.NEQ, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{ast.NewIdent("selectFields")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "select_"), "GetFields"))},
				),
			),
			nil,
		),
		newIfStmt(
			&ast.BinaryExpr{
				X: &ast.CallExpr{
					Fun: newSelectorExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "client"), "executor"),
						"FindFirstWithJoins",
					),
					Args: []ast.Expr{
						ast.NewIdent("ctx"),
						newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "client"), "table"),
						ast.NewIdent("selectFields"),
						ast.NewIdent("joins"),
						ast.NewIdent("whereClause"),
						ast.NewIdent("orderBy"),
						ast.NewIdent("include"),
						newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "client"), "relations"),
						&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("result")},
					},
				},
				Op: token.NEQ,
				Y:  ast.NewIdent("nil"),
			},
			newBlockStmt(
				newReturnStmt(ast.NewIdent("nil"), ast.NewIdent("err")),
			),
			nil,
		),
		newReturnStmt(&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("result")}, ast.NewIdent("nil")),
	}
	return newBlockStmt(stmts...)
}

// buildFieldFilterMethods builds field-specific filter methods for QueryBuilder
func buildFieldFilterMethods(model ModelInfo) []ast.Decl {
	var decls []ast.Decl
	modelName := model.Name

	recv := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("q")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
		},
	}

	for _, field := range model.Fields {
		goFieldName := field.GoName
		goType := field.GoType
		dbColumnName := toSnakeCase(field.Name)

		// Equals method
		params := &ast.FieldList{
			List: []*ast.Field{
				{Names: []*ast.Ident{ast.NewIdent("value")}, Type: parseTypeFromString(goType)},
			},
		}
		results := &ast.FieldList{
			List: []*ast.Field{
				{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
			},
		}
		body := newBlockStmt(
			newIfStmt(
				&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "where"), Op: token.EQL, Y: ast.NewIdent("nil")},
				newBlockStmt(
					newAssignStmt(
						[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "where")},
						token.ASSIGN,
						[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewWhereBuilder"))},
					),
				),
				nil,
			),
			&ast.ExprStmt{
				X: newCallExpr(
					newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "where"), "Equals"),
					newStringLit(dbColumnName),
					ast.NewIdent("value"),
				),
			},
			newReturnStmt(ast.NewIdent("q")),
		)
		decls = append(decls, newFuncDecl(
			goFieldName+"Equals",
			fmt.Sprintf("%sEquals filters where %s equals the value", goFieldName, field.Name),
			recv, params, results, body,
		))

		// NotEquals method
		body = newBlockStmt(
			newIfStmt(
				&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "where"), Op: token.EQL, Y: ast.NewIdent("nil")},
				newBlockStmt(
					newAssignStmt(
						[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "where")},
						token.ASSIGN,
						[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewWhereBuilder"))},
					),
				),
				nil,
			),
			&ast.ExprStmt{
				X: newCallExpr(
					newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "where"), "NotEquals"),
					newStringLit(dbColumnName),
					ast.NewIdent("value"),
				),
			},
			newReturnStmt(ast.NewIdent("q")),
		)
		decls = append(decls, newFuncDecl(
			goFieldName+"NotEquals",
			fmt.Sprintf("%sNotEquals filters where %s does not equal the value", goFieldName, field.Name),
			recv, params, results, body,
		))

		// Numeric comparison methods
		if isNumericType(goType) {
			// GreaterThan
			body = newBlockStmt(
				newIfStmt(
					&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "where"), Op: token.EQL, Y: ast.NewIdent("nil")},
					newBlockStmt(
						newAssignStmt(
							[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "where")},
							token.ASSIGN,
							[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewWhereBuilder"))},
						),
					),
					nil,
				),
				&ast.ExprStmt{
					X: newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "where"), "GreaterThan"),
						newStringLit(dbColumnName),
						ast.NewIdent("value"),
					),
				},
				newReturnStmt(ast.NewIdent("q")),
			)
			decls = append(decls, newFuncDecl(
				goFieldName+"GreaterThan",
				fmt.Sprintf("%sGreaterThan filters where %s is greater than the value", goFieldName, field.Name),
				recv, params, results, body,
			))

			// LessThan
			body = newBlockStmt(
				newIfStmt(
					&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "where"), Op: token.EQL, Y: ast.NewIdent("nil")},
					newBlockStmt(
						newAssignStmt(
							[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "where")},
							token.ASSIGN,
							[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewWhereBuilder"))},
						),
					),
					nil,
				),
				&ast.ExprStmt{
					X: newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "where"), "LessThan"),
						newStringLit(dbColumnName),
						ast.NewIdent("value"),
					),
				},
				newReturnStmt(ast.NewIdent("q")),
			)
			decls = append(decls, newFuncDecl(
				goFieldName+"LessThan",
				fmt.Sprintf("%sLessThan filters where %s is less than the value", goFieldName, field.Name),
				recv, params, results, body,
			))
		}

		// String Contains method
		if goType == "string" || strings.HasPrefix(goType, "*string") {
			params = &ast.FieldList{
				List: []*ast.Field{
					{Names: []*ast.Ident{ast.NewIdent("value")}, Type: ast.NewIdent("string")},
				},
			}
			body = newBlockStmt(
				newIfStmt(
					&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "where"), Op: token.EQL, Y: ast.NewIdent("nil")},
					newBlockStmt(
						newAssignStmt(
							[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "where")},
							token.ASSIGN,
							[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewWhereBuilder"))},
						),
					),
					nil,
				),
				&ast.ExprStmt{
					X: newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "where"), "Like"),
						newStringLit(dbColumnName),
						&ast.BinaryExpr{
							X: &ast.BinaryExpr{
								X:  newStringLit("%"),
								Op: token.ADD,
								Y:  ast.NewIdent("value"),
							},
							Op: token.ADD,
							Y:  newStringLit("%"),
						},
					),
				},
				newReturnStmt(ast.NewIdent("q")),
			)
			decls = append(decls, newFuncDecl(
				goFieldName+"Contains",
				fmt.Sprintf("%sContains filters where %s contains the substring", goFieldName, field.Name),
				recv, params, results, body,
			))
		}

		// Null check methods for optional fields
		if strings.HasPrefix(goType, "*") {
			params = &ast.FieldList{}
			body = newBlockStmt(
				newIfStmt(
					&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "where"), Op: token.EQL, Y: ast.NewIdent("nil")},
					newBlockStmt(
						newAssignStmt(
							[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "where")},
							token.ASSIGN,
							[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewWhereBuilder"))},
						),
					),
					nil,
				),
				&ast.ExprStmt{
					X: newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "where"), "IsNull"),
						newStringLit(dbColumnName),
					),
				},
				newReturnStmt(ast.NewIdent("q")),
			)
			decls = append(decls, newFuncDecl(
				goFieldName+"IsNull",
				fmt.Sprintf("%sIsNull filters where %s is NULL", goFieldName, field.Name),
				recv, params, results, body,
			))

			body = newBlockStmt(
				newIfStmt(
					&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "where"), Op: token.EQL, Y: ast.NewIdent("nil")},
					newBlockStmt(
						newAssignStmt(
							[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "where")},
							token.ASSIGN,
							[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewWhereBuilder"))},
						),
					),
					nil,
				),
				&ast.ExprStmt{
					X: newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "where"), "IsNotNull"),
						newStringLit(dbColumnName),
					),
				},
				newReturnStmt(ast.NewIdent("q")),
			)
			decls = append(decls, newFuncDecl(
				goFieldName+"IsNotNull",
				fmt.Sprintf("%sIsNotNull filters where %s is not NULL", goFieldName, field.Name),
				recv, params, results, body,
			))
		}
	}

	return decls
}

// buildOrderByMethods builds OrderBy methods for QueryBuilder
func buildOrderByMethods(model ModelInfo) []ast.Decl {
	var decls []ast.Decl
	modelName := model.Name

	recv := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("q")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
		},
	}

	params := &ast.FieldList{}
	results := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
		},
	}

	for _, field := range model.Fields {
		goFieldName := field.GoName
		dbColumnName := toSnakeCase(field.Name)

		// OrderByAsc
		body := newBlockStmt(
			newIfStmt(
				&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "orderBy"), Op: token.EQL, Y: ast.NewIdent("nil")},
				newBlockStmt(
					newAssignStmt(
						[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "orderBy")},
						token.ASSIGN,
						[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewOrderByBuilder"))},
					),
				),
				nil,
			),
			&ast.ExprStmt{
				X: newCallExpr(
					newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "orderBy"), "Asc"),
					newStringLit(dbColumnName),
				),
			},
			newReturnStmt(ast.NewIdent("q")),
		)
		decls = append(decls, newFuncDecl(
			"OrderBy"+goFieldName+"Asc",
			fmt.Sprintf("OrderBy%sAsc orders results by %s ascending", goFieldName, field.Name),
			recv, params, results, body,
		))

		// OrderByDesc
		body = newBlockStmt(
			newIfStmt(
				&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "orderBy"), Op: token.EQL, Y: ast.NewIdent("nil")},
				newBlockStmt(
					newAssignStmt(
						[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "orderBy")},
						token.ASSIGN,
						[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewOrderByBuilder"))},
					),
				),
				nil,
			),
			&ast.ExprStmt{
				X: newCallExpr(
					newSelectorExpr(newSelectorExpr(ast.NewIdent("q"), "orderBy"), "Desc"),
					newStringLit(dbColumnName),
				),
			},
			newReturnStmt(ast.NewIdent("q")),
		)
		decls = append(decls, newFuncDecl(
			"OrderBy"+goFieldName+"Desc",
			fmt.Sprintf("OrderBy%sDesc orders results by %s descending", goFieldName, field.Name),
			recv, params, results, body,
		))
	}

	return decls
}

// buildJoinIncludeSelectBuilders builds Join, Include, and Select builder types and methods
func buildJoinIncludeSelectBuilders(model ModelInfo) []ast.Decl {
	var decls []ast.Decl
	modelName := model.Name

	// JoinBuilder type
	joinBuilderFields := []*ast.Field{
		{Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "JoinBuilder")}},
		{Names: []*ast.Ident{ast.NewIdent("queryBuilder")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
	}
	joinBuilderType := newTypeDecl(
		modelName+"JoinBuilder",
		fmt.Sprintf("%sJoinBuilder builds JOIN clauses for %s", modelName, modelName),
		newStructType(joinBuilderFields),
	)
	decls = append(decls, joinBuilderType)

	// IncludeBuilder type
	includeBuilderFields := []*ast.Field{
		{Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "IncludeBuilder")}},
		{Names: []*ast.Ident{ast.NewIdent("queryBuilder")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
	}
	includeBuilderType := newTypeDecl(
		modelName+"IncludeBuilder",
		fmt.Sprintf("%sIncludeBuilder builds INCLUDE clauses for %s", modelName, modelName),
		newStructType(includeBuilderFields),
	)
	decls = append(decls, includeBuilderType)

	// SelectBuilder type
	selectBuilderFields := []*ast.Field{
		{Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "SelectBuilder")}},
		{Names: []*ast.Ident{ast.NewIdent("queryBuilder")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
	}
	selectBuilderType := newTypeDecl(
		modelName+"SelectBuilder",
		fmt.Sprintf("%sSelectBuilder builds SELECT clauses for %s", modelName, modelName),
		newStructType(selectBuilderFields),
	)
	decls = append(decls, selectBuilderType)

	// QueryBuilder.Join() method
	recv := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("q")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
		},
	}
	params := &ast.FieldList{}
	results := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "JoinBuilder")}},
		},
	}
	body := newBlockStmt(
		newIfStmt(
			&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "joins"), Op: token.EQL, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "joins")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewJoinBuilder"))},
				),
			),
			nil,
		),
		newReturnStmt(
			&ast.UnaryExpr{
				Op: token.AND,
				X: newCompositeLit(
					ast.NewIdent(modelName+"JoinBuilder"),
					[]ast.Expr{
						newKeyValueExpr("JoinBuilder", newSelectorExpr(ast.NewIdent("q"), "joins")),
						newKeyValueExpr("queryBuilder", ast.NewIdent("q")),
					},
				),
			},
		),
	)
	decls = append(decls, newFuncDecl("Join", "Join starts building JOIN clauses", recv, params, results, body))

	// JoinBuilder.Done() method
	recv = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("j")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "JoinBuilder")}},
		},
	}
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
		},
	}
	body = newBlockStmt(
		newReturnStmt(newSelectorExpr(ast.NewIdent("j"), "queryBuilder")),
	)
	decls = append(decls, newFuncDecl("Done", "Done returns to the query builder", recv, params, results, body))

	// QueryBuilder.Include() method
	recv = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("q")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
		},
	}
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "IncludeBuilder")}},
		},
	}
	body = newBlockStmt(
		newIfStmt(
			&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "include"), Op: token.EQL, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "include")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewIncludeBuilder"))},
				),
			),
			nil,
		),
		newReturnStmt(
			&ast.UnaryExpr{
				Op: token.AND,
				X: newCompositeLit(
					ast.NewIdent(modelName+"IncludeBuilder"),
					[]ast.Expr{
						newKeyValueExpr("IncludeBuilder", newSelectorExpr(ast.NewIdent("q"), "include")),
						newKeyValueExpr("queryBuilder", ast.NewIdent("q")),
					},
				),
			},
		),
	)
	decls = append(decls, newFuncDecl("Include", "Include starts building an INCLUDE clause for relations", recv, params, results, body))

	// IncludeBuilder.Done() method
	recv = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("i")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "IncludeBuilder")}},
		},
	}
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
		},
	}
	body = newBlockStmt(
		newReturnStmt(newSelectorExpr(ast.NewIdent("i"), "queryBuilder")),
	)
	decls = append(decls, newFuncDecl("Done", "Done returns to the query builder", recv, params, results, body))

	// Generate include methods for relation fields
	for _, field := range model.Fields {
		if field.IsRelation {
			goFieldName := field.GoName
			relatedModel := field.RelationTo

			recv = &ast.FieldList{
				List: []*ast.Field{
					{Names: []*ast.Ident{ast.NewIdent("i")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "IncludeBuilder")}},
				},
			}
			params = &ast.FieldList{}
			results = &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.StarExpr{X: ast.NewIdent(relatedModel + "IncludeBuilder")}},
				},
			}
			body = newBlockStmt(
				&ast.ExprStmt{
					X: newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("i"), "IncludeBuilder"), "Include"),
						newStringLit(field.Name),
					),
				},
				newReturnStmt(
					&ast.UnaryExpr{
						Op: token.AND,
						X: newCompositeLit(
							ast.NewIdent(relatedModel+"IncludeBuilder"),
							[]ast.Expr{
								newKeyValueExpr("IncludeBuilder", newSelectorExpr(ast.NewIdent("i"), "IncludeBuilder")),
								newKeyValueExpr("queryBuilder", newSelectorExpr(ast.NewIdent("i"), "queryBuilder")),
							},
						),
					},
				),
			)
			decls = append(decls, newFuncDecl(
				goFieldName,
				fmt.Sprintf("%s includes the %s relation", goFieldName, field.Name),
				recv, params, results, body,
			))
		}
	}

	// QueryBuilder.Select() method
	recv = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("q")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
		},
	}
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "SelectBuilder")}},
		},
	}
	body = newBlockStmt(
		newIfStmt(
			&ast.BinaryExpr{X: newSelectorExpr(ast.NewIdent("q"), "select_"), Op: token.EQL, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{newSelectorExpr(ast.NewIdent("q"), "select_")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewSelectBuilder"))},
				),
			),
			nil,
		),
		newReturnStmt(
			&ast.UnaryExpr{
				Op: token.AND,
				X: newCompositeLit(
					ast.NewIdent(modelName+"SelectBuilder"),
					[]ast.Expr{
						newKeyValueExpr("SelectBuilder", newSelectorExpr(ast.NewIdent("q"), "select_")),
						newKeyValueExpr("queryBuilder", ast.NewIdent("q")),
					},
				),
			},
		),
	)
	decls = append(decls, newFuncDecl("Select", "Select starts building a SELECT clause for fields", recv, params, results, body))

	// SelectBuilder.Done() method
	recv = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("s")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "SelectBuilder")}},
		},
	}
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "QueryBuilder")}},
		},
	}
	body = newBlockStmt(
		newReturnStmt(newSelectorExpr(ast.NewIdent("s"), "queryBuilder")),
	)
	decls = append(decls, newFuncDecl("Done", "Done returns to the query builder", recv, params, results, body))

	// Generate select methods for non-relation fields
	for _, field := range model.Fields {
		if !field.IsRelation {
			goFieldName := field.GoName

			recv = &ast.FieldList{
				List: []*ast.Field{
					{Names: []*ast.Ident{ast.NewIdent("s")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "SelectBuilder")}},
				},
			}
			params = &ast.FieldList{}
			results = &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "SelectBuilder")}},
				},
			}
			body = newBlockStmt(
				&ast.ExprStmt{
					X: newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("s"), "SelectBuilder"), "Field"),
						newStringLit(field.Name),
					),
				},
				newReturnStmt(ast.NewIdent("s")),
			)
			decls = append(decls, newFuncDecl(
				goFieldName,
				fmt.Sprintf("%s selects the %s field", goFieldName, field.Name),
				recv, params, results, body,
			))
		}
	}

	return decls
}

// buildCRUDMethods builds Create, Update, and Delete methods
func buildCRUDMethods(model ModelInfo) []ast.Decl {
	var decls []ast.Decl
	modelName := model.Name

	recv := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("c")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "Client")}},
		},
	}

	// Create method
	params := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
			{Names: []*ast.Ident{ast.NewIdent("data")}, Type: ast.NewIdent(modelName)},
		},
	}
	results := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName)}},
			{Type: ast.NewIdent("error")},
		},
	}
	body := newBlockStmt(
		newAssignStmt(
			[]ast.Expr{ast.NewIdent("result"), ast.NewIdent("err")},
			token.DEFINE,
			[]ast.Expr{
				newCallExpr(
					newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "executor"), "Create"),
					ast.NewIdent("ctx"),
					newSelectorExpr(ast.NewIdent("c"), "table"),
					ast.NewIdent("data"),
				),
			},
		),
		newIfStmt(
			&ast.BinaryExpr{X: ast.NewIdent("err"), Op: token.NEQ, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newReturnStmt(ast.NewIdent("nil"), ast.NewIdent("err")),
			),
			nil,
		),
		newAssignStmt(
			[]ast.Expr{ast.NewIdent("created"), ast.NewIdent("ok")},
			token.DEFINE,
			[]ast.Expr{
				&ast.TypeAssertExpr{
					X:    ast.NewIdent("result"),
					Type: &ast.StarExpr{X: ast.NewIdent(modelName)},
				},
			},
		),
		newIfStmt(
			&ast.BinaryExpr{X: ast.NewIdent("ok"), Op: token.EQL, Y: ast.NewIdent("true")},
			newBlockStmt(
				newReturnStmt(ast.NewIdent("created"), ast.NewIdent("nil")),
			),
			nil,
		),
		newReturnStmt(&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("data")}, ast.NewIdent("nil")),
	)
	decls = append(decls, newFuncDecl("Create", fmt.Sprintf("Create creates a new %s record", modelName), recv, params, results, body))

	// UpdateBuilder type
	updateBuilderFields := []*ast.Field{
		{Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "UpdateBuilder")}},
		{Names: []*ast.Ident{ast.NewIdent("client")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "Client")}},
	}
	updateBuilderType := newTypeDecl(
		modelName+"UpdateBuilder",
		fmt.Sprintf("%sUpdateBuilder builds UPDATE queries for %s", modelName, modelName),
		newStructType(updateBuilderFields),
	)
	decls = append(decls, updateBuilderType)

	// Update() method
	params = &ast.FieldList{}
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "UpdateBuilder")}},
		},
	}
	body = newBlockStmt(
		newReturnStmt(
			&ast.UnaryExpr{
				Op: token.AND,
				X: newCompositeLit(
					ast.NewIdent(modelName+"UpdateBuilder"),
					[]ast.Expr{
						newKeyValueExpr("UpdateBuilder", newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewUpdateBuilder"))),
						newKeyValueExpr("client", ast.NewIdent("c")),
					},
				),
			},
		),
	)
	decls = append(decls, newFuncDecl("Update", "Update starts building an UPDATE query", recv, params, results, body))

	// UpdateWhereBuilder type
	updateWhereBuilderFields := []*ast.Field{
		{Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "WhereBuilder")}},
		{Names: []*ast.Ident{ast.NewIdent("updateBuilder")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "UpdateBuilder")}},
	}
	updateWhereBuilderType := newTypeDecl(
		modelName+"UpdateWhereBuilder",
		fmt.Sprintf("%sUpdateWhereBuilder is a WhereBuilder for Update operations", modelName),
		newStructType(updateWhereBuilderFields),
	)
	decls = append(decls, updateWhereBuilderType)

	// UpdateBuilder.Where() method
	recv = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("u")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "UpdateBuilder")}},
		},
	}
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "UpdateWhereBuilder")}},
		},
	}
	body = newBlockStmt(
		newIfStmt(
			&ast.BinaryExpr{
				X:  newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("u"), "UpdateBuilder"), "GetWhere")),
				Op: token.EQL,
				Y:  ast.NewIdent("nil"),
			},
			newBlockStmt(
				&ast.ExprStmt{
					X: newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("u"), "UpdateBuilder"), "Where")),
				},
			),
			nil,
		),
		newReturnStmt(
			&ast.UnaryExpr{
				Op: token.AND,
				X: newCompositeLit(
					ast.NewIdent(modelName+"UpdateWhereBuilder"),
					[]ast.Expr{
						newKeyValueExpr("WhereBuilder", newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("u"), "UpdateBuilder"), "Where"))),
						newKeyValueExpr("updateBuilder", ast.NewIdent("u")),
					},
				),
			},
		),
	)
	decls = append(decls, newFuncDecl("Where", "Where adds a WHERE clause to the UPDATE query", recv, params, results, body))

	// UpdateBuilder.Execute() method
	params = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
		},
	}
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName)}},
			{Type: ast.NewIdent("error")},
		},
	}
	body = newBlockStmt(
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("result")},
						Type:  ast.NewIdent(modelName),
					},
				},
			},
		},
		newAssignStmt(
			[]ast.Expr{ast.NewIdent("whereClause")},
			token.DEFINE,
			[]ast.Expr{newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("u"), "UpdateBuilder"), "GetWhere"))},
		),
		newIfStmt(
			&ast.BinaryExpr{
				X: &ast.CallExpr{
					Fun: newSelectorExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("u"), "client"), "executor"),
						"Update",
					),
					Args: []ast.Expr{
						ast.NewIdent("ctx"),
						newSelectorExpr(newSelectorExpr(ast.NewIdent("u"), "client"), "table"),
						newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("u"), "UpdateBuilder"), "GetSet")),
						ast.NewIdent("whereClause"),
						&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("result")},
					},
				},
				Op: token.NEQ,
				Y:  ast.NewIdent("nil"),
			},
			newBlockStmt(
				newReturnStmt(ast.NewIdent("nil"), ast.NewIdent("err")),
			),
			nil,
		),
		newReturnStmt(&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("result")}, ast.NewIdent("nil")),
	)
	// Fix the Update call
	body.List[2] = newIfStmt(
		&ast.BinaryExpr{
			X: &ast.CallExpr{
				Fun: newSelectorExpr(
					newSelectorExpr(newSelectorExpr(ast.NewIdent("u"), "client"), "executor"),
					"Update",
				),
				Args: []ast.Expr{
					ast.NewIdent("ctx"),
					newSelectorExpr(newSelectorExpr(ast.NewIdent("u"), "client"), "table"),
					newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("u"), "UpdateBuilder"), "GetSet")),
					ast.NewIdent("whereClause"),
					&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("result")},
				},
			},
			Op: token.NEQ,
			Y:  ast.NewIdent("nil"),
		},
		newBlockStmt(
			newReturnStmt(ast.NewIdent("nil"), ast.NewIdent("err")),
		),
		nil,
	)
	decls = append(decls, newFuncDecl("Execute", "Execute executes the UPDATE query", recv, params, results, body))

	// Generate Set methods for UpdateBuilder
	for _, field := range model.Fields {
		goFieldName := field.GoName
		goType := field.GoType
		dbColumnName := toSnakeCase(field.Name)

		recv = &ast.FieldList{
			List: []*ast.Field{
				{Names: []*ast.Ident{ast.NewIdent("u")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "UpdateBuilder")}},
			},
		}
		params = &ast.FieldList{
			List: []*ast.Field{
				{Names: []*ast.Ident{ast.NewIdent("value")}, Type: parseTypeFromString(goType)},
			},
		}
		results = &ast.FieldList{
			List: []*ast.Field{
				{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "UpdateBuilder")}},
			},
		}
		body = newBlockStmt(
			&ast.ExprStmt{
				X: newCallExpr(
					newSelectorExpr(newSelectorExpr(ast.NewIdent("u"), "UpdateBuilder"), "Set"),
					newStringLit(dbColumnName),
					ast.NewIdent("value"),
				),
			},
			newReturnStmt(ast.NewIdent("u")),
		)
		decls = append(decls, newFuncDecl(
			"Set"+goFieldName,
			fmt.Sprintf("Set%s sets the %s field", goFieldName, field.Name),
			recv, params, results, body,
		))
	}

	// DeleteBuilder type
	deleteBuilderFields := []*ast.Field{
		{Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "WhereBuilder")}},
		{Names: []*ast.Ident{ast.NewIdent("client")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "Client")}},
	}
	deleteBuilderType := newTypeDecl(
		modelName+"DeleteBuilder",
		fmt.Sprintf("%sDeleteBuilder builds DELETE queries for %s", modelName, modelName),
		newStructType(deleteBuilderFields),
	)
	decls = append(decls, deleteBuilderType)

	// Delete() method
	recv = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("c")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "Client")}},
		},
	}
	params = &ast.FieldList{}
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "DeleteBuilder")}},
		},
	}
	body = newBlockStmt(
		newReturnStmt(
			&ast.UnaryExpr{
				Op: token.AND,
				X: newCompositeLit(
					ast.NewIdent(modelName+"DeleteBuilder"),
					[]ast.Expr{
						newKeyValueExpr("WhereBuilder", newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewWhereBuilder"))),
						newKeyValueExpr("client", ast.NewIdent("c")),
					},
				),
			},
		),
	)
	decls = append(decls, newFuncDecl("Delete", "Delete starts building a DELETE query", recv, params, results, body))

	// DeleteBuilder.Execute() method
	recv = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("d")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "DeleteBuilder")}},
		},
	}
	params = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
		},
	}
	results = &ast.FieldList{
		List: []*ast.Field{
			{Type: ast.NewIdent("error")},
		},
	}
	body = newBlockStmt(
		newAssignStmt(
			[]ast.Expr{ast.NewIdent("whereClause")},
			token.DEFINE,
			[]ast.Expr{newCallExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("d"), "WhereBuilder"), "Build"))},
		),
		newReturnStmt(
			newCallExpr(
				newSelectorExpr(newSelectorExpr(newSelectorExpr(ast.NewIdent("d"), "client"), "executor"), "Delete"),
				ast.NewIdent("ctx"),
				newSelectorExpr(newSelectorExpr(ast.NewIdent("d"), "client"), "table"),
				ast.NewIdent("whereClause"),
			),
		),
	)
	decls = append(decls, newFuncDecl("Execute", "Execute executes the DELETE query", recv, params, results, body))

	// Generate filter methods for DeleteBuilder (similar to WhereBuilder)
	for _, field := range model.Fields {
		goFieldName := field.GoName
		goType := field.GoType
		dbColumnName := toSnakeCase(field.Name)

		recv = &ast.FieldList{
			List: []*ast.Field{
				{Names: []*ast.Ident{ast.NewIdent("d")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "DeleteBuilder")}},
			},
		}
		params = &ast.FieldList{
			List: []*ast.Field{
				{Names: []*ast.Ident{ast.NewIdent("value")}, Type: parseTypeFromString(goType)},
			},
		}
		results = &ast.FieldList{
			List: []*ast.Field{
				{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "DeleteBuilder")}},
			},
		}
		body = newBlockStmt(
			&ast.ExprStmt{
				X: newCallExpr(
					newSelectorExpr(newSelectorExpr(ast.NewIdent("d"), "WhereBuilder"), "Equals"),
					newStringLit(dbColumnName),
					ast.NewIdent("value"),
				),
			},
			newReturnStmt(ast.NewIdent("d")),
		)
		decls = append(decls, newFuncDecl(
			goFieldName+"Equals",
			fmt.Sprintf("%sEquals filters where %s equals the value", goFieldName, field.Name),
			recv, params, results, body,
		))

		body = newBlockStmt(
			&ast.ExprStmt{
				X: newCallExpr(
					newSelectorExpr(newSelectorExpr(ast.NewIdent("d"), "WhereBuilder"), "NotEquals"),
					newStringLit(dbColumnName),
					ast.NewIdent("value"),
				),
			},
			newReturnStmt(ast.NewIdent("d")),
		)
		decls = append(decls, newFuncDecl(
			goFieldName+"NotEquals",
			fmt.Sprintf("%sNotEquals filters where %s does not equal the value", goFieldName, field.Name),
			recv, params, results, body,
		))

		if isNumericType(goType) {
			body = newBlockStmt(
				&ast.ExprStmt{
					X: newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("d"), "WhereBuilder"), "GreaterThan"),
						newStringLit(dbColumnName),
						ast.NewIdent("value"),
					),
				},
				newReturnStmt(ast.NewIdent("d")),
			)
			decls = append(decls, newFuncDecl(
				goFieldName+"GreaterThan",
				fmt.Sprintf("%sGreaterThan filters where %s is greater than the value", goFieldName, field.Name),
				recv, params, results, body,
			))

			body = newBlockStmt(
				&ast.ExprStmt{
					X: newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("d"), "WhereBuilder"), "LessThan"),
						newStringLit(dbColumnName),
						ast.NewIdent("value"),
					),
				},
				newReturnStmt(ast.NewIdent("d")),
			)
			decls = append(decls, newFuncDecl(
				goFieldName+"LessThan",
				fmt.Sprintf("%sLessThan filters where %s is less than the value", goFieldName, field.Name),
				recv, params, results, body,
			))
		}
	}

	return decls
}

// buildAggregationMethods builds Count, Sum, Avg, Min, Max methods
func buildAggregationMethods(model ModelInfo) []ast.Decl {
	var decls []ast.Decl
	modelName := model.Name

	recv := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("c")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "Client")}},
		},
	}

	// Count method
	params := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
		},
	}
	results := &ast.FieldList{
		List: []*ast.Field{
			{Type: ast.NewIdent("int64")},
			{Type: ast.NewIdent("error")},
		},
	}
	body := newBlockStmt(
		newReturnStmt(
			newCallExpr(
				newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "executor"), "Count"),
				ast.NewIdent("ctx"),
				newSelectorExpr(ast.NewIdent("c"), "table"),
				ast.NewIdent("nil"),
			),
		),
	)
	decls = append(decls, newFuncDecl("Count", "Count counts the number of records", recv, params, results, body))

	// CountWhere method
	params = &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
			{Names: []*ast.Ident{ast.NewIdent("where")}, Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "WhereBuilder")}},
		},
	}
	body = newBlockStmt(
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("whereClause")},
						Type:  &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("sqlgen"), "WhereClause")},
					},
				},
			},
		},
		newIfStmt(
			&ast.BinaryExpr{X: ast.NewIdent("where"), Op: token.NEQ, Y: ast.NewIdent("nil")},
			newBlockStmt(
				newAssignStmt(
					[]ast.Expr{ast.NewIdent("whereClause")},
					token.ASSIGN,
					[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("where"), "Build"))},
				),
			),
			nil,
		),
		newReturnStmt(
			newCallExpr(
				newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "executor"), "Count"),
				ast.NewIdent("ctx"),
				newSelectorExpr(ast.NewIdent("c"), "table"),
				ast.NewIdent("whereClause"),
			),
		),
	)
	decls = append(decls, newFuncDecl("CountWhere", "CountWhere counts records matching the WHERE clause", recv, params, results, body))

	// Sum, Avg, Min, Max for numeric fields
	for _, field := range model.Fields {
		if !field.IsRelation && isNumericType(field.GoType) {
			goFieldName := field.GoName
			dbColumnName := toSnakeCase(field.Name)

			// Sum
			params = &ast.FieldList{
				List: []*ast.Field{
					{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
				},
			}
			results = &ast.FieldList{
				List: []*ast.Field{
					{Type: ast.NewIdent("float64")},
					{Type: ast.NewIdent("error")},
				},
			}
			body = newBlockStmt(
				newReturnStmt(
					newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "executor"), "Sum"),
						ast.NewIdent("ctx"),
						newSelectorExpr(ast.NewIdent("c"), "table"),
						newStringLit(dbColumnName),
						ast.NewIdent("nil"),
					),
				),
			)
			decls = append(decls, newFuncDecl("Sum"+goFieldName, fmt.Sprintf("Sum%s calculates the sum of %s", goFieldName, field.Name), recv, params, results, body))

			// SumWhere
			params = &ast.FieldList{
				List: []*ast.Field{
					{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
					{Names: []*ast.Ident{ast.NewIdent("where")}, Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "WhereBuilder")}},
				},
			}
			body = newBlockStmt(
				&ast.DeclStmt{
					Decl: &ast.GenDecl{
						Tok: token.VAR,
						Specs: []ast.Spec{
							&ast.ValueSpec{
								Names: []*ast.Ident{ast.NewIdent("whereClause")},
								Type:  &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("sqlgen"), "WhereClause")},
							},
						},
					},
				},
				newIfStmt(
					&ast.BinaryExpr{X: ast.NewIdent("where"), Op: token.NEQ, Y: ast.NewIdent("nil")},
					newBlockStmt(
						newAssignStmt(
							[]ast.Expr{ast.NewIdent("whereClause")},
							token.ASSIGN,
							[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("where"), "Build"))},
						),
					),
					nil,
				),
				newReturnStmt(
					newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "executor"), "Sum"),
						ast.NewIdent("ctx"),
						newSelectorExpr(ast.NewIdent("c"), "table"),
						newStringLit(dbColumnName),
						ast.NewIdent("whereClause"),
					),
				),
			)
			decls = append(decls, newFuncDecl("Sum"+goFieldName+"Where", fmt.Sprintf("Sum%sWhere calculates the sum of %s with WHERE clause", goFieldName, field.Name), recv, params, results, body))

			// Avg
			params = &ast.FieldList{
				List: []*ast.Field{
					{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
				},
			}
			body = newBlockStmt(
				newReturnStmt(
					newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "executor"), "Avg"),
						ast.NewIdent("ctx"),
						newSelectorExpr(ast.NewIdent("c"), "table"),
						newStringLit(dbColumnName),
						ast.NewIdent("nil"),
					),
				),
			)
			decls = append(decls, newFuncDecl("Avg"+goFieldName, fmt.Sprintf("Avg%s calculates the average of %s", goFieldName, field.Name), recv, params, results, body))

			// AvgWhere
			params = &ast.FieldList{
				List: []*ast.Field{
					{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
					{Names: []*ast.Ident{ast.NewIdent("where")}, Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "WhereBuilder")}},
				},
			}
			body = newBlockStmt(
				&ast.DeclStmt{
					Decl: &ast.GenDecl{
						Tok: token.VAR,
						Specs: []ast.Spec{
							&ast.ValueSpec{
								Names: []*ast.Ident{ast.NewIdent("whereClause")},
								Type:  &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("sqlgen"), "WhereClause")},
							},
						},
					},
				},
				newIfStmt(
					&ast.BinaryExpr{X: ast.NewIdent("where"), Op: token.NEQ, Y: ast.NewIdent("nil")},
					newBlockStmt(
						newAssignStmt(
							[]ast.Expr{ast.NewIdent("whereClause")},
							token.ASSIGN,
							[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("where"), "Build"))},
						),
					),
					nil,
				),
				newReturnStmt(
					newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "executor"), "Avg"),
						ast.NewIdent("ctx"),
						newSelectorExpr(ast.NewIdent("c"), "table"),
						newStringLit(dbColumnName),
						ast.NewIdent("whereClause"),
					),
				),
			)
			decls = append(decls, newFuncDecl("Avg"+goFieldName+"Where", fmt.Sprintf("Avg%sWhere calculates the average of %s with WHERE clause", goFieldName, field.Name), recv, params, results, body))

			// Min
			params = &ast.FieldList{
				List: []*ast.Field{
					{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
				},
			}
			body = newBlockStmt(
				newReturnStmt(
					newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "executor"), "Min"),
						ast.NewIdent("ctx"),
						newSelectorExpr(ast.NewIdent("c"), "table"),
						newStringLit(dbColumnName),
						ast.NewIdent("nil"),
					),
				),
			)
			decls = append(decls, newFuncDecl("Min"+goFieldName, fmt.Sprintf("Min%s finds the minimum value of %s", goFieldName, field.Name), recv, params, results, body))

			// MinWhere
			params = &ast.FieldList{
				List: []*ast.Field{
					{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
					{Names: []*ast.Ident{ast.NewIdent("where")}, Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "WhereBuilder")}},
				},
			}
			body = newBlockStmt(
				&ast.DeclStmt{
					Decl: &ast.GenDecl{
						Tok: token.VAR,
						Specs: []ast.Spec{
							&ast.ValueSpec{
								Names: []*ast.Ident{ast.NewIdent("whereClause")},
								Type:  &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("sqlgen"), "WhereClause")},
							},
						},
					},
				},
				newIfStmt(
					&ast.BinaryExpr{X: ast.NewIdent("where"), Op: token.NEQ, Y: ast.NewIdent("nil")},
					newBlockStmt(
						newAssignStmt(
							[]ast.Expr{ast.NewIdent("whereClause")},
							token.ASSIGN,
							[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("where"), "Build"))},
						),
					),
					nil,
				),
				newReturnStmt(
					newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "executor"), "Min"),
						ast.NewIdent("ctx"),
						newSelectorExpr(ast.NewIdent("c"), "table"),
						newStringLit(dbColumnName),
						ast.NewIdent("whereClause"),
					),
				),
			)
			decls = append(decls, newFuncDecl("Min"+goFieldName+"Where", fmt.Sprintf("Min%sWhere finds the minimum value of %s with WHERE clause", goFieldName, field.Name), recv, params, results, body))

			// Max
			params = &ast.FieldList{
				List: []*ast.Field{
					{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
				},
			}
			body = newBlockStmt(
				newReturnStmt(
					newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "executor"), "Max"),
						ast.NewIdent("ctx"),
						newSelectorExpr(ast.NewIdent("c"), "table"),
						newStringLit(dbColumnName),
						ast.NewIdent("nil"),
					),
				),
			)
			decls = append(decls, newFuncDecl("Max"+goFieldName, fmt.Sprintf("Max%s finds the maximum value of %s", goFieldName, field.Name), recv, params, results, body))

			// MaxWhere
			params = &ast.FieldList{
				List: []*ast.Field{
					{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: newSelectorExpr(ast.NewIdent("context"), "Context")},
					{Names: []*ast.Ident{ast.NewIdent("where")}, Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "WhereBuilder")}},
				},
			}
			body = newBlockStmt(
				&ast.DeclStmt{
					Decl: &ast.GenDecl{
						Tok: token.VAR,
						Specs: []ast.Spec{
							&ast.ValueSpec{
								Names: []*ast.Ident{ast.NewIdent("whereClause")},
								Type:  &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("sqlgen"), "WhereClause")},
							},
						},
					},
				},
				newIfStmt(
					&ast.BinaryExpr{X: ast.NewIdent("where"), Op: token.NEQ, Y: ast.NewIdent("nil")},
					newBlockStmt(
						newAssignStmt(
							[]ast.Expr{ast.NewIdent("whereClause")},
							token.ASSIGN,
							[]ast.Expr{newCallExpr(newSelectorExpr(ast.NewIdent("where"), "Build"))},
						),
					),
					nil,
				),
				newReturnStmt(
					newCallExpr(
						newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "executor"), "Max"),
						ast.NewIdent("ctx"),
						newSelectorExpr(ast.NewIdent("c"), "table"),
						newStringLit(dbColumnName),
						ast.NewIdent("whereClause"),
					),
				),
			)
			decls = append(decls, newFuncDecl("Max"+goFieldName+"Where", fmt.Sprintf("Max%sWhere finds the maximum value of %s with WHERE clause", goFieldName, field.Name), recv, params, results, body))
		}
	}

	return decls
}

// buildModelClientDecls builds all AST declarations for a model client
func buildModelClientDecls(model ModelInfo) []ast.Decl {
	var decls []ast.Decl
	modelName := model.Name

	// 0. ModelClient struct
	clientFields := []*ast.Field{
		{Names: []*ast.Ident{ast.NewIdent("client")}, Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("client"), "PrismaClient")}},
		{Names: []*ast.Ident{ast.NewIdent("executor")}, Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("executor"), "Executor")}},
		{Names: []*ast.Ident{ast.NewIdent("table")}, Type: ast.NewIdent("string")},
		{Names: []*ast.Ident{ast.NewIdent("relations")}, Type: &ast.MapType{
			Key:   ast.NewIdent("string"),
			Value: newSelectorExpr(ast.NewIdent("executor"), "RelationMetadata"),
		}},
	}
	clientStruct := newTypeDecl(
		modelName+"Client",
		fmt.Sprintf("%sClient provides methods for %s operations", modelName, modelName),
		newStructType(clientFields),
	)
	decls = append(decls, clientStruct)

	// 1. WhereBuilder type
	whereBuilderFields := []*ast.Field{
		{Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "WhereBuilder")}},
		{Names: []*ast.Ident{ast.NewIdent("client")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "Client")}},
	}
	whereBuilderType := newTypeDecl(
		modelName+"WhereBuilder",
		fmt.Sprintf("%sWhereBuilder builds WHERE clauses for %s queries", modelName, modelName),
		newStructType(whereBuilderFields),
	)
	decls = append(decls, whereBuilderType)

	// 2. QueryBuilder type
	queryBuilderFields := []*ast.Field{
		{Names: []*ast.Ident{ast.NewIdent("where")}, Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "WhereBuilder")}},
		{Names: []*ast.Ident{ast.NewIdent("joins")}, Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "JoinBuilder")}},
		{Names: []*ast.Ident{ast.NewIdent("orderBy")}, Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "OrderByBuilder")}},
		{Names: []*ast.Ident{ast.NewIdent("include")}, Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "IncludeBuilder")}},
		{Names: []*ast.Ident{ast.NewIdent("select_")}, Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "SelectBuilder")}},
		{Names: []*ast.Ident{ast.NewIdent("limit")}, Type: &ast.StarExpr{X: ast.NewIdent("int")}},
		{Names: []*ast.Ident{ast.NewIdent("offset")}, Type: &ast.StarExpr{X: ast.NewIdent("int")}},
		{Names: []*ast.Ident{ast.NewIdent("client")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "Client")}},
	}
	queryBuilderType := newTypeDecl(
		modelName+"QueryBuilder",
		fmt.Sprintf("%sQueryBuilder builds complete queries for %s", modelName, modelName),
		newStructType(queryBuilderFields),
	)
	decls = append(decls, queryBuilderType)

	// 3. Where() method on Client
	whereMethodRecv := &ast.FieldList{
		List: []*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("c")}, Type: &ast.StarExpr{X: ast.NewIdent(modelName + "Client")}},
		},
	}
	whereMethodParams := &ast.FieldList{}
	whereMethodResults := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: ast.NewIdent(modelName + "WhereBuilder")}},
		},
	}
	whereMethodBody := newBlockStmt(
		newReturnStmt(
			&ast.UnaryExpr{
				Op: token.AND,
				X: newCompositeLit(
					ast.NewIdent(modelName+"WhereBuilder"),
					[]ast.Expr{
						newKeyValueExpr("WhereBuilder", newCallExpr(newSelectorExpr(ast.NewIdent("builder"), "NewWhereBuilder"))),
						newKeyValueExpr("client", ast.NewIdent("c")),
					},
				),
			},
		),
	)
	whereMethod := newFuncDecl("Where", "Where starts building a WHERE clause", whereMethodRecv, whereMethodParams, whereMethodResults, whereMethodBody)
	decls = append(decls, whereMethod)

	// 3.5. Subquery() method on Client
	subqueryParams := &ast.FieldList{}
	subqueryResults := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: newSelectorExpr(ast.NewIdent("builder"), "SubqueryBuilder")}},
		},
	}
	subqueryBody := newBlockStmt(
		newReturnStmt(
			newCallExpr(
				newSelectorExpr(ast.NewIdent("builder"), "NewSubqueryBuilder"),
				newSelectorExpr(ast.NewIdent("c"), "table"),
				newCallExpr(
					newSelectorExpr(newSelectorExpr(ast.NewIdent("c"), "executor"), "GetGenerator"),
				),
			),
		),
	)
	subqueryMethod := newFuncDecl("Subquery", "Subquery creates a subquery builder for this model", whereMethodRecv, subqueryParams, subqueryResults, subqueryBody)
	decls = append(decls, subqueryMethod)

	// 4. FindMany methods
	decls = append(decls, buildFindManyMethods(model)...)

	// 5. Query() method
	decls = append(decls, buildQueryMethod(model)...)

	// 6. FindFirst methods
	decls = append(decls, buildFindFirstMethods(model)...)

	// 7. WhereBuilder Execute methods
	decls = append(decls, buildWhereBuilderExecuteMethods(model)...)

	// 8. QueryBuilder methods
	decls = append(decls, buildQueryBuilderMethods(model)...)

	// 9. Field-specific filter methods
	decls = append(decls, buildFieldFilterMethods(model)...)

	// 10. OrderBy methods
	decls = append(decls, buildOrderByMethods(model)...)

	// 11. Join, Include, Select builders
	decls = append(decls, buildJoinIncludeSelectBuilders(model)...)

	// 12. CRUD methods
	decls = append(decls, buildCRUDMethods(model)...)

	// 13. Aggregation methods
	decls = append(decls, buildAggregationMethods(model)...)

	return decls
}

// GenerateClientFile generates the client.go file using AST
func GenerateClientFile(models []ModelInfo, provider string, outputDir string) error {
	// Create AST file
	file := newFile("generated")

	// Add header comment
	file.Comments = []*ast.CommentGroup{
		{
			List: []*ast.Comment{
				{Text: "// Code generated by prisma-go. DO NOT EDIT."},
			},
		},
	}

	// Add imports
	imports := []string{
		"context",
		"database/sql",
		"time",
		"github.com/satishbabariya/prisma-go/query/builder",
		"github.com/satishbabariya/prisma-go/query/columns",
		"github.com/satishbabariya/prisma-go/query/executor",
		"github.com/satishbabariya/prisma-go/query/sqlgen",
		"github.com/satishbabariya/prisma-go/runtime/client",
	}
	addImports(file, imports)

	// Add PrismaClient struct
	file.Decls = append(file.Decls, buildPrismaClientStruct(models))

	// Add NewPrismaClient function
	file.Decls = append(file.Decls, buildNewPrismaClientFunc(models, provider))

	// Add raw SQL methods
	rawMethods := buildRawSQLMethods()
	file.Decls = append(file.Decls, rawMethods...)

	// Generate all model client declarations using AST
	debug.Debug("Starting client code generation", "model_count", len(models))
	startTime := time.Now()

	for i, model := range models {
		if i%10 == 0 && len(models) > 10 {
			debug.Debug("Generating client code", "progress", fmt.Sprintf("%d/%d", i, len(models)), "elapsed", time.Since(startTime))
		}

		// Generate all declarations for this model client
		modelDecls := buildModelClientDecls(model)
		file.Decls = append(file.Decls, modelDecls...)
	}

	debug.Debug("Client code generation completed", "total_elapsed", time.Since(startTime), "models", len(models), "total_decls", len(file.Decls))

	// Write AST file to disk
	filePath := filepath.Join(outputDir, "client.go")
	debug.Debug("Writing AST file to disk", "path", filePath, "decl_count", len(file.Decls))
	writeStart := time.Now()
	err := writeASTFile(file, filePath)
	if err != nil {
		return err
	}
	debug.Debug("AST file written successfully", "elapsed", time.Since(writeStart))
	return nil
}
