# Prisma-Go V3 - Clean Architecture

This directory contains the v3 Clean Architecture implementation of Prisma-Go.

## Architecture Overview

The v3 implementation follows Clean Architecture principles with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────┐
│                    FRAMEWORKS & DRIVERS                  │
│  (CLI, Web API, Database Drivers, File System)          │
│                                                           │
│  cmd/prisma/    adapters/database/    adapters/storage/  │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│              INTERFACE ADAPTERS (Ports)                   │
│  (Controllers, Presenters, Gateways, Repositories)       │
│                                                           │
│  repository/    service/    config/                       │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│              APPLICATION BUSINESS RULES                   │
│  (Use Cases, Application Services)                        │
│                                                           │
│  service/migration_service.go                             │
│  service/generate_service.go                              │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│           ENTERPRISE BUSINESS RULES (Domain)              │
│  (Entities, Value Objects, Domain Services)               │
│                                                           │
│  core/schema/    core/migration/    core/query/           │
└─────────────────────────────────────────────────────────┘
```

## Directory Structure

- **cmd/** - CLI entrypoints and dependency injection
- **pkg/** - Public API (stable interfaces for external use)
- **internal/** - Internal implementation (not exposed)
  - **core/** - Domain layer (business rules and entities)
  - **adapters/** - Infrastructure adapters (database, storage, etc.)
  - **repository/** - Data access layer
  - **service/** - Application services (use cases)
  - **config/** - Configuration management
  - **utils/** - Internal utilities
- **runtime/** - Runtime client library (used by generated code)
- **examples/** - Example projects
- **test/** - Test infrastructure
- **docs/** - Documentation

## Key Principles

1. **Dependency Inversion**: Dependencies point inward, domain has no external dependencies
2. **Interface Segregation**: Small, focused interfaces
3. **Single Responsibility**: Each component has one clear purpose
4. **Open/Closed**: Open for extension, closed for modification
5. **Dependency Injection**: All dependencies injected, easily testable

## Getting Started

```bash
# Build
make build

# Run tests
make test

# Run integration tests
make test-integration

# Format code
make fmt

# Run linter
make lint
```

## Development

See [CONTRIBUTING.md](../CONTRIBUTING.md) for development guidelines.

## Documentation

- [Architecture Documentation](docs/architecture/)
- [User Guides](docs/guides/)
- [API Reference](docs/api/)
