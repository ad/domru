package handlers

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Events ...
func (h *Handler) Events(w http.ResponseWriter, r *http.Request) (string, error) {
	var (
		body   []byte
		err    error
		client = h.Client
	)

	query := r.URL.Query()
	placeID := query.Get("placeID")

	url := fmt.Sprintf(API_EVENTS, placeID)
	log.Println("/eventsHandler", url)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	rt := WithHeader(client.Transport)
	rt.Set("Content-Type", "application/json; charset=UTF-8")
	rt.Set("Operator", *h.Operator)
	rt.Set("Authorization", "Bearer "+*h.Token)
	client.Transport = rt

	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	log.Printf("%#v", resp)

	if resp.StatusCode == 409 { // Conflict (tokent already expired)
		body = []byte("token can't be refreshed")
	}

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return "", err
	}

	return string(body), nil
}

// EventsHandler ...
func (h *Handler) EventsHandler(w http.ResponseWriter, r *http.Request) {
	// log.Println("/eventsHandler")

	data, err := h.Events(w, r)
	if err != nil {
		log.Println("eventsHandler", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")
	
	if _, err := w.Write([]byte(data)); err != nil {
		log.Println("eventsHandler", err.Error())
	}
}