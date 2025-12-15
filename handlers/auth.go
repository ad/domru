package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ad/domru/multiconfig"
)

// LoginHandler ...
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	ingressPath := r.Header.Get("X-Ingress-Path")

	// log.Println(r.Method, "/login", ingressPath)

	w.Header().Set("Content-Type", "text/html")

	var loginError string

	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			loginError = fmt.Sprintf("ParseForm() err: %v", err)
		} else {
			phone := r.FormValue("phone")
			accounts, err := h.Accounts(&phone)
			if err != nil {
				loginError = fmt.Sprintf("login error: %v", err.Error())
			} else {
				if n, err := strconv.Atoi(phone); err == nil {
					h.Config.Login = n

					if err = h.Config.WriteConfig(); err != nil {
						log.Println("error on write config file ", err)
					}
				}

				h.UserAccounts = accounts
				// log.Printf("got accounts %+v\n", accounts)

				data := AccountsPageData{accounts, phone, ingressPath, loginError}

				var tmpl []byte
				var err error
				if tmpl, err = h.TemplateFs.ReadFile("templates/accounts.html"); err != nil {
					fmt.Println(err)
				}

				t := template.New("t")
				t, err = t.Parse(string(tmpl))
				if err != nil {
					loginError = err.Error()
				} else {
					err = t.Execute(w, data)
					if err != nil {
						loginError = err.Error()
					}
				}
			}

			if loginError != "" {
				log.Println(loginError)
			}
			return
		}
	}

	var tmpl []byte
	var err error
	if tmpl, err = h.TemplateFs.ReadFile("templates/login.html"); err != nil {
		fmt.Println(err)
	}

	data := LoginPageData{loginError, strconv.Itoa(h.Config.Login), ingressPath}

	t := template.New("t")
	t, err = t.Parse(string(tmpl))
	if err != nil {
		loginError = err.Error()
	} else {
		err = t.Execute(w, data)
		if err != nil {
			loginError = err.Error()
		}
	}

	if loginError != "" {
		log.Println(loginError)
	}
}

// LoginAddressHandler ...
func (h *Handler) LoginAddressHandler(w http.ResponseWriter, r *http.Request) {
	ingressPath := r.Header.Get("X-Ingress-Path")

	// log.Println(r.Method, "/login/address", ingressPath)

	w.Header().Set("Content-Type", "text/html")

	var loginError, phone, index string

	if err := r.ParseForm(); err != nil {
		loginError = fmt.Sprintf("ParseForm() err: %v", err)
	} else {
		phone = r.FormValue("phone")
		index = r.FormValue("index")

		if accountIndex, err := strconv.Atoi(index); err != nil {
			loginError = fmt.Sprintf("ParseForm() err: %v", err)
		} else {
			if accountIndex < 0 || accountIndex > len(h.UserAccounts)-1 {
				loginError = "Selected wrong account"
			} else {
				account := h.UserAccounts[accountIndex]
				h.Account = &account
				result, err := h.RequestCode(&phone, account)
				if err != nil {
					loginError = fmt.Sprintf("loginAddress error: %v", err.Error())
				}

				if n, err := strconv.Atoi(phone); err == nil {
					h.Config.Login = n
				}

				h.Config.Operator = int(h.Account.OperatorID)
				if err = h.Config.WriteConfig(); err != nil {
					log.Println("error on write config file ", err)
				}

				if !result && loginError == "" {
					loginError = "Error on sms send"
				}

			}
		}

	}

	if loginError != "" {
		log.Println(loginError)
	}

	data := SMSPageData{phone, index, ingressPath, loginError}

	var tmpl []byte
	var err error
	if tmpl, err = h.TemplateFs.ReadFile("templates/sms.html"); err != nil {
		fmt.Println(err)
	}

	t := template.New("t")
	t, err = t.Parse(string(tmpl))
	if err != nil {
		loginError = err.Error()
	} else {
		err = t.Execute(w, data)
		if err != nil {
			loginError = err.Error()
		}
	}

	if loginError != "" {
		log.Println(loginError)
	}
}

