#!/bin/bash

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}MariaDB Migration Test Runner${NC}"
echo -e "${BLUE}=========================================${NC}"

# Usage function
function show_usage {
  echo -e "${YELLOW}Usage:${NC}"
  echo -e "  $0 [config] [mode]"
  echo -e ""
  echo -e "${YELLOW}Available configurations:${NC}"
  echo -e "  local     - Local migration test (migration-config.json)"
  echo -e "  remote    - Remote migration test (remote-migration-config.json)"
  echo -e "  vm        - Server-VM migration test (server-to-vm-config.json)"
  echo -e "  relay     - Relay migration test (relay-migration-config.json)"
  echo -e "  test      - Test relay migration (test-relay-config.json)"
  echo -e ""
  echo -e "${YELLOW}Available modes:${NC}"
  echo -e "  full      - Run complete migration (default)"
  echo -e "  backup    - Run backup step only"
  echo -e "  transfer  - Run transfer step only"
  echo -e "  restore   - Run restore step only"
  echo -e ""
  echo -e "${YELLOW}Examples:${NC}"
  echo -e "  $0 local full    - Run local migration test"
  echo -e "  $0 relay backup  - Run only backup step for relay migration"
  echo -e "  $0 test          - Run test relay migration"
  echo -e ""
}

# Check arguments
if [ $# -lt 1 ]; then
  show_usage
  exit 1
fi

CONFIG=$1
MODE=${2:-full}

# Configuration file mapping
case $CONFIG in
  local)
    CONFIG_FILE="migration-config.json"
    ;;
  remote)
    CONFIG_FILE="remote-migration-config.json"
    ;;
  vm)
    CONFIG_FILE="server-to-vm-config.json"
    ;;
  relay)
    CONFIG_FILE="relay-migration-config.json"
    ;;
  test)
    CONFIG_FILE="test-relay-config.json"
    ;;
  *)
    echo -e "${RED}Error: Invalid configuration '$CONFIG'${NC}"
    show_usage
    exit 1
    ;;
esac

# Construct execution command
CMD="go run main.go --config=$CONFIG_FILE --verbose"

# Add mode flags
case $MODE in
  full)
    # Default mode, no additional flags needed
    ;;
  backup)
    CMD="$CMD --backup"
    ;;
  transfer)
    CMD="$CMD --transfer"
    ;;
  restore)
    CMD="$CMD --restore"
    ;;
  *)
    echo -e "${RED}Error: Invalid mode '$MODE'${NC}"
    show_usage
    exit 1
    ;;
esac

echo -e "${GREEN}Executing: $CMD${NC}"
echo -e "${BLUE}=========================================${NC}"

# Execute command
eval $CMD

exit_code=$?
if [ $exit_code -eq 0 ]; then
  echo -e "${GREEN}Migration test completed successfully!${NC}"
else
  echo -e "${RED}Migration test failed with exit code $exit_code${NC}"
fi
