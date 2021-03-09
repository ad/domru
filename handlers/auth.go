package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type Account struct {
	OperatorID   int64  `json:"operatorId"`
	SubscriberID int64  `json:"subscriberId"`
	AccountID    string `json:"accountId"`
	PlaceID      int64  `json:"placeId"`
	Address      string `json:"address"`
	ProfileID    string `json:"profileId"`
}

// Auth ...
func (h *Handler) Auth(username, password *string) (string, error) {
	var (
		err      error
		respBody []byte
		body     bytes.Buffer
		client   = h.Client

		values = map[string]io.Reader{
			"username":   strings.NewReader(*username),
			"password":   strings.NewReader(*password),
			"rememberMe": strings.NewReader("1"),
		}
	)

	writer := multipart.NewWriter(&body)

	for key, r := range values {
		var fw io.Writer
		if x, ok := r.(io.Closer); ok {
			defer func() {
				err2 := x.Close()
				if err2 != nil {
					log.Println(err2)
				}
			}()
		}

		if fw, err = writer.CreateFormField(key); err != nil {
			return "", err
		}
		if _, err = io.Copy(fw, r); err != nil {
			return "", err
		}

	}

	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("writerClose: %w", err)
	}

	req, err := http.NewRequest("POST", API_AUTH, &body)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:77.0) Gecko/20100101 Firefox/77.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", writer.FormDataContentType()) //
	req.Header.Set("Referer", "https://interzet.domru.ru/user/login")
	req.Header.Set("Domain", "interzet")
	req.Header.Set("Host", "api-auth.domru.ru")
	req.Header.Set("Origin", "https://interzet.domru.ru")
	req.Header.Set("TE", "Trailers")

	if req.Close && req.Body != nil {
		defer func() {
			err2 := req.Body.Close()
			if err2 != nil {
				log.Println(err2)
			}
		}()
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			log.Println(err2)
		}
	}()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("wrong response code: %d", resp.StatusCode)
	}

	type authResponse struct {
		Status int `json:"status"`
		Data   struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}

	if respBody, err = ioutil.ReadAll(resp.Body); err != nil {
		return string(respBody), err
	}

	var authResp authResponse
	err = json.Unmarshal(respBody, &authResp)
	if err != nil {
		return "", fmt.Errorf("Json parse error: %w", err)
	}

	h.Config.Token = authResp.Data.AccessToken
	h.Config.RefreshToken = authResp.Data.RefreshToken

	return authResp.Data.AccessToken, nil
}

// Accounts ...
func (h *Handler) Accounts(username *string) (a *Account, err error) {
	var (
		body   []byte
		client = h.Client
	)

	url := fmt.Sprintf(API_AUTH_LOGIN, *username)
	log.Println("/accountsHandler", url)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*30))
	defer cancel()
	request = request.WithContext(ctx)

	rt := WithHeader(client.Transport)
	rt.Set("Host", "myhome.novotelecom.ru")
	rt.Set("Content-Type", "application/json")
	rt.Set("Connection", "keep-alive")
	rt.Set("Accept", "*/*")
	rt.Set("User-Agent", API_USER_AGENT)
	rt.Set("Authorization", "")
	rt.Set("Accept-Language", "en-us")
	rt.Set("Accept-Encoding", "gzip, deflate, br")

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
		return nil, fmt.Errorf("token can't be refreshed")
	}

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return nil, err
	}

	var accounts []Account
	if err = json.Unmarshal(body, &accounts); err != nil {
		return nil, err
	}

	for i := range accounts {
		if accounts[i].AccountID != "" {
			return &accounts[i], nil
		}
	}

	return nil, fmt.Errorf("account not found")
}

// AuthHandler ...
func (h *Handler) AuthHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/authHandler")

	data, err := h.Auth(&h.Config.Login, &h.Config.Password)
	if err != nil {
		log.Println("authHandler", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write([]byte(data)); err != nil {
		log.Println("authHandler", err.Error())
	}
}

// AccountsHandler ...
func (h *Handler) AccountsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/accountsHandler")

	data, err := h.Accounts(&h.Config.Login)
	if err != nil {
		log.Println("accountsHandler", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")

	b, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return
	}

	if _, err := w.Write([]byte(b)); err != nil {
		log.Println("accountsHandler", err.Error())
	}
}
