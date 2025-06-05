# MariaDB Data Migration Example with transx

This example demonstrates how to perform MariaDB database migration using the transx library.

## Overview

This example performs the following tasks:

1. Database backup from source MariaDB (mariadb-dump / mysqldump)
2. Transfer backup files to the target server (rsync)
3. Database restoration on the target MariaDB

Supported migration modes:

1. **Direct Mode**: One-step direct transfer between source and destination

   - Local-to-local (testing on the same machine)
   - Local-to-remote (local source to a remote destination)
   - Remote-to-local (remote source to local destination)

2. **Relay Mode**: Two-step transfer where both source and destination are remote
   - Source → Local Machine → Destination (local system acts as an intermediary)

## Prerequisites

- Go 1.16 or higher
- Docker installed
- For remote scenarios: SSH access and key files

## Environment Setup

1. Run the environment setup script to configure the test environment:

```bash
chmod +x setup_environment.sh
./setup_environment.sh
```

This script performs the following tasks:

- Installs Docker if it's not already installed
- Creates `mariadb_source` and `mariadb_target` containers
- Generates test data in the source database

## How to Run the Example

### Using the Simple Migration Script

The easiest way to run migrations is to use the included migrate.sh script:

```bash
chmod +x migrate.sh
./migrate.sh [direct|relay] [flags]
```

Examples:

```bash
# Run direct mode migration (default)
./migrate.sh direct

# Run relay mode migration
./migrate.sh relay

# Run direct mode with verbose logging
./migrate.sh direct --verbose

# Run only the backup step in relay mode
./migrate.sh relay --backup
```

### Using the Test Migration Script

For more advanced testing scenarios, you can use the test_migration.sh script:

```bash
chmod +x test_migration.sh
./test_migration.sh [config] [mode]
```

Examples:

```bash
# Test direct mode migration
./test_migration.sh direct

# Test relay mode migration
./test_migration.sh relay

# Run only the backup step for relay migration
./test_migration.sh relay backup

# Run test relay migration
./test_migration.sh test
```

### Preparing JSON Configuration Files

Choose one of the two available configuration templates based on your migration needs:

#### Direct Mode Configuration (direct-mode-config.json)

For local-to-local, local-to-remote, or remote-to-local migrations:

```json
{
  "source": {
    "username": "ubuntu",
    "hostIP": "", // Leave empty for local source, or set to IP/hostname for remote
    "sshPort": 22,
    "path": "/path/to/backup", // Source backup directory path
    "sshPrivateKey": "~/.ssh/id_rsa",
    "backupCmd": "docker exec mariadb_source mariadb-dump -u root -p'password' database_name > /path/to/backup/dump.sql"
  },
  "destination": {
    "username": "ubuntu",
    "hostIP": "", // Leave empty for local destination, or set to IP/hostname for remote
    "sshPort": 22,
    "path": "/path/to/restore", // Destination restore directory path
    "sshPrivateKey": "~/.ssh/id_rsa",
    "restoreCmd": "docker exec -i mariadb_target mariadb -u root -p'password' database_name < /path/to/restore/dump.sql"
  },
  "rsyncOptions": {
    "compress": true,
    "archive": true,
    "verbose": true,
    "delete": false,
    "progress": true,
    "insecureSkipHostKeyVerification": false,
    "exclude": ["*.tmp", "*.log"],
    "extraArgs": ["--checksum", "--timeout=300"]
  }
}
```

#### Remote Migration Configuration (remote-migration-config.json)

```json
{
  "source": {
    "username": "ubuntu",
    "hostIP": "",
    "sshPort": 22,
    "path": "/home/ubuntu/mariadb_dump",
    "sshPrivateKey": "~/.ssh/id_rsa",
    "backupCmd": "docker exec mariadb_source mariadb-dump -u root -p'your_root_password' poc_db > /home/ubuntu/mariadb_dump/poc_db_dump.sql"
  },
  "destination": {
    "username": "ubuntu",
    "hostIP": "15.165.228.224",
    "sshPort": 22,
    "path": "/home/ubuntu/mariadb_dump",
    "sshPrivateKey": "~/.ssh/kimy-aws.pem",
    "restoreCmd": "docker exec -i mariadb_target mariadb -u root -p'your_root_password' poc_db < /home/ubuntu/mariadb_dump/poc_db_dump.sql"
  },
  "rsyncOptions": {
    "compress": true,
    "archive": true,
    "verbose": true,
    "delete": false,
    "progress": true,
    "insecureSkipHostKeyVerification": true
  }
}
```

### Running Migration

#### Local Migration Test

When source and destination are on the same system:

```bash
go run main.go --config=migration-config.json
```

#### Migration to Remote System

Migrating data to a remote system:

```bash
go run main.go --config=remote-migration-config.json
```

### Using the run_migration.sh Script

The `run_migration.sh` script provides a more user-friendly way to run migrations with either direct or relay mode:

```bash
chmod +x run_migration.sh
./run_migration.sh [options]
```

Examples:

```bash
# Run direct mode migration locally
./run_migration.sh --mode direct

# Run direct mode migration to a remote server
./run_migration.sh --mode direct --dest-host 192.168.1.20 --dest-user ubuntu

# Run relay mode migration (through local machine)
./run_migration.sh --mode relay --source-host 192.168.1.10 --source-user user1 --dest-host 192.168.1.20 --dest-user user2
```

Options:

