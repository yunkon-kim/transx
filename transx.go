package transx

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Transfer method constants
const (
	TransferMethodLocal = "local"
	TransferMethodRsync = "rsync"
	TransferMethodHttp  = "http"
)

// HTTP operation constants for object storage (user-facing)
const (
	HttpOperationDownload = "download" // GET operation
	HttpOperationUpload   = "upload"   // PUT operation
)

// Internal HTTP operation constants (not exposed to users)
const (
	httpOperationDeleteInternal = "delete" // DELETE operation (internal use only)
)

// HTTP method constants
const (
	HttpMethodGet    = "GET"
	HttpMethodPut    = "PUT"
	HttpMethodDelete = "DELETE"
)

// DataMigrationModel defines a single data migration task supporting multiple protocols.
type DataMigrationModel struct {
	Source                     EndpointDetails  `json:"source"`                               // Source endpoint configuration
	Destination                EndpointDetails  `json:"destination"`                          // Destination endpoint configuration
	SourceTransferOptions      *TransferOptions `json:"sourceTransferOptions,omitempty"`      // Source-specific transfer options
	DestinationTransferOptions *TransferOptions `json:"destinationTransferOptions,omitempty"` // Destination-specific transfer options
}

// EndpointDetails defines the source/destination endpoint for data transfer and backup/restore operations.
// Simple unified structure supporting SSH-based rsync, S3 Presigned URLs, and local filesystem transfers.
type EndpointDetails struct {
	// Endpoint configuration (auto-detects protocol based on provided fields)
	Endpoint string `json:"endpoint,omitempty"` // SSH host/IP OR S3 presigned URL (e.g., "server.com", "https://bucket.s3.amazonaws.com/upload?...")
	Port     int    `json:"port,omitempty"`     // SSH port (0 uses default 22, ignored for S3)

	// Data location (required)
	DataPath string `json:"dataPath"` // Local path, remote path, or S3 bucket path (e.g., "/data", "s3://bucket/path")

	// Command execution
	BackupCmd  string `json:"backupCmd,omitempty"`  // Backup command string to be executed on this endpoint
	RestoreCmd string `json:"restoreCmd,omitempty"` // Restore command string to be executed on this endpoint
}

// TransferOptions defines options for various data transfer methods.
type TransferOptions struct {
	// Transfer method specification (required)
	Method string `json:"method"` // Transfer method: "local", "rsync", or "http"

	// Rsync-specific options
	RsyncOptions *RsyncOption `json:"rsyncOptions,omitempty"`

	// HTTP-based object storage options
	HttpTransferOptions *HttpTransferOption `json:"httpTransferOptions,omitempty"`
}

// RsyncOption defines rsync-specific transfer options and SSH connection options.
type RsyncOption struct {
	// Transfer behavior options
	Verbose  bool `json:"verbose"`  // Enable verbose logging
	DryRun   bool `json:"dryRun"`   // Perform a trial run with no changes made
	Progress bool `json:"progress"` // Show progress during transfer

	// Rsync-specific options
	Compress  bool     `json:"compress"`            // -z, --compress: Compress file data during the transfer
	Archive   bool     `json:"archive"`             // -a, --archive: Archive mode; equals -rlptgoD (no -H,-A,-X)
	Delete    bool     `json:"delete"`              // --delete: Delete extraneous files from dest dirs
	RsyncPath string   `json:"rsyncPath,omitempty"` // Path to the rsync executable (if empty, uses system PATH)
	Exclude   []string `json:"exclude,omitempty"`   // --exclude=PATTERN: List of patterns to exclude
	Include   []string `json:"include,omitempty"`   // --include=PATTERN: List of patterns to include

	// TransferDirContentsOnly, if true, adds a trailing slash to source paths
	// to transfer only the contents of the directory and not the directory itself.
	TransferDirContentsOnly bool `json:"transferDirContentsOnly"`

	// SSH connection & authentication options (integrated)
	Username          string `json:"username,omitempty"`          // SSH username
	SSHPrivateKeyPath string `json:"sshPrivateKeyPath,omitempty"` // SSH private key path

	// InsecureSkipHostKeyVerification, if true, relaxes host key checking for SSH connections.
	// Adds "-o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=/dev/null" options.
	// Warning: This can be a security risk and should only be used in trusted environments.
	InsecureSkipHostKeyVerification bool `json:"insecureSkipHostKeyVerification"`
	ConnectTimeout                  int  `json:"connectTimeout,omitempty"` // SSH connection timeout in seconds
}

