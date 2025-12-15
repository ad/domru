package multiconfig

import (
	"embed"
	"fmt"
)

// MockHandlerFactory is a test implementation of HandlerFactory
type MockHandlerFactory struct{}

// CreateHandler creates a mock API handler for testing
func (f *MockHandlerFactory) CreateHandler(accountConfig *AccountConfig, templateFs embed.FS) APIHandler {
	return &MockAPIHandler{
		config: accountConfig,
	}
}

// MockAPIHandler is a test implementation of APIHandler
type MockAPIHandler struct {
	config *AccountConfig
}

// Cameras returns mock camera data specific to the account
func (m *MockAPIHandler) Cameras() (string, error) {
	// Return different data based on account ID to ensure proper isolation
	if m.config.Token == "invalid-token" {
		return "Not authorized", fmt.Errorf("invalid token")
	}
	
	// Return account-specific camera data
	cameraID := 1
	if m.config.ID != "" {
		// Use account ID to generate unique camera ID
		for _, char := range m.config.ID {
			cameraID += int(char)
		}
	}
	
	return fmt.Sprintf(`{"data":[{"ID":%d,"Name":"Camera-%s","IsActive":1}]}`, cameraID, m.config.ID), nil
}

// Places returns mock place data specific to the account
func (m *MockAPIHandler) Places() (string, error) {
	// Return different data based on account ID to ensure proper isolation
	if m.config.Token == "invalid-token" {
		return "Not authorized", fmt.Errorf("invalid token")
	}
	
	// Return account-specific place data
	placeID := 1
	if m.config.ID != "" {
		// Use account ID to generate unique place ID
		for _, char := range m.config.ID {
			placeID += int(char)
		}
	}
	
	return fmt.Sprintf(`{"data":[{"id":%d,"place":{"id":%d,"address":{"visibleAddress":"Address-%s"},"accessControls":[]},"subscriber":{"id":1,"name":"User-%s","accountId":"%s"},"blocked":false}]}`, placeID, placeID, m.config.ID, m.config.ID, m.config.ID), nil
}

// GetFinances returns mock finance data specific to the account
func (m *MockAPIHandler) GetFinances() (*FinanceData, error) {
	// Return error for invalid tokens
	if m.config.Token == "invalid-token" {
		return nil, fmt.Errorf("invalid token")
	}
	
	return &FinanceData{
		Balance:     100.0,
		BlockType:   "none",
		AmountSum:   0.0,
		TargetDate:  "2024-12-31",
		PaymentLink: "http://test.com",
		Blocked:     false,
	}, nil
}

// UpdateTokens updates the mock tokens
func (m *MockAPIHandler) UpdateTokens(token, refreshToken string) {
	m.config.Token = token
	m.config.RefreshToken = refreshToken
}