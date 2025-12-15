package multiconfig

import (
	"embed"
	"fmt"
	"sync"
)

// MultiAccountAggregator manages multiple accounts and provides aggregated data access
type MultiAccountAggregator struct {
	storage       *AccountStorage
	clientPool    *AccountClientPool
	dataMerger    *DataMerger
	mutex         sync.RWMutex
	accounts      map[string]*AccountConfig
	handlerFactory HandlerFactory
	templateFs    embed.FS
}

// NewMultiAccountAggregator creates a new multi-account aggregator
func NewMultiAccountAggregator(storage *AccountStorage, handlerFactory HandlerFactory, templateFs embed.FS) *MultiAccountAggregator {
	aggregator := &MultiAccountAggregator{
		storage:        storage,
		clientPool:     NewAccountClientPool(),
		accounts:       make(map[string]*AccountConfig),
		handlerFactory: handlerFactory,
		templateFs:     templateFs,
	}
	aggregator.dataMerger = NewDataMerger(aggregator)
	return aggregator
}

// LoadAccounts loads all accounts from storage and initializes clients
func (m *MultiAccountAggregator) LoadAccounts() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	accounts, err := m.storage.LoadAccounts()
	if err != nil {
		return fmt.Errorf("failed to load accounts: %w", err)
	}

	m.accounts = accounts

	// Initialize clients for enabled accounts
	for id, account := range accounts {
		if account.Enabled {
			client := NewAccountAPIClient(account, m.handlerFactory, m.templateFs)
			m.clientPool.AddClient(id, client)
		}
	}

	return nil
}

// GetAccounts returns a copy of all accounts
func (m *MultiAccountAggregator) GetAccounts() map[string]*AccountConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]*AccountConfig)
	for id, account := range m.accounts {
		// Create a copy to prevent external modification
		accountCopy := *account
		result[id] = &accountCopy
	}
	return result
}

// GetAccount returns a specific account by ID
func (m *MultiAccountAggregator) GetAccount(accountID string) (*AccountConfig, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	account, exists := m.accounts[accountID]
	if !exists {
		return nil, fmt.Errorf("account %s not found", accountID)
	}

	// Return a copy to prevent external modification
	accountCopy := *account
	return &accountCopy, nil
}

// AddAccount adds a new account
func (m *MultiAccountAggregator) AddAccount(account *AccountConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Add to storage
	if err := m.storage.AddAccount(account); err != nil {
		return fmt.Errorf("failed to add account to storage: %w", err)
	}

	// Add to local cache
	m.accounts[account.ID] = account

	// Initialize client if account is enabled
	if account.Enabled {
		client := NewAccountAPIClient(account, m.handlerFactory, m.templateFs)
		m.clientPool.AddClient(account.ID, client)
	}

	return nil
}

// RemoveAccount removes an account by ID
func (m *MultiAccountAggregator) RemoveAccount(accountID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if account exists
	if _, exists := m.accounts[accountID]; !exists {
		return fmt.Errorf("account %s not found", accountID)
	}

	// Remove from storage
	if err := m.storage.RemoveAccount(accountID); err != nil {
		return fmt.Errorf("failed to remove account from storage: %w", err)
	}

	// Remove from local cache
	delete(m.accounts, accountID)

	// Remove client
	m.clientPool.RemoveClient(accountID)

	return nil
}

// EnableAccount enables an account and initializes its client
func (m *MultiAccountAggregator) EnableAccount(accountID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	account, exists := m.accounts[accountID]
	if !exists {
		return fmt.Errorf("account %s not found", accountID)
	}

	if account.Enabled {
		return nil // Already enabled
	}

	// Update account status
	account.Enabled = true

	// Save to storage
	if err := m.storage.SaveAccounts(m.accounts); err != nil {
		account.Enabled = false // Rollback
		return fmt.Errorf("failed to save account status: %w", err)
	}

	// Initialize client
	client := NewAccountAPIClient(account, m.handlerFactory, m.templateFs)
	m.clientPool.AddClient(accountID, client)

	return nil
}

// DisableAccount disables an account and removes its client
func (m *MultiAccountAggregator) DisableAccount(accountID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	account, exists := m.accounts[accountID]
	if !exists {
		return fmt.Errorf("account %s not found", accountID)
	}

	if !account.Enabled {
		return nil // Already disabled
	}

	// Update account status
	account.Enabled = false

	// Save to storage
	if err := m.storage.SaveAccounts(m.accounts); err != nil {
		account.Enabled = true // Rollback
		return fmt.Errorf("failed to save account status: %w", err)
	}

	// Remove client
	m.clientPool.RemoveClient(accountID)

	return nil
}

// GetEnabledAccounts returns only enabled accounts
func (m *MultiAccountAggregator) GetEnabledAccounts() map[string]*AccountConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]*AccountConfig)
	for id, account := range m.accounts {
		if account.Enabled {
			// Create a copy to prevent external modification
			accountCopy := *account
			result[id] = &accountCopy
		}
	}
	return result
}

