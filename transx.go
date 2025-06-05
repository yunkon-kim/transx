package transx

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// DataMigrationModel defines a single rsync data migration task.
type DataMigrationModel struct {
	Source       EndpointDetails
	Destination  EndpointDetails
	RsyncOptions RsyncOption
}

// EndpointDetails defines the source/destination endpoint for rsync or the target for backup/restore operations.
type EndpointDetails struct {
	// For remote endpoints
	Username string // Username for SSH connection (e.g., "user")
	HostIP   string // Hostname or IP address for SSH connection (e.g., "server.example.com" or "192.168.1.100")
	SSHPort  int    // SSH port (0 or unspecified uses default 22)

	// DataPath for both local and remote operations
	DataPath string // Data path (e.g., "/home/user/data" for remote or "/var/backups/data" for local)

	SSHPrivateKeyPath string // Path to the SSH private key file (used for remote connections with key authentication)
	BackupCmd         string // Backup command string to be executed on this endpoint
	RestoreCmd        string // Restore command string to be executed on this endpoint
}

// RsyncOption defines options to be applied when executing rsync commands and SSH connection options.
type RsyncOption struct {
	Compress  bool     // -z, --compress: Compress file data during the transfer
	Archive   bool     // -a, --archive: Archive mode; equals -rlptgoD (no -H,-A,-X)
	Verbose   bool     // -v, --verbose: Increase verbosity
	Delete    bool     // --delete: Delete extraneous files from dest dirs
	Progress  bool     // --progress: Show progress during transfer
	DryRun    bool     // -n, --dry-run: Perform a trial run with no changes made
	RsyncPath string   // Path to the rsync executable (if empty, uses system PATH)
	Exclude   []string // --exclude=PATTERN: List of patterns to exclude
	Include   []string // --include=PATTERN: List of patterns to include
	// ExtraArgs []string // List of other rsync arguments to pass directly

	// InsecureSkipHostKeyVerification, if true, relaxes host key checking for SSH connections.
	// Adds "-o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=/dev/null" options.
	// Warning: This can be a security risk and should only be used in trusted environments.
	InsecureSkipHostKeyVerification bool
}

// isRemote determines if the EndpointDetails represent a remote endpoint.
// A remote endpoint must have a HostIP. Username and RemotePath are also typical.
func (e *EndpointDetails) isRemote() bool {
	return strings.TrimSpace(e.HostIP) != ""
}

// getRsyncPath constructs the path string suitable for rsync (e.g., "user@host:/path" or "/local/path").
func (e *EndpointDetails) getRsyncPath() string {
	if e.isRemote() {
		if strings.TrimSpace(e.Username) != "" {
			return fmt.Sprintf("%s@%s:%s", e.Username, e.HostIP, e.DataPath)
		}
		return fmt.Sprintf("%s:%s", e.HostIP, e.DataPath) // Username might be optional if SSH config handles it
	}
	return e.DataPath
}

// IsRelayMode determines if both source and destination endpoints are remote.
// This is used to identify relay migration scenarios where data needs to flow through the local machine
// as an intermediary between two remote endpoints.
func (task *DataMigrationModel) IsRelayMode() bool {
	return task.Source.isRemote() && task.Destination.isRemote()
}

