# transx

A Go-ba# Configure transfer options
options := transx.TransferOptions{
Method: "rsync", // or "object-storage-api"
// ... other configuration
}ata migration library that supports multiple transfer methods for moving data between databases and storage systems.

## Features

- **ETL Pattern**: Extract, Transfer, Load architecture for reliable data migration
- **Multiple Transfer Methods**: Support for `rsync` and `object-storage-api` transfers
- **Explicit Configuration**: User-specified transfer methods (no auto-detection)
- **Direct & Relay Modes**: Flexible migration patterns
- **Data Integrity**: Built-in validation and verification

## Quick Start

```go
import "github.com/yunkon-kim/transx"

// Configure transfer options
options := transx.TransferOptions{
    Method: "rsync", // or "object-storage-api"
    // ... other configuration
}

// Perform migration
err := transx.Migrate(sourceConfig, destConfig, options)
```

## Transfer Methods

| Method               | Description                                   | Use Case                                        |
| -------------------- | --------------------------------------------- | ----------------------------------------------- |
| `rsync`              | SSH-based file transfers using rsync          | Remote server migrations                        |
| `object-storage-api` | HTTP-based transfers with Object Storage APIs | S3-compatible storage (CB-Spider, AWS S3, etc.) |

Note: Direct local file operations are handled automatically based on endpoint configuration (empty endpoint = local).

## Architecture

The library follows the **DataMigrationModel** using ETL pattern:

1. **Extract (Backup)**: Export data from source systems
2. **Transfer**: Move data using specified transfer method
3. **Load (Restore)**: Import data into destination systems

## Examples

See [examples/mariadb-migration](examples/mariadb-migration/) for a complete MariaDB migration example with:

- Direct mode migration
- Relay mode migration
- Individual step execution
- Comprehensive testing

## Installation

```bash
go get github.com/yunkon-kim/transx
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