// HttpTransferOption defines HTTP-based object storage transfer options.
type HttpTransferOption struct {
	// Operation specification
	Operation string `json:"operation"` // HTTP operation: "download", "upload", or "delete"
	Method    string `json:"method"`    // HTTP method: "GET", "PUT", or "DELETE"

	// Request configuration
	Timeout         int               `json:"timeout,omitempty"`    // HTTP request timeout in seconds
	MaxRetries      int               `json:"maxRetries,omitempty"` // Maximum number of retry attempts
	ChunkSize       int64             `json:"chunkSize,omitempty"`  // Chunk size for multipart uploads (bytes)
	Headers         map[string]string `json:"headers,omitempty"`    // Additional HTTP headers
	FollowRedirects bool              `json:"followRedirects"`      // Follow HTTP redirects
	VerifySSL       bool              `json:"verifySSL"`            // Verify SSL certificates
	UserAgent       string            `json:"userAgent,omitempty"`  // Custom User-Agent string
}

// IsValidTransferMethod checks if the given transfer method is supported.
func IsValidTransferMethod(method string) bool {
	switch method {
	case TransferMethodLocal, TransferMethodRsync, TransferMethodHttp:
		return true
	default:
		return false
	}
}

// IsValidHttpOperation checks if the given HTTP operation is supported.
func IsValidHttpOperation(operation string) bool {
	switch operation {
	case HttpOperationDownload, HttpOperationUpload:
		return true
	default:
		return false
	}
}

// IsValidHttpMethod checks if the given HTTP method is supported.
func IsValidHttpMethod(method string) bool {
	switch method {
	case HttpMethodGet, HttpMethodPut, HttpMethodDelete:
		return true
	default:
		return false
	}
}

// GetHttpMethodForOperation returns the appropriate HTTP method for an operation.
func GetHttpMethodForOperation(operation string) string {
	switch operation {
	case HttpOperationDownload:
		return HttpMethodGet
	case HttpOperationUpload:
		return HttpMethodPut
	case httpOperationDeleteInternal:
		return HttpMethodDelete
	default:
		return HttpMethodGet // Default fallback
	}
}

// GetHost returns the endpoint (SSH host/IP or S3 URL).
func (e *EndpointDetails) GetHost() string {
	return e.Endpoint
}

// GetPort returns the SSH port.
func (e *EndpointDetails) GetPort() int {
	return e.Port
}

// GetUploadURL returns the upload URL for S3/Object Storage.
func (e *EndpointDetails) GetUploadURL() string {
	if strings.HasPrefix(e.Endpoint, "http://") || strings.HasPrefix(e.Endpoint, "https://") {
		return e.Endpoint
	}
	return ""
}

// GetDownloadURL returns the download URL for S3/Object Storage.
func (e *EndpointDetails) GetDownloadURL() string {
	if strings.HasPrefix(e.Endpoint, "http://") || strings.HasPrefix(e.Endpoint, "https://") {
		return e.Endpoint
	}
	return ""
}

// IsRemote determines if the EndpointDetails represent a remote endpoint.
// Supports multiple transfer methods: rsync (SSH-based) and Object Storage.
func (e *EndpointDetails) IsRemote() bool {
	transferMethod := e.DefaultTransferMethod()
	switch transferMethod {
	case TransferMethodRsync:
		return strings.TrimSpace(e.GetHost()) != ""
	case TransferMethodHttp:
		return true // Object storage is always considered remote
	case TransferMethodLocal:
		return false
	default:
		return strings.TrimSpace(e.GetHost()) != "" // Default to SSH-based detection
	}
}

// DefaultTransferMethod provides a default transfer method based on endpoint configuration.
// This is used as fallback when Method is not explicitly specified in TransferOptions.
func (e *EndpointDetails) DefaultTransferMethod() string {
	// Auto-detect based on endpoint patterns
	endpoint := e.GetHost()

	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return TransferMethodHttp
	}

	if endpoint != "" {
		return TransferMethodRsync
	}

	return TransferMethodLocal
}

