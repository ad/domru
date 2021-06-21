package handlers

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Refresh ...
func (h *Handler) Refresh(refreshToken *string) (string, string, error) {
	var (
		body   []byte
		err    error
		client = http.DefaultClient
	)

	url := API_REFRESH_SESSION
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	request = request.WithContext(ctx)

	operator := strconv.Itoa(h.Config.Operator)

	rt := WithHeader(client.Transport)
	rt.Set("Content-Type", "application/json; charset=UTF-8")
	rt.Set("Operator", operator)
	rt.Set("Bearer", h.Config.RefreshToken)
	client.Transport = rt

	resp, err := client.Do(request)
	if err != nil {
		return "", "", err
	}

	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			log.Println(err2)
		}
	}()

	// log.Printf("%#v", resp)

	if resp.StatusCode == 409 { // Conflict (tokent already expired)
		return "token can't be refreshed", "", nil
	}

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return "", "", err
	}

	var authResp ConfirmResponse
	if err = json.Unmarshal(body, &authResp); err != nil {
		return "", "", err
	}

	return authResp.AccessToken, authResp.RefreshToken, nil
}
