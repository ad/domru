package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// HANetwork ...
func (h *Handler) HANetwork() (string, error) {
	var (
		body             []byte
		err              error
		client           = &http.Client{}
		supervisor_token string
	)

	if val, ok := os.LookupEnv("SUPERVISOR_TOKEN"); ok {
		supervisor_token = val
		log.Printf("supervisor_token %s", supervisor_token)
	} else {
		return "", fmt.Errorf("supervisor token not set")
	}

	url := API_HA_NETWORK

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	request = request.WithContext(ctx)

	request.Header = http.Header{
		"Content-Type":  []string{"application/json; charset=UTF-8"},
		"Authorization": []string{"Bearer " + supervisor_token},
	}

	resp, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("supervisor ip request %s", err.Error())
	}

	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			log.Println(err2)
		}
	}()

	// log.Printf("%+v", resp)

	if body, err = io.ReadAll(resp.Body); err != nil {
		return "", fmt.Errorf("supervisor ip ReadAll %s", err.Error())
	}

	// log.Println(string(body))

	var haconfig HAConfig

	if err := json.Unmarshal(body, &haconfig); err != nil {
		return "", fmt.Errorf("supervisor ip Unmarshal %s", err.Error())
	}

	if haconfig.Result == "ok" && len(haconfig.Data.Interfaces) > 0 {
		address := strings.Split(haconfig.Data.Interfaces[0].Ipv4.Address[0], "/")
		return address[0], nil
	}

	return "", fmt.Errorf("supervisor ip not found")
}

// // HANetworkHandler ...
// func (h *Handler) HANetworkHandler(w http.ResponseWriter, r *http.Request) {
// 	log.Println("/HANetworkHandler")

// 	data, err := h.HANetwork()
// 	if err != nil {
// 		log.Println("HANetworkHandler", err.Error())
// 	}

// 	w.Header().Set("Content-Type", "application/json")

// 	if _, err := w.Write([]byte(data)); err != nil {
// 		log.Println("HANetworkHandler", err.Error())
// 	}
// }
