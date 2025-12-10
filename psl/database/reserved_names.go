// Package parserdatabase provides reserved name checking functionality.
package database

// IsReservedTypeName checks if a name is a reserved type name for the Prisma Client API.
func IsReservedTypeName(name string) bool {
	_, exists := reservedNames[name]
	return exists
}

// reservedNames contains all reserved names that cannot be used as type names.
// The source of the following list is from prisma-client. Any edit should be done in both places.
// https://github.com/prisma/prisma/blob/master/src/packages/client/src/generation/generateClient.ts#L443
var reservedNames = map[string]bool{
	"PrismaClient": true,
	// JavaScript keywords
	"async":      true,
	"await":      true,
	"break":      true,
	"case":       true,
	"catch":      true,
	"class":      true,
	"const":      true,
	"continue":   true,
	"debugger":   true,
	"default":    true,
	"delete":     true,
	"do":         true,
	"else":       true,
	"enum":       true,
	"export":     true,
	"extends":    true,
	"false":      true,
	"finally":    true,
	"for":        true,
	"function":   true,
	"if":         true,
	"implements": true,
	"import":     true,
	"in":         true,
	"instanceof": true,
	"interface":  true,
	"let":        true,
	"new":        true,
	"null":       true,
	"package":    true,
	"private":    true,
	"protected":  true,
	"public":     true,
	"return":     true,
	"super":      true,
	"switch":     true,
	"this":       true,
	"throw":      true,
	"true":       true,
	"try":        true,
	"typeof":     true,
	"using":      true,
	"var":        true,
	"void":       true,
	"while":      true,
	"with":       true,
	"yield":      true,
}
