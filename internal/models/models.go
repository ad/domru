package models

// Centralized data structures migrated from handlers.

type Account struct {
	OperatorID   int64  `json:"operatorId"`
	SubscriberID int64  `json:"subscriberId"`
	AccountID    string `json:"accountId"`
	PlaceID      int64  `json:"placeId"`
	Address      string `json:"address"`
	ProfileID    string `json:"profileId"`
}

type ConfirmRequest struct {
	Confirm      string `json:"confirm1"`
	SubscriberID int64  `json:"subscriberId"`
	Login        string `json:"login"`
	OperatorID   int64  `json:"operatorId"`
	AccountID    string `json:"accountId"`
	ProfileID    string `json:"profileId"`
}

type ConfirmResponse struct {
	OperatorID   int64  `json:"operatorId"`
	TokenType    string `json:"tokenType"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type Places struct {
	Data []struct {
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
	} `json:"data"`
}

type Cameras struct {
	Data []struct {
		ID       int    `json:"ID"`
		Name     string `json:"Name"`
		IsActive int    `json:"IsActive"`
	} `json:"data"`
}

type Finances struct {
	Balance     float64 `json:"balance"`
	BlockType   string  `json:"blockType"`
	AmountSum   float64 `json:"amountSum"`
	TargetDate  string  `json:"targetDate"`
	PaymentLink string  `json:"paymentLink"`
	Blocked     bool    `json:"blocked"`
}

type EventsInputModel struct {
	Data []struct {
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
	} `json:"data,omitempty"`
}

type HomePageData struct {
	HassioIngress string
	HostIP        string
	Port          string
	LoginError    string
	Phone         string
	Token         string
	RefreshToken  string
	Cameras       Cameras
	Places        Places
	Finances      Finances
}

type HAConfig struct {
	Result string `json:"result"`
	Data   struct {
		Interfaces []struct {
			Ipv4 struct {
				Address []string `json:"address"`
			} `json:"ipv4"`
		} `json:"interfaces"`
	} `json:"data"`
}
