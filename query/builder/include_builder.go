// Package builder provides include builder functionality for relations.
package builder

// IncludeBuilder builds include clauses for relations
type IncludeBuilder struct {
	includes map[string]*NestedInclude // Map of top-level includes
	current  *NestedInclude            // Current include being built
}

// NewIncludeBuilder creates a new include builder
func NewIncludeBuilder() *IncludeBuilder {
	return &IncludeBuilder{
		includes: make(map[string]*NestedInclude),
	}
}

// Include adds a relation to include and returns the builder for chaining
func (i *IncludeBuilder) Include(relation string) *IncludeBuilder {
	if _, exists := i.includes[relation]; !exists {
		i.includes[relation] = NewNestedInclude(relation)
	}
	i.current = i.includes[relation]
	return i
}

// GetIncludes returns the included relations as a flat map
// This is for backward compatibility with existing code
func (i *IncludeBuilder) GetIncludes() map[string]bool {
	result := make(map[string]bool)
	for relation := range i.includes {
		result[relation] = true
	}
	return result
}

// GetNestedIncludes returns the full nested include structure
func (i *IncludeBuilder) GetNestedIncludes() map[string]*NestedInclude {
	return i.includes
}

// GetFlattenedIncludes returns all includes in dot notation
// Example: {posts: {author: {}}} becomes {"posts": true, "posts.author": true}
func (i *IncludeBuilder) GetFlattenedIncludes() map[string]bool {
	result := make(map[string]bool)
	for relation, nested := range i.includes {
		// Add the top-level relation
		result[relation] = true
		// Add all nested relations with dot notation
		for nestedRelation := range nested.Flatten() {
			if nestedRelation != "" && nestedRelation != relation {
				result[nestedRelation] = true
			}
		}
	}
	return result
}

// HasIncludes returns true if there are any includes
func (i *IncludeBuilder) HasIncludes() bool {
	return len(i.includes) > 0
}
