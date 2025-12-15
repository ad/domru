package multiconfig

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// AccountConfig represents a single account configuration
type AccountConfig struct {
	ID           string `json:"id"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh"`
	Login        int    `json:"login"`
	Operator     int    `json:"operator"`
	UUID         string `json:"uuid"`
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
}

// MultiAccountConfig represents the complete multi-account configuration
type MultiAccountConfig struct {
	Accounts       map[string]*AccountConfig `json:"accounts"`
	PrimaryAccount string                    `json:"primary_account"`
	Version        string                    `json:"version"`
}

// AccountStorage handles reading and writing account configurations
type AccountStorage struct {
	configDir  string
	configFile string
	backupDir  string
}

// NewAccountStorage creates a new AccountStorage instance
func NewAccountStorage(configDir string) *AccountStorage {
	return &AccountStorage{
		configDir:  configDir,
		configFile: filepath.Join(configDir, "accounts.json"),
		backupDir:  filepath.Join(configDir, "backups"),
	}
}

// LoadAccounts loads all accounts from the configuration file
func (s *AccountStorage) LoadAccounts() (map[string]*AccountConfig, error) {
	log.Printf("[AccountStorage] Loading accounts from %s", s.configFile)
	config, err := s.loadConfig()
	if err != nil {
		log.Printf("[AccountStorage] Failed to load accounts: %v", err)
		return nil, err
	}
	log.Printf("[AccountStorage] Successfully loaded %d accounts", len(config.Accounts))
	return config.Accounts, nil
}

