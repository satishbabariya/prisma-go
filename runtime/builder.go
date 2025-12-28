package runtime

import (
	"github.com/satishbabariya/prisma-go/pkg/domain"
)

// BuildFilter converts a map of conditions/operators into a domain.Filter.
func BuildFilter(where map[string]interface{}) domain.Filter {
	if len(where) == 0 {
		return domain.Filter{}
	}

	var conditions []domain.Condition
	var nestedFilters []domain.Filter
	var operator = domain.AND // Default high-level operator is AND

	for key, val := range where {
		switch key {
		case "AND":
			// value should be slice of maps or single map
			if list, ok := val.([]interface{}); ok {
				for _, item := range list {
					if m, ok := item.(map[string]interface{}); ok {
						nestedFilters = append(nestedFilters, BuildFilter(m))
					}
				}
			} else if m, ok := val.(map[string]interface{}); ok {
				nestedFilters = append(nestedFilters, BuildFilter(m))
			} else if list, ok := val.([]map[string]interface{}); ok {
				for _, m := range list {
					nestedFilters = append(nestedFilters, BuildFilter(m))
				}
			}

		case "OR":
			// For OR, we usually create a nested filter group with OR operator
			// But if we are inside a BuildFilter call, we are building ONE filter object.
			// This structure of BuildFilter implies strict structure matching.
			// Prisma "OR" is a list of conditions.
			// Example: OR: [{email: ...}, {name: ...}]
			// We should create a nested filter with OR operator containing these.
			// However, domain.Filter has only ONE operator for its list.
			// So "OR": [...] means we append a nested Filter{Operator: OR, Nested: [BuildFilter(item1), ...]}
			var orConditions []domain.Filter
			if list, ok := val.([]interface{}); ok {
				for _, item := range list {
					if m, ok := item.(map[string]interface{}); ok {
						orConditions = append(orConditions, BuildFilter(m))
					}
				}
			} else if list, ok := val.([]map[string]interface{}); ok {
				for _, m := range list {
					orConditions = append(orConditions, BuildFilter(m))
				}
			}
			if len(orConditions) > 0 {
				nestedFilters = append(nestedFilters, domain.Filter{
					Operator:      domain.OR,
					NestedFilters: orConditions,
				})
			}

		case "NOT":
			// NOT usually takes a single condition map or list
			var notFilter domain.Filter
			if m, ok := val.(map[string]interface{}); ok {
				notFilter = BuildFilter(m)
			} else if list, ok := val.([]interface{}); ok {
				// Implicit AND inside NOT?
				// NOT: [{a:1}, {b:2}] -> NOT (a=1 AND b=2)
				var andGroup []domain.Filter
				for _, item := range list {
					if m, ok := item.(map[string]interface{}); ok {
						andGroup = append(andGroup, BuildFilter(m))
					}
				}
				notFilter = domain.Filter{
					Operator:      domain.AND,
					NestedFilters: andGroup,
				}
			}
			nestedFilters = append(nestedFilters, domain.Filter{
				Operator:      domain.NOT,
				NestedFilters: []domain.Filter{notFilter},
			})

		default:
			// Field condition or Relation filter
			// Key is field name
			// Value can be direct value (equals) or map of operators
			if nestedMap, ok := val.(map[string]interface{}); ok {
				// Check if it's a relation filter (some, every, none, is, isNot)
				// Or field operators (equals, contains, etc.)
				for op, opVal := range nestedMap {
					switch op {
					// Relation filters
					case "some":
						if m, ok := opVal.(map[string]interface{}); ok {
							conditions = append(conditions, domain.Condition{
								Field:    key,
								Operator: domain.Some,
								Value:    BuildFilter(m), // Value for relation filter is a sub-filter
							})
						}
					case "every":
						if m, ok := opVal.(map[string]interface{}); ok {
							conditions = append(conditions, domain.Condition{
								Field:    key,
								Operator: domain.Every,
								Value:    BuildFilter(m),
							})
						}
					case "none":
						if m, ok := opVal.(map[string]interface{}); ok {
							conditions = append(conditions, domain.Condition{
								Field:    key,
								Operator: domain.None,
								Value:    BuildFilter(m),
							})
						}
					case "is": // One-to-one relation filter
						// Treat 'is' as checking relation fields.
						// Actually Prisma uses 'is' for 1:1 same as direct equality sometimes or sub-filter.
						// For Go client, let's treat it as nested filter.
						// But usually 1:1 uses 'is' to match exact fields or 'is: { ... }'
						// Let's assume sub-filter.
						if m, ok := opVal.(map[string]interface{}); ok {
							// For 1:1, we can reuse 'Some' logic or have specific 'Is'?
							// Compiler handles all as subqueries/joins.
							// Let's map to 'Some' for now effectively "Exists related matching..."
							conditions = append(conditions, domain.Condition{
								Field:    key,
								Operator: domain.Some, // logical equivalent for 1:1 usually
								Value:    BuildFilter(m),
							})
						} else if opVal == nil {
							// is: null -> isNull
							conditions = append(conditions, domain.Condition{
								Field:    key,
								Operator: domain.IsNull,
								Value:    true,
							})
						}

					case "isNot":
						// isNot: { ... } -> NOT (is: { ... })
						// or isNot: null -> isNotNull
						if opVal == nil {
							conditions = append(conditions, domain.Condition{
								Field:    key,
								Operator: domain.IsNull,
								Value:    false, // IS NOT NULL
							})
						} else if m, ok := opVal.(map[string]interface{}); ok {
							// NOT (Some(...))
							conditions = append(conditions, domain.Condition{
								Field:    key,
								Operator: domain.None, // logical equivalent for 1:1
								Value:    BuildFilter(m),
							})
						}

					// Field operators
					case "equals":
						conditions = append(conditions, domain.Condition{Field: key, Operator: domain.Equals, Value: opVal})
					case "not":
						conditions = append(conditions, domain.Condition{Field: key, Operator: domain.NotEquals, Value: opVal})
					case "in":
						conditions = append(conditions, domain.Condition{Field: key, Operator: domain.In, Value: opVal})
					case "notIn":
						conditions = append(conditions, domain.Condition{Field: key, Operator: domain.NotIn, Value: opVal})
					case "lt":
						conditions = append(conditions, domain.Condition{Field: key, Operator: domain.Lt, Value: opVal})
					case "lte":
						conditions = append(conditions, domain.Condition{Field: key, Operator: domain.Lte, Value: opVal})
					case "gt":
						conditions = append(conditions, domain.Condition{Field: key, Operator: domain.Gt, Value: opVal})
					case "gte":
						conditions = append(conditions, domain.Condition{Field: key, Operator: domain.Gte, Value: opVal})
					case "contains":
						conditions = append(conditions, domain.Condition{Field: key, Operator: domain.Contains, Value: opVal})
					case "startsWith":
						conditions = append(conditions, domain.Condition{Field: key, Operator: domain.StartsWith, Value: opVal})
					case "endsWith":
						conditions = append(conditions, domain.Condition{Field: key, Operator: domain.EndsWith, Value: opVal})
					case "mode":
						// Special case: modifies previous condition?
						// Map structure is usually { field: { contains: "foo", mode: "insensitive" } }
						// We need to associate mode with the condition.
						// Limitation: Iterate loop order is random. "mode" might come before or after.
						// We might need to iterate twice or store mode?
						// For now, simple implementation assumes mode applies to sibling operators.
						// We will post-process this loop or check map specifically.
					}
				}

				// Handle Mode/Insensitive
				if mode, ok := nestedMap["mode"].(string); ok && mode == "insensitive" {
					for i := range conditions {
						// Apply to last added conditions for this field
						if conditions[i].Field == key {
							conditions[i].Mode = domain.ModeInsensitive
						}
					}
				}

			} else {
				// Direct equality implicit
				conditions = append(conditions, domain.Condition{
					Field:    key,
					Operator: domain.Equals,
					Value:    val,
				})
			}
		}
	}

	return domain.Filter{
		Conditions:    conditions,
		NestedFilters: nestedFilters,
		Operator:      operator,
	}
}
