package handlers

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// Cameras ...
func (h *Handler) Cameras() (string, error) {
	var (
		body   []byte
		err    error
		client = h.Client
	)

	url := API_CAMERAS
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*30))
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

// CamerasHandler ...
func (h *Handler) CamerasHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/camerasHandler")

	data, err := h.Cameras()
	if err != nil {
		log.Println("camerasHandler", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write([]byte(data)); err != nil {
		log.Println("camerasHandler", err.Error())
	}
}
