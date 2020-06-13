package handlers

import (
	"net/http"
)

type Handler struct {
	Addr         *string
	Token        *string
	RefreshToken *string
	Login        *string
	Password     *string
	Operator     *string

	Client *http.Client
}

func NewHandlers(addr, token, refreshToken, login, password, operator *string) (h *Handler) {
	h = &Handler{
		Addr:         addr,
		Token:        token,
		RefreshToken: refreshToken,
		Login:        login,
		Password:     password,
		Operator:     operator,

		Client: http.DefaultClient,
	}

	return h
}

// Header ...
type Header struct {
	http.Header
	rt http.RoundTripper
}

// WithHeader ...
func WithHeader(rt http.RoundTripper) Header {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return Header{Header: make(http.Header), rt: rt}
}

// RoundTrip ...
func (h Header) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range h.Header {
		req.Header[k] = v
	}

	return h.rt.RoundTrip(req)
}
