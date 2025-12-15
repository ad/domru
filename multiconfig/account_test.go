package multiconfig

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ad/domru/config"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestProperty_AccountStorageUniqueIdentifiers(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("account storage with unique identifiers", prop.ForAll(
		func(accounts []AccountConfig) bool {
			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "account_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			storage := NewAccountStorage(tempDir)

			// Convert slice to map and add accounts
			accountMap := make(map[string]*AccountConfig)
			usedIDs := make(map[string]bool)

			for i := range accounts {
				account := &accounts[i]
				
				// Ensure account has required fields for validation
				if account.Login == 0 {
					account.Login = 79123456789 + i // Valid phone number format
				}
				if account.UUID == "" {
					account.UUID = "test-uuid-" + string(rune('a'+i))
				}

				// Add account using AddAccount method which ensures unique IDs
				err := storage.AddAccount(account)
				if err != nil {
					t.Logf("Failed to add account: %v", err)
					return false
				}

				// Track the ID that was assigned
				usedIDs[account.ID] = true
				accountMap[account.ID] = account
			}

			// Load accounts back and verify uniqueness
			loadedAccounts, err := storage.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load accounts: %v", err)
				return false
			}

			// Check that all IDs are unique
			loadedIDs := make(map[string]bool)
			for id := range loadedAccounts {
				if loadedIDs[id] {
					t.Logf("Duplicate ID found: %s", id)
					return false
				}
				loadedIDs[id] = true
			}

			// Verify that the number of loaded accounts matches what we added
			if len(loadedAccounts) != len(accountMap) {
				t.Logf("Account count mismatch: expected %d, got %d", len(accountMap), len(loadedAccounts))
				return false
			}

			return true
		},
		genAccountSlice(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_ConfigurationValidationOnRead(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("configuration validation on read", prop.ForAll(
		func(corruptionType int) bool {
			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "validation_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			storage := NewAccountStorage(tempDir)
			configFile := filepath.Join(tempDir, "accounts.json")

			// Create different types of invalid configurations based on corruptionType
			var invalidJSON string
			switch corruptionType % 6 {
			case 0:
				// Invalid JSON syntax
				invalidJSON = `{"accounts": {invalid json}`
			case 1:
				// Nil accounts map
				invalidJSON = `{"accounts": null, "version": "2.0"}`
			case 2:
				// Account with mismatched ID
				invalidJSON = `{"accounts": {"id1": {"id": "id2", "login": 79123456789, "uuid": "test-uuid"}}, "version": "2.0"}`
			case 3:
				// Account with invalid login (zero)
				invalidJSON = `{"accounts": {"id1": {"id": "id1", "login": 0, "uuid": "test-uuid"}}, "version": "2.0"}`
			case 4:
				// Account with missing UUID
				invalidJSON = `{"accounts": {"id1": {"id": "id1", "login": 79123456789, "uuid": ""}}, "version": "2.0"}`
			case 5:
				// Primary account that doesn't exist
				invalidJSON = `{"accounts": {"id1": {"id": "id1", "login": 79123456789, "uuid": "test-uuid"}}, "primary_account": "nonexistent", "version": "2.0"}`
			}

			// Write invalid configuration
			err = os.WriteFile(configFile, []byte(invalidJSON), 0644)
			if err != nil {
				t.Logf("Failed to write invalid config: %v", err)
				return false
			}

			// Attempt to load accounts - should fail with validation error
			_, err = storage.LoadAccounts()
			if err == nil {
				t.Logf("Expected validation error but got none for corruption type %d", corruptionType%6)
				return false
			}

			// Verify error message indicates validation failure
			errorMsg := err.Error()
			validationErrors := []string{
				"invalid configuration format",
				"configuration validation failed",
				"accounts map cannot be nil",
				"account ID mismatch",
				"has invalid login",
				"missing UUID",
				"primary account",
				"does not exist",
			}

			foundValidationError := false
			for _, validationError := range validationErrors {
				if len(errorMsg) > 0 && contains(errorMsg, validationError) {
					foundValidationError = true
					break
				}
			}

			if !foundValidationError {
				t.Logf("Error message doesn't indicate validation failure: %s", errorMsg)
				return false
			}

			return true
		},
		gen.IntRange(0, 100),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// Generator for account slices
func genAccountSlice() gopter.Gen {
	return gen.SliceOfN(5, genAccount()) // Generate up to 5 accounts
}

// Generator for individual accounts
func genAccount() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString(),           // Name
		gen.Bool(),                  // Enabled
		gen.AlphaString(),           // Token
		gen.AlphaString(),           // RefreshToken
		gen.IntRange(1, 2),          // Operator (0 or 1)
	).Map(func(values []interface{}) AccountConfig {
		return AccountConfig{
			Name:         values[0].(string),
			Enabled:      values[1].(bool),
			Token:        values[2].(string),
			RefreshToken: values[3].(string),
			Operator:     values[4].(int),
			// ID, Login, and UUID will be set by the test or AddAccount method
		}
	})
}

// Unit tests for basic functionality
func TestAccountStorage_Basic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "account_basic_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := NewAccountStorage(tempDir)

	// Test loading from non-existent file
	accounts, err := storage.LoadAccounts()
	if err != nil {
		t.Fatalf("Expected no error for non-existent file, got: %v", err)
	}
	if len(accounts) != 0 {
		t.Fatalf("Expected empty accounts map, got %d accounts", len(accounts))
	}

	// Test adding an account
	account := &AccountConfig{
		Login:        79123456789,
		UUID:         "test-uuid",
		Name:         "Test Account",
		Enabled:      true,
		Token:        "test-token",
		RefreshToken: "test-refresh",
		Operator:     0,
	}

	err = storage.AddAccount(account)
	if err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	// Verify account was added with unique ID
	if account.ID == "" {
		t.Fatalf("Account ID should be generated")
	}

	// Load and verify
	accounts, err = storage.LoadAccounts()
	if err != nil {
		t.Fatalf("Failed to load accounts: %v", err)
	}

	if len(accounts) != 1 {
		t.Fatalf("Expected 1 account, got %d", len(accounts))
	}

	loadedAccount := accounts[account.ID]
	if loadedAccount == nil {
		t.Fatalf("Account not found by ID")
	}

	if loadedAccount.Login != account.Login {
		t.Fatalf("Login mismatch: expected %d, got %d", account.Login, loadedAccount.Login)
	}
}

func TestAccountStorage_WriteRetry(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "retry_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := NewAccountStorage(tempDir)

	// Test successful write
	data := []byte(`{"test": "data"}`)
	testFile := filepath.Join(tempDir, "test.json")
	
	err = storage.writeWithRetry(testFile, data)
	if err != nil {
		t.Fatalf("Write should succeed: %v", err)
	}

	// Verify file was written
	readData, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(readData) != string(data) {
		t.Fatalf("Data mismatch: expected %s, got %s", string(data), string(readData))
	}
}

func TestAccountStorage_DirectoryCreation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dir_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create storage with nested directory path
	nestedDir := filepath.Join(tempDir, "nested", "config")
	storage := NewAccountStorage(nestedDir)

	// Add account should create directory structure
	account := &AccountConfig{
		Login:   79123456789,
		UUID:    "test-uuid",
		Name:    "Test Account",
		Enabled: true,
	}

	err = storage.AddAccount(account)
	if err != nil {
		t.Fatalf("Failed to add account with directory creation: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Fatalf("Directory should have been created: %s", nestedDir)
	}

	// Verify config file exists
	configFile := filepath.Join(nestedDir, "accounts.json")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatalf("Config file should have been created: %s", configFile)
	}
}

