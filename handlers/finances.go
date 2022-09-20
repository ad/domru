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
	rt.Set("Content-Type", "application/json; charset=UTF-8")
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

// Finances ...
func (h *Handler) Crash() ([]byte, error) {
	var (
		body   []byte
		err    error
		client = http.DefaultClient
	)

	url := "https://api-profile.dom.ru/v1/ppr"

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	request = request.WithContext(ctx)

	// operator := strconv.Itoa(h.Config.Operator)

	rt := WithHeader(client.Transport)
	rt.Set("Content-Type", "application/json; charset=UTF-8")
	rt.Set("Domain", "interzet")
	// rt.Set("Authorization", "Bearer "+h.Config.Token)
	rt.Set("Authorization", "Bearer eyAiYWxnIjogIlNIMSIsICJ0eXAiOiAiSldUIiwgImsiOiAiMzAiIH0.2eHTEMGZNc1el2Rru5BPLUhy7f9sQOU9_9gCoQL3NBix7xmZe_pciOOzOXMG7hPYD1EU4cPP3jialcej2Z9s8Ds4j8Tuhqg3LQ_F4STPzNHKMgPa__gSbUbYwJ1zHml0M6bGby911L78jqRZ2JU7qg1EI7owqTTSFqst_5b6ldAHcoHonreWmDfwDOAZl2lo0VrAfEQMVC_Z8nggv1jT1Q1Qq6ntBFjetwB5GX83teilLN9i7XhJM1jxSWBugM-jPYcIAoLxHF9PwC3vxadepYqKjYVW_oLvtfdOWbGR25WPZXFPqzVE8oJiILtaaA-AfqGK6yXV4q-lrTm-OUepUg.QThEQTk2OEQwQ0JFMEQxREI3MDcwMDgzOTQyRkJDQzY4MDdFOUQ5Mw")
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

	data, err := h.Crash()
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
