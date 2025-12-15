package multiconfig

import (
	"encoding/json"
	"fmt"
	"sync"
)

// DataMerger component for combining data from multiple accounts
type DataMerger struct {
	aggregator *MultiAccountAggregator
}

// NewDataMerger creates a new DataMerger instance
func NewDataMerger(aggregator *MultiAccountAggregator) *DataMerger {
	return &DataMerger{
		aggregator: aggregator,
	}
}

// CameraWithAccount represents a camera with account information
type CameraWithAccount struct {
	ID          int    `json:"ID"`
	Name        string `json:"Name"`
	IsActive    int    `json:"IsActive"`
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
}

// PlaceWithAccount represents a place with account information
type PlaceWithAccount struct {
	ID    int `json:"id"`
	Place struct {
		ID      int `json:"id"`
		Address struct {
			VisibleAddress string `json:"visibleAddress"`
		} `json:"address"`
		AccessControls []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"accessControls"`
	} `json:"place"`
	Subscriber struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		AccountID string `json:"accountId"`
	} `json:"subscriber"`
	Blocked     bool   `json:"blocked"`
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
}

// FinanceWithAccount represents finances with account information
type FinanceWithAccount struct {
	Balance     float64 `json:"balance"`
	BlockType   string  `json:"blockType"`
	AmountSum   float64 `json:"amountSum"`
	TargetDate  string  `json:"targetDate"`
	PaymentLink string  `json:"paymentLink"`
	Blocked     bool    `json:"blocked"`
	AccountID   string  `json:"account_id"`
	AccountName string  `json:"account_name"`
}

// EventWithAccount represents an event with account information
type EventWithAccount struct {
	ID            string `json:"id,omitempty"`
	PlaceID       int    `json:"placeId,omitempty"`
	EventTypeName string `json:"eventTypeName,omitempty"`
	Timestamp     string `json:"timestamp,omitempty"`
	Message       string `json:"message,omitempty"`
	Source        struct {
		Type string `json:"type,omitempty"`
		ID   int    `json:"id,omitempty"`
	} `json:"source,omitempty"`
	Value struct {
		Type  string `json:"type,omitempty"`
		Value bool   `json:"value,omitempty"`
	} `json:"value,omitempty"`
	EventStatusValue interface{}   `json:"eventStatusValue,omitempty"`
	Actions          []interface{} `json:"actions,omitempty"`
	AccountID        string        `json:"account_id"`
	AccountName      string        `json:"account_name"`
}

// AggregatedData contains all aggregated data from multiple accounts
type AggregatedData struct {
	Cameras   []CameraWithAccount   `json:"cameras"`
	Places    []PlaceWithAccount    `json:"places"`
	Finances  []FinanceWithAccount  `json:"finances"`
	Events    []EventWithAccount    `json:"events"`
}

// GetAggregatedCameras returns cameras from all enabled accounts with account labels
func (m *DataMerger) GetAggregatedCameras() ([]CameraWithAccount, error) {
	var result []CameraWithAccount
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	clients := m.aggregator.GetAllClients()

	for accountID, client := range clients {
		if !client.IsEnabled() {
			continue
		}

		wg.Add(1)
		go func(accID string, c *AccountAPIClient) {
			defer wg.Done()

			camerasJSON, err := c.GetHandler().Cameras()
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to get cameras for account %s: %w", accID, err))
				mu.Unlock()
				return
			}

			// Skip if token expired
			if camerasJSON == "token can't be refreshed" {
				return
			}

			var cameras CamerasResponse
			if err := json.Unmarshal([]byte(camerasJSON), &cameras); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to unmarshal cameras for account %s: %w", accID, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			for _, camera := range cameras.Data {
				result = append(result, CameraWithAccount{
					ID:          camera.ID,
					Name:        camera.Name,
					IsActive:    camera.IsActive,
					AccountID:   accID,
					AccountName: c.GetAccountName(),
				})
			}
			mu.Unlock()
		}(accountID, client)
	}

	wg.Wait()

	if len(errors) > 0 && len(result) == 0 {
		return nil, fmt.Errorf("failed to get cameras from any account: %v", errors)
	}

	return result, nil
}

// GetAggregatedPlaces returns places from all enabled accounts with account identification
func (m *DataMerger) GetAggregatedPlaces() ([]PlaceWithAccount, error) {
	var result []PlaceWithAccount
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	clients := m.aggregator.GetAllClients()

	for accountID, client := range clients {
		if !client.IsEnabled() {
			continue
		}

		wg.Add(1)
		go func(accID string, c *AccountAPIClient) {
			defer wg.Done()

			placesJSON, err := c.GetHandler().Places()
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to get places for account %s: %w", accID, err))
				mu.Unlock()
				return
			}

			// Skip if token expired
			if placesJSON == "token can't be refreshed" {
				return
			}

			var places PlacesResponse
			if err := json.Unmarshal([]byte(placesJSON), &places); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to unmarshal places for account %s: %w", accID, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			for _, place := range places.Data {
				result = append(result, PlaceWithAccount{
					ID:          place.ID,
					Place:       place.Place,
					Subscriber:  place.Subscriber,
					Blocked:     place.Blocked,
					AccountID:   accID,
					AccountName: c.GetAccountName(),
				})
			}
			mu.Unlock()
		}(accountID, client)
	}

	wg.Wait()

	if len(errors) > 0 && len(result) == 0 {
		return nil, fmt.Errorf("failed to get places from any account: %v", errors)
	}

	return result, nil
}

// GetAggregatedFinances returns finances from all enabled accounts with account labels
func (m *DataMerger) GetAggregatedFinances() ([]FinanceWithAccount, error) {
	var result []FinanceWithAccount
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	clients := m.aggregator.GetAllClients()

	for accountID, client := range clients {
		if !client.IsEnabled() {
			continue
		}

		wg.Add(1)
		go func(accID string, c *AccountAPIClient) {
			defer wg.Done()

			finances, err := c.GetHandler().GetFinances()
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to get finances for account %s: %w", accID, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			result = append(result, FinanceWithAccount{
				Balance:     finances.Balance,
				BlockType:   finances.BlockType,
				AmountSum:   finances.AmountSum,
				TargetDate:  finances.TargetDate,
				PaymentLink: finances.PaymentLink,
				Blocked:     finances.Blocked,
				AccountID:   accID,
				AccountName: c.GetAccountName(),
			})
			mu.Unlock()
		}(accountID, client)
	}

	wg.Wait()

	if len(errors) > 0 && len(result) == 0 {
		return nil, fmt.Errorf("failed to get finances from any account: %v", errors)
	}

	return result, nil
}

// GetAggregatedEvents returns events from all enabled accounts for a specific place with account labels
func (m *DataMerger) GetAggregatedEvents(placeID string) ([]EventWithAccount, error) {
	var result []EventWithAccount

	// Note: Events aggregation is not implemented yet as it requires HTTP request context
	// This would need to be implemented at the handler level where proper HTTP requests
	// can be constructed with the placeID parameter

	return result, nil
}

// GetAllAggregatedData returns all aggregated data from all enabled accounts
func (m *DataMerger) GetAllAggregatedData() (*AggregatedData, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	data := &AggregatedData{
		Cameras:   make([]CameraWithAccount, 0),
		Places:    make([]PlaceWithAccount, 0),
		Finances:  make([]FinanceWithAccount, 0),
		Events:    make([]EventWithAccount, 0),
	}

	// Get cameras
	wg.Add(1)
	go func() {
		defer wg.Done()
		cameras, err := m.GetAggregatedCameras()
		if err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("failed to get aggregated cameras: %w", err))
			mu.Unlock()
			return
		}
		mu.Lock()
		data.Cameras = cameras
		mu.Unlock()
	}()

	// Get places
	wg.Add(1)
	go func() {
		defer wg.Done()
		places, err := m.GetAggregatedPlaces()
		if err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("failed to get aggregated places: %w", err))
			mu.Unlock()
			return
		}
		mu.Lock()
		data.Places = places
		mu.Unlock()
	}()

	// Get finances
	wg.Add(1)
	go func() {
		defer wg.Done()
		finances, err := m.GetAggregatedFinances()
		if err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("failed to get aggregated finances: %w", err))
			mu.Unlock()
			return
		}
		mu.Lock()
		data.Finances = finances
		mu.Unlock()
	}()

	wg.Wait()

	if len(errors) > 0 {
		return data, fmt.Errorf("errors occurred while aggregating data: %v", errors)
	}

	return data, nil
}