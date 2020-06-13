package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Door ...
func (h *Handler) Door() (string, error) {
	var (
		body   []byte
		err    error
		client = h.Client
	)

	type doorData struct {
		Name string `json:"name"`
	}

	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(&doorData{Name: "accessControlOpen"})

	url := fmt.Sprintf(API_OPEN_DOOR, 936129, 5351)
	request, err := http.NewRequest("POST", url, buf)
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

// DoorHandler ...
func (h *Handler) DoorHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/doorHandler")

	data, err := h.Door()
	if err != nil {
		data = err.Error()
		log.Println("doorHandler", err.Error())
	}
	if _, err := w.Write([]byte(data)); err != nil {
		log.Println("doorHandler", err.Error())
	}
}
