package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
)

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
			defer x.Close()
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
		return "", fmt.Errorf("writerClose: %v", err)
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
		defer req.Body.Close()
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

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
		return "", fmt.Errorf("Json parse error: %s", err)
	}

	h.Token = &authResp.Data.AccessToken
	h.RefreshToken = &authResp.Data.RefreshToken

	return authResp.Data.AccessToken, nil
}

// AuthHandler ...
func (h *Handler) AuthHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/authHandler")

	data, err := h.Auth(h.Login, h.Password)
	if err != nil {
		log.Println("authHandler", err.Error())
	}
	if _, err := w.Write([]byte(data)); err != nil {
		log.Println("authHandler", err.Error())
	}
}
