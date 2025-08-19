---
applyTo: "**"
---

# `transx` Package - Copilot Instructions

## Project Overview

`transx` is a Go-based data migration library that supports multiple transfer methods for moving data between databases and storage systems. The library is designed for flexibility, allowing users to specify transfer methods explicitly rather than relying on auto-detection.

## Core Architecture

### Package Structure

The `transx` package provides:

- Core migration functionality in `transx.go`
- Example implementations in `examples/` directory
- Comprehensive testing suite for validation

### Library Design Principles

- **Explicit over Implicit**: Users must specify transfer methods rather than relying on auto-detection
- **Modular Architecture**: Support for multiple transfer protocols through unified interface
- **Configuration-Driven**: JSON-based configuration for maximum flexibility

### Core Migration Model: Extract, Transfer, Load (ETL)

The `transx` package implements a generalized **DataMigrationModel** based on the ETL pattern, which is fundamental to the library's architecture:

#### ETL Pattern Implementation

1. **Extract (Backup)**: Export data from source systems into transferable format

   - Database dumps, file exports, serialization
   - Source-agnostic data extraction
   - Maintains data integrity and consistency

2. **Transfer**: Move extracted data between systems using specified transfer methods

   - Method-agnostic transfer (local, rsync, http, etc.)
   - Support for both direct and relay modes
   - Configurable transfer protocols and authentication

3. **Load (Restore)**: Import transferred data into destination systems
   - Data restoration and deserialization
   - Target-specific data loading
   - Validation and integrity verification

#### DataMigrationModel Generalization

The **DataMigrationModel** abstracts the migration process to support various data sources and destinations:

- **Source-Agnostic**: Works with databases, files, object storage, APIs, etc.
- **Destination-Flexible**: Supports heterogeneous target systems
- **Transfer-Independent**: Separation of data processing from transfer mechanics
- **Mode-Adaptive**: Supports both direct (source→destination) and relay (source→relay→destination) patterns

#### Design Benefits

- **Modularity**: Each ETL phase can be executed independently or as part of a complete workflow
- **Testability**: Individual phases can be tested in isolation
- **Flexibility**: Mix and match different extraction, transfer, and loading strategies
- **Reliability**: Built-in validation and rollback capabilities
- **Scalability**: Support for large-scale data migrations through configurable chunking and parallel processing

### Transfer Methods

The library supports multiple transfer methods that must be explicitly specified by users. Currently supported methods include:

- `local`: Direct local file operations
- `rsync`: SSH-based file transfers using rsync
- `http`: HTTP-based transfers for object storage (S3-compatible)

**Important**: Transfer methods are user-specified via the `Method` field in `TransferOptions`, not auto-detected.

**Extensibility**: The architecture is designed to support additional transfer methods. When adding new methods, follow the existing patterns for validation, configuration, and implementation.

### Transfer Options Structure

```go
type TransferOptions struct {
    Method string `json:"method"` // Required: transfer method (e.g., "local", "rsync", "http")
    // ... other fields
}
```

### Method-Specific Operations

Different transfer methods may support different operations. For example:

#### HTTP Operations

For HTTP transfers, support these operations:

- `download` (GET): Download files from object storage
- `upload` (PUT): Upload files to object storage
- `delete` (DELETE): **Internal use only** - not exposed to end users

**Critical**: DELETE operation should remain internal and not be exposed as a user-facing method/operation.

## Naming Conventions

### Go Code Standards

- Use Go standard naming conventions (CamelCase for exported, camelCase for unexported)
- Struct names should be descriptive and consistent:
  - `HttpTransferOption` (not `S3Option`) for HTTP-based transfers
  - `TransferOptions` for main configuration structure
- Function names should clearly indicate their purpose:
  - `DetectTransferMethod()` → `GetTransferMethod()` (when user specifies method)
  - `IsValidTransferMethod()` for validation

### Configuration Files

- Use descriptive, clear JSON configuration files
- Separate source and destination transfer options:
  - `sourceTransferOptions`
  - `destinationTransferOptions`

## Validation Requirements

### Transfer Method Validation

Always validate user-specified transfer methods:

```go
func IsValidTransferMethod(method string) bool {
    validMethods := []string{"local", "rsync", "http"} // Update this list when adding new methods
    // validation logic
}
```

