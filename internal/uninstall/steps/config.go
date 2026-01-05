// Package steps provides uninstallation step implementations for Igor.
// Each step represents a discrete phase of the NVIDIA driver uninstallation process.
package steps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tungetti/igor/internal/install"
)

// State keys for configuration cleanup.
const (
	// StateConfigsCleaned indicates configs were removed.
	StateConfigsCleaned = "configs_cleaned"
	// StateCleanedConfigs is the list of removed config paths.
	StateCleanedConfigs = "cleaned_configs"
	// StateBackedUpConfigs is the list of configs that were backed up.
	StateBackedUpConfigs = "backed_up_configs"
	// StateBackupDir is the directory where backups are stored.
	StateBackupDir = "backup_dir"
)

// Default backup directory for configuration files.
const DefaultBackupDir = "/var/lib/igor/backup/configs"

// DefaultNvidiaConfigPaths contains common NVIDIA configuration file paths.
var DefaultNvidiaConfigPaths = []string{
	"/etc/X11/xorg.conf.d/20-nvidia.conf",
	"/etc/modprobe.d/nvidia.conf",
	"/etc/modprobe.d/nvidia-blacklists-nouveau.conf",
	"/etc/modprobe.d/blacklist-nouveau.conf",
	"/usr/share/X11/xorg.conf.d/nvidia.conf",
}

// NouveauBlacklistPaths contains paths to nouveau blacklist files.
var NouveauBlacklistPaths = []string{
	"/etc/modprobe.d/nvidia-blacklists-nouveau.conf",
	"/etc/modprobe.d/blacklist-nouveau.conf",
	"/etc/modprobe.d/blacklist_nouveau.conf",
}

// XorgConfigPaths contains paths to X.org nvidia configuration.
var XorgConfigPaths = []string{
	"/etc/X11/xorg.conf.d/20-nvidia.conf",
	"/etc/X11/xorg.conf.d/10-nvidia.conf",
	"/usr/share/X11/xorg.conf.d/nvidia.conf",
}

// ModprobeConfigPaths contains paths to modprobe.d nvidia configuration.
var ModprobeConfigPaths = []string{
	"/etc/modprobe.d/nvidia.conf",
	"/etc/modprobe.d/nvidia-drm.conf",
	"/etc/modprobe.d/nvidia-graphics-drivers.conf",
}

// PersistenceConfigPaths contains paths to nvidia-persistenced configuration.
var PersistenceConfigPaths = []string{
	"/etc/nvidia-persistenced.conf",
	"/etc/systemd/system/nvidia-persistenced.service.d/override.conf",
}

// AllowedConfigDirs contains the directories that are safe to remove files from.
// This prevents path traversal attacks by limiting operations to known safe directories.
var AllowedConfigDirs = []string{
	"/etc/",
	"/usr/share/",
	"/var/lib/igor/",
}

// ConfigCleanupStep removes NVIDIA configuration files from the system.
type ConfigCleanupStep struct {
	install.BaseStep
	configPaths       []string // Specific paths to remove
	backupDir         string   // Directory to backup configs before removal
	createBackup      bool     // Whether to backup before removing
	removeBlacklist   bool     // Remove nouveau blacklist files
	removeXorgConf    bool     // Remove X.org nvidia config
	removeModprobe    bool     // Remove modprobe.d nvidia configs
	removePersistence bool     // Remove nvidia-persistenced config
	fileChecker       FileChecker
}

// FileChecker abstracts file system operations for testing.
type FileChecker interface {
	FileExists(path string) bool
}

// RealFileChecker implements FileChecker using the real filesystem.
type RealFileChecker struct{}

// FileExists checks if a file exists.
func (r *RealFileChecker) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Ensure RealFileChecker implements FileChecker.
var _ FileChecker = (*RealFileChecker)(nil)

// ConfigCleanupStepOption configures the ConfigCleanupStep.
type ConfigCleanupStepOption func(*ConfigCleanupStep)