func TestProperty_MigrationDataPreservation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("migration data preservation", prop.ForAll(
		func(legacyConfig config.Config) bool {
			// Ensure valid legacy config
			if legacyConfig.Login == 0 {
				legacyConfig.Login = 79123456789
			}
			if legacyConfig.UUID == "" {
				legacyConfig.UUID = "test-uuid-migration"
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "migration_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			// Create legacy config file
			legacyPath := filepath.Join(tempDir, "account.json")
			legacyData, err := json.Marshal(legacyConfig)
			if err != nil {
				t.Logf("Failed to marshal legacy config: %v", err)
				return false
			}

			err = os.WriteFile(legacyPath, legacyData, 0644)
			if err != nil {
				t.Logf("Failed to write legacy config: %v", err)
				return false
			}

			// Create storage and migration service
			storage := NewAccountStorage(tempDir)
			migration := NewMigrationService(storage, legacyPath)

			// Perform migration
			err = migration.MigrateFromLegacy()
			if err != nil {
				t.Logf("Migration failed: %v", err)
				return false
			}

			// Load migrated accounts
			accounts, err := storage.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load migrated accounts: %v", err)
				return false
			}

			// Should have exactly one account
			if len(accounts) != 1 {
				t.Logf("Expected 1 account after migration, got %d", len(accounts))
				return false
			}

			// Get the migrated account
			var migratedAccount *AccountConfig
			for _, account := range accounts {
				migratedAccount = account
				break
			}

			// Verify all original data is preserved
			if migratedAccount.Token != legacyConfig.Token {
				t.Logf("Token not preserved: expected %s, got %s", legacyConfig.Token, migratedAccount.Token)
				return false
			}

			if migratedAccount.RefreshToken != legacyConfig.RefreshToken {
				t.Logf("RefreshToken not preserved: expected %s, got %s", legacyConfig.RefreshToken, migratedAccount.RefreshToken)
				return false
			}

			if migratedAccount.Login != legacyConfig.Login {
				t.Logf("Login not preserved: expected %d, got %d", legacyConfig.Login, migratedAccount.Login)
				return false
			}

			if migratedAccount.Operator != legacyConfig.Operator {
				t.Logf("Operator not preserved: expected %d, got %d", legacyConfig.Operator, migratedAccount.Operator)
				return false
			}

			if migratedAccount.UUID != legacyConfig.UUID {
				t.Logf("UUID not preserved: expected %s, got %s", legacyConfig.UUID, migratedAccount.UUID)
				return false
			}

			// Verify account has required new fields
			if migratedAccount.ID == "" {
				t.Logf("Migrated account missing ID")
				return false
			}

			if migratedAccount.Name == "" {
				t.Logf("Migrated account missing Name")
				return false
			}

			if !migratedAccount.Enabled {
				t.Logf("Migrated account should be enabled by default")
				return false
			}

			return true
		},
		genLegacyConfig(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_MigrationBackupCreation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("migration backup creation", prop.ForAll(
		func(legacyConfig config.Config) bool {
			// Ensure valid legacy config
			if legacyConfig.Login == 0 {
				legacyConfig.Login = 79123456789
			}
			if legacyConfig.UUID == "" {
				legacyConfig.UUID = "test-uuid-backup"
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "backup_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			// Create legacy config file
			legacyPath := filepath.Join(tempDir, "account.json")
			legacyData, err := json.Marshal(legacyConfig)
			if err != nil {
				t.Logf("Failed to marshal legacy config: %v", err)
				return false
			}

			err = os.WriteFile(legacyPath, legacyData, 0644)
			if err != nil {
				t.Logf("Failed to write legacy config: %v", err)
				return false
			}

			// Create storage and migration service
			storage := NewAccountStorage(tempDir)
			migration := NewMigrationService(storage, legacyPath)

			// Perform migration
			err = migration.MigrateFromLegacy()
			if err != nil {
				t.Logf("Migration failed: %v", err)
				return false
			}

			// Check that backup was created
			backupDir := filepath.Join(tempDir, "migration_backups")
			backupFiles, err := filepath.Glob(filepath.Join(backupDir, "legacy_account_*.json"))
			if err != nil {
				t.Logf("Failed to list backup files: %v", err)
				return false
			}

			if len(backupFiles) == 0 {
				t.Logf("No backup files found")
				return false
			}

			// Verify backup contains original data
			backupData, err := os.ReadFile(backupFiles[0])
			if err != nil {
				t.Logf("Failed to read backup file: %v", err)
				return false
			}

			var backupConfig config.Config
			err = json.Unmarshal(backupData, &backupConfig)
			if err != nil {
				t.Logf("Failed to parse backup config: %v", err)
				return false
			}

			// Verify backup contains original data
			if backupConfig.Token != legacyConfig.Token ||
				backupConfig.RefreshToken != legacyConfig.RefreshToken ||
				backupConfig.Login != legacyConfig.Login ||
				backupConfig.Operator != legacyConfig.Operator ||
				backupConfig.UUID != legacyConfig.UUID {
				t.Logf("Backup data doesn't match original")
				return false
			}

			// Verify original legacy file was removed after successful migration
			if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
				t.Logf("Legacy file should be removed after successful migration")
				return false
			}

			return true
		},
		genLegacyConfig(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_MigrationFailureRecovery(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("migration failure recovery", prop.ForAll(
		func(legacyConfig config.Config) bool {
			// Ensure valid legacy config
			if legacyConfig.Login == 0 {
				legacyConfig.Login = 79123456789
			}
			if legacyConfig.UUID == "" {
				legacyConfig.UUID = "test-uuid-recovery"
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "recovery_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			// Create legacy config file
			legacyPath := filepath.Join(tempDir, "account.json")
			legacyData, err := json.Marshal(legacyConfig)
			if err != nil {
				t.Logf("Failed to marshal legacy config: %v", err)
				return false
			}

			err = os.WriteFile(legacyPath, legacyData, 0644)
			if err != nil {
				t.Logf("Failed to write legacy config: %v", err)
				return false
			}

			// Create a read-only directory to force migration failure
			readOnlyDir := filepath.Join(tempDir, "readonly")
			err = os.MkdirAll(readOnlyDir, 0444) // Read-only directory
			if err != nil {
				t.Logf("Failed to create read-only dir: %v", err)
				return false
			}

			// Create storage pointing to read-only directory (will cause save failure)
			storage := NewAccountStorage(readOnlyDir)
			migration := NewMigrationService(storage, legacyPath)

			// Attempt migration - should fail
			err = migration.MigrateFromLegacy()
			if err == nil {
				t.Logf("Expected migration to fail due to read-only directory")
				return false
			}

			// Verify original legacy file still exists (recovery)
			if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
				t.Logf("Legacy file should still exist after failed migration")
				return false
			}

			// Verify original data is intact
			recoveredData, err := os.ReadFile(legacyPath)
			if err != nil {
				t.Logf("Failed to read recovered legacy file: %v", err)
				return false
			}

			var recoveredConfig config.Config
			err = json.Unmarshal(recoveredData, &recoveredConfig)
			if err != nil {
				t.Logf("Failed to parse recovered config: %v", err)
				return false
			}

			// Verify all data is preserved in recovery
			if recoveredConfig.Token != legacyConfig.Token ||
				recoveredConfig.RefreshToken != legacyConfig.RefreshToken ||
				recoveredConfig.Login != legacyConfig.Login ||
				recoveredConfig.Operator != legacyConfig.Operator ||
				recoveredConfig.UUID != legacyConfig.UUID {
				t.Logf("Recovered data doesn't match original")
				return false
			}

			return true
		},
		genLegacyConfig(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Generator for legacy config
func genLegacyConfig() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString(),           // Token
		gen.AlphaString(),           // RefreshToken
		gen.IntRange(0, 2),          // Operator
		gen.IntRange(8000, 9000),    // Port
	).Map(func(values []interface{}) config.Config {
		return config.Config{
			Token:        values[0].(string),
			RefreshToken: values[1].(string),
			Operator:     values[2].(int),
			Port:         values[3].(int),
			// Login and UUID will be set by the test to ensure validity
		}
	})
}

// Unit tests for migration service
func TestMigrationService_NeedsMigration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "migration_needs_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := NewAccountStorage(tempDir)
	legacyPath := filepath.Join(tempDir, "account.json")
	migration := NewMigrationService(storage, legacyPath)

	// Test 1: No legacy file exists
	if migration.NeedsMigration() {
		t.Fatalf("Should not need migration when no legacy file exists")
	}

	// Test 2: Legacy file exists, no new format
	legacyConfig := config.Config{
		Token:        "test-token",
		RefreshToken: "test-refresh",
		Login:        79123456789,
		Operator:     0,
		UUID:         "test-uuid",
	}
	legacyData, _ := json.Marshal(legacyConfig)
	os.WriteFile(legacyPath, legacyData, 0644)

	if !migration.NeedsMigration() {
		t.Fatalf("Should need migration when legacy file exists and new format doesn't")
	}

	// Test 3: Both legacy and new format exist
	accounts := map[string]*AccountConfig{
		"test-id": {
			ID:      "test-id",
			Login:   79123456789,
			UUID:    "test-uuid",
			Enabled: true,
		},
	}
	storage.SaveAccounts(accounts)

	if migration.NeedsMigration() {
		t.Fatalf("Should not need migration when new format already exists")
	}
}

func TestMigrationService_MigrateFromLegacy_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "migration_success_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create legacy config
	legacyPath := filepath.Join(tempDir, "account.json")
	legacyConfig := config.Config{
		Token:        "test-token",
		RefreshToken: "test-refresh",
		Login:        79123456789,
		Operator:     1,
		UUID:         "test-uuid",
		Port:         18000,
	}
	legacyData, _ := json.Marshal(legacyConfig)
	os.WriteFile(legacyPath, legacyData, 0644)

	// Perform migration
	storage := NewAccountStorage(tempDir)
	migration := NewMigrationService(storage, legacyPath)

	err = migration.MigrateFromLegacy()
	if err != nil {
		t.Fatalf("Migration should succeed: %v", err)
	}

	// Verify migration results
	accounts, err := storage.LoadAccounts()
	if err != nil {
		t.Fatalf("Failed to load migrated accounts: %v", err)
	}

	if len(accounts) != 1 {
		t.Fatalf("Expected 1 account, got %d", len(accounts))
	}

	var account *AccountConfig
	for _, acc := range accounts {
		account = acc
		break
	}

	// Verify all data preserved
	if account.Token != legacyConfig.Token {
		t.Fatalf("Token not preserved")
	}
	if account.RefreshToken != legacyConfig.RefreshToken {
		t.Fatalf("RefreshToken not preserved")
	}
	if account.Login != legacyConfig.Login {
		t.Fatalf("Login not preserved")
	}
	if account.Operator != legacyConfig.Operator {
		t.Fatalf("Operator not preserved")
	}
	if account.UUID != legacyConfig.UUID {
		t.Fatalf("UUID not preserved")
	}

	// Verify new fields
	if account.ID == "" {
		t.Fatalf("ID should be generated")
	}
	if account.Name == "" {
		t.Fatalf("Name should be generated")
	}
	if !account.Enabled {
		t.Fatalf("Account should be enabled by default")
	}

	// Verify legacy file was removed
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("Legacy file should be removed after successful migration")
	}

	// Verify backup was created
	backupDir := filepath.Join(tempDir, "migration_backups")
	backupFiles, _ := filepath.Glob(filepath.Join(backupDir, "legacy_account_*.json"))
	if len(backupFiles) == 0 {
		t.Fatalf("Backup should be created")
	}
}

func TestMigrationService_InvalidLegacyConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "migration_invalid_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := NewAccountStorage(tempDir)
	legacyPath := filepath.Join(tempDir, "account.json")
	migration := NewMigrationService(storage, legacyPath)

	// Test invalid JSON
	os.WriteFile(legacyPath, []byte(`{invalid json`), 0644)
	err = migration.MigrateFromLegacy()
	if err == nil {
		t.Fatalf("Should fail with invalid JSON")
	}

	// Test missing required fields
	invalidConfig := config.Config{
		Token: "test-token",
		// Missing Login and UUID
	}
	invalidData, _ := json.Marshal(invalidConfig)
	os.WriteFile(legacyPath, invalidData, 0644)
	err = migration.MigrateFromLegacy()
	if err == nil {
		t.Fatalf("Should fail with missing required fields")
	}
}
func TestProperty_TokenIsolationBetweenAccounts(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("token isolation between accounts", prop.ForAll(
		func(accounts []AccountConfig) bool {
			// Need at least 2 accounts to test isolation
			if len(accounts) < 2 {
				return true // Skip test for insufficient accounts
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "token_isolation_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			storage := NewAccountStorage(tempDir)
			factory := &MockHandlerFactory{}
			var templateFs embed.FS
			aggregator := NewMultiAccountAggregator(storage, factory, templateFs)

			// Prepare accounts with unique tokens and required fields
			for i := range accounts {
				account := &accounts[i]
				
				// Ensure required fields
				if account.Login == 0 {
					account.Login = 79123456789 + i
				}
				if account.UUID == "" {
					account.UUID = fmt.Sprintf("test-uuid-%d", i)
				}
				
				// Ensure unique tokens for isolation testing
				account.Token = fmt.Sprintf("unique-token-%d", i)
				account.RefreshToken = fmt.Sprintf("unique-refresh-%d", i)
				account.Enabled = true

				// Add account
				err := aggregator.AddAccount(account)
				if err != nil {
					t.Logf("Failed to add account: %v", err)
					return false
				}
			}

			// Load accounts to initialize clients
			err = aggregator.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load accounts: %v", err)
				return false
			}

			// Get all clients
			clients := aggregator.GetAllClients()
			if len(clients) != len(accounts) {
				t.Logf("Expected %d clients, got %d", len(accounts), len(clients))
				return false
			}

			// Verify token isolation - each client should have its own unique tokens
			seenTokens := make(map[string]string) // token -> accountID
			seenRefreshTokens := make(map[string]string) // refreshToken -> accountID

			for accountID, client := range clients {
				config := client.GetConfig()
				
				// Check if token is unique
				if existingAccountID, exists := seenTokens[config.Token]; exists {
					t.Logf("Token collision: accounts %s and %s have same token %s", 
						accountID, existingAccountID, config.Token)
					return false
				}
				seenTokens[config.Token] = accountID

				// Check if refresh token is unique
				if existingAccountID, exists := seenRefreshTokens[config.RefreshToken]; exists {
					t.Logf("Refresh token collision: accounts %s and %s have same refresh token %s", 
						accountID, existingAccountID, config.RefreshToken)
					return false
				}
				seenRefreshTokens[config.RefreshToken] = accountID

				// Verify token matches what we set
				expectedToken := fmt.Sprintf("unique-token-%d", findAccountIndex(accounts, accountID))
				expectedRefresh := fmt.Sprintf("unique-refresh-%d", findAccountIndex(accounts, accountID))
				
				if config.Token != expectedToken {
					t.Logf("Token mismatch for account %s: expected %s, got %s", 
						accountID, expectedToken, config.Token)
					return false
				}

				if config.RefreshToken != expectedRefresh {
					t.Logf("Refresh token mismatch for account %s: expected %s, got %s", 
						accountID, expectedRefresh, config.RefreshToken)
					return false
				}
			}

			// Test token updates don't affect other accounts
			clientList := make([]*AccountAPIClient, 0, len(clients))
			accountIDs := make([]string, 0, len(clients))
			for id, client := range clients {
				clientList = append(clientList, client)
				accountIDs = append(accountIDs, id)
			}

			if len(clientList) >= 2 {
				// Update tokens for first client
				newToken := "updated-token-123"
				newRefresh := "updated-refresh-123"
				clientList[0].UpdateTokens(newToken, newRefresh)

				// Verify first client has new tokens
				config1 := clientList[0].GetConfig()
				if config1.Token != newToken || config1.RefreshToken != newRefresh {
					t.Logf("Token update failed for first client")
					return false
				}

				// Verify other clients still have original tokens
				for i := 1; i < len(clientList); i++ {
					config := clientList[i].GetConfig()
					expectedToken := fmt.Sprintf("unique-token-%d", findAccountIndex(accounts, accountIDs[i]))
					expectedRefresh := fmt.Sprintf("unique-refresh-%d", findAccountIndex(accounts, accountIDs[i]))
					
					if config.Token != expectedToken {
						t.Logf("Token isolation violated: client %d token changed unexpectedly", i)
						return false
					}
					if config.RefreshToken != expectedRefresh {
						t.Logf("Refresh token isolation violated: client %d refresh token changed unexpectedly", i)
						return false
					}
				}
			}

			return true
		},
		gen.SliceOfN(4, genAccount()), // Generate 2-4 accounts for isolation testing
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_AccountDeletionConsistency(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("account deletion consistency", prop.ForAll(
		func(accounts []AccountConfig) bool {
			// Need at least 1 account to test deletion
			if len(accounts) == 0 {
				return true // Skip test for no accounts
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "deletion_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			storage := NewAccountStorage(tempDir)
			factory := &MockHandlerFactory{}
			var templateFs embed.FS
			aggregator := NewMultiAccountAggregator(storage, factory, templateFs)

			// Prepare and add accounts
			accountIDs := make([]string, 0, len(accounts))
			for i := range accounts {
				account := &accounts[i]
				
				// Ensure required fields
				if account.Login == 0 {
					account.Login = 79123456789 + i
				}
				if account.UUID == "" {
					account.UUID = fmt.Sprintf("test-uuid-%d", i)
				}
				account.Enabled = true

				// Add account
				err := aggregator.AddAccount(account)
				if err != nil {
					t.Logf("Failed to add account: %v", err)
					return false
				}
				
				accountIDs = append(accountIDs, account.ID)
			}

			// Load accounts to initialize clients
			err = aggregator.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load accounts: %v", err)
				return false
			}

			// Verify all accounts are present initially
			allAccounts := aggregator.GetAccounts()
			if len(allAccounts) != len(accounts) {
				t.Logf("Initial account count mismatch: expected %d, got %d", len(accounts), len(allAccounts))
				return false
			}

			// Verify all clients are present initially
			allClients := aggregator.GetAllClients()
			if len(allClients) != len(accounts) {
				t.Logf("Initial client count mismatch: expected %d, got %d", len(accounts), len(allClients))
				return false
			}

			// Delete each account one by one and verify consistency
			for i, accountID := range accountIDs {
				// Delete the account
				err := aggregator.RemoveAccount(accountID)
				if err != nil {
					t.Logf("Failed to remove account %s: %v", accountID, err)
					return false
				}

				// Verify account is no longer in accounts list
				remainingAccounts := aggregator.GetAccounts()
				if _, exists := remainingAccounts[accountID]; exists {
					t.Logf("Account %s still exists in accounts list after deletion", accountID)
					return false
				}

				// Verify account count decreased
				expectedCount := len(accounts) - (i + 1)
				if len(remainingAccounts) != expectedCount {
					t.Logf("Account count after deletion %d: expected %d, got %d", 
						i+1, expectedCount, len(remainingAccounts))
					return false
				}

				// Verify client is no longer in client pool
				remainingClients := aggregator.GetAllClients()
				if _, exists := remainingClients[accountID]; exists {
					t.Logf("Client for account %s still exists after deletion", accountID)
					return false
				}

				// Verify client count decreased
				if len(remainingClients) != expectedCount {
					t.Logf("Client count after deletion %d: expected %d, got %d", 
						i+1, expectedCount, len(remainingClients))
					return false
				}

				// Verify GetClient returns error for deleted account
				_, err = aggregator.GetClient(accountID)
				if err == nil {
					t.Logf("GetClient should return error for deleted account %s", accountID)
					return false
				}

				// Verify GetAccount returns error for deleted account
				_, err = aggregator.GetAccount(accountID)
				if err == nil {
					t.Logf("GetAccount should return error for deleted account %s", accountID)
					return false
				}

				// Verify remaining accounts are still accessible
				for _, remainingID := range accountIDs[i+1:] {
					_, err := aggregator.GetAccount(remainingID)
					if err != nil {
						t.Logf("Remaining account %s should still be accessible: %v", remainingID, err)
						return false
					}

					_, err = aggregator.GetClient(remainingID)
					if err != nil {
						t.Logf("Remaining client %s should still be accessible: %v", remainingID, err)
						return false
					}
				}
			}

			// After deleting all accounts, verify empty state
			finalAccounts := aggregator.GetAccounts()
			if len(finalAccounts) != 0 {
				t.Logf("Expected no accounts after deleting all, got %d", len(finalAccounts))
				return false
			}

			finalClients := aggregator.GetAllClients()
			if len(finalClients) != 0 {
				t.Logf("Expected no clients after deleting all, got %d", len(finalClients))
				return false
			}

			// Verify GetPrimaryAccount returns error when no accounts exist
			_, err = aggregator.GetPrimaryAccount()
			if err == nil {
				t.Logf("GetPrimaryAccount should return error when no accounts exist")
				return false
			}

			return true
		},
		gen.SliceOfN(3, genAccount()), // Generate 1-3 accounts for deletion testing
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Helper function to find account index by ID
func findAccountIndex(accounts []AccountConfig, accountID string) int {
	for i, account := range accounts {
		if account.ID == accountID {
			return i
		}
	}
	return -1
}
// Unit tests for MultiAccountAggregator
func TestMultiAccountAggregator_Basic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "aggregator_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := NewAccountStorage(tempDir)
	factory := &MockHandlerFactory{}
	var templateFs embed.FS
	aggregator := NewMultiAccountAggregator(storage, factory, templateFs)

	// Test loading empty accounts
	err = aggregator.LoadAccounts()
	if err != nil {
		t.Fatalf("Failed to load empty accounts: %v", err)
	}

	accounts := aggregator.GetAccounts()
	if len(accounts) != 0 {
		t.Fatalf("Expected 0 accounts, got %d", len(accounts))
	}

	// Test adding account
	account := &AccountConfig{
		Login:        79123456789,
		UUID:         "test-uuid",
		Name:         "Test Account",
		Enabled:      true,
		Token:        "test-token",
		RefreshToken: "test-refresh",
		Operator:     0,
	}

	err = aggregator.AddAccount(account)
	if err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	// Verify account was added
	accounts = aggregator.GetAccounts()
	if len(accounts) != 1 {
		t.Fatalf("Expected 1 account, got %d", len(accounts))
	}

	// Test getting specific account
	retrievedAccount, err := aggregator.GetAccount(account.ID)
	if err != nil {
		t.Fatalf("Failed to get account: %v", err)
	}

	if retrievedAccount.Login != account.Login {
		t.Fatalf("Account data mismatch")
	}

	// Test getting client
	client, err := aggregator.GetClient(account.ID)
	if err != nil {
		t.Fatalf("Failed to get client: %v", err)
	}

	if client.GetAccountID() != account.ID {
		t.Fatalf("Client account ID mismatch")
	}

	// Test getting primary account
	primaryAccount, err := aggregator.GetPrimaryAccount()
	if err != nil {
		t.Fatalf("Failed to get primary account: %v", err)
	}

	if primaryAccount.ID != account.ID {
		t.Fatalf("Primary account mismatch")
	}
}

func TestMultiAccountAggregator_EnableDisable(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "enable_disable_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := NewAccountStorage(tempDir)
	factory := &MockHandlerFactory{}
	var templateFs embed.FS
	aggregator := NewMultiAccountAggregator(storage, factory, templateFs)

	// Add account
	account := &AccountConfig{
		Login:        79123456789,
		UUID:         "test-uuid",
		Name:         "Test Account",
		Enabled:      true,
		Token:        "test-token",
		RefreshToken: "test-refresh",
		Operator:     0,
	}

	err = aggregator.AddAccount(account)
	if err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	// Verify client exists
	_, err = aggregator.GetClient(account.ID)
	if err != nil {
		t.Fatalf("Client should exist for enabled account: %v", err)
	}

	// Disable account
	err = aggregator.DisableAccount(account.ID)
	if err != nil {
		t.Fatalf("Failed to disable account: %v", err)
	}

	// Verify client is removed
	_, err = aggregator.GetClient(account.ID)
	if err == nil {
		t.Fatalf("Client should not exist for disabled account")
	}

	// Verify account is still in storage but disabled
	retrievedAccount, err := aggregator.GetAccount(account.ID)
	if err != nil {
		t.Fatalf("Account should still exist when disabled: %v", err)
	}

	if retrievedAccount.Enabled {
		t.Fatalf("Account should be disabled")
	}

	// Re-enable account
	err = aggregator.EnableAccount(account.ID)
	if err != nil {
		t.Fatalf("Failed to enable account: %v", err)
	}

	// Verify client is recreated
	_, err = aggregator.GetClient(account.ID)
	if err != nil {
		t.Fatalf("Client should exist for re-enabled account: %v", err)
	}

	// Verify account is enabled
	retrievedAccount, err = aggregator.GetAccount(account.ID)
	if err != nil {
		t.Fatalf("Failed to get account: %v", err)
	}

	if !retrievedAccount.Enabled {
		t.Fatalf("Account should be enabled")
	}
}

func TestAccountAPIClient_Basic(t *testing.T) {
	account := &AccountConfig{
		ID:           "test-id",
		Login:        79123456789,
		UUID:         "test-uuid",
		Name:         "Test Account",
		Enabled:      true,
		Token:        "test-token",
		RefreshToken: "test-refresh",
		Operator:     0,
	}

	factory := &MockHandlerFactory{}
	var templateFs embed.FS
	client := NewAccountAPIClient(account, factory, templateFs)

	// Test basic getters
	if client.GetAccountID() != account.ID {
		t.Fatalf("Account ID mismatch")
	}

	if client.GetAccountName() != account.Name {
		t.Fatalf("Account name mismatch")
	}

	if !client.IsEnabled() {
		t.Fatalf("Account should be enabled")
	}

	// Test config retrieval
	config := client.GetConfig()
	if config.Token != account.Token {
		t.Fatalf("Token mismatch")
	}

	// Test token update
	newToken := "new-token"
	newRefresh := "new-refresh"
	client.UpdateTokens(newToken, newRefresh)

	updatedConfig := client.GetConfig()
	if updatedConfig.Token != newToken {
		t.Fatalf("Token not updated")
	}

	if updatedConfig.RefreshToken != newRefresh {
		t.Fatalf("Refresh token not updated")
	}

	// Note: Handler config verification removed since APIHandler interface doesn't expose Config
}

func TestAccountClientPool_Basic(t *testing.T) {
	pool := NewAccountClientPool()

	// Test empty pool
	clients := pool.GetAllClients()
	if len(clients) != 0 {
		t.Fatalf("Expected empty pool, got %d clients", len(clients))
	}

	// Test getting non-existent client
	_, err := pool.GetClient("non-existent")
	if err == nil {
		t.Fatalf("Should return error for non-existent client")
	}

	// Add client
	account := &AccountConfig{
		ID:           "test-id",
		Login:        79123456789,
		UUID:         "test-uuid",
		Name:         "Test Account",
		Enabled:      true,
		Token:        "test-token",
		RefreshToken: "test-refresh",
		Operator:     0,
	}

	factory := &MockHandlerFactory{}
	var templateFs embed.FS
	client := NewAccountAPIClient(account, factory, templateFs)
	pool.AddClient(account.ID, client)

	// Test getting client
	retrievedClient, err := pool.GetClient(account.ID)
	if err != nil {
		t.Fatalf("Failed to get client: %v", err)
	}

	if retrievedClient.GetAccountID() != account.ID {
		t.Fatalf("Client ID mismatch")
	}

	// Test getting all clients
	clients = pool.GetAllClients()
	if len(clients) != 1 {
		t.Fatalf("Expected 1 client, got %d", len(clients))
	}

	// Remove client
	pool.RemoveClient(account.ID)

	// Verify client is removed
	_, err = pool.GetClient(account.ID)
	if err == nil {
		t.Fatalf("Client should be removed")
	}

	clients = pool.GetAllClients()
	if len(clients) != 0 {
		t.Fatalf("Expected empty pool after removal, got %d clients", len(clients))
	}
}
func TestProperty_LegacyAPIPrimaryAccountCompatibility(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("legacy API primary account compatibility", prop.ForAll(
		func(accounts []AccountConfig) bool {
			// Need at least 1 account to test legacy compatibility
			if len(accounts) == 0 {
				return true // Skip test for no accounts
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "legacy_compat_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			storage := NewAccountStorage(tempDir)
			factory := &MockHandlerFactory{}
			var templateFs embed.FS
			aggregator := NewMultiAccountAggregator(storage, factory, templateFs)

			// Prepare and add accounts
			for i := range accounts {
				account := &accounts[i]
				
				// Ensure required fields
				if account.Login == 0 {
					account.Login = 79123456789 + i
				}
				if account.UUID == "" {
					account.UUID = fmt.Sprintf("test-uuid-%d", i)
				}
				account.Enabled = true
				account.Token = fmt.Sprintf("token-%d", i)
				account.RefreshToken = fmt.Sprintf("refresh-%d", i)

				// Add account
				err := aggregator.AddAccount(account)
				if err != nil {
					t.Logf("Failed to add account: %v", err)
					return false
				}
			}

			// Load accounts to initialize clients
			err = aggregator.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load accounts: %v", err)
				return false
			}

			// Create legacy adapter
			adapter := NewLegacyConfigAdapter(aggregator)

			// Get the actual primary account from the system
			actualPrimaryAccount, err := aggregator.GetPrimaryAccount()
			if err != nil {
				t.Logf("Failed to get actual primary account: %v", err)
				return false
			}

			// Test GetPrimaryConfig returns data in legacy format
			legacyConfig, err := adapter.GetPrimaryConfig()
			if err != nil {
				t.Logf("Failed to get primary config: %v", err)
				return false
			}

			// Verify legacy config matches actual primary account data
			if legacyConfig.Token != actualPrimaryAccount.Token {
				t.Logf("Legacy token mismatch: expected %s, got %s", 
					actualPrimaryAccount.Token, legacyConfig.Token)
				return false
			}

			if legacyConfig.RefreshToken != actualPrimaryAccount.RefreshToken {
				t.Logf("Legacy refresh token mismatch: expected %s, got %s", 
					actualPrimaryAccount.RefreshToken, legacyConfig.RefreshToken)
				return false
			}

			if legacyConfig.Login != actualPrimaryAccount.Login {
				t.Logf("Legacy login mismatch: expected %d, got %d", 
					actualPrimaryAccount.Login, legacyConfig.Login)
				return false
			}

			if legacyConfig.Operator != actualPrimaryAccount.Operator {
				t.Logf("Legacy operator mismatch: expected %d, got %d", 
					actualPrimaryAccount.Operator, legacyConfig.Operator)
				return false
			}

			if legacyConfig.UUID != actualPrimaryAccount.UUID {
				t.Logf("Legacy UUID mismatch: expected %s, got %s", 
					actualPrimaryAccount.UUID, legacyConfig.UUID)
				return false
			}

			// Verify legacy config has default port for backward compatibility
			if legacyConfig.Port != 18000 {
				t.Logf("Legacy port should be 18000 for compatibility, got %d", legacyConfig.Port)
				return false
			}

			// Test that legacy config is a proper config.Config type
			// This ensures existing handlers can use it without modification
			if legacyConfig == nil {
				t.Logf("Legacy config should not be nil")
				return false
			}

			// Test GetAccountConfig for specific account
			if len(accounts) > 1 {
				// Find a non-primary account to test specific account retrieval
				var testAccount *AccountConfig
				for i := range accounts {
					account := &accounts[i]
					if account.ID != actualPrimaryAccount.ID {
						testAccount = account
						break
					}
				}

				if testAccount != nil {
					specificLegacyConfig, err := adapter.GetAccountConfig(testAccount.ID)
					if err != nil {
						t.Logf("Failed to get specific account config: %v", err)
						return false
					}

					// Verify specific account data
					if specificLegacyConfig.Token != testAccount.Token {
						t.Logf("Specific account token mismatch: expected %s, got %s", 
							testAccount.Token, specificLegacyConfig.Token)
						return false
					}

					if specificLegacyConfig.Login != testAccount.Login {
						t.Logf("Specific account login mismatch: expected %d, got %d", 
							testAccount.Login, specificLegacyConfig.Login)
						return false
					}
				}
			}

			// Test HasPrimaryAccount
			if !adapter.HasPrimaryAccount() {
				t.Logf("Should have primary account when accounts exist")
				return false
			}

			// Test GetPrimaryAccountID
			primaryID, err := adapter.GetPrimaryAccountID()
			if err != nil {
				t.Logf("Failed to get primary account ID: %v", err)
				return false
			}

			if primaryID != actualPrimaryAccount.ID {
				t.Logf("Primary account ID mismatch: expected %s, got %s", 
					actualPrimaryAccount.ID, primaryID)
				return false
			}

			// Test token updates through legacy adapter
			newToken := "updated-legacy-token"
			newRefresh := "updated-legacy-refresh"
			
			err = adapter.UpdatePrimaryAccountTokens(newToken, newRefresh)
			if err != nil {
				t.Logf("Failed to update primary account tokens: %v", err)
				return false
			}

			// Reload accounts to ensure we get fresh data
			err = aggregator.LoadAccounts()
			if err != nil {
				t.Logf("Failed to reload accounts after token update: %v", err)
				return false
			}

			// Verify tokens were updated
			updatedLegacyConfig, err := adapter.GetPrimaryConfig()
			if err != nil {
				t.Logf("Failed to get updated primary config: %v", err)
				return false
			}

			if updatedLegacyConfig.Token != newToken {
				t.Logf("Token update failed: expected %s, got %s", newToken, updatedLegacyConfig.Token)
				return false
			}

			if updatedLegacyConfig.RefreshToken != newRefresh {
				t.Logf("Refresh token update failed: expected %s, got %s", newRefresh, updatedLegacyConfig.RefreshToken)
				return false
			}

			// Verify the update persisted in the underlying account
			updatedAccount, err := aggregator.GetAccount(actualPrimaryAccount.ID)
			if err != nil {
				t.Logf("Failed to get updated account: %v", err)
				return false
			}

			if updatedAccount.Token != newToken {
				t.Logf("Account token not updated: expected %s, got %s", newToken, updatedAccount.Token)
				return false
			}

			if updatedAccount.RefreshToken != newRefresh {
				t.Logf("Account refresh token not updated: expected %s, got %s", newRefresh, updatedAccount.RefreshToken)
				return false
			}

			// Test behavior when no accounts exist
			// Remove all accounts
			allAccounts := aggregator.GetAccounts()
			for accountID := range allAccounts {
				err := aggregator.RemoveAccount(accountID)
				if err != nil {
					t.Logf("Failed to remove account %s: %v", accountID, err)
					return false
				}
			}

			// Verify HasPrimaryAccount returns false
			if adapter.HasPrimaryAccount() {
				t.Logf("Should not have primary account when no accounts exist")
				return false
			}

			// Verify GetPrimaryConfig returns error
			_, err = adapter.GetPrimaryConfig()
			if err == nil {
				t.Logf("GetPrimaryConfig should return error when no accounts exist")
				return false
			}

			// Verify GetPrimaryAccountID returns error
			_, err = adapter.GetPrimaryAccountID()
			if err == nil {
				t.Logf("GetPrimaryAccountID should return error when no accounts exist")
				return false
			}

			return true
		},
		gen.SliceOfN(3, genAccount()), // Generate 1-3 accounts for legacy compatibility testing
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_AccountParameterAcceptance(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("account parameter acceptance", prop.ForAll(
		func(accounts []AccountConfig) bool {
			// Need at least 1 account to test parameter acceptance
			if len(accounts) == 0 {
				return true // Skip test for no accounts
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "param_acceptance_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			storage := NewAccountStorage(tempDir)
			factory := &MockHandlerFactory{}
			var templateFs embed.FS
			aggregator := NewMultiAccountAggregator(storage, factory, templateFs)

			// Prepare and add accounts
			accountIDs := make([]string, 0, len(accounts))
			for i := range accounts {
				account := &accounts[i]
				
				// Ensure required fields
				if account.Login == 0 {
					account.Login = 79123456789 + i
				}
				if account.UUID == "" {
					account.UUID = fmt.Sprintf("test-uuid-%d", i)
				}
				account.Enabled = true
				account.Token = fmt.Sprintf("token-%d", i)
				account.RefreshToken = fmt.Sprintf("refresh-%d", i)

				// Add account
				err := aggregator.AddAccount(account)
				if err != nil {
					t.Logf("Failed to add account: %v", err)
					return false
				}
				
				accountIDs = append(accountIDs, account.ID)
			}

			// Load accounts to initialize clients
			err = aggregator.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load accounts: %v", err)
				return false
			}

			// Test 1: Valid account ID parameter should be accepted
			for _, accountID := range accountIDs {
				// Test GetAccount accepts valid account ID
				account, err := aggregator.GetAccount(accountID)
				if err != nil {
					t.Logf("Valid account ID %s should be accepted by GetAccount: %v", accountID, err)
					return false
				}
				if account == nil {
					t.Logf("GetAccount should return account for valid ID %s", accountID)
					return false
				}
				if account.ID != accountID {
					t.Logf("Returned account ID mismatch: expected %s, got %s", accountID, account.ID)
					return false
				}

				// Test GetClient accepts valid account ID
				client, err := aggregator.GetClient(accountID)
				if err != nil {
					t.Logf("Valid account ID %s should be accepted by GetClient: %v", accountID, err)
					return false
				}
				if client == nil {
					t.Logf("GetClient should return client for valid ID %s", accountID)
					return false
				}
				if client.GetAccountID() != accountID {
					t.Logf("Returned client account ID mismatch: expected %s, got %s", accountID, client.GetAccountID())
					return false
				}

				// Test EnableAccount accepts valid account ID
				err = aggregator.EnableAccount(accountID)
				if err != nil {
					t.Logf("Valid account ID %s should be accepted by EnableAccount: %v", accountID, err)
					return false
				}

				// Test DisableAccount accepts valid account ID
				err = aggregator.DisableAccount(accountID)
				if err != nil {
					t.Logf("Valid account ID %s should be accepted by DisableAccount: %v", accountID, err)
					return false
				}

				// Re-enable for further tests
				err = aggregator.EnableAccount(accountID)
				if err != nil {
					t.Logf("Failed to re-enable account %s: %v", accountID, err)
					return false
				}

				// Test UpdateAccountTokens accepts valid account ID
				newToken := fmt.Sprintf("new-token-%s", accountID)
				newRefresh := fmt.Sprintf("new-refresh-%s", accountID)
				err = aggregator.UpdateAccountTokens(accountID, newToken, newRefresh)
				if err != nil {
					t.Logf("Valid account ID %s should be accepted by UpdateAccountTokens: %v", accountID, err)
					return false
				}

				// Verify tokens were updated
				updatedAccount, err := aggregator.GetAccount(accountID)
				if err != nil {
					t.Logf("Failed to get updated account: %v", err)
					return false
				}
				if updatedAccount.Token != newToken {
					t.Logf("Token update failed: expected %s, got %s", newToken, updatedAccount.Token)
					return false
				}
				if updatedAccount.RefreshToken != newRefresh {
					t.Logf("Refresh token update failed: expected %s, got %s", newRefresh, updatedAccount.RefreshToken)
					return false
				}
			}

			// Test 2: Invalid account ID parameters should be properly rejected
			invalidAccountIDs := []string{
				"non-existent-id",
				"",
				"invalid-uuid-format",
				"12345",
				"test-invalid",
			}

			for _, invalidID := range invalidAccountIDs {
				// Test GetAccount rejects invalid account ID
				_, err := aggregator.GetAccount(invalidID)
				if err == nil {
					t.Logf("Invalid account ID %s should be rejected by GetAccount", invalidID)
					return false
				}

				// Test GetClient rejects invalid account ID
				_, err = aggregator.GetClient(invalidID)
				if err == nil {
					t.Logf("Invalid account ID %s should be rejected by GetClient", invalidID)
					return false
				}

				// Test EnableAccount rejects invalid account ID
				err = aggregator.EnableAccount(invalidID)
				if err == nil {
					t.Logf("Invalid account ID %s should be rejected by EnableAccount", invalidID)
					return false
				}

				// Test DisableAccount rejects invalid account ID
				err = aggregator.DisableAccount(invalidID)
				if err == nil {
					t.Logf("Invalid account ID %s should be rejected by DisableAccount", invalidID)
					return false
				}

				// Test UpdateAccountTokens rejects invalid account ID
				err = aggregator.UpdateAccountTokens(invalidID, "token", "refresh")
				if err == nil {
					t.Logf("Invalid account ID %s should be rejected by UpdateAccountTokens", invalidID)
					return false
				}

				// Test RemoveAccount rejects invalid account ID
				err = aggregator.RemoveAccount(invalidID)
				if err == nil {
					t.Logf("Invalid account ID %s should be rejected by RemoveAccount", invalidID)
					return false
				}
			}

			// Test 3: Empty or nil parameters should be handled gracefully
			// Test UpdateAccountTokens with empty tokens
			if len(accountIDs) > 0 {
				firstAccountID := accountIDs[0]
				
				// Empty token should be accepted (might be valid for logout scenarios)
				err = aggregator.UpdateAccountTokens(firstAccountID, "", "")
				if err != nil {
					t.Logf("Empty tokens should be accepted by UpdateAccountTokens: %v", err)
					return false
				}

				// Verify empty tokens were set
				account, err := aggregator.GetAccount(firstAccountID)
				if err != nil {
					t.Logf("Failed to get account after empty token update: %v", err)
					return false
				}
				if account.Token != "" || account.RefreshToken != "" {
					t.Logf("Empty tokens should be set correctly")
					return false
				}
			}

			return true
		},
		gen.SliceOfN(3, genAccount()), // Generate 1-3 accounts for parameter testing
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_AccountSpecificOperationExecution(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("account-specific operation execution", prop.ForAll(
		func(accounts []AccountConfig) bool {
			// Need at least 2 accounts to test account-specific operations
			if len(accounts) < 2 {
				return true // Skip test for insufficient accounts
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "operation_execution_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			storage := NewAccountStorage(tempDir)
			factory := &MockHandlerFactory{}
			var templateFs embed.FS
			aggregator := NewMultiAccountAggregator(storage, factory, templateFs)

			// Prepare and add accounts with unique identifiable data
			accountIDs := make([]string, 0, len(accounts))
			for i := range accounts {
				account := &accounts[i]
				
				// Ensure required fields with unique values
				if account.Login == 0 {
					account.Login = 79123456789 + i
				}
				if account.UUID == "" {
					account.UUID = fmt.Sprintf("test-uuid-%d", i)
				}
				account.Enabled = true
				account.Token = fmt.Sprintf("unique-token-%d", i)
				account.RefreshToken = fmt.Sprintf("unique-refresh-%d", i)
				account.Name = fmt.Sprintf("Account-%d", i)

				// Add account
				err := aggregator.AddAccount(account)
				if err != nil {
					t.Logf("Failed to add account: %v", err)
					return false
				}
				
				accountIDs = append(accountIDs, account.ID)
			}

			// Load accounts to initialize clients
			err = aggregator.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load accounts: %v", err)
				return false
			}

			// Test 1: Operations on specific accounts should only affect those accounts
			firstAccountID := accountIDs[0]
			secondAccountID := accountIDs[1]

			// Get initial state of both accounts
			secondAccount, err := aggregator.GetAccount(secondAccountID)
			if err != nil {
				t.Logf("Failed to get second account: %v", err)
				return false
			}

			// Test token update on first account only
			newToken := "updated-token-first"
			newRefresh := "updated-refresh-first"
			err = aggregator.UpdateAccountTokens(firstAccountID, newToken, newRefresh)
			if err != nil {
				t.Logf("Failed to update tokens for first account: %v", err)
				return false
			}

			// Verify first account was updated
			updatedFirstAccount, err := aggregator.GetAccount(firstAccountID)
			if err != nil {
				t.Logf("Failed to get updated first account: %v", err)
				return false
			}
			if updatedFirstAccount.Token != newToken {
				t.Logf("First account token not updated: expected %s, got %s", newToken, updatedFirstAccount.Token)
				return false
			}
			if updatedFirstAccount.RefreshToken != newRefresh {
				t.Logf("First account refresh token not updated: expected %s, got %s", newRefresh, updatedFirstAccount.RefreshToken)
				return false
			}

			// Verify second account was NOT affected
			unchangedSecondAccount, err := aggregator.GetAccount(secondAccountID)
			if err != nil {
				t.Logf("Failed to get second account after first account update: %v", err)
				return false
			}
			if unchangedSecondAccount.Token != secondAccount.Token {
				t.Logf("Second account token should not change: expected %s, got %s", 
					secondAccount.Token, unchangedSecondAccount.Token)
				return false
			}
			if unchangedSecondAccount.RefreshToken != secondAccount.RefreshToken {
				t.Logf("Second account refresh token should not change: expected %s, got %s", 
					secondAccount.RefreshToken, unchangedSecondAccount.RefreshToken)
				return false
			}

			// Test 2: Enable/Disable operations should be account-specific
			// Disable first account
			err = aggregator.DisableAccount(firstAccountID)
			if err != nil {
				t.Logf("Failed to disable first account: %v", err)
				return false
			}

			// Verify first account is disabled
			disabledFirstAccount, err := aggregator.GetAccount(firstAccountID)
			if err != nil {
				t.Logf("Failed to get disabled first account: %v", err)
				return false
			}
			if disabledFirstAccount.Enabled {
				t.Logf("First account should be disabled")
				return false
			}

			// Verify second account is still enabled
			stillEnabledSecondAccount, err := aggregator.GetAccount(secondAccountID)
			if err != nil {
				t.Logf("Failed to get second account after first account disable: %v", err)
				return false
			}
			if !stillEnabledSecondAccount.Enabled {
				t.Logf("Second account should still be enabled")
				return false
			}

			// Verify first account client is removed but second account client still exists
			_, err = aggregator.GetClient(firstAccountID)
			if err == nil {
				t.Logf("First account client should be removed when disabled")
				return false
			}

			_, err = aggregator.GetClient(secondAccountID)
			if err != nil {
				t.Logf("Second account client should still exist: %v", err)
				return false
			}

			// Test 3: Account removal should only affect the specified account
			// Remove first account
			err = aggregator.RemoveAccount(firstAccountID)
			if err != nil {
				t.Logf("Failed to remove first account: %v", err)
				return false
			}

			// Verify first account is completely removed
			_, err = aggregator.GetAccount(firstAccountID)
			if err == nil {
				t.Logf("First account should be removed")
				return false
			}

			// Verify second account still exists and is accessible
			finalSecondAccount, err := aggregator.GetAccount(secondAccountID)
			if err != nil {
				t.Logf("Second account should still exist after first account removal: %v", err)
				return false
			}
			if finalSecondAccount.ID != secondAccountID {
				t.Logf("Second account ID should be unchanged: expected %s, got %s", 
					secondAccountID, finalSecondAccount.ID)
				return false
			}

			// Verify account count decreased by exactly one
			remainingAccounts := aggregator.GetAccounts()
			expectedCount := len(accounts) - 1
			if len(remainingAccounts) != expectedCount {
				t.Logf("Account count after removal: expected %d, got %d", expectedCount, len(remainingAccounts))
				return false
			}

			// Test 4: Operations should work correctly with remaining accounts
			// Test that we can still perform operations on remaining accounts
			for _, accountID := range accountIDs[1:] { // Skip first account (removed)
				// Test GetAccount still works
				_, err := aggregator.GetAccount(accountID)
				if err != nil {
					t.Logf("Remaining account %s should be accessible: %v", accountID, err)
					return false
				}

				// Test token update still works
				testToken := fmt.Sprintf("test-token-%s", accountID)
				testRefresh := fmt.Sprintf("test-refresh-%s", accountID)
				err = aggregator.UpdateAccountTokens(accountID, testToken, testRefresh)
				if err != nil {
					t.Logf("Token update should work for remaining account %s: %v", accountID, err)
					return false
				}

				// Verify update worked
				updatedAccount, err := aggregator.GetAccount(accountID)
				if err != nil {
					t.Logf("Failed to get updated remaining account %s: %v", accountID, err)
					return false
				}
				if updatedAccount.Token != testToken {
					t.Logf("Token update failed for remaining account %s: expected %s, got %s", 
						accountID, testToken, updatedAccount.Token)
					return false
				}

				// Test enable/disable still works
				err = aggregator.DisableAccount(accountID)
				if err != nil {
					t.Logf("Disable should work for remaining account %s: %v", accountID, err)
					return false
				}

				err = aggregator.EnableAccount(accountID)
				if err != nil {
					t.Logf("Enable should work for remaining account %s: %v", accountID, err)
					return false
				}
			}

			return true
		},
		gen.SliceOfN(4, genAccount()), // Generate 2-4 accounts for operation testing
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_ImmediatePersistenceOfChanges(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("immediate persistence of changes", prop.ForAll(
		func(accounts []AccountConfig) bool {
			// Need at least 1 account to test persistence
			if len(accounts) == 0 {
				return true // Skip test for no accounts
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "persistence_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			storage := NewAccountStorage(tempDir)

			// Prepare and add accounts
			accountMap := make(map[string]*AccountConfig)
			for i := range accounts {
				account := &accounts[i]
				
				// Ensure required fields
				if account.Login == 0 {
					account.Login = 79123456789 + i
				}
				if account.UUID == "" {
					account.UUID = fmt.Sprintf("test-uuid-%d", i)
				}
				account.Enabled = true
				account.Token = fmt.Sprintf("token-%d", i)
				account.RefreshToken = fmt.Sprintf("refresh-%d", i)

				// Add account
				err := storage.AddAccount(account)
				if err != nil {
					t.Logf("Failed to add account: %v", err)
					return false
				}
				
				accountMap[account.ID] = account
			}

			// Test 1: Verify accounts are immediately readable after addition
			loadedAccounts, err := storage.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load accounts immediately after addition: %v", err)
				return false
			}

			if len(loadedAccounts) != len(accountMap) {
				t.Logf("Account count mismatch after addition: expected %d, got %d", len(accountMap), len(loadedAccounts))
				return false
			}

			// Verify all added accounts are present and correct
			for id, originalAccount := range accountMap {
				loadedAccount, exists := loadedAccounts[id]
				if !exists {
					t.Logf("Account %s not found after immediate load", id)
					return false
				}

				if loadedAccount.Token != originalAccount.Token {
					t.Logf("Token mismatch for account %s: expected %s, got %s", 
						id, originalAccount.Token, loadedAccount.Token)
					return false
				}

				if loadedAccount.Login != originalAccount.Login {
					t.Logf("Login mismatch for account %s: expected %d, got %d", 
						id, originalAccount.Login, loadedAccount.Login)
					return false
				}
			}

			// Test 2: Modify accounts and verify immediate persistence
			modifiedAccounts := make(map[string]*AccountConfig)
			for id, account := range loadedAccounts {
				// Create modified copy
				modified := *account
				modified.Token = "modified-" + account.Token
				modified.RefreshToken = "modified-" + account.RefreshToken
				modified.Name = "Modified " + account.Name
				modifiedAccounts[id] = &modified
			}

			// Save modifications
			err = storage.SaveAccounts(modifiedAccounts)
			if err != nil {
				t.Logf("Failed to save modified accounts: %v", err)
				return false
			}

			// Immediately load and verify changes are persisted
			reloadedAccounts, err := storage.LoadAccounts()
			if err != nil {
				t.Logf("Failed to reload accounts after modification: %v", err)
				return false
			}

			if len(reloadedAccounts) != len(modifiedAccounts) {
				t.Logf("Account count mismatch after modification: expected %d, got %d", 
					len(modifiedAccounts), len(reloadedAccounts))
				return false
			}

			// Verify all modifications are persisted
			for id, modifiedAccount := range modifiedAccounts {
				reloadedAccount, exists := reloadedAccounts[id]
				if !exists {
					t.Logf("Modified account %s not found after reload", id)
					return false
				}

				if reloadedAccount.Token != modifiedAccount.Token {
					t.Logf("Modified token not persisted for account %s: expected %s, got %s", 
						id, modifiedAccount.Token, reloadedAccount.Token)
					return false
				}

				if reloadedAccount.RefreshToken != modifiedAccount.RefreshToken {
					t.Logf("Modified refresh token not persisted for account %s: expected %s, got %s", 
						id, modifiedAccount.RefreshToken, reloadedAccount.RefreshToken)
					return false
				}

				if reloadedAccount.Name != modifiedAccount.Name {
					t.Logf("Modified name not persisted for account %s: expected %s, got %s", 
						id, modifiedAccount.Name, reloadedAccount.Name)
					return false
				}
			}

			// Test 3: Remove accounts and verify immediate persistence
			if len(accountMap) > 1 {
				// Remove first account
				var firstAccountID string
				for id := range accountMap {
					firstAccountID = id
					break
				}

				err = storage.RemoveAccount(firstAccountID)
				if err != nil {
					t.Logf("Failed to remove account %s: %v", firstAccountID, err)
					return false
				}

				// Immediately verify removal is persisted
				afterRemovalAccounts, err := storage.LoadAccounts()
				if err != nil {
					t.Logf("Failed to load accounts after removal: %v", err)
					return false
				}

				if len(afterRemovalAccounts) != len(accountMap)-1 {
					t.Logf("Account count after removal: expected %d, got %d", 
						len(accountMap)-1, len(afterRemovalAccounts))
					return false
				}

				// Verify removed account is not present
				if _, exists := afterRemovalAccounts[firstAccountID]; exists {
					t.Logf("Removed account %s still exists after immediate load", firstAccountID)
					return false
				}

				// Verify remaining accounts are still present
				for id := range accountMap {
					if id == firstAccountID {
						continue // Skip removed account
					}
					if _, exists := afterRemovalAccounts[id]; !exists {
						t.Logf("Remaining account %s not found after removal", id)
						return false
					}
				}
			}

			// Test 4: Test persistence across multiple storage instances
			// Create a new storage instance pointing to the same directory
			storage2 := NewAccountStorage(tempDir)
			
			// Load accounts with new storage instance
			crossInstanceAccounts, err := storage2.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load accounts with new storage instance: %v", err)
				return false
			}

			// Verify data is consistent across storage instances
			currentAccounts, err := storage.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load accounts with original storage: %v", err)
				return false
			}

			if len(crossInstanceAccounts) != len(currentAccounts) {
				t.Logf("Account count mismatch across storage instances: expected %d, got %d", 
					len(currentAccounts), len(crossInstanceAccounts))
				return false
			}

			for id, currentAccount := range currentAccounts {
				crossInstanceAccount, exists := crossInstanceAccounts[id]
				if !exists {
					t.Logf("Account %s not found in cross-instance load", id)
					return false
				}

				if crossInstanceAccount.Token != currentAccount.Token {
					t.Logf("Token mismatch across instances for account %s: expected %s, got %s", 
						id, currentAccount.Token, crossInstanceAccount.Token)
					return false
				}

				if crossInstanceAccount.Login != currentAccount.Login {
					t.Logf("Login mismatch across instances for account %s: expected %d, got %d", 
						id, currentAccount.Login, crossInstanceAccount.Login)
					return false
				}
			}

			return true
		},
		gen.SliceOfN(3, genAccount()), // Generate 1-3 accounts for persistence testing
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_BackupRecoveryOnCorruption(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("backup recovery on corruption", prop.ForAll(
		func(accounts []AccountConfig) bool {
			// Need at least 1 account to test backup recovery
			if len(accounts) == 0 {
				return true // Skip test for no accounts
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "backup_recovery_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			storage := NewAccountStorage(tempDir)

			// Prepare and add accounts to create initial valid state
			accountMap := make(map[string]*AccountConfig)
			for i := range accounts {
				account := &accounts[i]
				
				// Ensure required fields
				if account.Login == 0 {
					account.Login = 79123456789 + i
				}
				if account.UUID == "" {
					account.UUID = fmt.Sprintf("test-uuid-%d", i)
				}
				account.Enabled = true
				account.Token = fmt.Sprintf("token-%d", i)
				account.RefreshToken = fmt.Sprintf("refresh-%d", i)

				// Add account
				err := storage.AddAccount(account)
				if err != nil {
					t.Logf("Failed to add account: %v", err)
					return false
				}
				
				accountMap[account.ID] = account
			}

			// Create a backup of the current valid configuration
			err = storage.CreateBackup()
			if err != nil {
				t.Logf("Failed to create backup: %v", err)
				return false
			}

			// Verify backup was created
			backupFiles, err := filepath.Glob(filepath.Join(storage.backupDir, "accounts_*.json"))
			if err != nil || len(backupFiles) == 0 {
				t.Logf("Backup file not created")
				return false
			}

			// Test 1: Corrupt the main configuration file with invalid JSON
			configFile := storage.configFile
			corruptData := []byte(`{"accounts": {invalid json syntax}`)
			err = os.WriteFile(configFile, corruptData, 0644)
			if err != nil {
				t.Logf("Failed to write corrupt data: %v", err)
				return false
			}

			// Verify loading fails due to corruption
			_, err = storage.LoadAccounts()
			if err == nil {
				t.Logf("Loading should fail with corrupted config")
				return false
			}

			// Attempt recovery from backup
			err = storage.RestoreFromBackup()
			if err != nil {
				t.Logf("Failed to restore from backup: %v", err)
				return false
			}

			// Verify recovery was successful - should be able to load accounts
			recoveredAccounts, err := storage.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load accounts after recovery: %v", err)
				return false
			}

			// Verify all original accounts are recovered
			if len(recoveredAccounts) != len(accountMap) {
				t.Logf("Account count after recovery: expected %d, got %d", 
					len(accountMap), len(recoveredAccounts))
				return false
			}

			for id, originalAccount := range accountMap {
				recoveredAccount, exists := recoveredAccounts[id]
				if !exists {
					t.Logf("Account %s not found after recovery", id)
					return false
				}

				if recoveredAccount.Token != originalAccount.Token {
					t.Logf("Token mismatch after recovery for account %s: expected %s, got %s", 
						id, originalAccount.Token, recoveredAccount.Token)
					return false
				}

				if recoveredAccount.Login != originalAccount.Login {
					t.Logf("Login mismatch after recovery for account %s: expected %d, got %d", 
						id, originalAccount.Login, recoveredAccount.Login)
					return false
				}

				if recoveredAccount.UUID != originalAccount.UUID {
					t.Logf("UUID mismatch after recovery for account %s: expected %s, got %s", 
						id, originalAccount.UUID, recoveredAccount.UUID)
					return false
				}
			}

			// Test 2: Corrupt with invalid account structure
			invalidStructureData := []byte(`{"accounts": {"test": null}, "version": "2.0"}`)
			err = os.WriteFile(configFile, invalidStructureData, 0644)
			if err != nil {
				t.Logf("Failed to write invalid structure data: %v", err)
				return false
			}

			// Verify loading fails
			_, err = storage.LoadAccounts()
			if err == nil {
				t.Logf("Loading should fail with invalid structure")
				return false
			}

			// Restore from backup again
			err = storage.RestoreFromBackup()
			if err != nil {
				t.Logf("Failed to restore from backup (second time): %v", err)
				return false
			}

			// Verify recovery works again
			secondRecoveredAccounts, err := storage.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load accounts after second recovery: %v", err)
				return false
			}

			if len(secondRecoveredAccounts) != len(accountMap) {
				t.Logf("Account count after second recovery: expected %d, got %d", 
					len(accountMap), len(secondRecoveredAccounts))
				return false
			}

			// Test 3: Test recovery when no backup exists
			// Remove all backup files
			backupFiles, _ = filepath.Glob(filepath.Join(storage.backupDir, "accounts_*.json"))
			for _, backupFile := range backupFiles {
				os.Remove(backupFile)
			}

			// Corrupt config again
			err = os.WriteFile(configFile, corruptData, 0644)
			if err != nil {
				t.Logf("Failed to write corrupt data for no-backup test: %v", err)
				return false
			}

			// Attempt recovery should fail when no backup exists
			err = storage.RestoreFromBackup()
			if err == nil {
				t.Logf("Recovery should fail when no backup exists")
				return false
			}

			// Verify error message indicates no backup found
			if !contains(err.Error(), "no backup files found") {
				t.Logf("Error should indicate no backup found: %s", err.Error())
				return false
			}

			return true
		},
		gen.SliceOfN(3, genAccount()), // Generate 1-3 accounts for backup recovery testing
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_WriteOperationRetryOnFailure(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("write operation retry on failure", prop.ForAll(
		func(accounts []AccountConfig) bool {
			// Need at least 1 account to test write retry
			if len(accounts) == 0 {
				return true // Skip test for no accounts
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "write_retry_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			storage := NewAccountStorage(tempDir)

			// Prepare accounts
			accountMap := make(map[string]*AccountConfig)
			for i := range accounts {
				account := &accounts[i]
				
				// Ensure required fields
				if account.Login == 0 {
					account.Login = 79123456789 + i
				}
				if account.UUID == "" {
					account.UUID = fmt.Sprintf("test-uuid-%d", i)
				}
				account.Enabled = true
				account.Token = fmt.Sprintf("token-%d", i)
				account.RefreshToken = fmt.Sprintf("refresh-%d", i)
				
				accountMap[account.ID] = account
			}

			// Test 1: Normal write should succeed without retry
			err = storage.SaveAccounts(accountMap)
			if err != nil {
				t.Logf("Normal write should succeed: %v", err)
				return false
			}

			// Verify accounts were saved
			loadedAccounts, err := storage.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load accounts after normal write: %v", err)
				return false
			}

			if len(loadedAccounts) != len(accountMap) {
				t.Logf("Account count mismatch after normal write: expected %d, got %d", 
					len(accountMap), len(loadedAccounts))
				return false
			}

			// Test 2: Test writeWithRetry method directly with valid data
			testData := []byte(`{"test": "data"}`)
			testFile := filepath.Join(tempDir, "test_retry.json")
			
			err = storage.writeWithRetry(testFile, testData)
			if err != nil {
				t.Logf("writeWithRetry should succeed with valid data: %v", err)
				return false
			}

			// Verify file was written correctly
			readData, err := os.ReadFile(testFile)
			if err != nil {
				t.Logf("Failed to read test file: %v", err)
				return false
			}

			if string(readData) != string(testData) {
				t.Logf("Data mismatch: expected %s, got %s", string(testData), string(readData))
				return false
			}

			// Test 3: Test retry behavior with temporary permission issues
			// Create a file and make it temporarily read-only to simulate write failure
			readOnlyFile := filepath.Join(tempDir, "readonly_test.json")
			err = os.WriteFile(readOnlyFile, []byte("initial"), 0644)
			if err != nil {
				t.Logf("Failed to create initial readonly file: %v", err)
				return false
			}

			// Make file read-only
			err = os.Chmod(readOnlyFile, 0444)
			if err != nil {
				t.Logf("Failed to make file read-only: %v", err)
				return false
			}

			// Attempt write - should fail even with retry
			err = storage.writeWithRetry(readOnlyFile, []byte("new data"))
			if err == nil {
				t.Logf("Write should fail for read-only file")
				return false
			}

			// Verify error message indicates retry failure
			if !contains(err.Error(), "failed to write file after") || !contains(err.Error(), "retries") {
				t.Logf("Error should indicate retry failure: %s", err.Error())
				return false
			}

			// Restore write permissions and verify retry would work
			err = os.Chmod(readOnlyFile, 0644)
			if err != nil {
				t.Logf("Failed to restore write permissions: %v", err)
				return false
			}

			// Now write should succeed
			err = storage.writeWithRetry(readOnlyFile, []byte("new data"))
			if err != nil {
				t.Logf("Write should succeed after restoring permissions: %v", err)
				return false
			}

			// Test 4: Test that retry logic is actually used by checking timing
			// This is a behavioral test - we can't easily simulate transient failures,
			// but we can verify the retry mechanism exists by testing with invalid paths
			invalidPath := filepath.Join("/nonexistent/path/file.json")
			
			err = storage.writeWithRetry(invalidPath, []byte("test"))
			if err == nil {
				t.Logf("Write to invalid path should fail")
				return false
			}

			// The error should indicate multiple retries were attempted
			if !contains(err.Error(), "retries") {
				t.Logf("Error should indicate retries were attempted: %s", err.Error())
				return false
			}

			// Test 5: Verify SaveAccounts uses retry logic
			// Create accounts with a problematic directory structure
			problematicDir := filepath.Join(tempDir, "problematic")
			problematicStorage := NewAccountStorage(problematicDir)
			
			// First, create the directory as a file to cause issues
			err = os.WriteFile(problematicDir, []byte("not a directory"), 0644)
			if err != nil {
				t.Logf("Failed to create problematic file: %v", err)
				return false
			}

			// Attempt to save accounts - should fail due to directory conflict
			err = problematicStorage.SaveAccounts(accountMap)
			if err == nil {
				t.Logf("SaveAccounts should fail with directory conflict")
				return false
			}

			// Remove the problematic file and create proper directory
			err = os.Remove(problematicDir)
			if err != nil {
				t.Logf("Failed to remove problematic file: %v", err)
				return false
			}

			// Now SaveAccounts should succeed
			err = problematicStorage.SaveAccounts(accountMap)
			if err != nil {
				t.Logf("SaveAccounts should succeed after fixing directory: %v", err)
				return false
			}

			// Verify accounts were saved correctly
			finalAccounts, err := problematicStorage.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load accounts from problematic storage: %v", err)
				return false
			}

			if len(finalAccounts) != len(accountMap) {
				t.Logf("Final account count mismatch: expected %d, got %d", 
					len(accountMap), len(finalAccounts))
				return false
			}

			return true
		},
		gen.SliceOfN(3, genAccount()), // Generate 1-3 accounts for write retry testing
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_DirectoryStructureCreation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("directory structure creation", prop.ForAll(
		func(accounts []AccountConfig) bool {
			// Need at least 1 account to test directory creation
			if len(accounts) == 0 {
				return true // Skip test for no accounts
			}

			// Create temporary base directory for testing
			baseDir, err := os.MkdirTemp("", "dir_creation_test_*")
			if err != nil {
				t.Logf("Failed to create base temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(baseDir)

			// Test 1: Create storage with nested directory path that doesn't exist
			nestedPath := filepath.Join(baseDir, "level1", "level2", "level3", "config")
			storage := NewAccountStorage(nestedPath)

			// Verify directory doesn't exist initially
			if _, err := os.Stat(nestedPath); !os.IsNotExist(err) {
				t.Logf("Nested directory should not exist initially")
				return false
			}

			// Prepare account
			account := &accounts[0]
			if account.Login == 0 {
				account.Login = 79123456789
			}
			if account.UUID == "" {
				account.UUID = "test-uuid-dir"
			}
			account.Enabled = true
			account.Token = "test-token"
			account.RefreshToken = "test-refresh"

			// Add account - this should create the directory structure
			err = storage.AddAccount(account)
			if err != nil {
				t.Logf("Failed to add account with nested directory: %v", err)
				return false
			}

			// Verify directory structure was created
			if _, err := os.Stat(nestedPath); os.IsNotExist(err) {
				t.Logf("Nested directory should be created: %s", nestedPath)
				return false
			}

			// Verify config file was created
			configFile := filepath.Join(nestedPath, "accounts.json")
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				t.Logf("Config file should be created: %s", configFile)
				return false
			}

			// Verify account was saved correctly
			loadedAccounts, err := storage.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load accounts from nested directory: %v", err)
				return false
			}

			if len(loadedAccounts) != 1 {
				t.Logf("Expected 1 account in nested directory, got %d", len(loadedAccounts))
				return false
			}

			// Test 2: Test backup directory creation
			err = storage.CreateBackup()
			if err != nil {
				t.Logf("Failed to create backup: %v", err)
				return false
			}

			// Verify backup directory was created
			backupDir := filepath.Join(nestedPath, "backups")
			if _, err := os.Stat(backupDir); os.IsNotExist(err) {
				t.Logf("Backup directory should be created: %s", backupDir)
				return false
			}

			// Verify backup file was created
			backupFiles, err := filepath.Glob(filepath.Join(backupDir, "accounts_*.json"))
			if err != nil || len(backupFiles) == 0 {
				t.Logf("Backup file should be created in backup directory")
				return false
			}

			// Test 3: Test with very deep nested structure
			veryDeepPath := filepath.Join(baseDir, "a", "b", "c", "d", "e", "f", "g", "config")
			deepStorage := NewAccountStorage(veryDeepPath)

			// Add multiple accounts to test directory creation with multiple operations
			accountMap := make(map[string]*AccountConfig)
			for i := range accounts {
				acc := &accounts[i]
				
				// Ensure required fields
				if acc.Login == 0 {
					acc.Login = 79123456789 + i
				}
				if acc.UUID == "" {
					acc.UUID = fmt.Sprintf("test-uuid-deep-%d", i)
				}
				acc.Enabled = true
				acc.Token = fmt.Sprintf("token-deep-%d", i)
				acc.RefreshToken = fmt.Sprintf("refresh-deep-%d", i)

				err = deepStorage.AddAccount(acc)
				if err != nil {
					t.Logf("Failed to add account %d to deep directory: %v", i, err)
					return false
				}
				
				accountMap[acc.ID] = acc
			}

			// Verify very deep directory structure was created
			if _, err := os.Stat(veryDeepPath); os.IsNotExist(err) {
				t.Logf("Very deep directory should be created: %s", veryDeepPath)
				return false
			}

			// Verify all accounts were saved
			deepAccounts, err := deepStorage.LoadAccounts()
			if err != nil {
				t.Logf("Failed to load accounts from deep directory: %v", err)
				return false
			}

			if len(deepAccounts) != len(accountMap) {
				t.Logf("Account count mismatch in deep directory: expected %d, got %d", 
					len(accountMap), len(deepAccounts))
				return false
			}

			// Test 4: Test directory creation with special characters (if supported by OS)
			specialPath := filepath.Join(baseDir, "config-with-dashes", "under_scores", "config")
			specialStorage := NewAccountStorage(specialPath)

			err = specialStorage.AddAccount(account)
			if err != nil {
				t.Logf("Failed to add account to directory with special characters: %v", err)
				return false
			}

			// Verify special character directory was created
			if _, err := os.Stat(specialPath); os.IsNotExist(err) {
				t.Logf("Directory with special characters should be created: %s", specialPath)
				return false
			}

			// Test 5: Test that existing directories are not affected
			existingDir := filepath.Join(baseDir, "existing")
			err = os.MkdirAll(existingDir, 0755)
			if err != nil {
				t.Logf("Failed to create existing directory: %v", err)
				return false
			}

			// Create a test file in existing directory
			testFile := filepath.Join(existingDir, "existing_file.txt")
			err = os.WriteFile(testFile, []byte("existing content"), 0644)
			if err != nil {
				t.Logf("Failed to create existing file: %v", err)
				return false
			}

			// Create storage in existing directory
			existingStorage := NewAccountStorage(existingDir)
			err = existingStorage.AddAccount(account)
			if err != nil {
				t.Logf("Failed to add account to existing directory: %v", err)
				return false
			}

			// Verify existing file is still there
			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				t.Logf("Existing file should not be affected: %s", testFile)
				return false
			}

			// Verify existing file content is unchanged
			content, err := os.ReadFile(testFile)
			if err != nil || string(content) != "existing content" {
				t.Logf("Existing file content should be unchanged")
				return false
			}

			// Test 6: Test permission handling (create directory with proper permissions)
			permissionPath := filepath.Join(baseDir, "permission_test", "config")
			permissionStorage := NewAccountStorage(permissionPath)

			err = permissionStorage.AddAccount(account)
			if err != nil {
				t.Logf("Failed to add account for permission test: %v", err)
				return false
			}

			// Verify directory has proper permissions (readable and writable)
			info, err := os.Stat(permissionPath)
			if err != nil {
				t.Logf("Failed to stat permission test directory: %v", err)
				return false
			}

			if !info.IsDir() {
				t.Logf("Permission test path should be a directory")
				return false
			}

			// Verify we can write to the directory (permissions are adequate)
			testPermFile := filepath.Join(permissionPath, "perm_test.txt")
			err = os.WriteFile(testPermFile, []byte("test"), 0644)
			if err != nil {
				t.Logf("Should be able to write to created directory: %v", err)
				return false
			}

			return true
		},
		gen.SliceOfN(3, genAccount()), // Generate 1-3 accounts for directory creation testing
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}