// Validate checks if the fields of DataMigrationModel satisfy basic requirements for an rsync task.
func Validate(task DataMigrationModel) error {
	sourceRsyncPath := task.Source.getRsyncPath()
	destRsyncPath := task.Destination.getRsyncPath()

	if strings.TrimSpace(sourceRsyncPath) == "" || strings.TrimSpace(task.Source.DataPath) == "" {
		return fmt.Errorf("source path must be provided for rsync task")
	}
	if strings.TrimSpace(destRsyncPath) == "" || strings.TrimSpace(task.Destination.DataPath) == "" {
		return fmt.Errorf("destination path must be provided for rsync task")
	}

	// Validate SSH port for source if it's a remote endpoint
	if task.Source.isRemote() {
		if task.Source.SSHPort != 0 && (task.Source.SSHPort < 1 || task.Source.SSHPort > 65535) {
			return fmt.Errorf("source SSH port %d is out of valid range (1-65535)", task.Source.SSHPort)
		}
		if strings.TrimSpace(task.Source.HostIP) == "" {
			return fmt.Errorf("source HostIP must be provided for remote rsync task")
		}
	}
	// Validate SSH port for destination if it's a remote endpoint
	if task.Destination.isRemote() {
		if task.Destination.SSHPort != 0 && (task.Destination.SSHPort < 1 || task.Destination.SSHPort > 65535) {
			return fmt.Errorf("destination SSH port %d is out of valid range (1-65535)", task.Destination.SSHPort)
		}
		if strings.TrimSpace(task.Destination.HostIP) == "" {
			return fmt.Errorf("destination HostIP must be provided for remote rsync task")
		}
	}
	// The existence of SSHPrivateKey path etc. will be handled by the ssh command at runtime.
	// The Validate function primarily checks for structural issues.
	return nil
}