// WithConfigPaths sets specific configuration paths to remove.
func WithConfigPaths(paths []string) ConfigCleanupStepOption {
	return func(s *ConfigCleanupStep) {
		s.configPaths = append(s.configPaths, paths...)
	}
}

// WithBackupDir sets the backup directory for removed configs.
// Default is "/var/lib/igor/backup/configs".
func WithBackupDir(dir string) ConfigCleanupStepOption {
	return func(s *ConfigCleanupStep) {
		s.backupDir = dir
	}
}

// WithCreateBackup enables/disables backup before removal.
// Default is true.
func WithCreateBackup(backup bool) ConfigCleanupStepOption {
	return func(s *ConfigCleanupStep) {
		s.createBackup = backup
	}
}

// WithRemoveBlacklist enables removal of nouveau blacklist files.
// Default is true.
func WithRemoveBlacklist(remove bool) ConfigCleanupStepOption {
	return func(s *ConfigCleanupStep) {
		s.removeBlacklist = remove
	}
}

// WithRemoveXorgConf enables removal of X.org nvidia config.
// Default is true.
func WithRemoveXorgConf(remove bool) ConfigCleanupStepOption {
	return func(s *ConfigCleanupStep) {
		s.removeXorgConf = remove
	}
}

// WithRemoveModprobe enables removal of modprobe.d nvidia configs.
// Default is true.
func WithRemoveModprobe(remove bool) ConfigCleanupStepOption {
	return func(s *ConfigCleanupStep) {
		s.removeModprobe = remove
	}
}

// WithRemovePersistence enables removal of nvidia-persistenced config.
// Default is true.
func WithRemovePersistence(remove bool) ConfigCleanupStepOption {
	return func(s *ConfigCleanupStep) {
		s.removePersistence = remove
	}
}

// WithFileChecker sets a custom file checker for testing.
func WithFileChecker(checker FileChecker) ConfigCleanupStepOption {
	return func(s *ConfigCleanupStep) {
		s.fileChecker = checker
	}
}

