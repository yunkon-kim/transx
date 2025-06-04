#!/bin/bash

# Example script for actual cloud migration using the transx library

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default settings
SOURCE_HOST=""
SOURCE_USER="$(whoami)"
SOURCE_KEY="$HOME/.ssh/id_rsa"
SOURCE_PATH="$HOME/mariadb_dump"
DEST_HOST=""
DEST_USER=""
DEST_KEY="$HOME/.ssh/id_rsa"
DEST_PATH="/home/ubuntu/mariadb_dump"
DB_NAME="poc_db"
DB_PASSWORD="your_root_password"
VERBOSE="-v"
COMPRESS="-z"

# Function to display usage
usage() {
    echo -e "${BLUE}Usage:${NC} $0 [options]"
    echo ""
    echo "Options:"
    echo "  -sh, --source-host      Source host (empty for local)"
    echo "  -su, --source-user      Source user [default: current user]"
    echo "  -sk, --source-key       Source SSH private key [default: ~/.ssh/id_rsa]"
    echo "  -sp, --source-path      Source path [default: ~/mariadb_dump]"
    echo "  -dh, --dest-host        Destination host (required for remote migration)"
    echo "  -du, --dest-user        Destination user (required for remote migration)"
    echo "  -dk, --dest-key         Destination SSH private key [default: ~/.ssh/id_rsa]"
    echo "  -dp, --dest-path        Destination path [default: /home/ubuntu/mariadb_dump]"
    echo "  -db, --database         Database name [default: poc_db]"
    echo "  -pw, --password         Database root password [default: your_root_password]"
    echo "  -q, --quiet             Disable verbose mode"
    echo "  -nc, --no-compress      Disable compression during transfer"
    echo "  -h, --help              Show this help message"
    exit 1
}

# Parse arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        -sh|--source-host) SOURCE_HOST="$2"; shift ;;
        -su|--source-user) SOURCE_USER="$2"; shift ;;
        -sk|--source-key) SOURCE_KEY="$2"; shift ;;
        -sp|--source-path) SOURCE_PATH="$2"; shift ;;
        -dh|--dest-host) DEST_HOST="$2"; shift ;;
        -du|--dest-user) DEST_USER="$2"; shift ;;
        -dk|--dest-key) DEST_KEY="$2"; shift ;;
        -dp|--dest-path) DEST_PATH="$2"; shift ;;
        -db|--database) DB_NAME="$2"; shift ;;
        -pw|--password) DB_PASSWORD="$2"; shift ;;
        -q|--quiet) VERBOSE="" ;;
        -nc|--no-compress) COMPRESS="" ;;
        -h|--help) usage ;;
        *) echo -e "${RED}Unknown parameter: $1${NC}"; usage ;;
    esac
    shift
done

# Check if destination host was provided
if [ -z "$DEST_HOST" ]; then
    echo -e "${YELLOW}No destination host specified. Running local migration test.${NC}"
else
    # Check if user was provided
    if [ -z "$DEST_USER" ]; then
        echo -e "${RED}Error: Destination user is required for remote migration${NC}"
        usage
    fi
    echo -e "${GREEN}Running remote migration to ${DEST_USER}@${DEST_HOST}${NC}"
fi

# Construct command
CMD_ARGS=""
[ -n "$SOURCE_HOST" ] && CMD_ARGS="$CMD_ARGS --src-host=$SOURCE_HOST"
[ -n "$SOURCE_USER" ] && CMD_ARGS="$CMD_ARGS --src-user=$SOURCE_USER"
[ -n "$SOURCE_KEY" ] && CMD_ARGS="$CMD_ARGS --src-key=$SOURCE_KEY"
[ -n "$SOURCE_PATH" ] && CMD_ARGS="$CMD_ARGS --src-path=$SOURCE_PATH"
[ -n "$DEST_HOST" ] && CMD_ARGS="$CMD_ARGS --dst-host=$DEST_HOST"
[ -n "$DEST_USER" ] && CMD_ARGS="$CMD_ARGS --dst-user=$DEST_USER"
[ -n "$DEST_KEY" ] && CMD_ARGS="$CMD_ARGS --dst-key=$DEST_KEY"
[ -n "$DEST_PATH" ] && CMD_ARGS="$CMD_ARGS --dst-path=$DEST_PATH"
[ -n "$DB_NAME" ] && CMD_ARGS="$CMD_ARGS --db=$DB_NAME"
[ -n "$DB_PASSWORD" ] && CMD_ARGS="$CMD_ARGS --password=$DB_PASSWORD"
[ -n "$VERBOSE" ] && CMD_ARGS="$CMD_ARGS $VERBOSE"
[ -n "$COMPRESS" ] && CMD_ARGS="$CMD_ARGS $COMPRESS"

# Run Go migration
echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}Running MariaDB Migration with transx${NC}"
echo -e "${BLUE}=========================================${NC}"

echo -e "${YELLOW}Executing: go run main.go $CMD_ARGS${NC}"
go run main.go $CMD_ARGS

# Check results
if [ $? -eq 0 ]; then
    echo -e "${GREEN}Migration completed successfully!${NC}"
    
    # Verify target database locally
    if [ -z "$DEST_HOST" ]; then
        echo -e "${YELLOW}Verifying migrated data...${NC}"
        docker exec -i mariadb_target mariadb -u root -p"$DB_PASSWORD" "$DB_NAME" -e "SELECT * FROM products;"
        echo -e "${GREEN}Data verification completed!${NC}"
    else
        echo -e "${YELLOW}To verify data on the remote server, run:${NC}"
        echo -e "${GREEN}ssh ${DEST_USER}@${DEST_HOST} 'docker exec -i mariadb_target mariadb -u root -p\"$DB_PASSWORD\" \"$DB_NAME\" -e \"SELECT * FROM products;\"'${NC}"
    fi
else
    echo -e "${RED}Migration failed!${NC}"
fi
