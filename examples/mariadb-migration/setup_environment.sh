#!/bin/bash

# Environment validation script for MariaDB Server-VM migration
# This script assumes that rsync and MariaDB environment are already configured,
# and verifies that the necessary tools and settings are properly set up.

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}MariaDB Migration Environment Check Script${NC}"
echo -e "${BLUE}=========================================${NC}"

# Check for required tools
echo -e "${YELLOW}Checking for required tools...${NC}"

# Check for rsync
if ! command -v rsync &> /dev/null; then
    echo -e "${RED}rsync is not installed. Please install rsync:${NC}"
    echo -e "    sudo apt-get update && sudo apt-get install -y rsync"
    echo -e "${RED}rsync is required for the migration process${NC}"
    echo -e "${YELLOW}Continuing environment check...${NC}"
else
    echo -e "${GREEN}✓ rsync is installed${NC}"
fi

# Check for ssh
if ! command -v ssh &> /dev/null; then
    echo -e "${RED}ssh client is not installed. Please install OpenSSH client:${NC}"
    echo -e "    sudo apt-get update && sudo apt-get install -y openssh-client"
    echo -e "${RED}SSH client is required for remote operations${NC}"
    echo -e "${YELLOW}Continuing environment check...${NC}"
else
    echo -e "${GREEN}✓ ssh client is installed${NC}"
fi

# Check for Docker (if used on VM target)
if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}Warning: Docker is not installed on this machine.${NC}"
    echo -e "${YELLOW}If this is the VM target, Docker is required for running MariaDB container:${NC}"
    echo -e "    curl -sSL get.docker.com | sh"
    echo -e "    sudo usermod -aG docker \${USER}"
    echo -e "${YELLOW}If this is the source server (with native MariaDB), you can ignore this warning.${NC}"
else
    echo -e "${GREEN}✓ Docker is installed${NC}"
    
    # Check if MariaDB container is running (when used on VM target)
    if docker ps | grep -q "mariadb_target"; then
        echo -e "${GREEN}✓ MariaDB target container is running${NC}"
    else
        echo -e "${YELLOW}Warning: MariaDB target container is not running${NC}"
        echo -e "${YELLOW}If this is the VM target, make sure the container is running:${NC}"
        echo -e "    docker run -d --name mariadb_target \\"
        echo -e "      -e MARIADB_ROOT_PASSWORD=your_root_password \\"
        echo -e "      -p 3306:3306 \\"
        echo -e "      -v /home/ubuntu/mariadb_backup:/backup \\"
        echo -e "      mariadb:latest"
        echo -e "${YELLOW}If this is the source server, you can ignore this warning.${NC}"
    fi
fi

# Check and create backup directory (if needed)
echo -e "${YELLOW}Checking for backup directory...${NC}"
if [ ! -d ~/mariadb_backup ]; then
    echo -e "${YELLOW}Creating backup directory...${NC}"
    mkdir -p ~/mariadb_backup
    echo -e "${GREEN}✓ Backup directory created: ~/mariadb_backup${NC}"
else
    echo -e "${GREEN}✓ Backup directory exists: ~/mariadb_backup${NC}"
fi

# Check temporary directory (for relay migration)
echo -e "${YELLOW}Checking for temporary directory permissions...${NC}"
if [ -w /tmp ]; then
    echo -e "${GREEN}✓ Temporary directory is writable (required for relay mode)${NC}"
else
    echo -e "${RED}Warning: Temporary directory (/tmp) is not writable${NC}"
    echo -e "${RED}Relay migration mode may fail${NC}"
fi

# Check SSH key permissions
echo -e "${YELLOW}Checking SSH key permissions...${NC}"
if [ -f ~/.ssh/id_rsa ]; then
    permissions=$(stat -c "%a" ~/.ssh/id_rsa)
    if [ "$permissions" != "600" ]; then
        echo -e "${YELLOW}Fixing SSH key permissions...${NC}"
        chmod 600 ~/.ssh/id_rsa
    fi
    echo -e "${GREEN}✓ SSH key permissions verified${NC}"
else
    echo -e "${YELLOW}Warning: Default SSH key (~/.ssh/id_rsa) not found.${NC}"
    echo -e "${YELLOW}Make sure to specify the correct SSH key in your config file.${NC}"
fi

echo -e "${BLUE}=========================================${NC}"
echo -e "${GREEN}Environment check completed!${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""
echo -e "${YELLOW}Available migration configuration files:${NC}"
echo -e "${GREEN}direct-mode-config.json${NC} - Direct mode migration (local-to-local, local-to-remote, or remote-to-local)"
echo -e "${GREEN}relay-mode-config.json${NC} - Relay mode migration (both source and destination are remote)"
echo -e "${GREEN}test-relay-config.json${NC} - Test relay migration configuration"
echo ""
echo -e "${YELLOW}Check and modify configuration files:${NC}"
echo -e "${GREEN}cat direct-mode-config.json${NC} or ${GREEN}cat relay-mode-config.json${NC}"
echo -e "${YELLOW}Modify source/destination server information, SSH key paths, backup/restore commands as needed.${NC}"
echo ""
echo -e "${YELLOW}How to run migration:${NC}"
echo -e "${GREEN}go run main.go --config=direct-mode-config.json${NC} (Direct mode migration)"
echo -e "${GREEN}go run main.go --config=relay-mode-config.json${NC} (Relay mode migration)"
echo -e "${YELLOW}The above commands will execute the complete migration process using the configuration file.${NC}"
echo ""
echo -e "${YELLOW}Step-by-step migration execution:${NC}"
echo -e "${GREEN}go run main.go --config=[CONFIG_FILE] --backup${NC} (Run backup only from source)"
echo -e "${GREEN}go run main.go --config=[CONFIG_FILE] --transfer${NC} (Run transfer only from source to destination)"
echo -e "${GREEN}go run main.go --config=[CONFIG_FILE] --restore${NC} (Run restore only on destination)"
echo -e "${GREEN}go run main.go --config=[CONFIG_FILE] --verbose${NC} (Show detailed logs and execution time)"
echo ""
echo -e "${YELLOW}Note: Before using in a production environment, verify the passwords and paths in the configuration file.${NC}"
echo ""
