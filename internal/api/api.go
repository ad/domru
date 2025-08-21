package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ad/domru/internal/constants"
	"github.com/ad/domru/internal/models"
	"github.com/ad/domru/internal/upstream"
)

var baseURL = "https://" + constants.APIHost

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Wrapper groups high-level API calls
type DeviceInfo interface {
	GetOperatorID() int
	GetUUID() string
}

type Wrapper struct {
	client HTTPDoer
	logger Logger
	device DeviceInfo
}

func New(client HTTPDoer) *Wrapper                  { return &Wrapper{client: client} }
func (w *Wrapper) WithLogger(l Logger) *Wrapper     { w.logger = l; return w }
func (w *Wrapper) WithDevice(d DeviceInfo) *Wrapper { w.device = d; return w }

// Logger lightweight interface to avoid dependency on concrete logging lib.
type Logger interface {
	Info(msg string, kv ...any)
	Error(msg string, kv ...any)
}

func (w *Wrapper) fetchJSON(ctx context.Context, url string, out interface{}, placeID int64) error {
	req := upstream.New(url).WithMethod(http.MethodGet).WithContext(ctx)
	if uc, ok := w.client.(*http.Client); ok {
		req.WithClient(uc)
	}
	if w.logger != nil {
		req.WithLogger(w.logger)
	}
	// dynamic headers
	if w.device != nil {
		ua := buildUserAgent(w.device.GetOperatorID(), w.device.GetUUID(), placeID)
		req.Set("User-Agent", ua)
	}
	req.Set("Host", constants.APIHost)
	if err := req.Send(out); err != nil {
		return mapUpstreamErr(err)
	}
	return nil
}

func buildUserAgent(operatorID int, uuid string, placeID int64) string {
	if placeID == 0 {
		placeID = 1
	}
	return fmt.Sprintf("%s | | %d | %s | %d", constants.BaseUserAgentCore, operatorID, uuid, placeID)
}

func (w *Wrapper) Cameras(ctx context.Context) (models.Cameras, error) {
	var out models.Cameras
	err := w.fetchJSON(ctx, fmt.Sprintf("%s/rest/v1/forpost/cameras", baseURL), &out, 0)
	return out, err
}

func (w *Wrapper) Places(ctx context.Context) (models.Places, error) {
	var out models.Places
	err := w.fetchJSON(ctx, fmt.Sprintf("%s/rest/v1/subscriberplaces", baseURL), &out, 0)
	return out, err
}

func (w *Wrapper) Finances(ctx context.Context) (models.Finances, error) {
	var out models.Finances
	err := w.fetchJSON(ctx, fmt.Sprintf("%s/rest/v1/subscribers/profiles/finances", baseURL), &out, 0)
	return out, err
}

func (w *Wrapper) Events(ctx context.Context, placeID string) (models.EventsInputModel, error) {
	var out models.EventsInputModel
	err := w.fetchJSON(ctx, fmt.Sprintf("%s/rest/v1/places/%s/events?allowExtentedActions=true", baseURL, placeID), &out, 0)
	return out, err
}

func (w *Wrapper) StreamURL(cameraID string, q url.Values) string {
	u := fmt.Sprintf("%s/rest/v1/forpost/cameras/%s/video", baseURL, cameraID)
	if len(q) > 0 {
		return u + "?" + q.Encode()
	}
	return u
}

func (w *Wrapper) SnapshotURL(placeID, accessControlID string) string {
	return fmt.Sprintf("%s/rest/v1/places/%s/accesscontrols/%s/videosnapshots", baseURL, placeID, accessControlID)
}

func (w *Wrapper) OpenDoorURL(placeID, accessControlID string) string {
	return fmt.Sprintf("%s/rest/v1/places/%s/accesscontrols/%s/actions", baseURL, placeID, accessControlID)
}

// Operators returns raw operators JSON (simple pass-through, rarely used)
func (w *Wrapper) Operators(ctx context.Context) (map[string]any, error) {
	url := fmt.Sprintf("%s/public/v1/operators", baseURL)
	var out map[string]any
	if err := w.fetchJSON(ctx, url, &out, 0); err != nil {
		return nil, err
	}
	return out, nil
}

// OpenDoor triggers an access control action
func (w *Wrapper) OpenDoor(ctx context.Context, placeID, accessControlID string) error {
	url := w.OpenDoorURL(placeID, accessControlID)
	payload := map[string]string{"name": "accessControlOpen"}
	req := upstream.New(url).WithMethod(http.MethodPost).WithJSONBody(payload).WithContext(ctx)
	if uc, ok := w.client.(*http.Client); ok {
		req.WithClient(uc)
	}
	if w.logger != nil {
		req.WithLogger(w.logger)
	}
	pid, _ := strconv.ParseInt(placeID, 10, 64)
	if w.device != nil {
		req.Set("User-Agent", buildUserAgent(w.device.GetOperatorID(), w.device.GetUUID(), pid))
	}
	req.Set("Host", constants.APIHost)
	if err := req.Send(nil); err != nil {
		return mapUpstreamErr(err)
	}
	return nil
}

// Snapshot fetches snapshot bytes and returns them (content-type assumed image/jpeg upstream)
func (w *Wrapper) Snapshot(ctx context.Context, placeID, accessControlID string) ([]byte, *upstream.UpstreamError, error) {
	url := w.SnapshotURL(placeID, accessControlID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	pid, _ := strconv.ParseInt(placeID, 10, 64)
	if w.device != nil {
		req.Header.Set("User-Agent", buildUserAgent(w.device.GetOperatorID(), w.device.GetUUID(), pid))
	}
	req.Host = constants.APIHost
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, &upstream.UpstreamError{StatusCode: resp.StatusCode, Body: string(data)}, nil
	}
	return data, nil, nil
}
