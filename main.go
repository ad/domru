package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/ad/domru/handlers"
)

var (
	addr         *string
	token        *string
	refreshToken *string
	login        *string
	password     *string
	operator     *string
)

func main() {
	addr = flag.String("addr", ":8080", "listen address")
	token = flag.String("token", "", "dom.ru token")
	refreshToken = flag.String("reshresh", "", "dom.ru refresh token")
	login = flag.String("login", "", "dom.ru login")
	password = flag.String("password", "", "dom.ru password")
	operator = flag.String("operator", "", "dom.ru operator")
	flag.Parse()

	h := handlers.NewHandlers(addr, token, refreshToken, login, password, operator)

	if *token != "" || *refreshToken != "" {
		if *refreshToken != "" {
			data, err := h.Refresh(h.RefreshToken)
			if err != nil {
				data = err.Error()
				log.Println("refresh token, error:", err.Error())
			} else {
				h.Token = &data
			}
		}
	} else if *login != "" && *password != "" {
		data, err := h.Auth(h.Login, h.Password)
		if err != nil {
			log.Println("login error", err.Error())
		} else {
			token = &data
		}
	} else {
		panic("auth/refresh token or login and password must be provided")
	}

	http.HandleFunc("/cameras", h.CamerasHandler)
	http.HandleFunc("/door", h.DoorHandler)
	http.HandleFunc("/events", h.EventsHandler)
	http.HandleFunc("/finances", h.FinancesHandler)
	http.HandleFunc("/operators", h.OperatorsHandler)
	http.HandleFunc("/places", h.PlacesHandler)
	http.HandleFunc("/snapshot", h.SnapshotHandler)
	http.HandleFunc("/stream", h.StreamHandler)
	http.HandleFunc("/token", h.TokenHandler)
	http.HandleFunc("/auth", h.AuthHandler)

	log.Println("start listening on", *addr, "with token", *token)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
