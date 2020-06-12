package main

import (
	"io/ioutil"
	"log"
	"net/http"
)

func places() (string, error) {
	var (
		body []byte
		err  error
	)

	url := API_SUBSCRIBER_PLACES
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	rt := WithHeader(client.Transport)
	rt.Set("Content-Type", "application/json; charset=UTF-8")
	rt.Set("Operator", operator)
	rt.Set("Authorization", "Bearer "+*token)
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

func placesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/placesHandler")

	data, err := places()
	if err != nil {
		data = err.Error()
		log.Println("placesHandler", err.Error())
	}
	if _, err := w.Write([]byte(data)); err != nil {
		log.Println("placesHandler", err.Error())
	}
}
