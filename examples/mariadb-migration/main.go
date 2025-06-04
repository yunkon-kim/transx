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

	"github.com/yunkon-kim/transx/pkg/transx"
)

func main() {
	var configFile string
	var backupOnly bool
	var transferOnly bool
	var restoreOnly bool
	var verbose bool

	// Setting up command-line flags
	flag.StringVar(&configFile, "config", "migration-config.json", "Migration configuration JSON file path")
	flag.BoolVar(&backupOnly, "backup", false, "Run only the backup step")
	flag.BoolVar(&transferOnly, "transfer", false, "Run only the data transfer step")
	flag.BoolVar(&restoreOnly, "restore", false, "Run only the restore step")
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
	var migrationTask transx.DataMigrationModel
	err = json.Unmarshal(jsonData, &migrationTask)
	if err != nil {
		log.Fatalf("Failed to parse config JSON: %v", err)
	}

	// Validate migration configuration file
	err = transx.Validate(migrationTask)
	if err != nil {
		log.Fatalf("Invalid migration configuration: %v", err)
	}

	// Detect and validate migration scenario
	isRelayMode := migrationTask.IsRelayMode()

	if strings.Contains(configFile, "server-to-vm") {
		fmt.Println("Server to VM migration mode detected.")

		// Check if source is a server and has the correct backup command
		if !strings.Contains(migrationTask.Source.BackupCmd, "mysqldump") && !strings.Contains(migrationTask.Source.BackupCmd, "mariadb-dump") {
			log.Printf("Warning: Server backup command might be incorrect. Expected mysqldump or mariadb-dump command.")
		}

		// Check if target is a VM and has Docker restore command
		if !strings.Contains(migrationTask.Destination.RestoreCmd, "docker exec") {
			log.Printf("Warning: VM restore command might be incorrect. Expected 'docker exec' for container operation.")
		}
	} else if strings.Contains(configFile, "relay-migration") || isRelayMode {
		// Validate relay migration mode (both source and destination are remote)
		if isRelayMode {
			fmt.Println("Relay migration mode detected: Source and destination are both remote.")
			fmt.Println("This machine will act as an intermediary relay for the data transfer.")

			// Display source and destination information
			fmt.Printf("Source: %s@%s:%s\n", migrationTask.Source.Username, migrationTask.Source.HostIP, migrationTask.Source.Path)
			fmt.Printf("Destination: %s@%s:%s\n", migrationTask.Destination.Username, migrationTask.Destination.HostIP, migrationTask.Destination.Path)
		} else {
			log.Printf("Warning: Relay migration configuration should have both remote source and destination.")
		}
	} else if migrationTask.Source.HostIP == "" && migrationTask.Destination.HostIP == "" {
		fmt.Println("Local migration mode detected.")
	} else if migrationTask.Source.HostIP == "" && migrationTask.Destination.HostIP != "" {
		fmt.Println("Local to remote migration mode detected.")
	} else if migrationTask.Source.HostIP != "" && migrationTask.Destination.HostIP == "" {
		fmt.Println("Remote to local migration mode detected.")
	}

	// Expand tilde (~) in SSH private key paths
	if strings.HasPrefix(migrationTask.Source.SSHPrivateKey, "~/") {
		homeDir, _ := os.UserHomeDir()
		migrationTask.Source.SSHPrivateKey = filepath.Join(homeDir, migrationTask.Source.SSHPrivateKey[2:])
	}
	if strings.HasPrefix(migrationTask.Destination.SSHPrivateKey, "~/") {
		homeDir, _ := os.UserHomeDir()
		migrationTask.Destination.SSHPrivateKey = filepath.Join(homeDir, migrationTask.Destination.SSHPrivateKey[2:])
	}

	// Execute migration steps
	if !transferOnly && !restoreOnly {
		// Step 1: Backup database in source environment
		fmt.Println("Step 1: Backing up database...")
		stepStartTime := time.Now()

		// Display backup command (in verbose mode)
		if verbose {
			fmt.Printf("Backup command: %s\n", migrationTask.Source.BackupCmd)
		}

		if err := transx.Backup(migrationTask.Source, migrationTask.RsyncOptions); err != nil {
			log.Fatalf("Database backup failed: %v", err)
		}

		if verbose {
			fmt.Printf("Database backup completed successfully! (Time: %s)\n", time.Since(stepStartTime))
		} else {
			fmt.Println("Database backup completed successfully!")
		}

		if backupOnly {
			if verbose {
				fmt.Printf("\nTotal execution time: %s\n", time.Since(startTime))
			}
			return
		}
	}

	if !backupOnly && !restoreOnly {
		// Step 2: Transfer backup file to destination environment
		fmt.Println("Step 2: Transferring backup file to destination...")
		stepStartTime := time.Now()

		// Display additional information for relay migration
		if migrationTask.IsRelayMode() {
			if verbose {
				fmt.Println("Relay transfer: Data will flow through this machine as an intermediary")
				fmt.Printf("Source path: %s\n", migrationTask.Source.Path)
				fmt.Printf("Destination path: %s\n", migrationTask.Destination.Path)
			}
		}

		if err := transx.Execute(migrationTask); err != nil {
			log.Fatalf("Data transfer failed: %v", err)
		}

		if verbose {
			fmt.Printf("Data transfer completed successfully! (Time: %s)\n", time.Since(stepStartTime))
		} else {
			fmt.Println("Data transfer completed successfully!")
		}

		if transferOnly {
			if verbose {
				fmt.Printf("\nTotal execution time: %s\n", time.Since(startTime))
			}
			return
		}
	}

	if !backupOnly && !transferOnly {
		// Step 3: Restore database in destination environment
		fmt.Println("Step 3: Restoring database on the destination...")
		stepStartTime := time.Now()

		// Display restore command (in verbose mode)
		if verbose {
			fmt.Printf("Restore command: %s\n", migrationTask.Destination.RestoreCmd)
		}

		if err := transx.Restore(migrationTask.Destination, migrationTask.RsyncOptions); err != nil {
			log.Fatalf("Database restore failed: %v", err)
		}

		if verbose {
			fmt.Printf("Database restore completed successfully! (Time: %s)\n", time.Since(stepStartTime))
		} else {
			fmt.Println("Database restore completed successfully!")
		}

		if restoreOnly && verbose {
			fmt.Printf("\nTotal execution time: %s\n", time.Since(startTime))
		}
	}

	// Display summary information when full migration is completed
	if !backupOnly && !transferOnly && !restoreOnly {
		totalTime := time.Since(startTime)
		fmt.Println("\n=== Migration Summary ===")
		fmt.Printf("Source: %s@%s:%s\n", migrationTask.Source.Username, migrationTask.Source.HostIP, migrationTask.Source.Path)
		fmt.Printf("Destination: %s@%s:%s\n", migrationTask.Destination.Username, migrationTask.Destination.HostIP, migrationTask.Destination.Path)
		fmt.Printf("Total migration time: %s\n", totalTime)
		fmt.Println("MariaDB migration completed successfully!")
	}
}
