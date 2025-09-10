package transx

import (
	"fmt"
	"strings"
)

// Backup executes the BackupCmd defined in the source EndpointDetails of the DataMigrationModel.
func Backup(dmm DataMigrationModel) error {
	// Use source endpoint for backup operations
	source := dmm.Source
	if strings.TrimSpace(source.BackupCmd) == "" {
		return fmt.Errorf("backup command is not defined for source")
	}

	// Get transfer options for SSH configuration
	transferOptions := dmm.SourceTransferOptions

	// Determine the source path for display
	// This allows us to handle both local and remote backups properly.
	// If it's a remote source, format it as "username@host:path" and the command will be executed remotely.
	// If it's a local source, just use the DataPath directly.
	sourcePath := source.DataPath
	if source.IsRemote() {
		var username string
		if transferOptions != nil && transferOptions.RsyncOptions != nil {
			username = transferOptions.RsyncOptions.Username
		}
		if strings.TrimSpace(username) != "" {
			sourcePath = fmt.Sprintf("%s@%s:%s", username, source.GetEndpoint(), source.DataPath)
		} else {
			sourcePath = fmt.Sprintf("%s:%s", source.GetEndpoint(), source.DataPath)
		}
		fmt.Printf("Executing backup command on remote server %s...\n", source.GetEndpoint())
	} else {
		fmt.Println("Executing backup command locally...")
	}

	fmt.Printf("Backup command: %s\n", source.BackupCmd)
	output, err := executeCommand(source.BackupCmd, source, transferOptions)
	if err != nil {
		return fmt.Errorf("backup command execution failed for source '%s': %w\nOutput:\n%s", sourcePath, err, string(output))
	}

	// Show output summary
	outputStr := string(output)
	if len(outputStr) > 200 {
		// Truncate very long output for display
		fmt.Printf("Backup command output (truncated): %s...\n", outputStr[:200])
	} else if len(outputStr) > 0 {
		fmt.Printf("Backup command output: %s\n", outputStr)
	}
	return nil
}

// Restore executes the RestoreCmd defined in the destination EndpointDetails of the DataMigrationModel.
func Restore(dmm DataMigrationModel) error {
	// Use destination endpoint for restore operations
	destination := dmm.Destination
	if strings.TrimSpace(destination.RestoreCmd) == "" {
		return fmt.Errorf("restore command is not defined for destination")
	}

	// Get transfer options for SSH configuration
	transferOptions := dmm.DestinationTransferOptions

	// Determine the destination path for display
	// This allows us to handle both local and remote restores properly.
	// If it's a remote destination, format it as "username@host:path" and the command will be executed remotely.
	// If it's a local destination, just use the DataPath directly.
	destinationDataPath := destination.DataPath
	if destination.IsRemote() {
		var username string
		if transferOptions != nil && transferOptions.RsyncOptions != nil {
			username = transferOptions.RsyncOptions.Username
		}
		if strings.TrimSpace(username) != "" {
			destinationDataPath = fmt.Sprintf("%s@%s:%s", username, destination.GetEndpoint(), destination.DataPath)
		} else {
			destinationDataPath = fmt.Sprintf("%s:%s", destination.GetEndpoint(), destination.DataPath)
		}
		fmt.Printf("Executing restore command on remote server %s...\n", destination.GetEndpoint())
	} else {
		fmt.Println("Executing restore command locally...")
	}

	fmt.Printf("Restore command: %s\n", destination.RestoreCmd)
	output, err := executeCommand(destination.RestoreCmd, destination, transferOptions)
	if err != nil {
		return fmt.Errorf("restore command execution failed for destination '%s': %w\nOutput:\n%s", destinationDataPath, err, string(output))
	}

	// Show output summary
	outputStr := string(output)
	if len(outputStr) > 200 {
		// Truncate very long output for display
		fmt.Printf("Restore command output (truncated): %s...\n", outputStr[:200])
	} else if len(outputStr) > 0 {
		fmt.Printf("Restore command output: %s\n", outputStr)
	}
	return nil
}

// Transfer runs the transfer command to transfer data as defined by the given DataMigrationModel.
func Transfer(dmm DataMigrationModel) error {
	if err := Validate(dmm); err != nil {
		return fmt.Errorf("data migration model validation failed: %w", err)
	}

	// Check if we're operating in relay mode (both source and destination are remote)
	isRelayMode := dmm.IsRelayMode()

	// Handle different transfer scenarios
	if !isRelayMode {
		fmt.Printf("Direct transfer: (%s) → (%s)\n", dmm.Source.Endpoint, dmm.Destination.Endpoint)
		// Direct mode: at least one endpoint is local
		return performDirectTransfer(dmm)
	} else {
		fmt.Printf("Relay transfer: (%s) → (relay node) → (%s)\n", dmm.Source.Endpoint, dmm.Destination.Endpoint)
		// Relay mode: both endpoints are remote
		return performRelayTransfer(dmm)
	}
}

// MigrateData manages the complete data migration workflow:
// 1. If Source.BackupCmd is available, perform Backup
// 2. Always perform Transfer
// 3. If Destination.RestoreCmd is available, perform Restore
// This provides a simple one-call approach to handle the entire data migration pipeline.
func MigrateData(dmm DataMigrationModel) error {
	// Step 1: Check and perform backup if BackupCmd is defined
	if strings.TrimSpace(dmm.Source.BackupCmd) != "" {
		fmt.Println("Step 1: Backing up data...")
		if err := Backup(dmm); err != nil {
			return fmt.Errorf("backup operation failed: %w", err)
		}
		fmt.Println("Backup completed successfully!")
	}

	// Step 2: Always perform the data transfer (core functionality)
	fmt.Println("Step 2: Transferring data to destination...")
	if err := Transfer(dmm); err != nil {
		return fmt.Errorf("data transfer failed: %w", err)
	}
	fmt.Println("Data transfer completed successfully!")

	// Step 3: Check and perform restore if RestoreCmd is defined
	if strings.TrimSpace(dmm.Destination.RestoreCmd) != "" {
		fmt.Println("Step 3: Restoring data...")
		if err := Restore(dmm); err != nil {
			return fmt.Errorf("restore operation failed: %w", err)
		}
		fmt.Println("Restore completed successfully!")
	}

	return nil
}
