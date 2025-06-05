#!/bin/bash

# Simple migration wrapper script for running migrations with either direct or relay mode

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default mode is direct
MODE="direct"

# Function to display usage
function show_usage {
  echo -e "${BLUE}Simple MariaDB Migration Tool${NC}"
  echo -e "${YELLOW}Usage:${NC}"
  echo -e "  $0 [direct|relay] [flags]"
  echo -e ""
  echo -e "${YELLOW}Modes:${NC}"
  echo -e "  direct    - Direct mode migration (default)"
  echo -e "  relay     - Relay mode migration (source → local → destination)"
  echo -e ""
  echo -e "${YELLOW}Flags:${NC}"
  echo -e "  --backup     Run only the backup step"
  echo -e "  --transfer   Run only the transfer step"
  echo -e "  --restore    Run only the restore step"
  echo -e "  --verbose    Enable verbose logging"
  echo -e ""
  echo -e "${YELLOW}Examples:${NC}"
  echo -e "  $0 direct --verbose      Run direct mode migration with verbose logging"
  echo -e "  $0 relay --backup        Run only the backup step in relay mode"
  echo -e ""
}

# Check if no arguments or help requested
if [ $# -eq 0 ] || [ "$1" == "--help" ] || [ "$1" == "-h" ]; then
  show_usage
  exit 0
fi

# Parse mode argument
if [ "$1" == "direct" ] || [ "$1" == "relay" ]; then
  MODE="$1"
  shift
fi

# Set configuration file based on mode
if [ "$MODE" == "relay" ]; then
  CONFIG="relay-mode-config.json"
else
  CONFIG="direct-mode-config.json"
fi

# Construct command to run
CMD_ARGS="--config=$CONFIG $@"

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}Running MariaDB Migration (${MODE} mode)${NC}"
echo -e "${BLUE}=========================================${NC}"

echo -e "${GREEN}Executing: go run main.go $CMD_ARGS${NC}"
go run main.go $CMD_ARGS

# Check result
if [ $? -eq 0 ]; then
  echo -e "${GREEN}Migration completed successfully!${NC}"
else
  echo -e "${RED}Migration failed!${NC}"
fi
