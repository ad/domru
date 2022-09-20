package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

type EventsInputModel struct {
	Data []struct {
		ID            string `json:"id,omitempty"`
		PlaceID       int    `json:"placeId,omitempty"`
		EventTypeName string `json:"eventTypeName,omitempty"`
		Timestamp     string `json:"timestamp,omitempty"`
		Message       string `json:"message,omitempty"`
		Source        struct {
			Type string `json:"type,omitempty"`
			ID   int    `json:"id,omitempty"`
		} `json:"source,omitempty"`
		Value struct {
			Type  string `json:"type,omitempty"`
			Value bool   `json:"value,omitempty"`
		} `json:"value,omitempty"`
		EventStatusValue interface{}   `json:"eventStatusValue,omitempty"`
		Actions          []interface{} `json:"actions,omitempty"`
	} `json:"data,omitempty"`
}

// Events ...
func (h *Handler) Events(w http.ResponseWriter, r *http.Request) (string, error) {
	var (
		body   []byte
		err    error
		client = http.DefaultClient
	)

	query := r.URL.Query()
	placeID := query.Get("placeID")

	if placeID == "" {
		return "provide placeID", fmt.Errorf("%s", "provide placeID")
	}

	url := fmt.Sprintf(API_EVENTS, placeID)
	// log.Println("/eventsHandler", url)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	request = request.WithContext(ctx)

	operator := strconv.Itoa(h.Config.Operator)

	rt := WithHeader(client.Transport)
	rt.Set("Content-Type", "application/json; charset=UTF-8")
	rt.Set("Operator", operator)
	rt.Set("Authorization", "Bearer "+h.Config.Token)
	client.Transport = rt

	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}

	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			log.Println(err2)
		}
	}()

	// log.Printf("%#v", resp)

	if resp.StatusCode == 409 { // Conflict (tokent already expired)
		return "token can't be refreshed", nil
	}

	if body, err = io.ReadAll(resp.Body); err != nil {
		return "", err
	}

	return string(body), nil
}

// Events ...
func (h *Handler) LastEvent(w http.ResponseWriter, r *http.Request) (events EventsInputModel, err error) {
	var (
		body   []byte
		client = http.DefaultClient
	)

	query := r.URL.Query()
	placeID := query.Get("placeID")

	if placeID == "" {
		return events, fmt.Errorf("%s", "provide placeID")
	}

	url := fmt.Sprintf(API_EVENTS, placeID)
	// log.Println("/eventsHandler", url)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return events, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	request = request.WithContext(ctx)

	operator := strconv.Itoa(h.Config.Operator)

	rt := WithHeader(client.Transport)
	rt.Set("Content-Type", "application/json; charset=UTF-8")
	rt.Set("Operator", operator)
	rt.Set("Authorization", "Bearer "+h.Config.Token)
	client.Transport = rt

	resp, err := client.Do(request)
	if err != nil {
		return events, err
	}

	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			log.Println(err2)
		}
	}()

	// log.Printf("%#v", resp)

	if resp.StatusCode == 409 { // Conflict (tokent already expired)
		return events, fmt.Errorf("%s", "token can't be refreshed")
	}

	if body, err = io.ReadAll(resp.Body); err != nil {
		return events, err
	}

	if err := json.Unmarshal(body, &events); err != nil {
		return events, fmt.Errorf("json parse error: %q", err)

	}

	return events, nil
}

// EventsHandler ...
func (h *Handler) EventsHandler(w http.ResponseWriter, r *http.Request) {
	data, err := h.Events(w, r)
	if err != nil {
		log.Println("eventsHandler", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write([]byte(data)); err != nil {
		log.Println("eventsHandler", err.Error())
	}
}

// EventsHandler ...
func (h *Handler) LastEventHandler(w http.ResponseWriter, r *http.Request) {
	data, err := h.LastEvent(w, r)
	if err != nil {
		log.Println("lastEventHandler", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")

	if len(data.Data) > 0 {
		b, err := json.Marshal(data.Data[0])
		if err != nil {
			log.Println("lastEventHandler", err.Error())
		}

		if _, err := w.Write(b); err != nil {
			log.Println("lastEventHandler", err.Error())
		}

		return
	}

	if _, err := w.Write([]byte(`{"error": "events not found"}`)); err != nil {
		log.Println("lastEventHandler", err.Error())
	}
}
