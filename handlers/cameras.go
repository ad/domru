package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Cameras ...
func (h *Handler) Cameras() (string, error) {
	var (
		body   []byte
		err    error
		client = http.DefaultClient
	)

	url := API_CAMERAS

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
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
		log.Printf("%+v %s %s", resp, operator, h.Config.Token)
		return "", err
	}

	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			log.Println(err2)
		}
	}()

	// log.Printf("%+v %s %s", resp, operator, h.Config.Token)

	if resp.StatusCode == 409 { // Conflict (tokent already expired)
		return "token can't be refreshed", nil
	}

	if body, err = ReadResponseBody(resp); err != nil {
		return "", err
	}

	return string(body), nil
}

// CamerasHandler ...
func (h *Handler) CamerasHandler(w http.ResponseWriter, r *http.Request) {
	// log.Println("/camerasHandler")

	w.Header().Set("Content-Type", "application/json")

	// Check if we have multi-account support
	if h.MultiAccountAggregator != nil {
		// Multi-account mode: return aggregated cameras from all accounts
		dataMerger := h.MultiAccountAggregator.GetDataMerger()
		cameras, err := dataMerger.GetAggregatedCameras()
		if err != nil {
			log.Println("multi-account camerasHandler", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert to JSON and return
		if jsonData, err := json.Marshal(cameras); err != nil {
			log.Println("camerasHandler JSON marshal error", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			if _, err := w.Write(jsonData); err != nil {
				log.Println("camerasHandler write error", err.Error())
			}
		}
	} else {
		// Legacy single-account mode
		data, err := h.Cameras()
		if err != nil {
			log.Println("camerasHandler", err.Error())
		}

		if _, err := w.Write([]byte(data)); err != nil {
			log.Println("camerasHandler", err.Error())
		}
	}
}