// HomeHandler ...
func (h *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	ingressPath := r.Header.Get("X-Ingress-Path")
	// log.Println(r.Method, "/", ingressPath)

	// Check if we have multi-account support
	isMultiAccount := h.MultiAccountAggregator != nil

	// For multi-account, check if we have any accounts
	if isMultiAccount {
		if !h.LegacyAdapter.HasPrimaryAccount() {
			http.Redirect(w, r, ingressPath+"/login", http.StatusSeeOther)
			return
		}
	} else {
		// Legacy single-account check
		if h.Config.Token == "" || h.Config.RefreshToken == "" {
			http.Redirect(w, r, ingressPath+"/login", http.StatusSeeOther)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html")

	var loginError string

	hostIP, err2 := h.HANetwork()
	if err2 != nil {
		// loginError = "hostIP got: " + err2.Error()
		hostIP = "localhost"
	}

	var cameras Cameras
	var places Places
	var finances *Finances

	// Multi-account data
	var multiAccountCameras []MultiAccountCamera
	var multiAccountPlaces []MultiAccountPlace
	var multiAccountFinances []MultiAccountFinance

	if isMultiAccount {
		// Get aggregated data from all accounts
		dataMerger := h.MultiAccountAggregator.GetDataMerger()

		// Get cameras from all accounts
		if aggCameras, err := dataMerger.GetAggregatedCameras(); err == nil {
			for _, camera := range aggCameras {
				multiAccountCameras = append(multiAccountCameras, MultiAccountCamera{
					ID:          camera.ID,
					Name:        camera.Name,
					IsActive:    camera.IsActive,
					AccountID:   camera.AccountID,
					AccountName: camera.AccountName,
				})
			}
		} else {
			loginError = "multi-account cameras error: " + err.Error()
		}

		// Get places from all accounts
		if aggPlaces, err := dataMerger.GetAggregatedPlaces(); err == nil {
			for _, place := range aggPlaces {
				multiAccountPlaces = append(multiAccountPlaces, MultiAccountPlace{
					ID:          place.ID,
					Place:       place.Place,
					Subscriber:  place.Subscriber,
					Blocked:     place.Blocked,
					AccountID:   place.AccountID,
					AccountName: place.AccountName,
				})
			}
		} else {
			loginError = "multi-account places error: " + err.Error()
		}

		// Get finances from all accounts
		if aggFinances, err := dataMerger.GetAggregatedFinances(); err == nil {
			for _, finance := range aggFinances {
				multiAccountFinances = append(multiAccountFinances, MultiAccountFinance{
					Balance:     finance.Balance,
					BlockType:   finance.BlockType,
					AmountSum:   finance.AmountSum,
					TargetDate:  finance.TargetDate,
					PaymentLink: finance.PaymentLink,
					Blocked:     finance.Blocked,
					AccountID:   finance.AccountID,
					AccountName: finance.AccountName,
				})
			}
		} else {
			loginError = "multi-account finances error: " + err.Error()
		}

		// For backward compatibility, also populate single-account data from primary account
		if primaryConfig, err := h.LegacyAdapter.GetPrimaryConfig(); err == nil {
			h.Config = primaryConfig // Update current config
		}
	}

	// Get single-account data (for backward compatibility or single-account mode)
	if loginError == "" {
		camerasData, err := h.Cameras()
		if err != nil {
			if !isMultiAccount {
				loginError = "cameras (" + camerasData + ") got " + err.Error()
			}
		} else {
			if err := json.Unmarshal([]byte(camerasData), &cameras); err != nil {
				if !isMultiAccount {
					loginError = "cameras (" + camerasData + ") Unmarshal got " + err.Error()
				}
			}
		}
	}

	if loginError == "" {
		placesData, err := h.Places()
		if err != nil {
			if !isMultiAccount {
				loginError = "places (" + placesData + ") got " + err.Error()
			}
		} else {
			if err := json.Unmarshal([]byte(placesData), &places); err != nil {
				if !isMultiAccount {
					loginError = "places (" + placesData + ") Unmarshal got " + err.Error()
				}
			}
		}
	}

	finances, _ = h.GetFinances()
	if finances == nil {
		finances = &Finances{}
	}

	data := HomePageData{
		HassioIngress: ingressPath,
		HostIP:        hostIP,
		Port:          strconv.Itoa(h.Config.Port),
		LoginError:    loginError,
		Phone:         strconv.Itoa(h.Config.Login),
		Token:         h.Config.Token,
		RefreshToken:  h.Config.RefreshToken,
		Cameras:       cameras,
		Places:        places,
		Finances:      *finances,
		
		// Multi-account data
		MultiAccountCameras:  multiAccountCameras,
		MultiAccountPlaces:   multiAccountPlaces,
		MultiAccountFinances: multiAccountFinances,
		IsMultiAccount:       isMultiAccount,
	}

	var tmpl []byte
	var err error
	if tmpl, err = h.TemplateFs.ReadFile("templates/home.html"); err != nil {
		fmt.Println("reafile templates/home.html error", err)
	}

	t := template.New("t")
	t, err = t.Parse(string(tmpl))
	if err != nil {
		loginError = "parse templates/home.html " + err.Error()
	} else {
		err = t.Execute(w, data)
		if err != nil {
			loginError = "execute templates/home.html " + err.Error()
		}
	}

	if loginError != "" {
		log.Println(loginError)
	}
}

// Accounts ...
func (h *Handler) Accounts(username *string) (a []Account, err error) {
	var (
		body   []byte
		client = http.DefaultClient
	)

	url := fmt.Sprintf(API_AUTH_LOGIN, *username)
	// log.Println("/accountsHandler", url)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	request = request.WithContext(ctx)

	rt := WithHeader(client.Transport)
	rt.Set("Host", API_HOST)
	rt.Set("Content-Type", "application/json")
	rt.Set("Connection", "keep-alive")
	rt.Set("Accept", "*/*")
	rt.Set("User-Agent", GenerateUserAgent(h.Config.Operator, h.Config.UUID, 0))
	rt.Set("Authorization", "")
	rt.Set("Accept-Language", "en-us")
	rt.Set("Accept-Encoding", "gzip, deflate, br")

	client.Transport = rt

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			log.Println(err2)
		}
	}()

	if resp.StatusCode == 409 { // Conflict (tokent already expired)
		return nil, fmt.Errorf("token can't be refreshed")
	}

	if body, err = ReadResponseBody(resp); err != nil {
		return nil, err
	}

	var accounts []Account
	if err = json.Unmarshal(body, &accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}

// RequestCode ...
func (h *Handler) RequestCode(username *string, account Account) (result bool, err error) {
	var (
		body   []byte
		client = http.DefaultClient
	)

	url := fmt.Sprintf(API_AUTH_CONFIRMATION, *username)
	// log.Println("/requestCodeHandler", url)

	b, err := json.Marshal(account)
	if err != nil {
		return false, err
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	request = request.WithContext(ctx)

	rt := WithHeader(client.Transport)
	rt.Set("Host", API_HOST)
	rt.Set("Content-Type", "application/json")
	rt.Set("User-Agent", GenerateUserAgent(h.Config.Operator, h.Config.UUID, account.PlaceID))
	rt.Set("Connection", "keep-alive")
	rt.Set("Accept", "*/*")
	rt.Set("Accept-Language", "en-us")
	rt.Set("Accept-Encoding", "gzip, deflate, br")
	rt.Set("Authorization", "")

	client.Transport = rt

	resp, err := client.Do(request)
	if err != nil {
		return false, err
	}

	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			log.Println(err2)
		}
	}()

	if resp.StatusCode == 409 { // Conflict (tokent already expired)
		return false, fmt.Errorf("token can't be refreshed")
	}

	if resp.StatusCode == 200 {
		return true, nil
	}

	if body, err = ReadResponseBody(resp); err != nil {
		return false, err
	}

	return false, fmt.Errorf("status %d\n%s", resp.StatusCode, body)
}

