package multiconfig

import (
	"fmt"

	"github.com/ad/domru/config"
)

// LegacyConfigAdapter provides backward compatibility by adapting multi-account system
// to the legacy single-account interface
type LegacyConfigAdapter struct {
	aggregator *MultiAccountAggregator
}

// NewLegacyConfigAdapter creates a new legacy configuration adapter
func NewLegacyConfigAdapter(aggregator *MultiAccountAggregator) *LegacyConfigAdapter {
	return &LegacyConfigAdapter{
		aggregator: aggregator,
	}
}

// GetPrimaryConfig returns the primary account configuration in legacy format
// This method ensures existing handlers continue working without changes
func (a *LegacyConfigAdapter) GetPrimaryConfig() (*config.Config, error) {
	primaryAccount, err := a.aggregator.GetPrimaryAccount()
	if err != nil {
		return nil, fmt.Errorf("failed to get primary account: %w", err)
	}

	// Convert AccountConfig to legacy Config format
	legacyConfig := &config.Config{
		Token:        primaryAccount.Token,
		RefreshToken: primaryAccount.RefreshToken,
		Login:        primaryAccount.Login,
		Operator:     primaryAccount.Operator,
		UUID:         primaryAccount.UUID,
		Port:         18000, // Default port for backward compatibility
	}

	return legacyConfig, nil
}

// GetAccountConfig returns a specific account configuration in legacy format
// This allows handlers to work with specific accounts when needed
func (a *LegacyConfigAdapter) GetAccountConfig(accountID string) (*config.Config, error) {
	account, err := a.aggregator.GetAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account %s: %w", accountID, err)
	}

	// Convert AccountConfig to legacy Config format
	legacyConfig := &config.Config{
		Token:        account.Token,
		RefreshToken: account.RefreshToken,
		Login:        account.Login,
		Operator:     account.Operator,
		UUID:         account.UUID,
		Port:         18000, // Default port for backward compatibility
	}

	return legacyConfig, nil
}

// UpdatePrimaryAccountTokens updates the tokens for the primary account
// This maintains compatibility with existing token refresh logic
func (a *LegacyConfigAdapter) UpdatePrimaryAccountTokens(token, refreshToken string) error {
	primaryAccount, err := a.aggregator.GetPrimaryAccount()
	if err != nil {
		return fmt.Errorf("failed to get primary account: %w", err)
	}

	// Use the aggregator's method to update account tokens
	return a.aggregator.UpdateAccountTokens(primaryAccount.ID, token, refreshToken)
}

// UpdateAccountTokens updates the tokens for a specific account
func (a *LegacyConfigAdapter) UpdateAccountTokens(accountID, token, refreshToken string) error {
	// Use the aggregator's method to update account tokens
	return a.aggregator.UpdateAccountTokens(accountID, token, refreshToken)
}

// HasPrimaryAccount checks if there is a primary account available
func (a *LegacyConfigAdapter) HasPrimaryAccount() bool {
	_, err := a.aggregator.GetPrimaryAccount()
	return err == nil
}

// GetPrimaryAccountID returns the ID of the primary account
func (a *LegacyConfigAdapter) GetPrimaryAccountID() (string, error) {
	primaryAccount, err := a.aggregator.GetPrimaryAccount()
	if err != nil {
		return "", fmt.Errorf("failed to get primary account: %w", err)
	}
	return primaryAccount.ID, nil
}