// GetPrimaryAccount returns the primary account for legacy compatibility
func (m *MultiAccountAggregator) GetPrimaryAccount() (*AccountConfig, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Load the current configuration to get the primary account ID
	config, err := m.storage.loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// If primary account is specified and exists, return it
	if config.PrimaryAccount != "" {
		if account, exists := m.accounts[config.PrimaryAccount]; exists && account.Enabled {
			// Return a copy to prevent external modification
			accountCopy := *account
			return &accountCopy, nil
		}
	}

	// Fallback: find first enabled account as primary
	for _, account := range m.accounts {
		if account.Enabled {
			// Return a copy to prevent external modification
			accountCopy := *account
			return &accountCopy, nil
		}
	}

	return nil, fmt.Errorf("no enabled accounts found")
}

// GetClient returns the API client for a specific account
func (m *MultiAccountAggregator) GetClient(accountID string) (*AccountAPIClient, error) {
	return m.clientPool.GetClient(accountID)
}

// GetAllClients returns all active API clients
func (m *MultiAccountAggregator) GetAllClients() map[string]*AccountAPIClient {
	return m.clientPool.GetAllClients()
}

// AccountClientPool manages API clients for multiple accounts
type AccountClientPool struct {
	clients map[string]*AccountAPIClient
	mutex   sync.RWMutex
}

// NewAccountClientPool creates a new client pool
func NewAccountClientPool() *AccountClientPool {
	return &AccountClientPool{
		clients: make(map[string]*AccountAPIClient),
	}
}

// AddClient adds a client for an account
func (p *AccountClientPool) AddClient(accountID string, client *AccountAPIClient) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.clients[accountID] = client
}

// RemoveClient removes a client for an account
func (p *AccountClientPool) RemoveClient(accountID string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	delete(p.clients, accountID)
}

// GetClient returns the client for a specific account
func (p *AccountClientPool) GetClient(accountID string) (*AccountAPIClient, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	client, exists := p.clients[accountID]
	if !exists {
		return nil, fmt.Errorf("client for account %s not found", accountID)
	}
	return client, nil
}

// GetAllClients returns all clients
func (p *AccountClientPool) GetAllClients() map[string]*AccountAPIClient {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make(map[string]*AccountAPIClient)
	for id, client := range p.clients {
		result[id] = client
	}
	return result
}

// AccountAPIClient wraps an API handler for individual account operations
type AccountAPIClient struct {
	config  *AccountConfig
	handler APIHandler
	factory HandlerFactory
}

// NewAccountAPIClient creates a new API client for an account
func NewAccountAPIClient(account *AccountConfig, factory HandlerFactory, templateFs embed.FS) *AccountAPIClient {
	// Create handler using the factory
	handler := factory.CreateHandler(account, templateFs)

	return &AccountAPIClient{
		config:  account,
		handler: handler,
		factory: factory,
	}
}

// GetConfig returns the account configuration
func (c *AccountAPIClient) GetConfig() *AccountConfig {
	// Return a copy to prevent external modification
	configCopy := *c.config
	return &configCopy
}

// GetHandler returns the underlying handler
func (c *AccountAPIClient) GetHandler() APIHandler {
	return c.handler
}

// UpdateTokens updates the authentication tokens for this client
func (c *AccountAPIClient) UpdateTokens(token, refreshToken string) {
	c.config.Token = token
	c.config.RefreshToken = refreshToken
	
	// Update the handler's tokens as well
	c.handler.UpdateTokens(token, refreshToken)
}

// IsEnabled returns whether this account is enabled
func (c *AccountAPIClient) IsEnabled() bool {
	return c.config.Enabled
}

// GetAccountID returns the account ID
func (c *AccountAPIClient) GetAccountID() string {
	return c.config.ID
}

// GetAccountName returns the account name
func (c *AccountAPIClient) GetAccountName() string {
	return c.config.Name
}

// GetDataMerger returns the data merger instance
func (m *MultiAccountAggregator) GetDataMerger() *DataMerger {
	return m.dataMerger
}

// UpdateAccountTokens updates the authentication tokens for a specific account
func (m *MultiAccountAggregator) UpdateAccountTokens(accountID, token, refreshToken string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if account exists
	account, exists := m.accounts[accountID]
	if !exists {
		return fmt.Errorf("account %s not found", accountID)
	}

	// Update tokens in the account configuration
	account.Token = token
	account.RefreshToken = refreshToken

	// Update tokens in the client if it exists
	if client, err := m.clientPool.GetClient(accountID); err == nil {
		client.UpdateTokens(token, refreshToken)
	}

	// Save the updated accounts
	return m.storage.SaveAccounts(m.accounts)
}