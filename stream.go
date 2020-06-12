package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func stream() ([]byte, error) {
	var (
		body []byte
		err  error
	)

	url := fmt.Sprintf(API_CAMERA_GET_STREAM, 284945)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return body, err
	}

	rt := WithHeader(client.Transport)
	rt.Set("Content-Type", "application/json; charset=UTF-8")
	rt.Set("Operator", operator)
	rt.Set("Authorization", "Bearer "+*token)
	client.Transport = rt

	resp, err := client.Do(request)
	if err != nil {
		return body, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 409 {
		body = []byte("token can't be refreshed")
	}

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return body, err
	}

	return body, nil
}

func streamHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/financesHandler")

	data, err := stream()
	if err != nil {
		data = []byte(err.Error())
		log.Println("financesHandler", err.Error())
	}

	if _, err := w.Write(data); err != nil {
		log.Println("financesHandler", err.Error())
	}
}
