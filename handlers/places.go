package handlers

import (
	"context"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Places ...
func (h *Handler) Places() (string, error) {
	var (
		body   []byte
		err    error
		client = http.DefaultClient
	)

	url := API_SUBSCRIBER_PLACES

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
		return "", err
	}

	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			log.Println(err2)
		}
	}()

	// log.Printf("Places: %s %#v", h.Config.Token, resp)

	if resp.StatusCode == 409 { // Conflict (tokent already expired)
		return "token can't be refreshed", nil
	}

	if body, err = io.ReadAll(resp.Body); err != nil {
		return "", err
	}

	return string(body), nil
}

// PlacesHandler ...
func (h *Handler) PlacesHandler(w http.ResponseWriter, r *http.Request) {
	// log.Println("/placesHandler")

	data, err := h.Places()
	if err != nil {
		data = err.Error()
		log.Println("placesHandler", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write([]byte(data)); err != nil {
		log.Println("placesHandler", err.Error())
	}
}
