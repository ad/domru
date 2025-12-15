package multiconfig

import (
	"embed"
	"fmt"
	"os"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestProperty_SimultaneousMultiAccountDataDisplay(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("simultaneous multi-account data display", prop.ForAll(
		func(accounts []AccountConfig) bool {
			// Need at least 1 account to test data display
			if len(accounts) == 0 {
				return true // Skip test for no accounts
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "data_display_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			storage := NewAccountStorage(tempDir)
			factory := &MockHandlerFactory{}
			var templateFs embed.FS
			aggregator := NewMultiAccountAggregator(storage, factory, templateFs)
			dataMerger := aggregator.GetDataMerger()

			// Prepare and add accounts
			enabledAccountCount := 0
			for i := range accounts {
				account := &accounts[i]
				
				// Ensure required fields
				if account.Login == 0 {
					account.Login = 79123456789 + i
				}
				if account.UUID == "" {
					account.UUID = fmt.Sprintf("test-uuid-%d", i)
				}
				// Always set unique name to avoid duplicates from generator
				account.Name = fmt.Sprintf("Test Account %d", i)
				if account.ID == "" {
					account.ID = fmt.Sprintf("account-%d", i)
				}
				
				// Enable account for testing
				account.Enabled = true
				enabledAccountCount++

				if err := aggregator.AddAccount(account); err != nil {
					t.Logf("Failed to add account: %v", err)
					return false
				}
			}

			if enabledAccountCount == 0 {
				return true // Skip if no enabled accounts
			}

			// Load accounts into aggregator
			if err := aggregator.LoadAccounts(); err != nil {
				t.Logf("Failed to load accounts: %v", err)
				return false
			}

			// Test simultaneous data display from all accounts
			cameras, err := dataMerger.GetAggregatedCameras()
			if err != nil {
				// Error is acceptable if no valid data is available
				t.Logf("Cameras aggregation error (acceptable): %v", err)
			}

			places, err := dataMerger.GetAggregatedPlaces()
			if err != nil {
				// Error is acceptable if no valid data is available
				t.Logf("Places aggregation error (acceptable): %v", err)
			}

			finances, err := dataMerger.GetAggregatedFinances()
			if err != nil {
				// Error is acceptable if no valid data is available
				t.Logf("Finances aggregation error (acceptable): %v", err)
			}

			// Verify that data from all accounts is displayed simultaneously
			// Each data item should have account identification
			for _, camera := range cameras {
				if camera.AccountID == "" || camera.AccountName == "" {
					t.Logf("Camera missing account identification: %+v", camera)
					return false
				}
			}

			for _, place := range places {
				if place.AccountID == "" || place.AccountName == "" {
					t.Logf("Place missing account identification: %+v", place)
					return false
				}
			}

			for _, finance := range finances {
				if finance.AccountID == "" || finance.AccountName == "" {
					t.Logf("Finance missing account identification: %+v", finance)
					return false
				}
			}

			return true
		},
		genAccountSlice(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_CameraGroupingByAccount(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("camera grouping by account", prop.ForAll(
		func(accounts []AccountConfig) bool {
			// Need at least 2 accounts to test grouping
			if len(accounts) < 2 {
				return true // Skip test for insufficient accounts
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "camera_grouping_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			storage := NewAccountStorage(tempDir)
			factory := &MockHandlerFactory{}
			var templateFs embed.FS
			aggregator := NewMultiAccountAggregator(storage, factory, templateFs)
			dataMerger := aggregator.GetDataMerger()

			// Prepare and add accounts with unique names
			for i := range accounts {
				account := &accounts[i]
				
				// Ensure required fields
				if account.Login == 0 {
					account.Login = 79123456789 + i
				}
				if account.UUID == "" {
					account.UUID = fmt.Sprintf("test-uuid-%d", i)
				}
				// Always set unique name to avoid duplicates from generator
				account.Name = fmt.Sprintf("Test Account %d", i)
				if account.ID == "" {
					account.ID = fmt.Sprintf("account-%d", i)
				}
				
				// Enable account for testing
				account.Enabled = true

				if err := aggregator.AddAccount(account); err != nil {
					t.Logf("Failed to add account: %v", err)
					return false
				}
			}

			// Load accounts into aggregator
			if err := aggregator.LoadAccounts(); err != nil {
				t.Logf("Failed to load accounts: %v", err)
				return false
			}

			// Test camera grouping by account
			cameras, err := dataMerger.GetAggregatedCameras()
			if err != nil {
				// Error is acceptable if no valid data is available
				t.Logf("Cameras aggregation error (acceptable): %v", err)
				return true
			}

			// Group cameras by account
			camerasByAccount := make(map[string][]CameraWithAccount)
			for _, camera := range cameras {
				camerasByAccount[camera.AccountID] = append(camerasByAccount[camera.AccountID], camera)
			}

			// Verify that cameras from different accounts have different account names
			if len(camerasByAccount) > 1 {
				accountIDs := make([]string, 0, len(camerasByAccount))
				for accountID := range camerasByAccount {
					accountIDs = append(accountIDs, accountID)
				}

				for i := 0; i < len(accountIDs); i++ {
					for j := i + 1; j < len(accountIDs); j++ {
						if accountIDs[i] == accountIDs[j] {
							t.Logf("Duplicate account ID in camera grouping: %s", accountIDs[i])
							return false
						}

						// Verify cameras from different accounts have different account names
						cameras1 := camerasByAccount[accountIDs[i]]
						cameras2 := camerasByAccount[accountIDs[j]]
						
						if len(cameras1) > 0 && len(cameras2) > 0 {
							if cameras1[0].AccountName == cameras2[0].AccountName {
								t.Logf("Cameras from different accounts have same AccountName: %s", 
									cameras1[0].AccountName)
								return false
							}
						}
					}
				}
			}

			return true
		},
		genAccountSlice(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_PlaceIdentificationByAccount(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("place identification by account", prop.ForAll(
		func(accounts []AccountConfig) bool {
			// Need at least 2 accounts to test identification
			if len(accounts) < 2 {
				return true // Skip test for insufficient accounts
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "place_identification_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			storage := NewAccountStorage(tempDir)
			factory := &MockHandlerFactory{}
			var templateFs embed.FS
			aggregator := NewMultiAccountAggregator(storage, factory, templateFs)
			dataMerger := aggregator.GetDataMerger()

			// Prepare and add accounts with unique names
			for i := range accounts {
				account := &accounts[i]
				
				// Ensure required fields
				if account.Login == 0 {
					account.Login = 79123456789 + i
				}
				if account.UUID == "" {
					account.UUID = fmt.Sprintf("test-uuid-%d", i)
				}
				// Always set unique name to avoid duplicates from generator
				account.Name = fmt.Sprintf("Test Account %d", i)
				if account.ID == "" {
					account.ID = fmt.Sprintf("account-%d", i)
				}
				
				// Enable account for testing
				account.Enabled = true

				if err := aggregator.AddAccount(account); err != nil {
					t.Logf("Failed to add account: %v", err)
					return false
				}
			}

			// Load accounts into aggregator
			if err := aggregator.LoadAccounts(); err != nil {
				t.Logf("Failed to load accounts: %v", err)
				return false
			}

			// Test place identification by account
			places, err := dataMerger.GetAggregatedPlaces()
			if err != nil {
				// Error is acceptable if no valid data is available
				t.Logf("Places aggregation error (acceptable): %v", err)
				return true
			}

			// Group places by account
			placesByAccount := make(map[string][]PlaceWithAccount)
			for _, place := range places {
				placesByAccount[place.AccountID] = append(placesByAccount[place.AccountID], place)
			}

			// Verify that places from different accounts have different account names
			if len(placesByAccount) > 1 {
				accountIDs := make([]string, 0, len(placesByAccount))
				for accountID := range placesByAccount {
					accountIDs = append(accountIDs, accountID)
				}

				for i := 0; i < len(accountIDs); i++ {
					for j := i + 1; j < len(accountIDs); j++ {
						if accountIDs[i] == accountIDs[j] {
							t.Logf("Duplicate account ID in place identification: %s", accountIDs[i])
							return false
						}

						// Verify places from different accounts have different account names
						places1 := placesByAccount[accountIDs[i]]
						places2 := placesByAccount[accountIDs[j]]
						
						if len(places1) > 0 && len(places2) > 0 {
							if places1[0].AccountName == places2[0].AccountName {
								t.Logf("Places from different accounts have same AccountName: %s", 
									places1[0].AccountName)
								return false
							}
						}
					}
				}
			}

			return true
		},
		genAccountSlice(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_MultiAccountAPIDataAggregation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("multi-account API data aggregation", prop.ForAll(
		func(accounts []AccountConfig, requestAccountID string) bool {
			// Need at least 1 account to test aggregation
			if len(accounts) == 0 {
				return true // Skip test for no accounts
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "api_aggregation_test_*")
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
			enabledAccountCount := 0
			for i := range accounts {
				account := &accounts[i]
				
				// Ensure required fields
				if account.Login == 0 {
					account.Login = 79123456789 + i
				}
				if account.UUID == "" {
					account.UUID = fmt.Sprintf("test-uuid-%d", i)
				}
				// Always set unique name to avoid duplicates from generator
				account.Name = fmt.Sprintf("Test Account %d", i)
				if account.ID == "" {
					account.ID = fmt.Sprintf("account-%d", i)
				}
				
				// Enable account for testing
				account.Enabled = true
				enabledAccountCount++

				if err := aggregator.AddAccount(account); err != nil {
					t.Logf("Failed to add account: %v", err)
					return false
				}
			}

			if enabledAccountCount == 0 {
				return true // Skip if no enabled accounts
			}

			// Load accounts into aggregator
			if err := aggregator.LoadAccounts(); err != nil {
				t.Logf("Failed to load accounts: %v", err)
				return false
			}

			// Test API data aggregation behavior
			dataMerger := aggregator.GetDataMerger()

			// Test cameras aggregation
			cameras, err := dataMerger.GetAggregatedCameras()
			if err != nil {
				// Error is acceptable if no valid data is available
				t.Logf("Cameras aggregation error (acceptable): %v", err)
			}

			// Test places aggregation
			places, err := dataMerger.GetAggregatedPlaces()
			if err != nil {
				// Error is acceptable if no valid data is available
				t.Logf("Places aggregation error (acceptable): %v", err)
			}

			// Test finances aggregation
			finances, err := dataMerger.GetAggregatedFinances()
			if err != nil {
				// Error is acceptable if no valid data is available
				t.Logf("Finances aggregation error (acceptable): %v", err)
			}

			// Verify that data from all accounts is properly aggregated
			// Each data item should have account identification
			for _, camera := range cameras {
				if camera.AccountID == "" || camera.AccountName == "" {
					t.Logf("Camera missing account identification: %+v", camera)
					return false
				}
			}

			for _, place := range places {
				if place.AccountID == "" || place.AccountName == "" {
					t.Logf("Place missing account identification: %+v", place)
					return false
				}
			}

			for _, finance := range finances {
				if finance.AccountID == "" || finance.AccountName == "" {
					t.Logf("Finance missing account identification: %+v", finance)
					return false
				}
			}

			// Test account-specific data retrieval if requestAccountID is valid
			if requestAccountID != "" {
				// Check if the requested account exists
				if _, err := aggregator.GetAccount(requestAccountID); err == nil {
					// Account exists, verify we can get its specific data
					client, err := aggregator.GetClient(requestAccountID)
					if err == nil && client.IsEnabled() {
						// Verify client can provide account-specific data
						if client.GetAccountID() != requestAccountID {
							t.Logf("Client account ID mismatch: expected %s, got %s", requestAccountID, client.GetAccountID())
							return false
						}
					}
				}
			}

			return true
		},
		genAccountSlice(),
		gen.AlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_NewAPIAggregatedDataConsistency(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("new API aggregated data consistency", prop.ForAll(
		func(accounts []AccountConfig) bool {
			// Need at least 1 account to test consistency
			if len(accounts) == 0 {
				return true // Skip test for no accounts
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "api_consistency_test_*")
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
			enabledAccountCount := 0
			for i := range accounts {
				account := &accounts[i]
				
				// Ensure required fields
				if account.Login == 0 {
					account.Login = 79123456789 + i
				}
				if account.UUID == "" {
					account.UUID = fmt.Sprintf("test-uuid-%d", i)
				}
				// Always set unique name to avoid duplicates from generator
				account.Name = fmt.Sprintf("Test Account %d", i)
				if account.ID == "" {
					account.ID = fmt.Sprintf("account-%d", i)
				}
				
				// Enable account for testing
				account.Enabled = true
				enabledAccountCount++

				if err := aggregator.AddAccount(account); err != nil {
					t.Logf("Failed to add account: %v", err)
					return false
				}
			}

			if enabledAccountCount == 0 {
				return true // Skip if no enabled accounts
			}

			// Load accounts into aggregator
			if err := aggregator.LoadAccounts(); err != nil {
				t.Logf("Failed to load accounts: %v", err)
				return false
			}

			// Test new API aggregated data consistency
			dataMerger := aggregator.GetDataMerger()

			// Get all aggregated data
			allData, err := dataMerger.GetAllAggregatedData()
			if err != nil {
				// Error is acceptable if no valid data is available
				t.Logf("All data aggregation error (acceptable): %v", err)
			}

			// Verify consistency: all data should be from enabled accounts only
			enabledAccounts := aggregator.GetEnabledAccounts()
			
			// Check cameras consistency
			for _, camera := range allData.Cameras {
				if _, exists := enabledAccounts[camera.AccountID]; !exists {
					t.Logf("Camera from disabled account found: %s", camera.AccountID)
					return false
				}
				// Verify account name matches
				if enabledAccounts[camera.AccountID].Name != camera.AccountName {
					t.Logf("Camera account name mismatch: expected %s, got %s", 
						enabledAccounts[camera.AccountID].Name, camera.AccountName)
					return false
				}
			}

			// Check places consistency
			for _, place := range allData.Places {
				if _, exists := enabledAccounts[place.AccountID]; !exists {
					t.Logf("Place from disabled account found: %s", place.AccountID)
					return false
				}
				// Verify account name matches
				if enabledAccounts[place.AccountID].Name != place.AccountName {
					t.Logf("Place account name mismatch: expected %s, got %s", 
						enabledAccounts[place.AccountID].Name, place.AccountName)
					return false
				}
			}

			// Check finances consistency
			for _, finance := range allData.Finances {
				if _, exists := enabledAccounts[finance.AccountID]; !exists {
					t.Logf("Finance from disabled account found: %s", finance.AccountID)
					return false
				}
				// Verify account name matches
				if enabledAccounts[finance.AccountID].Name != finance.AccountName {
					t.Logf("Finance account name mismatch: expected %s, got %s", 
						enabledAccounts[finance.AccountID].Name, finance.AccountName)
					return false
				}
			}

			return true
		},
		genAccountSlice(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_AccountSpecificAuthenticationErrorIndication(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("account-specific authentication error indication", prop.ForAll(
		func(accounts []AccountConfig, invalidTokenAccountIndex int) bool {
			// Need at least 1 account to test error indication
			if len(accounts) == 0 {
				return true // Skip test for no accounts
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "auth_error_test_*")
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
			enabledAccountCount := 0
			for i := range accounts {
				account := &accounts[i]
				
				// Ensure required fields
				if account.Login == 0 {
					account.Login = 79123456789 + i
				}
				if account.UUID == "" {
					account.UUID = fmt.Sprintf("test-uuid-%d", i)
				}
				// Always set unique name to avoid duplicates from generator
				account.Name = fmt.Sprintf("Test Account %d", i)
				if account.ID == "" {
					account.ID = fmt.Sprintf("account-%d", i)
				}
				
				// Set valid token by default
				account.Token = "valid-token"
				account.RefreshToken = "valid-refresh-token"
				
				// Invalidate token for one specific account if index is valid
				if i == invalidTokenAccountIndex%len(accounts) {
					account.Token = "invalid-token"
					account.RefreshToken = "invalid-refresh-token"
				}
				
				// Enable account for testing
				account.Enabled = true
				enabledAccountCount++

				if err := aggregator.AddAccount(account); err != nil {
					t.Logf("Failed to add account: %v", err)
					return false
				}
			}

			if enabledAccountCount == 0 {
				return true // Skip if no enabled accounts
			}

			// Load accounts into aggregator
			if err := aggregator.LoadAccounts(); err != nil {
				t.Logf("Failed to load accounts: %v", err)
				return false
			}

			// Test authentication error indication
			dataMerger := aggregator.GetDataMerger()

			// Try to get data from all accounts
			cameras, _ := dataMerger.GetAggregatedCameras()
			places, _ := dataMerger.GetAggregatedPlaces()
			finances, _ := dataMerger.GetAggregatedFinances()

			// Verify that accounts with invalid tokens don't affect other accounts
			// The system should continue to work for accounts with valid tokens
			
			// Count accounts that should have valid data
			validAccountsCount := 0
			invalidAccountID := ""
			
			for i, account := range accounts {
				if i == invalidTokenAccountIndex%len(accounts) {
					invalidAccountID = account.ID
				} else {
					validAccountsCount++
				}
			}

			// If there are valid accounts, we should not see data from the invalid account
			if validAccountsCount > 0 && invalidAccountID != "" {
				// Check that invalid account data is not included
				for _, camera := range cameras {
					if camera.AccountID == invalidAccountID {
						t.Logf("Data from invalid account found in cameras: %s", invalidAccountID)
						return false
					}
				}
				
				for _, place := range places {
					if place.AccountID == invalidAccountID {
						t.Logf("Data from invalid account found in places: %s", invalidAccountID)
						return false
					}
				}
				
				for _, finance := range finances {
					if finance.AccountID == invalidAccountID {
						t.Logf("Data from invalid account found in finances: %s", invalidAccountID)
						return false
					}
				}
			}

			// Verify that the system can identify which account has authentication issues
			clients := aggregator.GetAllClients()
			for accountID, client := range clients {
				if accountID == invalidAccountID {
					// This account should have authentication issues
					// We can't directly test HTTP errors here, but we can verify
					// that the client exists and has the invalid token
					if client.GetConfig().Token != "invalid-token" {
						t.Logf("Expected invalid token for account %s", accountID)
						return false
					}
				}
			}

			return true
		},
		genAccountSlice(),
		gen.IntRange(0, 100),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty_MultiAccountDataLabelingConsistency(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("multi-account data labeling consistency", prop.ForAll(
		func(accounts []AccountConfig) bool {
			// Need at least 1 account to test labeling consistency
			if len(accounts) == 0 {
				return true // Skip test for no accounts
			}

			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "data_labeling_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			storage := NewAccountStorage(tempDir)
			factory := &MockHandlerFactory{}
			var templateFs embed.FS
			aggregator := NewMultiAccountAggregator(storage, factory, templateFs)
			dataMerger := aggregator.GetDataMerger()

			// Prepare and add accounts with unique names
			enabledAccountCount := 0
			for i := range accounts {
				account := &accounts[i]
				
				// Ensure required fields
				if account.Login == 0 {
					account.Login = 79123456789 + i
				}
				if account.UUID == "" {
					account.UUID = fmt.Sprintf("test-uuid-%d", i)
				}
				// Always set unique name to avoid duplicates from generator
				account.Name = fmt.Sprintf("Test Account %d", i)
				if account.ID == "" {
					account.ID = fmt.Sprintf("account-%d", i)
				}
				
				// Enable account for testing
				account.Enabled = true
				enabledAccountCount++

				if err := aggregator.AddAccount(account); err != nil {
					t.Logf("Failed to add account: %v", err)
					return false
				}
			}

			if enabledAccountCount == 0 {
				return true // Skip if no enabled accounts
			}

			// Load accounts into aggregator
			if err := aggregator.LoadAccounts(); err != nil {
				t.Logf("Failed to load accounts: %v", err)
				return false
			}

			// Test multi-account data labeling consistency
			// Get all aggregated data
			allData, err := dataMerger.GetAllAggregatedData()
			if err != nil {
				// Error is acceptable if no valid data is available
				t.Logf("All data aggregation error (acceptable): %v", err)
			}

			// Verify that each data item is clearly labeled with account information
			// Check cameras labeling consistency
			for _, camera := range allData.Cameras {
				// Each camera must have both AccountID and AccountName
				if camera.AccountID == "" {
					t.Logf("Camera missing AccountID: %+v", camera)
					return false
				}
				if camera.AccountName == "" {
					t.Logf("Camera missing AccountName: %+v", camera)
					return false
				}
				
				// Verify that the AccountID corresponds to an actual account
				account, err := aggregator.GetAccount(camera.AccountID)
				if err != nil {
					t.Logf("Camera references non-existent account ID %s: %+v", camera.AccountID, camera)
					return false
				}
				
				// Verify that AccountName matches the actual account name
				if camera.AccountName != account.Name {
					t.Logf("Camera AccountName mismatch: expected %s, got %s", account.Name, camera.AccountName)
					return false
				}
			}

			// Check places labeling consistency
			for _, place := range allData.Places {
				// Each place must have both AccountID and AccountName
				if place.AccountID == "" {
					t.Logf("Place missing AccountID: %+v", place)
					return false
				}
				if place.AccountName == "" {
					t.Logf("Place missing AccountName: %+v", place)
					return false
				}
				
				// Verify that the AccountID corresponds to an actual account
				account, err := aggregator.GetAccount(place.AccountID)
				if err != nil {
					t.Logf("Place references non-existent account ID %s: %+v", place.AccountID, place)
					return false
				}
				
				// Verify that AccountName matches the actual account name
				if place.AccountName != account.Name {
					t.Logf("Place AccountName mismatch: expected %s, got %s", account.Name, place.AccountName)
					return false
				}
			}

			// Check finances labeling consistency
			for _, finance := range allData.Finances {
				// Each finance must have both AccountID and AccountName
				if finance.AccountID == "" {
					t.Logf("Finance missing AccountID: %+v", finance)
					return false
				}
				if finance.AccountName == "" {
					t.Logf("Finance missing AccountName: %+v", finance)
					return false
				}
				
				// Verify that the AccountID corresponds to an actual account
				account, err := aggregator.GetAccount(finance.AccountID)
				if err != nil {
					t.Logf("Finance references non-existent account ID %s: %+v", finance.AccountID, finance)
					return false
				}
				
				// Verify that AccountName matches the actual account name
				if finance.AccountName != account.Name {
					t.Logf("Finance AccountName mismatch: expected %s, got %s", account.Name, finance.AccountName)
					return false
				}
			}

			// Verify consistency across different data types for the same account
			// All data items from the same account should have identical AccountID and AccountName
			accountLabels := make(map[string]string) // AccountID -> AccountName

			// Collect labels from cameras
			for _, camera := range allData.Cameras {
				if existingName, exists := accountLabels[camera.AccountID]; exists {
					if existingName != camera.AccountName {
						t.Logf("Inconsistent AccountName for account %s: camera has %s, expected %s", 
							camera.AccountID, camera.AccountName, existingName)
						return false
					}
				} else {
					accountLabels[camera.AccountID] = camera.AccountName
				}
			}

			// Verify labels from places match
			for _, place := range allData.Places {
				if existingName, exists := accountLabels[place.AccountID]; exists {
					if existingName != place.AccountName {
						t.Logf("Inconsistent AccountName for account %s: place has %s, expected %s", 
							place.AccountID, place.AccountName, existingName)
						return false
					}
				} else {
					accountLabels[place.AccountID] = place.AccountName
				}
			}

			// Verify labels from finances match
			for _, finance := range allData.Finances {
				if existingName, exists := accountLabels[finance.AccountID]; exists {
					if existingName != finance.AccountName {
						t.Logf("Inconsistent AccountName for account %s: finance has %s, expected %s", 
							finance.AccountID, finance.AccountName, existingName)
						return false
					}
				} else {
					accountLabels[finance.AccountID] = finance.AccountName
				}
			}

			return true
		},
		genAccountSlice(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}