// NewConfigCleanupStep creates a new ConfigCleanupStep with the given options.
func NewConfigCleanupStep(opts ...ConfigCleanupStepOption) *ConfigCleanupStep {
	s := &ConfigCleanupStep{
		BaseStep:          install.NewBaseStep("config_cleanup", "Remove NVIDIA configuration files", true),
		configPaths:       make([]string, 0),
		backupDir:         DefaultBackupDir,
		createBackup:      true,
		removeBlacklist:   true,
		removeXorgConf:    true,
		removeModprobe:    true,
		removePersistence: true,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Execute removes NVIDIA configuration files from the system.
// It performs the following steps:
//  1. Checks for cancellation
//  2. Validates context has executor
//  3. Collects all config paths to remove based on options
//  4. Filters to only existing files
//  5. If no files to remove, returns skip
//  6. In dry-run mode, logs what would be removed and returns success
//  7. If createBackup is true, backs up each file before removal
//  8. Removes each configuration file
//  9. Tracks removed files in state
//  10. Returns success with count of removed files
func (s *ConfigCleanupStep) Execute(ctx *install.Context) install.StepResult {
	startTime := time.Now()

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled)
	}

	ctx.LogDebug("starting configuration cleanup")

	// Validate prerequisites
	if err := s.Validate(ctx); err != nil {
		return install.FailStep("validation failed", err).WithDuration(time.Since(startTime))
	}

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Collect all config paths to remove
	allPaths := s.collectConfigPaths()
	ctx.LogDebug("collected config paths", "count", len(allPaths))

	// Filter to only existing files
	existingPaths := s.filterExistingFiles(ctx, allPaths)
	ctx.LogDebug("existing config files", "count", len(existingPaths))

	// If no files to remove, skip
	if len(existingPaths) == 0 {
		ctx.Log("no NVIDIA configuration files to remove")
		return install.SkipStep("no NVIDIA configuration files to remove").
			WithDuration(time.Since(startTime))
	}

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Dry run mode
	if ctx.DryRun {
		ctx.Log("dry run: would remove NVIDIA configuration files")
		for _, path := range existingPaths {
			ctx.Log("dry run: would remove", "path", path)
		}
		if s.createBackup {
			ctx.Log("dry run: would backup configs to", "dir", s.backupDir)
		}
		return install.CompleteStep("dry run: NVIDIA configuration files would be removed").
			WithDuration(time.Since(startTime))
	}

	// Check for cancellation before starting removal
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Create backup directory if needed
	if s.createBackup {
		if err := s.createBackupDir(ctx); err != nil {
			ctx.LogError("failed to create backup directory", "dir", s.backupDir, "error", err)
			return install.FailStep("failed to create backup directory", err).
				WithDuration(time.Since(startTime))
		}
	}

	// Process each file: backup and remove
	ctx.Log("removing NVIDIA configuration files", "count", len(existingPaths))
	var cleanedConfigs []string
	var backedUpConfigs []string

	for _, configPath := range existingPaths {
		// Check for cancellation between file operations
		if ctx.IsCancelled() {
			// Store partial state
			ctx.SetState(StateConfigsCleaned, len(cleanedConfigs) > 0)
			ctx.SetState(StateCleanedConfigs, append([]string{}, cleanedConfigs...))
			ctx.SetState(StateBackedUpConfigs, append([]string{}, backedUpConfigs...))
			ctx.SetState(StateBackupDir, s.backupDir)
			return install.FailStep("step cancelled", context.Canceled).
				WithDuration(time.Since(startTime))
		}

		// Backup if enabled
		if s.createBackup {
			if err := backupFile(ctx, configPath, s.backupDir); err != nil {
				ctx.LogWarn("failed to backup config file, continuing anyway", "path", configPath, "error", err)
			} else {
				backedUpConfigs = append(backedUpConfigs, configPath)
				ctx.LogDebug("backed up config file", "path", configPath)
			}
		}

		// Remove the file
		if err := removeFile(ctx, configPath); err != nil {
			ctx.LogError("failed to remove config file", "path", configPath, "error", err)
			// Store partial state and fail
			ctx.SetState(StateConfigsCleaned, len(cleanedConfigs) > 0)
			ctx.SetState(StateCleanedConfigs, append([]string{}, cleanedConfigs...))
			ctx.SetState(StateBackedUpConfigs, append([]string{}, backedUpConfigs...))
			ctx.SetState(StateBackupDir, s.backupDir)
			return install.FailStep(fmt.Sprintf("failed to remove config file '%s'", configPath), err).
				WithDuration(time.Since(startTime))
		}

		cleanedConfigs = append(cleanedConfigs, configPath)
		ctx.LogDebug("removed config file", "path", configPath)
	}

	// Store state for rollback
	ctx.SetState(StateConfigsCleaned, true)
	ctx.SetState(StateCleanedConfigs, append([]string{}, cleanedConfigs...))
	ctx.SetState(StateBackedUpConfigs, append([]string{}, backedUpConfigs...))
	ctx.SetState(StateBackupDir, s.backupDir)

	ctx.Log("NVIDIA configuration files removed successfully", "count", len(cleanedConfigs))
	return install.CompleteStep(fmt.Sprintf("removed %d NVIDIA configuration files", len(cleanedConfigs))).
		WithDuration(time.Since(startTime)).
		WithCanRollback(len(backedUpConfigs) > 0)
}

