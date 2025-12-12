package parsing

import (
	"strconv"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
	v2 "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

func convertSchema(schema *v2.SchemaAst) *ast.SchemaAst {
	tops := make([]ast.Top, 0, len(schema.Tops))
	for _, top := range schema.Tops {
		if t := convertTop(top); t != nil {
			tops = append(tops, t)
		}
	}
	return &ast.SchemaAst{Tops: tops}
}

func convertTop(top v2.Top) ast.Top {
	switch t := top.(type) {
	case *v2.Model:
		return convertModel(t)
	case *v2.Enum:
		return convertEnum(t)
	case *v2.SourceConfig:
		return convertSource(t)
	case *v2.GeneratorConfig:
		return convertGenerator(t)
	case *v2.CompositeType:
		return convertCompositeType(t)
	}
	return nil
}

func convertModel(m *v2.Model) *ast.Model {
	return &ast.Model{
		Name:          convertIdentifier(m.Name),
		Fields:        convertFields(m.Fields),
		Attributes:    convertBlockAttributes(m.BlockAttributes),
		Documentation: convertCommentBlock(m.Documentation),
		IsView:        m.IsView(),
		ASTSpan:       convertPosToSpan(m.Pos, len("model")+len(m.GetName())), // Approximate
	}
}

func convertEnum(e *v2.Enum) *ast.Enum {
	values := make([]ast.EnumValue, len(e.Values))
	for i, v := range e.Values {
		values[i] = ast.EnumValue{
			Name:          convertIdentifier(v.Name),
			Attributes:    convertAttributes(v.Attributes),
			Documentation: convertCommentBlock(v.Documentation),
			ASTSpan:       convertPosToSpan(v.Pos, len(v.GetName())),
		}
	}
	return &ast.Enum{
		Name:          convertIdentifier(e.Name),
		Values:        values,
		Attributes:    convertBlockAttributes(e.BlockAttributes),
		Documentation: convertCommentBlock(e.Documentation),
		ASTSpan:       convertPosToSpan(e.Pos, len("enum")+len(e.GetName())),
	}
}

func convertCompositeType(c *v2.CompositeType) *ast.CompositeType {
	return &ast.CompositeType{
		Name:   convertIdentifier(c.Name),
		Fields: convertFields(c.Fields),
		// Attributes:    convertBlockAttributes(c.BlockAttributes), // Not present in v2 CompositeType
		Documentation: convertCommentBlock(c.Documentation),
		ASTSpan:       convertPosToSpan(c.Pos, len("type")+len(c.Name.Name)),
	}
}

func convertSource(s *v2.SourceConfig) *ast.SourceConfig {
	return &ast.SourceConfig{
		Name:          convertIdentifier(s.Name),
		Properties:    convertConfigProperties(s.Properties),
		Documentation: convertCommentBlock(s.Documentation),
		ASTSpan:       convertPosToSpan(s.Pos, len("datasource")+len(s.Name.Name)),
	}
}

func convertGenerator(g *v2.GeneratorConfig) *ast.GeneratorConfig {
	return &ast.GeneratorConfig{
		Name:          convertIdentifier(g.Name),
		Properties:    convertConfigProperties(g.Properties),
		Documentation: convertCommentBlock(g.Documentation),
		ASTSpan:       convertPosToSpan(g.Pos, len("generator")+len(g.Name.Name)),
	}
}

func convertFields(fields []*v2.Field) []ast.Field {
	res := make([]ast.Field, len(fields))
	for i, f := range fields {
		res[i] = ast.Field{
			Name:          convertFieldName(f.Name),
			FieldType:     convertFieldType(f.Type),
			Arity:         convertArity(f.Arity),
			Attributes:    convertAttributes(f.Attributes),
			Documentation: convertCommentBlock(f.Documentation),
			ASTSpan:       convertPosToSpan(f.Pos, len(f.GetName())),
		}
	}
	return res
}

func convertFieldType(t *v2.FieldType) ast.FieldType {
	if t == nil {
		// Should not happen for valid schema field types
		return ast.FieldType{
			Type: ast.SupportedFieldType{
				Identifier: ast.Identifier{Name: "String"},
			},
		}
	}

	if t.IsUnsupported() {
		return ast.FieldType{
			Type: ast.UnsupportedFieldType{
				TypeName: *t.Unsupported,
			},
			ASTSpan: convertPosToSpan(t.Pos, len("Unsupported")+len(*t.Unsupported)+2),
		}
	}

	return ast.FieldType{
		Type: ast.SupportedFieldType{
			Identifier: ast.Identifier{Name: t.Name},
		},
		ASTSpan: convertPosToSpan(t.Pos, len(t.Name)),
	}
}

func convertArity(arity v2.FieldArity) ast.FieldArity {
	switch arity {
	case v2.FieldArityList:
		return ast.List
	case v2.FieldArityOptional:
		return ast.Optional
	case v2.FieldArityRequired:
		return ast.Required
	default:
		return ast.Required
	}
}

func convertAttributes(attrs []*v2.Attribute) []ast.Attribute {
	res := make([]ast.Attribute, len(attrs))
	for i, a := range attrs {
		res[i] = ast.Attribute{
			Name:      convertIdentifier(a.Name),
			Arguments: convertArguments(a.Arguments),
			Span:      convertPosToSpan(a.Pos, len(a.Name.Name)+1), // +1 for @
		}
	}
	return res
}

func convertBlockAttributes(attrs []*v2.BlockAttribute) []ast.Attribute {
	res := make([]ast.Attribute, len(attrs))
	for i, a := range attrs {
		res[i] = ast.Attribute{
			Name:      convertIdentifier(a.Name),
			Arguments: convertArguments(a.Arguments),
			Span:      convertPosToSpan(a.Pos, len(a.Name.Name)+2), // +2 for @@
		}
	}
	return res
}

func convertArguments(args *v2.ArgumentsList) ast.ArgumentsList {
	if args == nil || len(args.Arguments) == 0 {
		return ast.ArgumentsList{}
	}

	res := make([]ast.Argument, len(args.Arguments))
	for i, a := range args.Arguments {
		var name *ast.Identifier
		if a.Name != nil {
			n := convertIdentifier(a.Name)
			name = &n
		}

		val := convertExpression(a.Value)
		// Calculate span for argument
		// We use a rough estimate if value span is available
		var s diagnostics.Span
		if val != nil {
			s = val.Span()
		}

		res[i] = ast.Argument{
			Name:  name,
			Value: val,
			Span:  s,
		}
	}
	return ast.ArgumentsList{Arguments: res}
}

func convertConfigProperties(props []*v2.ConfigBlockProperty) []ast.ConfigBlockProperty {
	res := make([]ast.ConfigBlockProperty, len(props))
	for i, p := range props {
		res[i] = ast.ConfigBlockProperty{
			Name:    convertIdentifier(p.Name),
			Value:   convertExpressionPtr(p.Value),
			ASTSpan: convertPosToSpan(p.Pos, len(p.Name.Name)),
		}
	}
	return res
}

func convertExpressionPtr(expr v2.Expression) ast.Expression {
	if expr == nil {
		return nil
	}
	e := convertExpression(expr)
	return e
}

func convertExpression(expr v2.Expression) ast.Expression {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *v2.StringValue:
		return &ast.StringLiteral{Value: e.Value, ASTSpan: convertPosToSpan(e.Pos, len(e.Value)+2)}
	case *v2.NumericValue:
		span := convertPosToSpan(e.Pos, len(e.Value))
		// Try to parse int
		if i, err := strconv.Atoi(e.Value); err == nil {
			return &ast.IntLiteral{Value: i, ASTSpan: span}
		}
		// Try to parse float
		if f, err := strconv.ParseFloat(e.Value, 64); err == nil {
			return &ast.FloatLiteral{Value: f, ASTSpan: span}
		}
		return &ast.NumericValue{Value: e.Value, ASTSpan: span}
	case *v2.ConstantValue:
		span := convertPosToSpan(e.Pos, len(e.Value))
		if e.Value == "true" {
			return &ast.BooleanLiteral{Value: true, ASTSpan: span}
		}
		if e.Value == "false" {
			return &ast.BooleanLiteral{Value: false, ASTSpan: span}
		}
		return &ast.ConstantValue{Value: e.Value, ASTSpan: span}
	case *v2.FunctionCall:
		var args []ast.Expression
		if e.Arguments != nil {
			args = make([]ast.Expression, len(e.Arguments.Arguments))
			for i, arg := range e.Arguments.Arguments {
				args[i] = convertExpression(arg.Value)
			}
		}
		return &ast.FunctionCall{
			Name:      ast.Identifier{Name: e.Name, ASTSpan: convertPosToSpan(e.Pos, len(e.Name))},
			Arguments: args,
			ASTSpan:   convertPosToSpan(e.Pos, len(e.Name)), // Approximation
		}
	case *v2.ArrayExpression:
		elems := make([]ast.Expression, len(e.Elements))
		for i, el := range e.Elements {
			elems[i] = convertExpression(el)
		}
		return &ast.ArrayLiteral{Elements: elems, ASTSpan: convertPosToSpan(e.Pos, 2)} // []
	}
	return nil
}

func convertIdentifier(id *v2.Identifier) ast.Identifier {
	if id == nil {
		return ast.Identifier{}
	}
	return ast.Identifier{
		Name:    id.Name,
		ASTSpan: convertPosToSpan(id.Pos, len(id.Name)),
	}
}

func convertFieldName(fn *v2.FieldName) ast.Identifier {
	if fn == nil {
		return ast.Identifier{}
	}
	return ast.Identifier{
		Name:    fn.Name,
		ASTSpan: convertPosToSpan(fn.Pos, len(fn.Name)),
	}
}

func convertCommentBlock(c *v2.CommentBlock) *ast.Comment {
	if c == nil {
		return nil
	}
	text := c.GetText()
	// Span calculation for comments is tricky as they are multiple lines
	// Just taking 0 for now or first comment pos
	var span diagnostics.Span
	if len(c.Comments) > 0 {
		span = convertPosToSpan(c.Comments[0].Pos, len(text))
	}

	return &ast.Comment{Text: text, Span: span}
}

func convertPosToSpan(pos lexer.Position, length int) diagnostics.Span {
	return diagnostics.Span{
		Start:  pos.Offset,
		End:    pos.Offset + length,
		FileID: diagnostics.FileIDZero,
	}
}