// GetTransferMethod returns the transfer method from options or falls back to default detection.
func GetTransferMethod(options *TransferOptions, endpoint EndpointDetails) string {
	if options != nil && options.Method != "" {
		if IsValidTransferMethod(options.Method) {
			return options.Method
		}
	}
	return endpoint.DefaultTransferMethod()
}

// getEffectiveSourceOptions returns the effective source transfer options.
func (task *DataMigrationModel) getEffectiveSourceOptions() *TransferOptions {
	if task.SourceTransferOptions != nil {
		return task.SourceTransferOptions
	}

	// Default configuration based on source transfer method
	transferMethod := GetTransferMethod(task.SourceTransferOptions, task.Source)
	switch transferMethod {
	case TransferMethodRsync:
		return &TransferOptions{
			Method: TransferMethodRsync,
			RsyncOptions: &RsyncOption{
				Archive:        true,
				Compress:       true,
				ConnectTimeout: 30,
			},
		}
	case TransferMethodHttp:
		return &TransferOptions{
			Method: TransferMethodHttp,
			HttpTransferOptions: &HttpTransferOption{
				Operation:  HttpOperationDownload, // Default for source (read operation)
				Method:     HttpMethodGet,
				Timeout:    300,
				MaxRetries: 3,
				VerifySSL:  true,
			},
		}
	default:
		return &TransferOptions{
			Method: TransferMethodLocal,
		}
	}
}

// getEffectiveDestinationOptions returns the effective destination transfer options.
func (task *DataMigrationModel) getEffectiveDestinationOptions() *TransferOptions {
	if task.DestinationTransferOptions != nil {
		return task.DestinationTransferOptions
	}

	// Default configuration based on destination transfer method
	transferMethod := GetTransferMethod(task.DestinationTransferOptions, task.Destination)
	switch transferMethod {
	case TransferMethodRsync:
		return &TransferOptions{
			Method: TransferMethodRsync,
			RsyncOptions: &RsyncOption{
				Archive:        true,
				Compress:       true,
				ConnectTimeout: 30,
			},
		}
	case TransferMethodHttp:
		return &TransferOptions{
			Method: TransferMethodHttp,
			HttpTransferOptions: &HttpTransferOption{
				Operation:  HttpOperationUpload, // Default for destination (write operation)
				Method:     HttpMethodPut,
				Timeout:    300,
				MaxRetries: 3,
				VerifySSL:  true,
			},
		}
	default:
		return &TransferOptions{
			Method: TransferMethodLocal,
		}
	}
}

// GetRsyncPath constructs the path string suitable for rsync (e.g., "user@host:/path" or "/local/path").
func (e *EndpointDetails) GetRsyncPath(options *TransferOptions) string {
	if e.IsRemote() {
		var username string
		if options != nil && options.RsyncOptions != nil {
			username = options.RsyncOptions.Username
		}
		host := e.GetHost()
		if strings.TrimSpace(username) != "" {
			return fmt.Sprintf("%s@%s:%s", username, host, e.DataPath)
		}
		return fmt.Sprintf("%s:%s", host, e.DataPath) // Username might be optional if SSH config handles it
	}
	return e.DataPath
}

// IsRelayMode determines if both source and destination endpoints are remote.
// This is used to identify relay migration scenarios where data needs to flow through the local machine
// as an intermediary between two remote endpoints.
func (task *DataMigrationModel) IsRelayMode() bool {
	return task.Source.IsRemote() && task.Destination.IsRemote()
}