- `--mode`: Specify `direct` or `relay` migration mode
- `--source-host`, `--source-user`, `--source-key`: Source server connection details
- `--dest-host`, `--dest-user`, `--dest-key`: Destination server connection details

### Running Individual Steps

To run specific steps only, you can use the following flags:

```bash
# Run backup step only
go run main.go --config=direct-mode-config.json --backup

# Run transfer step only
go run main.go --config=direct-mode-config.json --transfer

# Run restore step only
go run main.go --config=direct-mode-config.json --restore
```

## Available Flags

| Flag         | Description                     | Default                   |
| ------------ | ------------------------------- | ------------------------- |
| `--config`   | Migration config JSON file path | `direct-mode-config.json` |
| `--backup`   | Run backup step only            | `false`                   |
| `--transfer` | Run transfer step only          | `false`                   |
| `--restore`  | Run restore step only           | `false`                   |
| `--verbose`  | Enable verbose logging          | `false`                   |

## Example Scenarios

### Scenario: Local Test

1. Run the environment setup script
2. Execute the local migration test command
3. Verify the database:

```bash
docker exec -it mariadb_target mariadb -u root -p'your_root_password' poc_db -e "SELECT * FROM products;"
```

### Scenario: Migration to a Remote Server

1. Run the environment setup script on the source system
2. Configure the MariaDB container on the remote system
3. Execute the remote migration command
4. Verify data on the remote system

### Scenario: Server to VM Migration

This scenario is for migrating a MariaDB database from a physical server to a VM in a real production environment.

#### Prerequisites

- Source server: MariaDB must be running and SSH access must be available
- Target VM: MariaDB container must be running and SSH access must be available
- Both systems must have rsync installed

#### Migration Configuration Settings (server-to-vm-config.json)

```json
{
  "source": {
    "username": "ubuntu",
    "hostIP": "192.168.1.10",
    "sshPort": 22,
    "path": "/var/lib/mysql/backup",
    "sshPrivateKey": "~/.ssh/id_rsa",
    "backupCmd": "sudo mysqldump -u root -p'your_root_password' --all-databases > /var/lib/mysql/backup/all_databases.sql"
  },
  "destination": {
    "username": "ubuntu",
    "hostIP": "192.168.1.20",
    "sshPort": 22,
    "path": "/home/ubuntu/mariadb_backup",
    "sshPrivateKey": "~/.ssh/vm_key.pem",
    "restoreCmd": "docker exec -i mariadb_target mysql -u root -p'your_root_password' < /home/ubuntu/mariadb_backup/all_databases.sql"
  },
  "rsyncOptions": {
    "compress": true,
    "archive": true,
    "verbose": true,
    "delete": false,
    "progress": true,
    "insecureSkipHostKeyVerification": true
  }
}
```

#### Running Migration

1. Run the environment validation script:

```bash
chmod +x setup_environment.sh
./setup_environment.sh
```

2. Run the complete migration:

```bash
go run main.go --config=server-to-vm-config.json
```

3. Step-by-step migration (if needed):

```bash
# Run backup only from the source server
go run main.go --config=server-to-vm-config.json --backup

# Run data transfer only from source server to VM
go run main.go --config=server-to-vm-config.json --transfer

# Run restore only on VM
go run main.go --config=server-to-vm-config.json --restore
```

## Troubleshooting

- **SSH connection failure**: Verify SSH key permissions (chmod 600 <key_file>)
- **Permission errors**: Check directory permissions
- **Container access denied**: Verify Docker group permissions, log out and log back in

### Scenario: Relay Migration (Remote-to-Remote)

This scenario is for cases where both source and target servers are remote. In this case, the current system acts as an intermediary.

#### How It Works

1. Execute backup command on source server (via SSH)
2. Download backup files from source server to local system (rsync)
3. Upload backup files from local system to target server (rsync)
4. Execute restore command on target server (via SSH)

#### Relay Migration Configuration Settings (relay-migration-config.json)

```json
{
  "source": {
    "username": "user1",
    "hostIP": "192.168.1.10",
    "sshPort": 22,
    "path": "/var/lib/mysql/backup",
    "sshPrivateKey": "~/.ssh/server1_key",
    "backupCmd": "sudo mysqldump -u root -p'password1' --all-databases > /var/lib/mysql/backup/all_databases.sql"
  },
  "destination": {
    "username": "user2",
    "hostIP": "192.168.1.20",
    "sshPort": 22,
    "path": "/home/user2/mariadb_backup",
    "sshPrivateKey": "~/.ssh/server2_key",
    "restoreCmd": "sudo mysql -u root -p'password2' < /home/user2/mariadb_backup/all_databases.sql"
  },
  "rsyncOptions": {
    "compress": true,
    "archive": true,
    "verbose": true,
    "delete": false,
    "progress": true,
    "insecureSkipHostKeyVerification": true
  }
}
```

#### Running Relay Migration

```bash
go run main.go --config=relay-migration-config.json
```

This command performs the following:

1. Connect to source server via SSH and execute backup command
2. Download backup files from source server to local temporary directory
3. Upload files from local temporary directory to target server
4. Connect to target server via SSH and execute restore command

#### Test Configuration for Relay Migration

To test the relay migration functionality locally, you can use the following test configuration file:

```bash
go run main.go --config=test-relay-config.json --verbose
```

This test configuration simulates migration between two directories on the local system and tests all steps of relay migration using SSH local connections.