**Note**: When adding new transfer methods, update the validation function accordingly.

### HTTP Operation Validation

For HTTP transfers, validate operations but keep DELETE internal:

```go
// User-facing operations only
validOperations := []string{"download", "upload"}
```

## Terminology Standards

### Relay Mode Descriptions

When describing relay mode operations, use precise terminology:

- ✅ "Relay mode (source → relay node → destination)"
- ❌ "Relay mode (source → local → destination)"

The intermediate node should be referred to as "relay node" not "local" to avoid confusion.

### Error Messages and Logging

- Use consistent terminology across all error messages
- Include context about which transfer method/operation failed
- Provide actionable error messages for users

## Testing Guidelines

### Comprehensive Testing Requirements

All changes must be tested with:

1. **Direct Mode**: Source → Destination direct transfer
2. **Relay Mode**: Source → Relay Node → Destination transfer
3. **Individual Steps**: Backup, Transfer, Restore operations separately
4. **Data Integrity**: Verify migrated data matches source data

### ETL Pattern Testing

The testing should validate each phase of the ETL pattern:

- **Extract Phase**: Verify backup/export functionality and data completeness
- **Transfer Phase**: Validate data movement across different transfer methods
- **Load Phase**: Confirm restoration accuracy and data integrity
- **End-to-End**: Complete migration workflow validation

### Example Testing Structure

```bash
# Direct mode test
./migrate.sh -m direct

# Relay mode test
./migrate.sh -m relay

# Individual step testing
./migrate.sh -s backup
./migrate.sh -s transfer
./migrate.sh -s restore
```

### Performance Expectations

- Direct mode: ~900ms for typical test data
- Relay mode: ~1000ms for typical test data
- Individual steps: backup ~85ms, transfer ~233ms, restore ~308ms

## Configuration Management

### Example Structure

Maintain separate configuration files for different modes:

- `direct-mode-config.json`: Direct transfer configuration
- `relay-mode-config.json`: Relay transfer with intermediate node

### Required Configuration Fields

Ensure all transfer configurations include:

- Explicit `method` specification
- Appropriate authentication details (SSH keys, HTTP credentials)
- Source and destination connection parameters

## Development Workflow

### Making Changes

1. Always test compilation after changes: `go build`
2. Run comprehensive tests in examples/mariadb-migration
3. Verify data integrity after migrations
4. Test both direct and relay modes

### Adding New Features

- Follow existing patterns for transfer method implementation
- Maintain backward compatibility with existing configurations
- Update example configurations and documentation
- Ensure proper validation for new methods/operations

### Adding New Transfer Methods

When implementing new transfer methods:

1. **Validation**: Add the new method to `IsValidTransferMethod()` function
2. **Configuration**: Create method-specific option structures following naming conventions
3. **Implementation**: Follow existing patterns in transfer logic
4. **Testing**: Add comprehensive tests for the new method
5. **Documentation**: Update configuration examples and method descriptions
6. **Operations**: Define method-specific operations (if applicable)

### Method Implementation Guidelines

- Each method should have its own configuration structure (e.g., `HttpTransferOption`, `RsyncTransferOption`)
- Follow consistent naming patterns for configuration fields
- Implement proper error handling and validation
- Support both direct and relay mode operations
- Provide clear error messages with method context

## Common Pitfalls to Avoid

1. **Auto-detection**: Never implement auto-detection of transfer methods - always require explicit user specification
2. **DELETE Exposure**: Keep DELETE operations internal, not user-facing
3. **Terminology Confusion**: Use "relay node" not "local" in relay mode descriptions
4. **Naming Inconsistency**: Maintain consistent naming (HttpTransferOption, not S3Option)
5. **Missing Validation**: Always validate user inputs for transfer methods and operations

## Integration Notes

### CB-Spider Compatibility

The library is designed to work with CB-Spider project for S3 Object Storage operations using presigned URLs. Ensure HTTP transfer methods support:

- Presigned URL authentication
- Standard S3-compatible operations (GET, PUT)
- Proper error handling for object storage scenarios

This instruction set should be referenced for all development work on the `transx` package to maintain consistency and architectural integrity.
