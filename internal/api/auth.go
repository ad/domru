package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ad/domru/internal/constants"
	"github.com/ad/domru/internal/models"
	"github.com/ad/domru/internal/upstream"
)

// Auth related endpoints encapsulation (Step 1 of refactor)

const (
	loginURLFmt       = "https://%s/auth/v2/login/%s"
	confirmURLFmt     = "https://%s/auth/v2/confirmation/%s"
	confirmSMSURLFmt  = "https://%s/auth/v2/auth/%s/confirmation"
	refreshSessionURL = "https://%s/auth/v2/session/refresh"
)

// Accounts retrieves accounts for login (phone)
func (w *Wrapper) Accounts(ctx context.Context, login string) ([]models.Account, error) {
	var out []models.Account
	url := fmt.Sprintf(loginURLFmt, constants.APIHost, login)
	r := upstream.New(url).WithMethod(http.MethodGet).WithContext(ctx)
	if err := r.Send(&out); err != nil {
		return nil, mapUpstreamErr(err)
	}
	return out, nil
}

// RequestCode triggers SMS code sending for selected account
func (w *Wrapper) RequestCode(ctx context.Context, login string, account models.Account) error {
	url := fmt.Sprintf(confirmURLFmt, constants.APIHost, login)
	r := upstream.New(url).WithMethod(http.MethodPost).WithJSONBody(account).WithContext(ctx)
	if err := r.Send(nil); err != nil {
		return mapUpstreamErr(err)
	}
	return nil
}

// ConfirmCode validates SMS code and returns tokens
func (w *Wrapper) ConfirmCode(ctx context.Context, login, code string, account models.Account) (string, string, error) {
	url := fmt.Sprintf(confirmSMSURLFmt, constants.APIHost, login)
	payload := models.ConfirmRequest{Confirm: code, SubscriberID: account.SubscriberID, Login: login, OperatorID: account.OperatorID, AccountID: account.AccountID, ProfileID: account.ProfileID}
	var resp models.ConfirmResponse
	r := upstream.New(url).WithMethod(http.MethodPost).WithJSONBody(payload).WithContext(ctx)
	if err := r.Send(&resp); err != nil {
		return "", "", mapUpstreamErr(err)
	}
	return resp.AccessToken, resp.RefreshToken, nil
}

// Refresh tokens using refresh token stored in config provider (handled outside normally).
func (w *Wrapper) Refresh(ctx context.Context, refreshToken string, operatorID int, ua string) (string, string, error) {
	r := upstream.New(fmt.Sprintf(refreshSessionURL, constants.APIHost)).WithMethod(http.MethodGet).WithContext(ctx)
	// Custom headers
	r.Set("Bearer", refreshToken)
	if ua != "" {
		r.Set("User-Agent", ua)
	}
	var out models.ConfirmResponse
	if err := r.Send(&out); err != nil {
		return "", "", mapUpstreamErr(err)
	}
	return out.AccessToken, out.RefreshToken, nil
}