// Rollback restores the backed up configuration files.
// This is only possible if backups were created during execution.
func (s *ConfigCleanupStep) Rollback(ctx *install.Context) error {
	// Check if we actually cleaned configs
	if !ctx.GetStateBool(StateConfigsCleaned) {
		ctx.LogDebug("no configs were cleaned, nothing to rollback")
		return nil
	}

	// Get the list of backed up configs
	backedUpConfigsRaw, ok := ctx.GetState(StateBackedUpConfigs)
	if !ok {
		ctx.LogDebug("backed up configs list not found in state, nothing to rollback")
		return nil
	}

	backedUpConfigs, ok := backedUpConfigsRaw.([]string)
	if !ok {
		ctx.LogDebug("backed up configs list has invalid type, nothing to rollback")
		return nil
	}

	if len(backedUpConfigs) == 0 {
		ctx.LogDebug("no configs were backed up, nothing to rollback")
		return nil
	}

	// Get backup directory
	backupDir := ctx.GetStateString(StateBackupDir)
	if backupDir == "" {
		backupDir = s.backupDir
	}

	// Validate executor
	if ctx.Executor == nil {
		return fmt.Errorf("executor not available for rollback")
	}

	ctx.Log("rolling back configuration cleanup (restoring configs)")

	var firstErr error
	for _, configPath := range backedUpConfigs {
		backupPath := getBackupPath(configPath, backupDir)
		ctx.LogDebug("restoring config file", "from", backupPath, "to", configPath)

		if err := restoreFile(ctx, backupPath, configPath); err != nil {
			ctx.LogError("failed to restore config file", "path", configPath, "error", err)
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to restore config '%s': %w", configPath, err)
			}
			// Continue trying to restore other files
		}
	}

	// Clear state
	ctx.DeleteState(StateConfigsCleaned)
	ctx.DeleteState(StateCleanedConfigs)
	ctx.DeleteState(StateBackedUpConfigs)
	ctx.DeleteState(StateBackupDir)

	ctx.LogDebug("configuration cleanup rollback completed")
	return firstErr
}

// Validate checks if the step can be executed with the given context.
// It ensures the Executor is available and validates paths for security.
func (s *ConfigCleanupStep) Validate(ctx *install.Context) error {
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}
	if ctx.Executor == nil {
		return fmt.Errorf("executor is required for configuration cleanup")
	}

	// Validate backup directory path
	if s.backupDir != "" && !isValidPath(s.backupDir) {
		return fmt.Errorf("invalid backup directory path: %q", s.backupDir)
	}

	// Validate custom config paths
	for _, path := range s.configPaths {
		if !isValidPath(path) {
			return fmt.Errorf("invalid config path: %q", path)
		}
	}

	return nil
}

// CanRollback returns true if backups can be created (createBackup is enabled).
func (s *ConfigCleanupStep) CanRollback() bool {
	return s.createBackup
}

// collectConfigPaths collects all config paths based on step options.
func (s *ConfigCleanupStep) collectConfigPaths() []string {
	seen := make(map[string]bool)
	var paths []string

	// Helper to add unique paths
	addPaths := func(toAdd []string) {
		for _, p := range toAdd {
			if p != "" && !seen[p] && isValidPath(p) {
				seen[p] = true
				paths = append(paths, p)
			}
		}
	}

	// Add specific config paths first
	addPaths(s.configPaths)

	// Add paths based on removal flags
	if s.removeBlacklist {
		addPaths(NouveauBlacklistPaths)
	}
	if s.removeXorgConf {
		addPaths(XorgConfigPaths)
	}
	if s.removeModprobe {
		addPaths(ModprobeConfigPaths)
	}
	if s.removePersistence {
		addPaths(PersistenceConfigPaths)
	}

	return paths
}

// filterExistingFiles filters the paths to only include files that exist.
func (s *ConfigCleanupStep) filterExistingFiles(ctx *install.Context, paths []string) []string {
	checker := s.getFileChecker()
	var existing []string

	for _, path := range paths {
		if checker.FileExists(path) {
			existing = append(existing, path)
		}
	}

	return existing
}

// getFileChecker returns the file checker (real or mock).
func (s *ConfigCleanupStep) getFileChecker() FileChecker {
	if s.fileChecker != nil {
		return s.fileChecker
	}
	return &RealFileChecker{}
}

