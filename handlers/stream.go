package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Stream ...
func (h *Handler) Stream(r *http.Request) (string, error) {
	var (
		body     string
		respBody []byte
		err      error
		client   = http.DefaultClient
	)

	query := r.URL.Query()
	cameraID := query.Get("cameraID")

	targetRawURL := fmt.Sprintf(API_CAMERA_GET_STREAM, cameraID)

	targetURL, _ := url.Parse(targetRawURL)
	targetURL.RawQuery = r.URL.RawQuery

	request, err := http.NewRequest("GET", targetURL.String(), nil)
	if err != nil {
		return body, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	request = request.WithContext(ctx)

	operator := strconv.Itoa(h.Config.Operator)

	rt := WithHeader(client.Transport)
	rt.Set("Host", API_HOST)
	rt.Set("Content-Type", "application/json; charset=UTF-8")
	rt.Set("User-Agent", GenerateUserAgent(h.Config.Operator, h.Config.UUID, 0))
	rt.Set("Operator", operator)
	rt.Set("Authorization", "Bearer "+h.Config.Token)
	client.Transport = rt

	resp, err := client.Do(request)
	if err != nil {
		return body, err
	}

	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			log.Println(err2)
		}
	}()

	if resp.StatusCode == 409 {
		return "token can't be refreshed", nil
	}

	if respBody, err = io.ReadAll(resp.Body); err != nil {
		return string(respBody), err
	}

	type streamResponse struct {
		Data struct {
			URL string `json:"URL"`
		} `json:"data"`
	}

	var streamResp streamResponse
	err = json.Unmarshal(respBody, &streamResp)
	if err != nil {
		return "", fmt.Errorf("json parse error: %w", err)
	}

	return streamResp.Data.URL, nil
}

// StreamHandler ...
func (h *Handler) StreamHandler(w http.ResponseWriter, r *http.Request) {
	// log.Println("/streamHandler")

	data, err := h.Stream(r)
	if err != nil {
		data = err.Error()
		log.Println("streamHandler", err.Error())

		if _, err := w.Write([]byte(data)); err != nil {
			log.Println("streamHandler", err.Error())
		}

		return
	}

	http.Redirect(w, r, data, http.StatusFound)
}
