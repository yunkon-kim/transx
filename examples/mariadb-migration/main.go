// filepath: /home/ubuntu/dev/yunkon-kim/transx/examples/mariadb-migration/main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yunkon-kim/transx"
)

func main() {
	var configFile string
	var verbose bool

	// Setting up command-line flags
	flag.StringVar(&configFile, "config", "direct-mode-config.json", "Migration configuration JSON file path")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.Parse()

	// Record start time (for performance measurement)
	startTime := time.Now()

	// Check configuration file path
	if !filepath.IsAbs(configFile) {
		// Convert relative path to absolute path
		workingDir, err := os.Getwd()
		if err == nil {
			configFile = filepath.Join(workingDir, configFile)
		}
	}

	// Read JSON file
	jsonData, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to read config file %s: %v", configFile, err)
	}

	// Parse JSON data
	var dmm transx.DataMigrationModel
	err = json.Unmarshal(jsonData, &dmm)
	if err != nil {
		log.Fatalf("Failed to parse config JSON: %v", err)
	}

	// Validate migration configuration file
	err = transx.Validate(dmm)
	if err != nil {
		log.Fatalf("Invalid migration configuration: %v", err)
	}

	// Detect and validate migration scenario
	isRelayMode := dmm.IsRelayMode()

	if isRelayMode {
		fmt.Println("Relay mode detected: Source and destination are both remote.")
		fmt.Println("This machine will act as an intermediary relay for the data transfer.")
		fmt.Printf("Source: %s@%s:%s\n", dmm.Source.Username, dmm.Source.HostIP, dmm.Source.DataPath)
		fmt.Printf("Destination: %s@%s:%s\n", dmm.Destination.Username, dmm.Destination.HostIP, dmm.Destination.DataPath)
	} else {
		fmt.Println("Direct mode detected.")

		// Check if it's entirely local or involves remote endpoints
		if dmm.Source.HostIP == "" && dmm.Destination.HostIP == "" {
			fmt.Println("Local-to-local migration (both source and destination are on this machine).")
		} else if dmm.Source.HostIP == "" && dmm.Destination.HostIP != "" {
			fmt.Println("Local-to-remote migration (source is on this machine).")
		} else if dmm.Source.HostIP != "" && dmm.Destination.HostIP == "" {
			fmt.Println("Remote-to-local migration (destination is on this machine).")
		}
	}

	// Expand tilde (~) in SSH private key paths
	if strings.HasPrefix(dmm.Source.SSHPrivateKeyPath, "~/") {
		homeDir, _ := os.UserHomeDir()
		dmm.Source.SSHPrivateKeyPath = filepath.Join(homeDir, dmm.Source.SSHPrivateKeyPath[2:])
	}
	if strings.HasPrefix(dmm.Destination.SSHPrivateKeyPath, "~/") {
		homeDir, _ := os.UserHomeDir()
		dmm.Destination.SSHPrivateKeyPath = filepath.Join(homeDir, dmm.Destination.SSHPrivateKeyPath[2:])
	}

	// Display commands (in verbose mode)
	if verbose {
		if dmm.Source.BackupCmd != "" {
			fmt.Printf("Backup command: %s\n", dmm.Source.BackupCmd)
		}
		if dmm.Destination.RestoreCmd != "" {
			fmt.Printf("Restore command: %s\n", dmm.Destination.RestoreCmd)
		}

		// Display additional information for relay migration
		if dmm.IsRelayMode() {
			fmt.Println("Relay transfer: Data will flow through this machine as an intermediary")
			fmt.Printf("Source path: %s\n", dmm.Source.DataPath)
			fmt.Printf("Destination path: %s\n", dmm.Destination.DataPath)
		}
	}

	// Execute the complete data migration workflow
	if err := transx.MigrateData(dmm); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Display summary information
	totalTime := time.Since(startTime)
	fmt.Println("\n=== Migration Summary ===")
	fmt.Printf("Source: %s@%s:%s\n", dmm.Source.Username, dmm.Source.HostIP, dmm.Source.DataPath)
	fmt.Printf("Destination: %s@%s:%s\n", dmm.Destination.Username, dmm.Destination.HostIP, dmm.Destination.DataPath)
	fmt.Printf("Total migration time: %s\n", totalTime)
	fmt.Println("MariaDB migration completed successfully!")
}
