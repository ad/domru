package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ad/domru/internal/constants"
)

type RefreshConfig interface {
	GetRefreshToken() string
	SetTokens(access, refresh string) error
	GetOperatorID() int
	GetUUID() string
}

type RefreshResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type TokenRefresherImpl struct {
	Cfg    RefreshConfig
	Client HTTPDoer
}

const refreshURL = "https://%s/auth/v2/session/refresh"

func (r *TokenRefresherImpl) RefreshToken() error {
	rtok := r.Cfg.GetRefreshToken()
	if rtok == "" {
		return fmt.Errorf("no refresh token")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(refreshURL, constants.APIHost), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Bearer", rtok)
	req.Header.Set("User-Agent", constants.BaseUserAgentCore)
	resp, err := r.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("refresh status %d", resp.StatusCode)
	}
	var rr RefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return err
	}
	return r.Cfg.SetTokens(rr.AccessToken, rr.RefreshToken)
}
