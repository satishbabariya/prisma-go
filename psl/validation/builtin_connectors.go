// Package pslcore provides builtin database connectors for Prisma schemas.
package validation

// BuiltinConnectors manages the registry of builtin connectors.
type BuiltinConnectors struct {
	connectors []ExtendedConnector
}

// NewBuiltinConnectors creates a new builtin connectors registry.
func NewBuiltinConnectors() *BuiltinConnectors {
	return &BuiltinConnectors{
		connectors: []ExtendedConnector{
			NewPostgresConnector(),
			NewMySqlConnector(),
			NewSqliteConnector(),
			NewMsSqlConnector(),
			NewCockroachConnector(),
			NewMongoDbConnector(),
		},
	}
}

// GetConnector returns a connector by provider name.
func (bc *BuiltinConnectors) GetConnector(providerName string) ExtendedConnector {
	for _, connector := range bc.connectors {
		if connector.IsProvider(providerName) {
			return connector
		}
	}
	return nil
}

// GetAllConnectors returns all available connectors.
func (bc *BuiltinConnectors) GetAllConnectors() []ExtendedConnector {
	return bc.connectors
}

// ConnectorRegistry represents a registry of connectors.
type ConnectorRegistry struct {
	connectors []ExtendedConnector
}

// NewConnectorRegistry creates a new connector registry.
func NewConnectorRegistry() *ConnectorRegistry {
	return &ConnectorRegistry{
		connectors: []ExtendedConnector{
			NewPostgresConnector(),
			NewMySqlConnector(),
			NewSqliteConnector(),
			NewMsSqlConnector(),
			NewCockroachConnector(),
			NewMongoDbConnector(),
		},
	}
}

// RegisterConnector registers a new connector.
func (cr *ConnectorRegistry) RegisterConnector(connector ExtendedConnector) {
	cr.connectors = append(cr.connectors, connector)
}

// GetConnector returns a connector by provider name.
func (cr *ConnectorRegistry) GetConnector(providerName string) ExtendedConnector {
	for _, connector := range cr.connectors {
		if connector.IsProvider(providerName) {
			return connector
		}
	}
	return nil
}

// GetConnectors returns all registered connectors.
func (cr *ConnectorRegistry) GetConnectors() []ExtendedConnector {
	return cr.connectors
}