// SendCode ...
func (h *Handler) SendCode(r *http.Request) (authToken, refreshToken string, err error) {
	var (
		body   []byte
		client = http.DefaultClient
	)

	url := fmt.Sprintf(API_AUTH_CONFIRMATION_SMS, strconv.Itoa(h.Config.Login))

	if err := r.ParseForm(); err != nil {
		return "", "", fmt.Errorf("ParseForm() err: %v", err)
	}

	code := r.FormValue("code")

	c := ConfirmRequest{
		Confirm:      code,
		SubscriberID: h.Account.SubscriberID,
		Login:        strconv.Itoa(h.Config.Login),
		OperatorID:   int64(h.Config.Operator),
		AccountID:    h.Account.AccountID,
		ProfileID:    h.Account.ProfileID,
	}

	b, err := json.Marshal(c)
	if err != nil {
		return "", "", err
	}

	// log.Println("/sms", url, string(b))

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		return "", "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	request = request.WithContext(ctx)

	rt := WithHeader(client.Transport)
	rt.Set("Host", API_HOST)
	rt.Set("Content-Type", "application/json")
	rt.Set("User-Agent", GenerateUserAgent(h.Config.Operator, h.Config.UUID, h.Account.PlaceID))
	rt.Set("Connection", "keep-alive")
	rt.Set("Accept", "*/*")
	rt.Set("Accept-Language", "en-us")
	rt.Set("Accept-Encoding", "gzip, deflate, br")
	rt.Set("Authorization", "")

	client.Transport = rt

	resp, err := client.Do(request)
	if err != nil {
		return "", "", err
	}

	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			log.Println(err2)
		}
	}()

	if resp.StatusCode == 409 { // Conflict (tokent already expired)
		return "", "", fmt.Errorf("token can't be refreshed")
	}

	if body, err = ReadResponseBody(resp); err != nil {
		return "", "", err
	}

	if resp.StatusCode == 200 {
		var authResp ConfirmResponse
		if err = json.Unmarshal(body, &authResp); err != nil {
			return "", "", err
		}

		return authResp.AccessToken, authResp.RefreshToken, nil
	}

	return "", "", fmt.Errorf("unknown error with status %d\n%s", resp.StatusCode, body)
}