// loadConfig loads the complete configuration including primary account info
func (s *AccountStorage) loadConfig() (*MultiAccountConfig, error) {
	log.Printf("[AccountStorage] Loading configuration from %s", s.configFile)
	
	// Ensure directory exists
	if err := s.ensureDir(s.configDir); err != nil {
		log.Printf("[AccountStorage] Failed to create config directory %s: %v", s.configDir, err)
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(s.configFile); os.IsNotExist(err) {
		log.Printf("[AccountStorage] Config file does not exist, returning empty configuration")
		// Return empty config if file doesn't exist
		return &MultiAccountConfig{
			Accounts: make(map[string]*AccountConfig),
			Version:  "2.0",
		}, nil
	}

	// Read file
	data, err := os.ReadFile(s.configFile)
	if err != nil {
		log.Printf("[AccountStorage] Failed to read config file %s: %v", s.configFile, err)
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var config MultiAccountConfig
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("[AccountStorage] Failed to parse JSON from %s: %v", s.configFile, err)
		return nil, fmt.Errorf("invalid configuration format: %w", err)
	}

	// Validate configuration
	if err := s.validateConfig(&config); err != nil {
		log.Printf("[AccountStorage] Configuration validation failed: %v", err)
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	log.Printf("[AccountStorage] Successfully loaded and validated configuration")
	return &config, nil
}

// SaveAccounts saves all accounts to the configuration file
func (s *AccountStorage) SaveAccounts(accounts map[string]*AccountConfig) error {
	log.Printf("[AccountStorage] Saving %d accounts to %s", len(accounts), s.configFile)
	
	// Ensure directory exists
	if err := s.ensureDir(s.configDir); err != nil {
		log.Printf("[AccountStorage] Failed to create config directory %s: %v", s.configDir, err)
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Try to preserve existing primary account
	primaryAccount := ""
	
	// First, try to load existing config to get current primary account
	if existingConfig, err := s.loadConfig(); err == nil && existingConfig.PrimaryAccount != "" {
		// Check if existing primary account is still enabled
		if account, exists := accounts[existingConfig.PrimaryAccount]; exists && account.Enabled {
			primaryAccount = existingConfig.PrimaryAccount
			log.Printf("[AccountStorage] Preserving existing primary account: %s", primaryAccount)
		}
	}

	// If no valid existing primary account, find first enabled account
	if primaryAccount == "" {
		for id, account := range accounts {
			if account.Enabled {
				primaryAccount = id
				log.Printf("[AccountStorage] Setting new primary account: %s", primaryAccount)
				break
			}
		}
	}

	config := MultiAccountConfig{
		Accounts:       accounts,
		PrimaryAccount: primaryAccount,
		Version:        "2.0",
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Printf("[AccountStorage] Failed to marshal config to JSON: %v", err)
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with retry logic
	log.Printf("[AccountStorage] Writing configuration with retry logic")
	if err := s.writeWithRetry(s.configFile, data); err != nil {
		log.Printf("[AccountStorage] Failed to save accounts after retries: %v", err)
		return err
	}
	
	log.Printf("[AccountStorage] Successfully saved accounts to %s", s.configFile)
	return nil
}

// AddAccount adds a new account with a unique ID
func (s *AccountStorage) AddAccount(account *AccountConfig) error {
	log.Printf("[AccountStorage] Adding new account with login %d", account.Login)
	
	accounts, err := s.LoadAccounts()
	if err != nil {
		log.Printf("[AccountStorage] Failed to load existing accounts: %v", err)
		return fmt.Errorf("failed to load existing accounts: %w", err)
	}

	// Generate unique ID if not provided
	if account.ID == "" {
		account.ID = uuid.New().String()
		log.Printf("[AccountStorage] Generated new account ID: %s", account.ID)
	}

	// Ensure ID is unique
	for accounts[account.ID] != nil {
		account.ID = uuid.New().String()
		log.Printf("[AccountStorage] ID collision detected, generated new ID: %s", account.ID)
	}

	// Set default values
	if account.UUID == "" {
		account.UUID = uuid.New().String()
		log.Printf("[AccountStorage] Generated device UUID: %s", account.UUID)
	}
	if account.Name == "" {
		account.Name = fmt.Sprintf("Account %d", account.Login)
		log.Printf("[AccountStorage] Generated account name: %s", account.Name)
	}

	accounts[account.ID] = account
	log.Printf("[AccountStorage] Account %s added to collection, saving to storage", account.ID)
	
	if err := s.SaveAccounts(accounts); err != nil {
		log.Printf("[AccountStorage] Failed to save accounts after adding %s: %v", account.ID, err)
		return err
	}
	
	log.Printf("[AccountStorage] Successfully added account %s", account.ID)
	return nil
}

// RemoveAccount removes an account by ID
func (s *AccountStorage) RemoveAccount(accountID string) error {
	log.Printf("[AccountStorage] Removing account %s", accountID)
	
	accounts, err := s.LoadAccounts()
	if err != nil {
		log.Printf("[AccountStorage] Failed to load existing accounts: %v", err)
		return fmt.Errorf("failed to load existing accounts: %w", err)
	}

	// Check if account exists before removal
	if _, exists := accounts[accountID]; !exists {
		log.Printf("[AccountStorage] Account %s not found for removal", accountID)
		return fmt.Errorf("account %s not found", accountID)
	}

	delete(accounts, accountID)
	log.Printf("[AccountStorage] Account %s removed from collection, saving to storage", accountID)
	
	if err := s.SaveAccounts(accounts); err != nil {
		log.Printf("[AccountStorage] Failed to save accounts after removing %s: %v", accountID, err)
		return err
	}
	
	log.Printf("[AccountStorage] Successfully removed account %s", accountID)
	return nil
}

// validateConfig validates the configuration structure and data
func (s *AccountStorage) validateConfig(config *MultiAccountConfig) error {
	if config.Accounts == nil {
		return fmt.Errorf("accounts map cannot be nil")
	}

	// Validate each account
	for id, account := range config.Accounts {
		if account == nil {
			return fmt.Errorf("account %s cannot be nil", id)
		}
		if account.ID != id {
			return fmt.Errorf("account ID mismatch: key=%s, account.ID=%s", id, account.ID)
		}
		if account.Login == 0 {
			return fmt.Errorf("account %s has invalid login", id)
		}
		if account.UUID == "" {
			return fmt.Errorf("account %s missing UUID", id)
		}
	}

	// Validate primary account exists if specified
	if config.PrimaryAccount != "" {
		if _, exists := config.Accounts[config.PrimaryAccount]; !exists {
			return fmt.Errorf("primary account %s does not exist", config.PrimaryAccount)
		}
	}

	return nil
}

// writeWithRetry writes data to file with retry logic
func (s *AccountStorage) writeWithRetry(filename string, data []byte) error {
	const maxRetries = 3
	const retryDelay = 100 * time.Millisecond

	log.Printf("[AccountStorage] Writing file %s with retry logic (max %d retries)", filename, maxRetries)
	
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := os.WriteFile(filename, data, 0644); err != nil {
			lastErr = err
			log.Printf("[AccountStorage] Write attempt %d failed for %s: %v", i+1, filename, err)
			if i < maxRetries-1 {
				sleepDuration := retryDelay * time.Duration(i+1) // Exponential backoff
				log.Printf("[AccountStorage] Retrying write in %v", sleepDuration)
				time.Sleep(sleepDuration)
			}
			continue
		}
		if i > 0 {
			log.Printf("[AccountStorage] Write succeeded on attempt %d for %s", i+1, filename)
		} else {
			log.Printf("[AccountStorage] Write succeeded on first attempt for %s", filename)
		}
		return nil
	}

	log.Printf("[AccountStorage] All write attempts failed for %s: %v", filename, lastErr)
	return fmt.Errorf("failed to write file after %d retries: %w", maxRetries, lastErr)
}

// ensureDir creates directory if it doesn't exist
func (s *AccountStorage) ensureDir(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		log.Printf("[AccountStorage] Creating directory structure: %s", dirPath)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			log.Printf("[AccountStorage] Failed to create directory %s: %v", dirPath, err)
			return err
		}
		log.Printf("[AccountStorage] Successfully created directory: %s", dirPath)
	} else {
		log.Printf("[AccountStorage] Directory already exists: %s", dirPath)
	}
	return nil
}

// CreateBackup creates a backup of the current configuration
func (s *AccountStorage) CreateBackup() error {
	log.Printf("[AccountStorage] Creating backup of configuration")
	
	if _, err := os.Stat(s.configFile); os.IsNotExist(err) {
		log.Printf("[AccountStorage] No configuration file to backup")
		return nil // No file to backup
	}

	// Ensure backup directory exists
	if err := s.ensureDir(s.backupDir); err != nil {
		log.Printf("[AccountStorage] Failed to create backup directory: %v", err)
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Read current config
	data, err := os.ReadFile(s.configFile)
	if err != nil {
		log.Printf("[AccountStorage] Failed to read config for backup: %v", err)
		return fmt.Errorf("failed to read config for backup: %w", err)
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(s.backupDir, fmt.Sprintf("accounts_%s.json", timestamp))

	// Write backup
	if err := os.WriteFile(backupFile, data, 0644); err != nil {
		log.Printf("[AccountStorage] Failed to write backup file %s: %v", backupFile, err)
		return fmt.Errorf("failed to write backup file: %w", err)
	}
	
	log.Printf("[AccountStorage] Successfully created backup: %s", backupFile)
	return nil
}

// RestoreFromBackup attempts to restore from the most recent backup
func (s *AccountStorage) RestoreFromBackup() error {
	log.Printf("[AccountStorage] Attempting to restore from backup")
	
	// Find most recent backup
	backupFiles, err := filepath.Glob(filepath.Join(s.backupDir, "accounts_*.json"))
	if err != nil || len(backupFiles) == 0 {
		log.Printf("[AccountStorage] No backup files found in %s", s.backupDir)
		return fmt.Errorf("no backup files found")
	}

	// Get the most recent backup (files are sorted by name which includes timestamp)
	var mostRecent string
	for _, file := range backupFiles {
		if file > mostRecent {
			mostRecent = file
		}
	}

	log.Printf("[AccountStorage] Using most recent backup: %s", mostRecent)

	// Read backup
	data, err := os.ReadFile(mostRecent)
	if err != nil {
		log.Printf("[AccountStorage] Failed to read backup file %s: %v", mostRecent, err)
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Validate backup before restoring
	var config MultiAccountConfig
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("[AccountStorage] Backup file %s is corrupted: %v", mostRecent, err)
		return fmt.Errorf("backup file is corrupted: %w", err)
	}

	if err := s.validateConfig(&config); err != nil {
		log.Printf("[AccountStorage] Backup file %s validation failed: %v", mostRecent, err)
		return fmt.Errorf("backup file validation failed: %w", err)
	}

	// Restore backup
	log.Printf("[AccountStorage] Restoring configuration from backup")
	if err := s.writeWithRetry(s.configFile, data); err != nil {
		log.Printf("[AccountStorage] Failed to restore backup: %v", err)
		return err
	}
	
	log.Printf("[AccountStorage] Successfully restored configuration from backup %s", mostRecent)
	return nil
}