package handlers

import (
	"compress/gzip"
	"embed"
	"io"
	"net/http"
	"strings"

	"github.com/ad/domru/config"
	"github.com/ad/domru/multiconfig"
)

type Handler struct {
	Config       *config.Config
	UserAccounts []Account
	Account      *Account

	// Multi-account support
	MultiAccountAggregator *multiconfig.MultiAccountAggregator
	LegacyAdapter         *multiconfig.LegacyConfigAdapter

	TemplateFs embed.FS
}

func NewHandlers(config *config.Config, templateFs embed.FS) (h *Handler) {
	h = &Handler{
		Config:     config,
		TemplateFs: templateFs,
	}

	return h
}

// NewMultiAccountHandlers creates a new handler with multi-account support
func NewMultiAccountHandlers(aggregator *multiconfig.MultiAccountAggregator, templateFs embed.FS) (h *Handler) {
	legacyAdapter := multiconfig.NewLegacyConfigAdapter(aggregator)
	
	// Get primary config for backward compatibility
	primaryConfig, err := legacyAdapter.GetPrimaryConfig()
	if err != nil {
		// Create empty config if no primary account exists
		primaryConfig = &config.Config{Port: 18000}
	}

	h = &Handler{
		Config:                 primaryConfig,
		MultiAccountAggregator: aggregator,
		LegacyAdapter:         legacyAdapter,
		TemplateFs:            templateFs,
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

// ReadResponseBody читает тело ответа с поддержкой gzip декомпрессии
func ReadResponseBody(resp *http.Response) ([]byte, error) {
	var reader io.Reader = resp.Body

	// Проверяем, сжат ли ответ gzip
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Encoding")), "gzip") {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	return io.ReadAll(reader)
}
