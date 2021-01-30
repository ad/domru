package handlers

import (
	"net/http"

	"github.com/ad/domru/config"
)

type Handler struct {
	Config       *config.Config

	Client *http.Client
}

func NewHandlers(config *config.Config) (h *Handler) {
	h = &Handler{
		Config: config,
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
