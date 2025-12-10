// Package builder provides nested include support for deep relation loading.
package builder

// NestedInclude represents a hierarchical include structure
type NestedInclude struct {
	Relation string                    // Relation name
	Nested   map[string]*NestedInclude // Nested includes
}

// NewNestedInclude creates a new nested include
func NewNestedInclude(relation string) *NestedInclude {
	return &NestedInclude{
		Relation: relation,
		Nested:   make(map[string]*NestedInclude),
	}
}

// Add adds a nested relation
func (n *NestedInclude) Add(relation string) *NestedInclude {
	if nested, exists := n.Nested[relation]; exists {
		return nested
	}
	nested := NewNestedInclude(relation)
	n.Nested[relation] = nested
	return nested
}

// Flatten converts nested includes to a flat map with dot notation
// Example: {posts: {author: {}}} becomes {"posts": true, "posts.author": true}
func (n *NestedInclude) Flatten() map[string]bool {
	result := make(map[string]bool)
	n.flattenRecursive("", result)
	return result
}

func (n *NestedInclude) flattenRecursive(prefix string, result map[string]bool) {
	currentPath := n.Relation
	if prefix != "" {
		currentPath = prefix + "." + n.Relation
	}
	
	if currentPath != "" {
		result[currentPath] = true
	}
	
	for _, nested := range n.Nested {
		nested.flattenRecursive(currentPath, result)
	}
}

// GetPath returns the full path of this include
func (n *NestedInclude) GetPath(prefix string) string {
	if prefix == "" {
		return n.Relation
	}
	return prefix + "." + n.Relation
}

// HasNested returns true if this include has nested relations
func (n *NestedInclude) HasNested() bool {
	return len(n.Nested) > 0
}

// GetNestedIncludes returns nested includes as a map
func (n *NestedInclude) GetNestedIncludes() map[string]*NestedInclude {
	return n.Nested
}

