package multiconfig

import "embed"

// APIHandler defines the interface for API operations needed by multiconfig
type APIHandler interface {
	Cameras() (string, error)
	Places() (string, error)
	GetFinances() (*FinanceData, error)
	UpdateTokens(token, refreshToken string)
}

// FinanceData represents the finance data structure
type FinanceData struct {
	Balance     float64 `json:"balance"`
	BlockType   string  `json:"blockType"`
	AmountSum   float64 `json:"amountSum"`
	TargetDate  string  `json:"targetDate"`
	PaymentLink string  `json:"paymentLink"`
	Blocked     bool    `json:"blocked"`
}

// CameraData represents camera data structure
type CameraData struct {
	ID       int    `json:"ID"`
	Name     string `json:"Name"`
	IsActive int    `json:"IsActive"`
}

// CamerasResponse represents the cameras API response
type CamerasResponse struct {
	Data []CameraData `json:"data"`
}

// PlaceData represents place data structure
type PlaceData struct {
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
	Blocked bool `json:"blocked"`
}

// PlacesResponse represents the places API response
type PlacesResponse struct {
	Data []PlaceData `json:"data"`
}

// HandlerFactory creates API handlers for accounts
type HandlerFactory interface {
	CreateHandler(config *AccountConfig, templateFs embed.FS) APIHandler
}