package auth

import (
	"errors"
	"net/http"
	"strconv"
)

// TokenProvider supplies current access token
type TokenProvider interface{ GetToken() (string, error) }

// TokenRefresher refreshes token when expired
type TokenRefresher interface{ RefreshToken() error }

// OperatorProvider supplies operator id
type OperatorProvider interface{ GetOperatorID() (int, error) }

// AutoClient injects Authorization / Operator headers and retries once after refresh.
type AutoClient struct {
	Base      *http.Client
	Tokens    TokenProvider
	Refresher TokenRefresher
	Operators OperatorProvider
	UserAgent string
}

func NewAutoClient(base *http.Client, tp TokenProvider, tr TokenRefresher, op OperatorProvider, ua string) *AutoClient {
	if base == nil {
		base = http.DefaultClient
	}
	return &AutoClient{Base: base, Tokens: tp, Refresher: tr, Operators: op, UserAgent: ua}
}

func (c *AutoClient) Do(req *http.Request) (*http.Response, error) {
	if err := c.decorate(req); err != nil {
		return nil, err
	}
	resp, err := c.Base.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == 409 {
		// try refresh once
		resp.Body.Close()
		if c.Refresher != nil {
			if rErr := c.Refresher.RefreshToken(); rErr == nil {
				if err := c.decorate(req); err != nil {
					return nil, err
				}
				return c.Base.Do(req)
			}
		}
	}
	return resp, nil
}

func (c *AutoClient) decorate(req *http.Request) error {
	token, err := c.Tokens.GetToken()
	if err != nil {
		return err
	}
	op, err := c.Operators.GetOperatorID()
	if err != nil {
		return err
	}
	if token == "" {
		return errors.New("empty token")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Operator", strconv.Itoa(op))
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	return nil
}