// Validate checks if the fields of DataMigrationModel satisfy basic requirements for an rsync task.
func Validate(task DataMigrationModel) error {
	sourceRsyncPath := task.Source.GetRsyncPath(nil)    // Basic validation without specific options
	destRsyncPath := task.Destination.GetRsyncPath(nil) // Basic validation without specific options

	if strings.TrimSpace(sourceRsyncPath) == "" || strings.TrimSpace(task.Source.DataPath) == "" {
		return fmt.Errorf("source path must be provided for rsync task")
	}
	if strings.TrimSpace(destRsyncPath) == "" || strings.TrimSpace(task.Destination.DataPath) == "" {
		return fmt.Errorf("destination path must be provided for rsync task")
	}

	// Validate SSH port for source if it's a remote endpoint
	if task.Source.IsRemote() {
		sourcePort := task.Source.GetPort()
		if sourcePort != 0 && (sourcePort < 1 || sourcePort > 65535) {
			return fmt.Errorf("source SSH port %d is out of valid range (1-65535)", sourcePort)
		}
		if strings.TrimSpace(task.Source.GetHost()) == "" {
			return fmt.Errorf("source HostIP must be provided for remote rsync task")
		}
	}
	// Validate SSH port for destination if it's a remote endpoint
	if task.Destination.IsRemote() {
		destPort := task.Destination.GetPort()
		if destPort != 0 && (destPort < 1 || destPort > 65535) {
			return fmt.Errorf("destination SSH port %d is out of valid range (1-65535)", destPort)
		}
		if strings.TrimSpace(task.Destination.GetHost()) == "" {
			return fmt.Errorf("destination HostIP must be provided for remote rsync task")
		}
	}
	// The existence of SSHPrivateKey path etc. will be handled by the ssh command at runtime.
	// The Validate function primarily checks for structural issues.
	return nil
}

// Transfer runs the transfer command to transfer data as defined by the given DataMigrationModel.
func Transfer(task DataMigrationModel) error {
	if err := Validate(task); err != nil {
		return fmt.Errorf("transfer task validation failed: %w", err)
	}

	// Get effective options
	sourceOptions := task.getEffectiveSourceOptions()
	destOptions := task.getEffectiveDestinationOptions()

	// Check if we're operating in relay mode (both source and destination are remote)
	isRelayMode := task.IsRelayMode()

	// Determine transfer method based on transfer methods
	sourceTransferMethod := GetTransferMethod(sourceOptions, task.Source)
	destTransferMethod := GetTransferMethod(destOptions, task.Destination)

	// Get verbose setting from rsync options (if available)
	verbose := false
	if sourceOptions.RsyncOptions != nil {
		verbose = sourceOptions.RsyncOptions.Verbose
	} else if destOptions.RsyncOptions != nil {
		verbose = destOptions.RsyncOptions.Verbose
	}

	if verbose {
		fmt.Printf("Transfer mode: %s\n", getTransferModeDescription(isRelayMode, sourceTransferMethod, destTransferMethod))
		fmt.Printf("Source transfer method: %s, Destination transfer method: %s\n", sourceTransferMethod, destTransferMethod)
	}

	// Handle different transfer scenarios
	if !isRelayMode {
		// Direct mode: at least one endpoint is local
		return handleDirectTransfer(task, sourceOptions, destOptions)
	} else {
		// Relay mode: both endpoints are remote
		return handleRelayTransfer(task, sourceOptions, destOptions)
	}
}

// getTransferModeDescription returns a description of the transfer mode
func getTransferModeDescription(isRelayMode bool, sourceTransferMethod, destTransferMethod string) string {
	if isRelayMode {
		return fmt.Sprintf("Relay mode (%s → relay node → %s)", sourceTransferMethod, destTransferMethod)
	}
	return fmt.Sprintf("Direct mode (%s → %s)", sourceTransferMethod, destTransferMethod)
}

// handleDirectTransfer handles direct transfer where at least one endpoint is local
func handleDirectTransfer(task DataMigrationModel, sourceOptions, destOptions *TransferOptions) error {
	sourceTransferMethod := GetTransferMethod(sourceOptions, task.Source)
	destTransferMethod := GetTransferMethod(destOptions, task.Destination)

	// For direct mode, use rsync if both endpoints support it
	if (sourceTransferMethod == TransferMethodLocal || sourceTransferMethod == TransferMethodRsync) && (destTransferMethod == TransferMethodLocal || destTransferMethod == TransferMethodRsync) {
		return performRsyncTransfer(task, sourceOptions)
	}

	// Handle mixed transfer method transfers
	if sourceTransferMethod == TransferMethodHttp && destTransferMethod == TransferMethodLocal {
		return performHttpDownload(task, sourceOptions)
	}
	if sourceTransferMethod == TransferMethodLocal && destTransferMethod == TransferMethodHttp {
		return performHttpUpload(task, destOptions)
	}

	return fmt.Errorf("unsupported direct transfer combination: %s → %s", sourceTransferMethod, destTransferMethod)
}

