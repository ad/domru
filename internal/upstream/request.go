package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/ad/domru/internal/constants"
)

// Default headers similar to mobile app
var defaultHeaders = map[string]string{
	"user-agent":      constants.BaseUserAgentCore,
	"content-type":    "application/json; charset=UTF-8",
	"connection":      "Keep-Alive",
	"accept-encoding": "gzip",
}

// Error from upstream service
type UpstreamError struct {
	StatusCode int
	Body       string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("upstream error: %d, body: %s", e.StatusCode, e.Body)
}

// Request builder
type Request struct {
	client  *http.Client
	url     string
	method  string
	body    []byte
	headers http.Header
	start   time.Time
	ctx     context.Context
	logger  Logger
}

func New(url string) *Request {
	h := http.Header{}
	for k, v := range defaultHeaders {
		h.Set(k, v)
	}
	return &Request{client: http.DefaultClient, url: url, headers: h, ctx: context.Background()}
}

func (r *Request) WithClient(c *http.Client) *Request { r.client = c; return r }
func (r *Request) WithMethod(m string) *Request       { r.method = m; return r }
func (r *Request) WithJSONBody(v interface{}) *Request {
	b, _ := json.Marshal(v)
	r.body = b
	return r
}
func (r *Request) Set(k, v string) *Request { r.headers.Set(k, v); return r }
func (r *Request) WithContext(ctx context.Context) *Request {
	if ctx != nil {
		r.ctx = ctx
	}
	return r
}
func (r *Request) WithLogger(l Logger) *Request { r.logger = l; return r }

// Logger interface (duplicated lightweight to avoid circular import). Implemented in higher layers.
type Logger interface {
	Info(msg string, kv ...any)
	Error(msg string, kv ...any)
}

func (r *Request) Send(out interface{}) error {
	if r.method == "" {
		r.method = http.MethodGet
	}
	var bodyReader io.Reader
	if len(r.body) > 0 {
		bodyReader = bytes.NewReader(r.body)
	}
	req, err := http.NewRequestWithContext(r.ctx, r.method, r.url, bodyReader)
	if err != nil {
		return err
	}
	for k, vals := range r.headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	start := time.Now()
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// gzip handled elsewhere if needed via transport; just read
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if r.logger != nil {
		r.logger.Info("upstream request", "method", r.method, "url", r.url, "dur", time.Since(start).String(), "status", resp.StatusCode)
	} else {
		log.Printf("upstream %s %s took %s status=%d", r.method, r.url, time.Since(start), resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return &UpstreamError{StatusCode: resp.StatusCode, Body: string(data)}
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("json decode: %w", err)
	}
	return nil
}
