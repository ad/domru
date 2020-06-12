package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

func snapshotHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/snapshotHandler")

	url := fmt.Sprintf(API_VIDEO_SNAPSHOT, 936129, 5351)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("snapshotHandler", err)
		return
	}

	rt := WithHeader(client.Transport)
	rt.Set("Authorization", "Bearer "+*token)
	rt.Set("Operator", operator)
	client.Transport = rt

	resp, err := client.Do(request)
	if err != nil {
		log.Println("snapshotHandler", "connect error")
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	if _, err := w.Write(body); err != nil {
		log.Println("snapshotHandler", "unable to write image.")
	}
}
