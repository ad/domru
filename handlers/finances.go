package handlers

import (
	"io/ioutil"
	"log"
	"net/http"
)

// Finances ...
func (h *Handler) Finances() (string, error) {
	var (
		body   []byte
		err    error
		client = h.Client
	)

	url := API_FINANCES
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

	if resp.StatusCode == 409 { // Conflict (tokent already expired)
		body = []byte("token can't be refreshed")
	}

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return "", err
	}

	return string(body), nil
}

// FinancesHandler ...
func (h *Handler) FinancesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/financesHandler")

	data, err := h.Finances()
	if err != nil {
		data = err.Error()
		log.Println("financesHandler", err.Error())
	}
	if _, err := w.Write([]byte(data)); err != nil {
		log.Println("financesHandler", err.Error())
	}
}