// AccountsHandler ...
func (h *Handler) AccountsHandler(w http.ResponseWriter, r *http.Request) {
	// log.Println("/accountsHandler")

	login := strconv.Itoa(h.Config.Login)

	data, err := h.Accounts(&login)
	if err != nil {
		log.Println("accountsHandler", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")

	b, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("Error: %s", err)

		return
	}

	if _, err := w.Write(b); err != nil {
		log.Println("accountsHandler", err.Error())
	}
}

// MultiAccountsHandler handles listing all accounts in multi-account mode
func (h *Handler) MultiAccountsHandler(w http.ResponseWriter, r *http.Request) {
	// Only work in multi-account mode
	if h.MultiAccountAggregator == nil {
		http.Error(w, "Multi-account mode not enabled", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleListAccounts(w, r)
	case http.MethodPost:
		h.handleAddAccount(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// AccountByIDHandler handles operations on specific accounts
func (h *Handler) AccountByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Only work in multi-account mode
	if h.MultiAccountAggregator == nil {
		http.Error(w, "Multi-account mode not enabled", http.StatusServiceUnavailable)
		return
	}

	// Extract account ID from URL path
	path := r.URL.Path
	// Expected format: /accounts/{id} or /accounts/{id}/action
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 || parts[0] != "accounts" {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	accountID := parts[1]
	if accountID == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	// Check for specific actions
	if len(parts) >= 3 {
		action := parts[2]
		switch action {
		case "enable":
			h.handleEnableAccount(w, r, accountID)
		case "disable":
			h.handleDisableAccount(w, r, accountID)
		default:
			http.Error(w, "Unknown action", http.StatusBadRequest)
		}
		return
	}

	// Handle basic CRUD operations
	switch r.Method {
	case http.MethodGet:
		h.handleGetAccount(w, r, accountID)
	case http.MethodPut:
		h.handleUpdateAccount(w, r, accountID)
	case http.MethodDelete:
		h.handleDeleteAccount(w, r, accountID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListAccounts returns all accounts
func (h *Handler) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts := h.MultiAccountAggregator.GetAccounts()
	
	// Convert to response format (without sensitive tokens)
	type AccountResponse struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Login   int    `json:"login"`
		UUID    string `json:"uuid"`
		Enabled bool   `json:"enabled"`
	}

	response := make([]AccountResponse, 0, len(accounts))
	for _, account := range accounts {
		response = append(response, AccountResponse{
			ID:      account.ID,
			Name:    account.Name,
			Login:   account.Login,
			UUID:    account.UUID,
			Enabled: account.Enabled,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode accounts response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleAddAccount adds a new account
func (h *Handler) handleAddAccount(w http.ResponseWriter, r *http.Request) {
	type AddAccountRequest struct {
		Name         string `json:"name"`
		Login        int    `json:"login"`
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
		UUID         string `json:"uuid"`
		Operator     int    `json:"operator"`
		Enabled      bool   `json:"enabled"`
	}

	var req AddAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Login == 0 {
		http.Error(w, "Login is required", http.StatusBadRequest)
		return
	}
	if req.UUID == "" {
		http.Error(w, "UUID is required", http.StatusBadRequest)
		return
	}

	// Create new account
	account := &multiconfig.AccountConfig{
		Name:         req.Name,
		Login:        req.Login,
		Token:        req.Token,
		RefreshToken: req.RefreshToken,
		UUID:         req.UUID,
		Operator:     req.Operator,
		Enabled:      req.Enabled,
	}

	// Add account
	if err := h.MultiAccountAggregator.AddAccount(account); err != nil {
		log.Printf("Failed to add account: %v", err)
		http.Error(w, "Failed to add account", http.StatusInternalServerError)
		return
	}

	// Return created account (without sensitive tokens)
	type AccountResponse struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Login   int    `json:"login"`
		UUID    string `json:"uuid"`
		Enabled bool   `json:"enabled"`
	}

	response := AccountResponse{
		ID:      account.ID,
		Name:    account.Name,
		Login:   account.Login,
		UUID:    account.UUID,
		Enabled: account.Enabled,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode account response: %v", err)
	}
}

// handleGetAccount returns a specific account
func (h *Handler) handleGetAccount(w http.ResponseWriter, r *http.Request, accountID string) {
	account, err := h.MultiAccountAggregator.GetAccount(accountID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	// Return account (without sensitive tokens)
	type AccountResponse struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Login   int    `json:"login"`
		UUID    string `json:"uuid"`
		Enabled bool   `json:"enabled"`
	}

	response := AccountResponse{
		ID:      account.ID,
		Name:    account.Name,
		Login:   account.Login,
		UUID:    account.UUID,
		Enabled: account.Enabled,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode account response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleUpdateAccount updates an existing account
func (h *Handler) handleUpdateAccount(w http.ResponseWriter, r *http.Request, accountID string) {
	// Check if account exists
	existingAccount, err := h.MultiAccountAggregator.GetAccount(accountID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	type UpdateAccountRequest struct {
		Name         *string `json:"name,omitempty"`
		Token        *string `json:"token,omitempty"`
		RefreshToken *string `json:"refresh_token,omitempty"`
		Enabled      *bool   `json:"enabled,omitempty"`
	}

	var req UpdateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Update fields if provided
	updated := false
	if req.Name != nil && *req.Name != existingAccount.Name {
		// Update name by removing and re-adding account with new name
		updatedAccount := *existingAccount
		updatedAccount.Name = *req.Name
		
		if err := h.MultiAccountAggregator.RemoveAccount(accountID); err != nil {
			log.Printf("Failed to remove account for name update: %v", err)
			http.Error(w, "Failed to update account", http.StatusInternalServerError)
			return
		}
		
		if err := h.MultiAccountAggregator.AddAccount(&updatedAccount); err != nil {
			log.Printf("Failed to re-add account after name update: %v", err)
			http.Error(w, "Failed to update account", http.StatusInternalServerError)
			return
		}
		updated = true
	}

	if req.Token != nil || req.RefreshToken != nil {
		token := existingAccount.Token
		refreshToken := existingAccount.RefreshToken
		
		if req.Token != nil {
			token = *req.Token
		}
		if req.RefreshToken != nil {
			refreshToken = *req.RefreshToken
		}
		
		if err := h.MultiAccountAggregator.UpdateAccountTokens(accountID, token, refreshToken); err != nil {
			log.Printf("Failed to update account tokens: %v", err)
			http.Error(w, "Failed to update account tokens", http.StatusInternalServerError)
			return
		}
		updated = true
	}

	if req.Enabled != nil {
		if *req.Enabled != existingAccount.Enabled {
			if *req.Enabled {
				if err := h.MultiAccountAggregator.EnableAccount(accountID); err != nil {
					log.Printf("Failed to enable account: %v", err)
					http.Error(w, "Failed to enable account", http.StatusInternalServerError)
					return
				}
			} else {
				if err := h.MultiAccountAggregator.DisableAccount(accountID); err != nil {
					log.Printf("Failed to disable account: %v", err)
					http.Error(w, "Failed to disable account", http.StatusInternalServerError)
					return
				}
			}
			updated = true
		}
	}

	if !updated {
		http.Error(w, "No changes provided", http.StatusBadRequest)
		return
	}

	// Return updated account
	updatedAccount, err := h.MultiAccountAggregator.GetAccount(accountID)
	if err != nil {
		log.Printf("Failed to get updated account: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	type AccountResponse struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Login   int    `json:"login"`
		UUID    string `json:"uuid"`
		Enabled bool   `json:"enabled"`
	}

	response := AccountResponse{
		ID:      updatedAccount.ID,
		Name:    updatedAccount.Name,
		Login:   updatedAccount.Login,
		UUID:    updatedAccount.UUID,
		Enabled: updatedAccount.Enabled,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode account response: %v", err)
	}
}

// handleDeleteAccount deletes an account
func (h *Handler) handleDeleteAccount(w http.ResponseWriter, r *http.Request, accountID string) {
	if err := h.MultiAccountAggregator.RemoveAccount(accountID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Account not found", http.StatusNotFound)
		} else {
			log.Printf("Failed to delete account: %v", err)
			http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleEnableAccount enables an account
func (h *Handler) handleEnableAccount(w http.ResponseWriter, r *http.Request, accountID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.MultiAccountAggregator.EnableAccount(accountID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Account not found", http.StatusNotFound)
		} else {
			log.Printf("Failed to enable account: %v", err)
			http.Error(w, "Failed to enable account", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "enabled"}`))
}

// handleDisableAccount disables an account
func (h *Handler) handleDisableAccount(w http.ResponseWriter, r *http.Request, accountID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.MultiAccountAggregator.DisableAccount(accountID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Account not found", http.StatusNotFound)
		} else {
			log.Printf("Failed to disable account: %v", err)
			http.Error(w, "Failed to disable account", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "disabled"}`))
}

// LoginSMSHandler ...
func (h *Handler) LoginSMSHandler(w http.ResponseWriter, r *http.Request) {
	// log.Println("/sms")

	access, refresh, err := h.SendCode(r)
	if err != nil {
		log.Println("sms", err.Error())
	}

	h.Config.Token = access
	h.Config.RefreshToken = refresh
	if err = h.Config.WriteConfig(); err != nil {
		log.Println("error on write config file ", err)
	}

	if _, err := w.Write([]byte(access + " / " + refresh)); err != nil {
		log.Println("sms", err.Error())
	}
}