// handleRelayTransfer handles relay transfer where both endpoints are remote
func handleRelayTransfer(task DataMigrationModel, sourceOptions, destOptions *TransferOptions) error {
	// For relay mode, we need to download from source and upload to destination
	sourceTransferMethod := GetTransferMethod(sourceOptions, task.Source)
	destTransferMethod := GetTransferMethod(destOptions, task.Destination)

	// Get verbose setting from rsync options (if available)
	verbose := false
	if sourceOptions.RsyncOptions != nil {
		verbose = sourceOptions.RsyncOptions.Verbose
	} else if destOptions.RsyncOptions != nil {
		verbose = destOptions.RsyncOptions.Verbose
	}

	if verbose {
		fmt.Printf("Relay transfer: %s → relay node → %s\n", sourceTransferMethod, destTransferMethod)
	}

	// Create temporary directory for relay
	tempDir, err := os.MkdirTemp("", "transx-relay-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Step 1: Download from source to local temp
	tempTask := task
	tempTask.Destination = EndpointDetails{DataPath: tempDir}
	tempTask.DestinationTransferOptions = &TransferOptions{Method: TransferMethodLocal} // Local destination

	if err := handleDirectTransfer(tempTask, sourceOptions, &TransferOptions{Method: TransferMethodLocal}); err != nil {
		return fmt.Errorf("relay step 1 (source → relay node) failed: %w", err)
	}

	// Step 2: Upload from local temp to destination
	tempTask = task
	tempTask.Source = EndpointDetails{DataPath: tempDir}
	tempTask.SourceTransferOptions = &TransferOptions{Method: TransferMethodLocal} // Local source

	if err := handleDirectTransfer(tempTask, &TransferOptions{Method: TransferMethodLocal}, destOptions); err != nil {
		return fmt.Errorf("relay step 2 (relay node → destination) failed: %w", err)
	}

	return nil
}

// performRsyncTransfer performs rsync-based transfer
func performRsyncTransfer(task DataMigrationModel, sourceOptions *TransferOptions) error {
	rsyncCmdPath := "rsync"
	if sourceOptions.RsyncOptions != nil && sourceOptions.RsyncOptions.RsyncPath != "" {
		rsyncCmdPath = sourceOptions.RsyncOptions.RsyncPath
	}

	var args []string
	// Configure basic rsync options
	if sourceOptions.RsyncOptions != nil && sourceOptions.RsyncOptions.Archive {
		args = append(args, "-a")
	}
	if sourceOptions.RsyncOptions != nil && sourceOptions.RsyncOptions.Compress {
		args = append(args, "-z")
	}
	if sourceOptions.RsyncOptions != nil && sourceOptions.RsyncOptions.Verbose {
		args = append(args, "-v")
	}
	if sourceOptions.RsyncOptions != nil && sourceOptions.RsyncOptions.Delete {
		args = append(args, "--delete")
	}
	if sourceOptions.RsyncOptions != nil && sourceOptions.RsyncOptions.Progress {
		args = append(args, "--progress")
	}
	if sourceOptions.RsyncOptions != nil && sourceOptions.RsyncOptions.DryRun {
		args = append(args, "-n") // or "--dry-run"
	}

	// Configure Exclude and Include options
	if sourceOptions.RsyncOptions != nil {
		for _, ex := range sourceOptions.RsyncOptions.Exclude {
			if strings.TrimSpace(ex) != "" {
				args = append(args, "--exclude="+ex)
			}
		}
		for _, inc := range sourceOptions.RsyncOptions.Include {
			if strings.TrimSpace(inc) != "" {
				args = append(args, "--include="+inc)
			}
		}
	}

	// Configure SSH options (-e)
	var sshOptString string
	var activeRemoteEndpointForRsync EndpointDetails
	operationInvolvesRemoteRsync := false

	if task.Source.IsRemote() {
		activeRemoteEndpointForRsync = task.Source
		operationInvolvesRemoteRsync = true
	} else if task.Destination.IsRemote() {
		activeRemoteEndpointForRsync = task.Destination
		operationInvolvesRemoteRsync = true
	}

	if operationInvolvesRemoteRsync && sourceOptions.RsyncOptions != nil && sourceOptions.RsyncOptions.SSHPrivateKeyPath != "" {
		var sshCmdParts []string
		sshCmdParts = append(sshCmdParts, "ssh")
		if strings.TrimSpace(sourceOptions.RsyncOptions.SSHPrivateKeyPath) != "" {
			sshCmdParts = append(sshCmdParts, "-i", sourceOptions.RsyncOptions.SSHPrivateKeyPath)
		}
		if activeRemoteEndpointForRsync.GetPort() != 0 { // If 0, use default port (22)
			sshCmdParts = append(sshCmdParts, "-p", strconv.Itoa(activeRemoteEndpointForRsync.GetPort()))
		}
		if sourceOptions.RsyncOptions != nil && sourceOptions.RsyncOptions.InsecureSkipHostKeyVerification {
			sshCmdParts = append(sshCmdParts, "-o", "StrictHostKeyChecking=accept-new")
			sshCmdParts = append(sshCmdParts, "-o", "UserKnownHostsFile=/dev/null")
		}

		// Set connection timeout if specified
		if sourceOptions.RsyncOptions != nil && sourceOptions.RsyncOptions.ConnectTimeout > 0 {
			connectTimeout := sourceOptions.RsyncOptions.ConnectTimeout
			sshCmdParts = append(sshCmdParts, "-o", fmt.Sprintf("ConnectTimeout=%d", connectTimeout))
		}

		sshOptString = strings.Join(sshCmdParts, " ")
	}

	if sshOptString != "" {
		args = append(args, "-e", sshOptString)
	}

	// Add source and destination paths
	sourceRsyncPath := task.Source.GetRsyncPath(sourceOptions)
	destinationRsyncPath := task.Destination.GetRsyncPath(sourceOptions) // Use sourceOptions for destination path too since it contains the SSH config

	// Handle TransferDirContentsOnly option
	if sourceOptions.RsyncOptions != nil && sourceOptions.RsyncOptions.TransferDirContentsOnly && !strings.HasSuffix(sourceRsyncPath, "/") {
		sourceRsyncPath += "/"
	}

	args = append(args, sourceRsyncPath, destinationRsyncPath)

	// Execute rsync command
	if sourceOptions.RsyncOptions != nil && sourceOptions.RsyncOptions.Verbose {
		fmt.Printf("Executing: %s %s\n", rsyncCmdPath, strings.Join(args, " "))
	}

	cmd := exec.Command(rsyncCmdPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// performHttpDownload downloads from HTTP object storage to local
func performHttpDownload(task DataMigrationModel, sourceOptions *TransferOptions) error {
	sourceURL := task.Source.GetDownloadURL()
	if sourceURL == "" {
		return fmt.Errorf("source does not have a valid download URL")
	}

	var httpTransferOptions *HttpTransferOption
	if sourceOptions != nil && sourceOptions.HttpTransferOptions != nil {
		httpTransferOptions = sourceOptions.HttpTransferOptions
	}

	// Get verbose setting from rsync options if available
	verbose := false
	if sourceOptions != nil && sourceOptions.RsyncOptions != nil {
		verbose = sourceOptions.RsyncOptions.Verbose
	}

	if verbose {
		fmt.Printf("Downloading from HTTP object storage: %s → %s\n", sourceURL, task.Destination.DataPath)
	}

	return DownloadFromObjectStorage(sourceURL, task.Destination.DataPath, httpTransferOptions)
}

// performHttpUpload uploads from local to HTTP object storage
func performHttpUpload(task DataMigrationModel, destOptions *TransferOptions) error {
	destURL := task.Destination.GetUploadURL()
	if destURL == "" {
		return fmt.Errorf("destination does not have a valid upload URL")
	}

	var httpTransferOptions *HttpTransferOption
	if destOptions != nil && destOptions.HttpTransferOptions != nil {
		httpTransferOptions = destOptions.HttpTransferOptions
	}

	// Get verbose setting from rsync options if available
	verbose := false
	if destOptions != nil && destOptions.RsyncOptions != nil {
		verbose = destOptions.RsyncOptions.Verbose
	}

	if verbose {
		fmt.Printf("Uploading to HTTP object storage: %s → %s\n", task.Source.DataPath, destURL)
	}

	return UploadToObjectStorage(task.Source.DataPath, destURL, httpTransferOptions)
}

// executeCommand executes the given command locally or remotely (via SSH).
func executeCommand(commandToExecute string, endpoint EndpointDetails, options *TransferOptions) ([]byte, error) {
	if strings.TrimSpace(commandToExecute) == "" {
		return nil, fmt.Errorf("command to execute cannot be empty")
	}

	if endpoint.IsRemote() { // Check if it's a remote endpoint
		if strings.TrimSpace(endpoint.GetHost()) == "" {
			return nil, fmt.Errorf("HostIP must be provided for remote command execution on endpoint")
		}

		userHost := endpoint.GetHost()
		var username string
		if options != nil && options.RsyncOptions != nil {
			username = options.RsyncOptions.Username
		}
		if strings.TrimSpace(username) != "" {
			userHost = fmt.Sprintf("%s@%s", username, endpoint.GetHost())
		}

		var sshCmdParts []string
		sshCmdParts = append(sshCmdParts, "ssh") // SSH command
		if options != nil && options.RsyncOptions != nil && strings.TrimSpace(options.RsyncOptions.SSHPrivateKeyPath) != "" {
			sshCmdParts = append(sshCmdParts, "-i", options.RsyncOptions.SSHPrivateKeyPath) // Private key
		}
		if endpoint.GetPort() != 0 { // SSH port (if not 0)
			sshCmdParts = append(sshCmdParts, "-p", strconv.Itoa(endpoint.GetPort()))
		}
		if options != nil && options.RsyncOptions != nil && options.RsyncOptions.InsecureSkipHostKeyVerification { // Skip host key verification option
			sshCmdParts = append(sshCmdParts, "-o", "StrictHostKeyChecking=accept-new")
			sshCmdParts = append(sshCmdParts, "-o", "UserKnownHostsFile=/dev/null")
		}

		// Add timeout for SSH connection
		connectTimeout := 30
		if options != nil && options.RsyncOptions != nil && options.RsyncOptions.ConnectTimeout > 0 {
			connectTimeout = options.RsyncOptions.ConnectTimeout
		}
		sshCmdParts = append(sshCmdParts, "-o", fmt.Sprintf("ConnectTimeout=%d", connectTimeout))

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

	// Get effective transfer options for SSH configuration
	transferOptions := dmm.getEffectiveSourceOptions()

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
			sourcePath = fmt.Sprintf("%s@%s:%s", username, source.GetHost(), source.DataPath)
		} else {
			sourcePath = fmt.Sprintf("%s:%s", source.GetHost(), source.DataPath)
		}
		fmt.Printf("Executing backup command on remote server %s...\n", source.GetHost())
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

	// Get effective transfer options for SSH configuration
	transferOptions := dmm.getEffectiveDestinationOptions()

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
			destinationDataPath = fmt.Sprintf("%s@%s:%s", username, destination.GetHost(), destination.DataPath)
		} else {
			destinationDataPath = fmt.Sprintf("%s:%s", destination.GetHost(), destination.DataPath)
		}
		fmt.Printf("Executing restore command on remote server %s...\n", destination.GetHost())
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

// UploadToObjectStorage uploads a file to object storage using presigned URL.
func UploadToObjectStorage(localFilePath string, presignedURL string, options *HttpTransferOption) error {
	if options == nil {
		options = &HttpTransferOption{
			Operation:  HttpOperationUpload,
			Method:     HttpMethodPut,
			Timeout:    300, // 5 minutes default
			MaxRetries: 3,
			VerifySSL:  true,
		}
	}

	// Ensure operation and method are set correctly for upload
	if options.Operation == "" {
		options.Operation = HttpOperationUpload
	}
	if options.Method == "" {
		options.Method = GetHttpMethodForOperation(options.Operation)
	}

	file, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	// Get file info for content length
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	req, err := http.NewRequest(options.Method, presignedURL, file)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.ContentLength = fileInfo.Size()

	// Add custom headers if specified
	if options.Headers != nil {
		for key, value := range options.Headers {
			req.Header.Set(key, value)
		}
	}

	if options.UserAgent != "" {
		req.Header.Set("User-Agent", options.UserAgent)
	}

	client := &http.Client{
		Timeout: time.Duration(options.Timeout) * time.Second,
	}

	// Retry logic
	maxRetries := options.MaxRetries
	if maxRetries == 0 {
		maxRetries = 1
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Reset file position for retries
		if attempt > 0 {
			file.Seek(0, 0)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed (attempt %d/%d): %w", attempt+1, maxRetries, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil // Success
		}

		lastErr = fmt.Errorf("upload failed with status %d (attempt %d/%d)", resp.StatusCode, attempt+1, maxRetries)
	}

	return lastErr
}

// DownloadFromObjectStorage downloads a file from object storage using presigned URL.
func DownloadFromObjectStorage(presignedURL string, localFilePath string, options *HttpTransferOption) error {
	if options == nil {
		options = &HttpTransferOption{
			Operation:  HttpOperationDownload,
			Method:     HttpMethodGet,
			Timeout:    300, // 5 minutes default
			MaxRetries: 3,
			VerifySSL:  true,
		}
	}

	// Ensure operation and method are set correctly for download
	if options.Operation == "" {
		options.Operation = HttpOperationDownload
	}
	if options.Method == "" {
		options.Method = GetHttpMethodForOperation(options.Operation)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(localFilePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	client := &http.Client{
		Timeout: time.Duration(options.Timeout) * time.Second,
	}

	// Retry logic
	maxRetries := options.MaxRetries
	if maxRetries == 0 {
		maxRetries = 1
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequest(options.Method, presignedURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		// Add custom headers if specified
		for key, value := range options.Headers {
			req.Header.Set(key, value)
		}

		// Set User-Agent if specified
		if options.UserAgent != "" {
			req.Header.Set("User-Agent", options.UserAgent)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed (attempt %d/%d): %w", attempt+1, maxRetries, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("download failed with status %d (attempt %d/%d)", resp.StatusCode, attempt+1, maxRetries)
			continue
		}

		// Create output file
		file, err := os.Create(localFilePath)
		if err != nil {
			lastErr = fmt.Errorf("failed to create local file: %w", err)
			continue
		}
		defer file.Close()

		// Copy data from response to file
		_, err = io.Copy(file, resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to write file data: %w", err)
			continue
		}

		return nil // Success
	}

	return lastErr
}

// DeleteFromObjectStorage deletes an object from object storage using presigned URL.
func DeleteFromObjectStorage(presignedURL string, options *HttpTransferOption) error {
	if options == nil {
		options = &HttpTransferOption{
			Operation:  httpOperationDeleteInternal,
			Method:     HttpMethodDelete,
			Timeout:    300, // 5 minutes default
			MaxRetries: 3,
			VerifySSL:  true,
		}
	}

	// Ensure operation and method are set correctly for delete
	if options.Operation == "" {
		options.Operation = httpOperationDeleteInternal
	}
	if options.Method == "" {
		options.Method = GetHttpMethodForOperation(options.Operation)
	}

	client := &http.Client{
		Timeout: time.Duration(options.Timeout) * time.Second,
	}

	// Retry logic
	maxRetries := options.MaxRetries
	if maxRetries == 0 {
		maxRetries = 1
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequest(options.Method, presignedURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		// Add custom headers if specified
		for key, value := range options.Headers {
			req.Header.Set(key, value)
		}

		// Set User-Agent if specified
		if options.UserAgent != "" {
			req.Header.Set("User-Agent", options.UserAgent)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed (attempt %d/%d): %w", attempt+1, maxRetries, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("delete failed with status %d (attempt %d/%d)", resp.StatusCode, attempt+1, maxRetries)
			continue
		}

		return nil // Success
	}

	return lastErr
}
