package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Door ...
func (h *Handler) Door(r *http.Request) (string, error) {
	var (
		body   []byte
		err    error
		client = http.DefaultClient
	)

	type doorData struct {
		Name string `json:"name"`
	}

	buf := new(bytes.Buffer)
	if err = json.NewEncoder(buf).Encode(&doorData{Name: "accessControlOpen"}); err != nil {
		return "", err
	}

	query := r.URL.Query()
	placeID := query.Get("placeID")
	accessControlID := query.Get("accessControlID")

	url := fmt.Sprintf(API_OPEN_DOOR, placeID, accessControlID)

	request, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	request = request.WithContext(ctx)

	operator := strconv.Itoa(h.Config.Operator)

	// Конвертируем placeID из строки в int64
	placeIDInt, _ := strconv.ParseInt(placeID, 10, 64)

	rt := WithHeader(client.Transport)
	rt.Set("Host", API_HOST)
	rt.Set("Content-Type", "application/json; charset=UTF-8")
	rt.Set("User-Agent", GenerateUserAgent(h.Config.Operator, h.Config.UUID, placeIDInt))
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

	if body, err = ReadResponseBody(resp); err != nil {
		return "", err
	}

	return string(body), nil
}

// DoorHandler ...
func (h *Handler) DoorHandler(w http.ResponseWriter, r *http.Request) {
	// log.Println("/doorHandler")

	data, err := h.Door(r)
	if err != nil {
		data = err.Error()
		log.Println("doorHandler", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write([]byte(data)); err != nil {
		log.Println("doorHandler", err.Error())
	}
}
