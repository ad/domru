package multiconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ad/domru/config"
	"github.com/google/uuid"
)

// MigrationService handles migration from legacy single-account configuration
type MigrationService struct {
	storage       *AccountStorage
	legacyPath    string
	backupDir     string
}

// NewMigrationService creates a new migration service
func NewMigrationService(storage *AccountStorage, legacyPath string) *MigrationService {
	return &MigrationService{
		storage:    storage,
		legacyPath: legacyPath,
		backupDir:  filepath.Join(filepath.Dir(storage.configFile), "migration_backups"),
	}
}

// NeedsMigration checks if migration is needed (legacy file exists and new format doesn't)
func (m *MigrationService) NeedsMigration() bool {
	// Check if legacy file exists
	if _, err := os.Stat(m.legacyPath); os.IsNotExist(err) {
		return false
	}

	// Check if new format already exists
	if _, err := os.Stat(m.storage.configFile); err == nil {
		return false // New format already exists, no migration needed
	}

	return true
}

// MigrateFromLegacy migrates from legacy configuration to multi-account format
func (m *MigrationService) MigrateFromLegacy() error {
	// Check if migration is needed
	if !m.NeedsMigration() {
		return nil
	}

	// Read legacy configuration
	legacyConfig, err := m.readLegacyConfig()
	if err != nil {
		return fmt.Errorf("failed to read legacy config: %w", err)
	}

	// Create backup of legacy configuration
	backupPath, err := m.createLegacyBackup()
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Convert to new format
	account, err := m.convertLegacyToAccount(legacyConfig)
	if err != nil {
		return fmt.Errorf("failed to convert legacy config: %w", err)
	}

	// Save in new format
	accounts := map[string]*AccountConfig{
		account.ID: account,
	}

	if err := m.storage.SaveAccounts(accounts); err != nil {
		// Migration failed, attempt to restore
		if restoreErr := m.restoreLegacyFromBackup(backupPath); restoreErr != nil {
			return fmt.Errorf("migration failed and restore failed: migration error: %w, restore error: %v", err, restoreErr)
		}
		return fmt.Errorf("migration failed, legacy config restored: %w", err)
	}

	// Migration successful, remove legacy file
	if err := os.Remove(m.legacyPath); err != nil {
		// Log warning but don't fail migration
		fmt.Printf("Warning: failed to remove legacy config file: %v\n", err)
	}

	return nil
}

// readLegacyConfig reads the legacy configuration file
func (m *MigrationService) readLegacyConfig() (*config.Config, error) {
	data, err := os.ReadFile(m.legacyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read legacy file: %w", err)
	}

	var legacyConfig config.Config
	if err := json.Unmarshal(data, &legacyConfig); err != nil {
		return nil, fmt.Errorf("failed to parse legacy config: %w", err)
	}

	return &legacyConfig, nil
}

// convertLegacyToAccount converts legacy config to new account format
func (m *MigrationService) convertLegacyToAccount(legacyConfig *config.Config) (*AccountConfig, error) {
	// Generate unique ID for the account
	accountID := uuid.New().String()

	// Preserve all data from legacy config
	account := &AccountConfig{
		ID:           accountID,
		Token:        legacyConfig.Token,
		RefreshToken: legacyConfig.RefreshToken,
		Login:        legacyConfig.Login,
		Operator:     legacyConfig.Operator,
		UUID:         legacyConfig.UUID,
		Name:         fmt.Sprintf("Migrated Account (%d)", legacyConfig.Login),
		Enabled:      true, // Default to enabled for migrated accounts
	}

	// Validate required fields
	if account.Login == 0 {
		return nil, fmt.Errorf("legacy config has invalid login")
	}
	if account.UUID == "" {
		return nil, fmt.Errorf("legacy config missing UUID")
	}

	return account, nil
}

// createLegacyBackup creates a backup of the legacy configuration
func (m *MigrationService) createLegacyBackup() (string, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(m.backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Read legacy file
	data, err := os.ReadFile(m.legacyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read legacy file for backup: %w", err)
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(m.backupDir, fmt.Sprintf("legacy_account_%s.json", timestamp))

	// Write backup
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	return backupPath, nil
}

// restoreLegacyFromBackup restores legacy configuration from backup
func (m *MigrationService) restoreLegacyFromBackup(backupPath string) error {
	// Read backup
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Restore to original location
	if err := os.WriteFile(m.legacyPath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore legacy file: %w", err)
	}

	return nil
}

// GetLegacyConfigPath returns the path to the legacy configuration file
func (m *MigrationService) GetLegacyConfigPath() string {
	return m.legacyPath
}

// CleanupBackups removes old migration backup files (keeps last 5)
func (m *MigrationService) CleanupBackups() error {
	backupFiles, err := filepath.Glob(filepath.Join(m.backupDir, "legacy_account_*.json"))
	if err != nil {
		return fmt.Errorf("failed to list backup files: %w", err)
	}

	// Keep only the 5 most recent backups
	if len(backupFiles) <= 5 {
		return nil
	}

	// Sort files by name (which includes timestamp)
	// Remove oldest files
	for i := 0; i < len(backupFiles)-5; i++ {
		if err := os.Remove(backupFiles[i]); err != nil {
			fmt.Printf("Warning: failed to remove old backup %s: %v\n", backupFiles[i], err)
		}
	}

	return nil
}