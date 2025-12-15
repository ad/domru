package handlers

import (
	"embed"

	"github.com/ad/domru/config"
	"github.com/ad/domru/multiconfig"
)

// HandlerFactory implements multiconfig.HandlerFactory
type HandlerFactory struct{}

// NewHandlerFactory creates a new handler factory
func NewHandlerFactory() *HandlerFactory {
	return &HandlerFactory{}
}

// CreateHandler creates an API handler for an account
func (f *HandlerFactory) CreateHandler(accountConfig *multiconfig.AccountConfig, templateFs embed.FS) multiconfig.APIHandler {
	// Convert AccountConfig to legacy Config for handlers
	legacyConfig := &config.Config{
		Token:        accountConfig.Token,
		RefreshToken: accountConfig.RefreshToken,
		Login:        accountConfig.Login,
		Operator:     accountConfig.Operator,
		UUID:         accountConfig.UUID,
		Port:         18000, // Default port
	}

	// Create handler with the legacy config
	handler := NewHandlers(legacyConfig, templateFs)
	
	// Return a wrapper that implements APIHandler
	return &HandlerAPIWrapper{handler: handler}
}

// HandlerAPIWrapper wraps Handler to implement multiconfig.APIHandler
type HandlerAPIWrapper struct {
	handler *Handler
}

// Cameras implements multiconfig.APIHandler
func (w *HandlerAPIWrapper) Cameras() (string, error) {
	return w.handler.Cameras()
}

// Places implements multiconfig.APIHandler
func (w *HandlerAPIWrapper) Places() (string, error) {
	return w.handler.Places()
}

// GetFinances implements multiconfig.APIHandler
func (w *HandlerAPIWrapper) GetFinances() (*multiconfig.FinanceData, error) {
	finances, err := w.handler.GetFinances()
	if err != nil {
		return nil, err
	}
	
	// Convert to multiconfig.FinanceData
	return &multiconfig.FinanceData{
		Balance:     finances.Balance,
		BlockType:   finances.BlockType,
		AmountSum:   finances.AmountSum,
		TargetDate:  finances.TargetDate,
		PaymentLink: finances.PaymentLink,
		Blocked:     finances.Blocked,
	}, nil
}

// UpdateTokens implements multiconfig.APIHandler
func (w *HandlerAPIWrapper) UpdateTokens(token, refreshToken string) {
	w.handler.Config.Token = token
	w.handler.Config.RefreshToken = refreshToken
}

// GetHandler returns the underlying handler for backward compatibility
func (w *HandlerAPIWrapper) GetHandler() *Handler {
	return w.handler
}