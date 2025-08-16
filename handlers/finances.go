package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Finances struct {
	Balance     float64 `json:"balance"`
	BlockType   string  `json:"blockType"`
	AmountSum   float64 `json:"amountSum"`
	TargetDate  string  `json:"targetDate"`
	PaymentLink string  `json:"paymentLink"`
	Blocked     bool    `json:"blocked"`
}

// Finances ...
func (h *Handler) Finances() ([]byte, error) {
	var (
		body   []byte
		err    error
		client = http.DefaultClient
	)

	url := API_FINANCES

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			log.Println(err2)
		}
	}()

	if resp.StatusCode == 409 { // Conflict (tokent already expired)
		return []byte("token can't be refreshed"), nil
	}

	if body, err = io.ReadAll(resp.Body); err != nil {
		return nil, err
	}

	return body, nil
}

// FinancesHandler ...
func (h *Handler) FinancesHandler(w http.ResponseWriter, r *http.Request) {
	// log.Println("/financesHandler")

	data, err := h.Finances()
	if err != nil {
		log.Println("financesHandler", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write(data); err != nil {
		log.Println("financesHandler", err.Error())
	}
}

func (h *Handler) GetFinances() (*Finances, error) {
	finances := &Finances{}

	data, err := h.Finances()
	if err != nil {
		return finances, err
	}

	if err = json.Unmarshal(data, &finances); err != nil {
		return finances, fmt.Errorf("error on unmarshal Finances %q", err.Error())
	}

	return finances, nil
}