// Transfer runs the rsync command to transfer data as defined by the given DataMigrationModel.
func Transfer(task DataMigrationModel) error {
	if err := Validate(task); err != nil {
		return fmt.Errorf("rsync task validation failed: %w", err)
	}

	// Check if we're operating in relay mode (both source and destination are remote)
	isRelayMode := task.IsRelayMode()

	rsyncCmdPath := task.RsyncOptions.RsyncPath
	if rsyncCmdPath == "" {
		rsyncCmdPath = "rsync" // Use system default rsync
	}

	var args []string
	// Configure basic rsync options
	if task.RsyncOptions.Archive {
		args = append(args, "-a")
	}
	if task.RsyncOptions.Compress {
		args = append(args, "-z")
	}
	if task.RsyncOptions.Verbose {
		args = append(args, "-v")
	}
	if task.RsyncOptions.Delete {
		args = append(args, "--delete")
	}
	if task.RsyncOptions.Progress {
		args = append(args, "--progress")
	}
	if task.RsyncOptions.DryRun {
		args = append(args, "-n") // or "--dry-run"
	}

	// Configure Exclude and Include options
	for _, ex := range task.RsyncOptions.Exclude {
		if strings.TrimSpace(ex) != "" {
			args = append(args, "--exclude="+ex)
		}
	}
	for _, inc := range task.RsyncOptions.Include {
		if strings.TrimSpace(inc) != "" {
			args = append(args, "--include="+inc)
		}
	}

	// // Configure extra rsync arguments
	// if len(task.RsyncOptions.ExtraArgs) > 0 {
	// 	args = append(args, task.RsyncOptions.ExtraArgs...)
	// }

	// Configure SSH options (-e)
	// rsync uses only one remote shell command.
	// If the source is remote, SSH settings for source connection are used.
	// If the source is local and destination is remote, SSH settings for destination connection are used.
	var sshOptString string
	var activeRemoteEndpointForRsync EndpointDetails
	operationInvolvesRemoteRsync := false

	if task.Source.isRemote() {
		activeRemoteEndpointForRsync = task.Source
		operationInvolvesRemoteRsync = true
	} else if task.Destination.isRemote() {
		activeRemoteEndpointForRsync = task.Destination
		operationInvolvesRemoteRsync = true
	}

	if operationInvolvesRemoteRsync && activeRemoteEndpointForRsync.SSHPrivateKeyPath != "" {
		var sshCmdParts []string
		sshCmdParts = append(sshCmdParts, "ssh")
		if strings.TrimSpace(activeRemoteEndpointForRsync.SSHPrivateKeyPath) != "" {
			sshCmdParts = append(sshCmdParts, "-i", activeRemoteEndpointForRsync.SSHPrivateKeyPath)
		}
		if activeRemoteEndpointForRsync.SSHPort != 0 { // If 0, use default port (22)
			sshCmdParts = append(sshCmdParts, "-p", strconv.Itoa(activeRemoteEndpointForRsync.SSHPort))
		}
		if task.RsyncOptions.InsecureSkipHostKeyVerification {
			sshCmdParts = append(sshCmdParts, "-o", "StrictHostKeyChecking=accept-new")
			sshCmdParts = append(sshCmdParts, "-o", "UserKnownHostsFile=/dev/null")
		}
		// Username and HostIP are part of the rsync path, not the -e ssh command for rsync
		sshOptString = strings.Join(sshCmdParts, " ")
	}

	if sshOptString != "" {
		args = append(args, "-e", sshOptString)
	}

	// Add source and destination paths
	sourceRsyncPath := task.Source.getRsyncPath()
	destinationRsyncPath := task.Destination.getRsyncPath()

	// Check if we need to use relay mode (both source and destination are remote)
	if isRelayMode {
		// For relay mode, we need to:
		// 1. Create a temporary directory on the local machine
		// 2. First download from source to the temp dir
		// 3. Then upload from the temp dir to the destination

		tempDir, err := os.MkdirTemp("", "transx-relay-*")
		if err != nil {
			return fmt.Errorf("failed to create temporary directory for relay transfer: %w", err)
		}
		defer os.RemoveAll(tempDir) // Clean up temp dir when done

		// Step 1: Download from source to temp dir
		downloadArgs := make([]string, len(args))
		copy(downloadArgs, args)
		downloadArgs = append(downloadArgs, sourceRsyncPath, tempDir+"/")

		fmt.Printf("Relay transfer mode: Downloading from source to local temp dir...\n")
		downloadCmd := exec.Command(rsyncCmdPath, downloadArgs...)
		downloadOutput, err := downloadCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("relay download failed from '%s' to temp dir\nCommand: %s %s\nError: %w\nOutput:\n%s",
				sourceRsyncPath, rsyncCmdPath, strings.Join(downloadArgs, " "), err, string(downloadOutput))
		}

		// Step 2: Upload from temp dir to destination
		uploadArgs := make([]string, len(args))
		copy(uploadArgs, args)
		uploadArgs = append(uploadArgs, tempDir+"/", destinationRsyncPath)

		fmt.Printf("Relay transfer mode: Uploading from local temp dir to destination...\n")
		uploadCmd := exec.Command(rsyncCmdPath, uploadArgs...)
		uploadOutput, err := uploadCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("relay upload failed from temp dir to '%s'\nCommand: %s %s\nError: %w\nOutput:\n%s",
				destinationRsyncPath, rsyncCmdPath, strings.Join(uploadArgs, " "), err, string(uploadOutput))
		}

		fmt.Printf("Relay transfer completed successfully!\n")
		return nil
	}

	// Standard direct transfer (not relay mode)
	args = append(args, sourceRsyncPath, destinationRsyncPath)

	// Create and execute the rsync command
	cmd := exec.Command(rsyncCmdPath, args...)
	// fmt.Println("Executing command:", cmd.String()) // For debugging

	output, err := cmd.CombinedOutput() // Get combined stdout and stderr
	if err != nil {
		// Improve error message by including the command and output for easier debugging
		return fmt.Errorf("rsync execution failed for task from '%s' to '%s'\nCommand: %s %s\nError: %w\nOutput:\n%s",
			sourceRsyncPath, destinationRsyncPath, rsyncCmdPath, strings.Join(args, " "), err, string(output))
	}
	return nil
}

