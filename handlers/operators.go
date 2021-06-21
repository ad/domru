package handlers

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// Operators ...
func (h *Handler) Operators() (string, error) {
	var (
		body   []byte
		err    error
		client = http.DefaultClient
	)

	url := API_OPERATORS

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	request = request.WithContext(ctx)

	rt := WithHeader(client.Transport)
	rt.Set("Content-Type", "application/json; charset=UTF-8")
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

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return "", err
	}

	return string(body), nil
}

// OperatorsHandler ...
func (h *Handler) OperatorsHandler(w http.ResponseWriter, r *http.Request) {
	// log.Println("/operators")

	data, err := h.Operators()
	if err != nil {
		data = err.Error()
		log.Println("operatorsHandler", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write([]byte(data)); err != nil {
		log.Println("operatorsHandler", err.Error())
	}
}