// createBackupDir creates the backup directory if it doesn't exist.
func (s *ConfigCleanupStep) createBackupDir(ctx *install.Context) error {
	result := ctx.Executor.ExecuteElevated(ctx.Context(), "mkdir", "-p", s.backupDir)
	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("failed to create backup directory: %s", errMsg)
	}
	return nil
}

// backupFile copies a file to the backup directory.
func backupFile(ctx *install.Context, src, backupDir string) error {
	// Validate source path
	if !isValidPath(src) {
		return fmt.Errorf("invalid source path: %q", src)
	}

	// Generate backup path
	backupPath := getBackupPath(src, backupDir)

	// Create parent directory in backup location
	backupParent := filepath.Dir(backupPath)
	mkdirResult := ctx.Executor.ExecuteElevated(ctx.Context(), "mkdir", "-p", backupParent)
	if mkdirResult.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(mkdirResult.Stderr))
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("failed to create backup parent directory: %s", errMsg)
	}

	// Copy the file to backup location
	result := ctx.Executor.ExecuteElevated(ctx.Context(), "cp", "-p", src, backupPath)
	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("failed to backup file: %s", errMsg)
	}

	return nil
}

// removeFile removes a file from the filesystem.
func removeFile(ctx *install.Context, path string) error {
	// Validate path
	if !isValidPath(path) {
		return fmt.Errorf("invalid path for removal: %q", path)
	}

	result := ctx.Executor.ExecuteElevated(ctx.Context(), "rm", "-f", path)
	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("failed to remove file: %s", errMsg)
	}

	return nil
}

// restoreFile restores a file from backup.
func restoreFile(ctx *install.Context, backupPath, originalPath string) error {
	// Validate paths
	if !isValidPath(backupPath) {
		return fmt.Errorf("invalid backup path: %q", backupPath)
	}
	if !isValidPath(originalPath) {
		return fmt.Errorf("invalid original path: %q", originalPath)
	}

	// Create parent directory for original path if needed
	parentDir := filepath.Dir(originalPath)
	mkdirResult := ctx.Executor.ExecuteElevated(ctx.Context(), "mkdir", "-p", parentDir)
	if mkdirResult.ExitCode != 0 {
		// Non-fatal, directory might already exist
		ctx.LogDebug("mkdir for parent directory returned non-zero", "dir", parentDir)
	}

	// Copy the backup back to original location
	result := ctx.Executor.ExecuteElevated(ctx.Context(), "cp", "-p", backupPath, originalPath)
	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("failed to restore file: %s", errMsg)
	}

	return nil
}

// getBackupPath generates the backup path for a source file.
// It preserves the directory structure within the backup directory.
func getBackupPath(srcPath, backupDir string) string {
	// Remove leading slash and join with backup dir
	relativePath := strings.TrimPrefix(srcPath, "/")
	return filepath.Join(backupDir, relativePath)
}

// isValidPath validates a path is safe (no traversal, no dangerous characters).
// It also ensures the path is within allowed directories.
func isValidPath(path string) bool {
	if path == "" {
		return false
	}

	// Check for path traversal
	if strings.Contains(path, "..") {
		return false
	}

	// Check for dangerous characters (command injection)
	dangerousChars := []string{";", "&", "|", "`", "$", "(", ")", "{", "}", "<", ">", "!", "\n", "\r", "'", "\""}
	for _, char := range dangerousChars {
		if strings.Contains(path, char) {
			return false
		}
	}

	// Path must be absolute
	if !filepath.IsAbs(path) {
		return false
	}

	// Check if path is within allowed directories
	// Clean the path first to handle any edge cases
	cleanPath := filepath.Clean(path)
	inAllowedDir := false
	for _, allowedDir := range AllowedConfigDirs {
		if strings.HasPrefix(cleanPath, allowedDir) {
			inAllowedDir = true
			break
		}
	}

	return inAllowedDir
}

// Ensure ConfigCleanupStep implements the Step interface.
var _ install.Step = (*ConfigCleanupStep)(nil)