// executeCommand executes the given command locally or remotely (via SSH).
// If endpoint is remote (has HostIP) and SSHPrivateKey is provided, it executes remotely.
// Otherwise, it executes locally.
// sshConfig provides SSH options (InsecureSkipHostKeyVerification) for remote execution.
func executeCommand(commandToExecute string, endpoint EndpointDetails, sshConfig RsyncOption) ([]byte, error) {
	if strings.TrimSpace(commandToExecute) == "" {
		return nil, fmt.Errorf("command to execute cannot be empty")
	}

	if endpoint.isRemote() { // Check if it's a remote endpoint
		if strings.TrimSpace(endpoint.HostIP) == "" {
			return nil, fmt.Errorf("HostIP must be provided for remote command execution on endpoint")
		}

		userHost := endpoint.HostIP
		if strings.TrimSpace(endpoint.Username) != "" {
			userHost = fmt.Sprintf("%s@%s", endpoint.Username, endpoint.HostIP)
		}

		var sshCmdParts []string
		sshCmdParts = append(sshCmdParts, "ssh") // SSH command
		if strings.TrimSpace(endpoint.SSHPrivateKeyPath) != "" {
			sshCmdParts = append(sshCmdParts, "-i", endpoint.SSHPrivateKeyPath) // Private key
		}
		if endpoint.SSHPort != 0 { // SSH port (if not 0)
			sshCmdParts = append(sshCmdParts, "-p", strconv.Itoa(endpoint.SSHPort))
		}
		if sshConfig.InsecureSkipHostKeyVerification { // Skip host key verification option
			sshCmdParts = append(sshCmdParts, "-o", "StrictHostKeyChecking=accept-new")
			sshCmdParts = append(sshCmdParts, "-o", "UserKnownHostsFile=/dev/null")
		}

		// Add timeout for SSH connection
		sshCmdParts = append(sshCmdParts, "-o", "ConnectTimeout=30")

		// For remote commands with sudo, we need the -t option to allocate a pseudo-tty
		if strings.Contains(commandToExecute, "sudo") {
			sshCmdParts = append(sshCmdParts, "-t")
		}

		sshCmdParts = append(sshCmdParts, userHost, commandToExecute) // user@host "command_to_execute"

		cmd := exec.Command(sshCmdParts[0], sshCmdParts[1:]...)
		fmt.Printf("Executing remote command on %s...\n", userHost) // For user feedback
		return cmd.CombinedOutput()
	} else {
		// Local execution
		// Use "sh -c" to handle complex shell commands
		cmd := exec.Command("sh", "-c", commandToExecute)
		fmt.Println("Executing local command...")
		return cmd.CombinedOutput()
	}
}

// Backup executes the BackupCmd defined in the source EndpointDetails of the DataMigrationModel.
func Backup(dmm DataMigrationModel) error {
	// Use source endpoint for backup operations
	source := dmm.Source
	if strings.TrimSpace(source.BackupCmd) == "" {
		return fmt.Errorf("backup command is not defined for source")
	}

	// Determine the source path for display
	// This allows us to handle both local and remote backups properly.
	// If it's a remote source, format it as "username@host:path" and the command will be executed remotely.
	// If it's a local source, just use the DataPath directly.
	sourcePath := source.DataPath
	if source.isRemote() {
		sourcePath = fmt.Sprintf("%s@%s:%s", source.Username, source.HostIP, source.DataPath)
		fmt.Printf("Executing backup command on remote server %s...\n", source.HostIP)
	} else {
		fmt.Println("Executing backup command locally...")
	}

	fmt.Printf("Backup command: %s\n", source.BackupCmd)
	output, err := executeCommand(source.BackupCmd, source, dmm.RsyncOptions)
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

	// Determine the destination path for display
	// This allows us to handle both local and remote restores properly.
	// If it's a remote destination, format it as "username@host:path" and the command will be executed remotely.
	// If it's a local destination, just use the DataPath directly.
	destinationDataPath := destination.DataPath
	if destination.isRemote() {
		destinationDataPath = fmt.Sprintf("%s@%s:%s", destination.Username, destination.HostIP, destination.DataPath)
		fmt.Printf("Executing restore command on remote server %s...\n", destination.HostIP)
	} else {
		fmt.Println("Executing restore command locally...")
	}

	fmt.Printf("Restore command: %s\n", destination.RestoreCmd)
	output, err := executeCommand(destination.RestoreCmd, destination, dmm.RsyncOptions)
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
