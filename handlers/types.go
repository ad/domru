package handlers

import "github.com/ad/domru/internal/models"

// Page data structs used by templates
type AccountsPageData struct {
	Accounts      []Account
	Phone         string
	HassioIngress string
	LoginError    string
}

type LoginPageData struct {
	LoginError    string
	Phone         string
	HassioIngress string
}

type SMSPageData struct {
	Phone         string
	Index         string
	HassioIngress string
	LoginError    string
}

// Aliases to centralized domain models (removed duplicate struct definitions)
type Account = models.Account
type ConfirmRequest = models.ConfirmRequest
type ConfirmResponse = models.ConfirmResponse
type Cameras = models.Cameras
type Places = models.Places
type Finances = models.Finances
type HomePageData = models.HomePageData

// HAConfig mirrors Home Assistant network info response
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
