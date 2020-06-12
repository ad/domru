package main

import (
	"flag"
	"log"
	"net/http"
)

var (
	addr         *string
	token        *string
	refreshToken *string
	login        *string
	password     *string
	operator     = "2"

	client = http.DefaultClient
)

func main() {
	addr = flag.String("addr", ":8080", "listen address")
	token = flag.String("token", "", "dom.ru token")
	refreshToken = flag.String("reshresh", "", "dom.ru refresh token")
	login = flag.String("login", "", "dom.ru login")
	password = flag.String("password", "", "dom.ru password")
	flag.Parse()

	if *token != "" || *refreshToken != "" {
		if *refreshToken != "" {
			data, err := refresh(refreshToken)
			if err != nil {
				data = err.Error()
				log.Println("refresh token, error:", err.Error())
			} else {
				token = &data
			}
		}
	} else if *login != "" && *password != "" {
		data, err := auth(*login, *password)
		if err != nil {
			log.Println("login error", err.Error())
		} else {
			token = &data
		}
	} else {
		log.Fatal("auth/refresh token or login and password must be provided")
	}

	http.HandleFunc("/cameras", camerasHandler)
	http.HandleFunc("/door", doorHandler)
	http.HandleFunc("/finances", financesHandler)
	http.HandleFunc("/places", placesHandler)
	http.HandleFunc("/snapshot", snapshotHandler)
	http.HandleFunc("/stream", streamHandler)
	http.HandleFunc("/token", tokenHandler)
	http.HandleFunc("/auth", authHandler)

	log.Println("start listening on", *addr, "with token", *token)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

// Header ...
type Header struct {
	http.Header
	rt http.RoundTripper
}

// WithHeader ...
func WithHeader(rt http.RoundTripper) Header {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return Header{Header: make(http.Header), rt: rt}
}

// RoundTrip ...
func (h Header) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range h.Header {
		req.Header[k] = v
	}

	return h.rt.RoundTrip(req)
}
