# MariaDB Data Migration Example with transx

This example demonstrates how to perform MariaDB database migration using the transx library.

## Overview

This example performs the following tasks:

1. Database backup from source MariaDB (mariadb-dump / mysqldump)
2. Transfer backup files to the target server (rsync)
3. Database restoration on the target MariaDB

Supported migration scenarios:

- Local migration (testing on the same machine)
- Remote migration (migrating to another machine)
- Server to VM migration (transferring database from existing server to VM)
- Relay migration (when both source and target are remote, the local system acts as an intermediary)

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

### Using the Simple Test Script

You can easily test various migration scenarios using the provided test script:

```bash
chmod +x test_migration.sh
./test_migration.sh [config] [mode]
```

Examples:

```bash
# Test local migration
./test_migration.sh local

# Test relay mode (for testing between 127.0.0.1)
./test_migration.sh test

# Run only the backup step for server-VM migration
./test_migration.sh vm backup
```

### Preparing JSON Configuration Files

Prepare a JSON configuration file for the migration task. Example files are as follows:

#### Local Migration Configuration (migration-config.json)

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
    "hostIP": "",
    "sshPort": 22,
    "path": "/home/ubuntu/mariadb_dump",
    "sshPrivateKey": "~/.ssh/id_rsa",
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

### Running Individual Steps

To run specific steps only, you can use the following flags:

```bash
# Run backup step only
go run main.go --config=migration-config.json --backup

# Run transfer step only
go run main.go --config=migration-config.json --transfer

# Run restore step only
go run main.go --config=migration-config.json --restore
```

## Available Flags

| Flag         | Description                     | Default                 |
| ------------ | ------------------------------- | ----------------------- |
| `--config`   | Migration config JSON file path | `migration-config.json` |
| `--backup`   | Run backup step only            | `false`                 |
| `--transfer` | Run transfer step only          | `false`                 |
| `--restore`  | Run restore step only           | `false`                 |

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
