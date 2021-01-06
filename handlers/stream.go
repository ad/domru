package handlers

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Stream ...
func (h *Handler) Stream(r *http.Request) ([]byte, error) {
	var (
		body   []byte
		err    error
		client = h.Client
	)

	query := r.URL.Query()
	cameraID := query.Get("cameraID")

	url := fmt.Sprintf(API_CAMERA_GET_STREAM, cameraID)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return body, err
	}

	rt := WithHeader(client.Transport)
	rt.Set("Content-Type", "application/json; charset=UTF-8")
	rt.Set("Operator", *h.Operator)
	rt.Set("Authorization", "Bearer "+*h.Token)
	client.Transport = rt

	resp, err := client.Do(request)
	if err != nil {
		return body, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 409 {
		body = []byte("token can't be refreshed")
	}

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return body, err
	}

	return body, nil
}

// StreamHandler ...
func (h *Handler) StreamHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/streamHandler")

	data, err := h.Stream(r)
	if err != nil {
		data = []byte(err.Error())
		log.Println("streamHandler", err.Error())
	}

	if _, err := w.Write(data); err != nil {
		log.Println("streamHandler", err.Error())
	}
}
