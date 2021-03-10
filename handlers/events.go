package handlers

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	request = request.WithContext(ctx)

	rt := WithHeader(client.Transport)
	rt.Set("Content-Type", "application/json; charset=UTF-8")
	rt.Set("Operator", h.Config.Operator)
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

	log.Printf("%#v", resp)

	if resp.StatusCode == 409 { // Conflict (tokent already expired)
		return "token can't be refreshed", nil
	}

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return "", err
	}

	return string(body), nil
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
