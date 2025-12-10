// Package builder provides nested write operations for relations.
package builder

// NestedWriteOperation represents a nested write operation on a relation
type NestedWriteOperation struct {
	Relation string
	Type     NestedWriteType
	Data     interface{} // Can be map, slice, or ID for connect/disconnect
}

// NestedWriteType represents the type of nested write operation
type NestedWriteType string

const (
	NestedWriteCreate     NestedWriteType = "create"
	NestedWriteUpdate     NestedWriteType = "update"
	NestedWriteDelete     NestedWriteType = "delete"
	NestedWriteConnect    NestedWriteType = "connect"
	NestedWriteDisconnect NestedWriteType = "disconnect"
	NestedWriteSet        NestedWriteType = "set"
	NestedWriteUpsert     NestedWriteType = "upsert"
)

// NestedWriteBuilder builds nested write operations
type NestedWriteBuilder struct {
	operations []*NestedWriteOperation
	relation   string
}

// NewNestedWriteBuilder creates a new nested write builder
func NewNestedWriteBuilder(relation string) *NestedWriteBuilder {
	return &NestedWriteBuilder{
		operations: []*NestedWriteOperation{},
		relation:   relation,
	}
}

// Create adds a nested create operation
func (n *NestedWriteBuilder) Create(data interface{}) *NestedWriteBuilder {
	n.operations = append(n.operations, &NestedWriteOperation{
		Relation: n.relation,
		Type:     NestedWriteCreate,
		Data:     data,
	})
	return n
}

// Update adds a nested update operation
func (n *NestedWriteBuilder) Update(data interface{}) *NestedWriteBuilder {
	n.operations = append(n.operations, &NestedWriteOperation{
		Relation: n.relation,
		Type:     NestedWriteUpdate,
		Data:     data,
	})
	return n
}

// Delete adds a nested delete operation
func (n *NestedWriteBuilder) Delete(where interface{}) *NestedWriteBuilder {
	n.operations = append(n.operations, &NestedWriteOperation{
		Relation: n.relation,
		Type:     NestedWriteDelete,
		Data:     where,
	})
	return n
}

// Connect adds a nested connect operation (connect existing records)
func (n *NestedWriteBuilder) Connect(ids interface{}) *NestedWriteBuilder {
	n.operations = append(n.operations, &NestedWriteOperation{
		Relation: n.relation,
		Type:     NestedWriteConnect,
		Data:     ids,
	})
	return n
}

// Disconnect adds a nested disconnect operation (disconnect records)
func (n *NestedWriteBuilder) Disconnect(ids interface{}) *NestedWriteBuilder {
	n.operations = append(n.operations, &NestedWriteOperation{
		Relation: n.relation,
		Type:     NestedWriteDisconnect,
		Data:     ids,
	})
	return n
}

// Set adds a nested set operation (replace all relations)
func (n *NestedWriteBuilder) Set(ids interface{}) *NestedWriteBuilder {
	n.operations = append(n.operations, &NestedWriteOperation{
		Relation: n.relation,
		Type:     NestedWriteSet,
		Data:     ids,
	})
	return n
}

// Upsert adds a nested upsert operation
func (n *NestedWriteBuilder) Upsert(where interface{}, create interface{}, update interface{}) *NestedWriteBuilder {
	n.operations = append(n.operations, &NestedWriteOperation{
		Relation: n.relation,
		Type:     NestedWriteUpsert,
		Data: map[string]interface{}{
			"where":  where,
			"create": create,
			"update": update,
		},
	})
	return n
}

// GetOperations returns all nested write operations
func (n *NestedWriteBuilder) GetOperations() []*NestedWriteOperation {
	return n.operations
}

// HasOperations returns true if there are any nested operations
func (n *NestedWriteBuilder) HasOperations() bool {
	return len(n.operations) > 0
}
