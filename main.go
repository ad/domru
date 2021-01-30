package main

import (
	"log"
	"net/http"

	"github.com/ad/domru/config"
	"github.com/ad/domru/handlers"
)

func main() {
	// Init Config
	config := config.InitConfig()

	// Init Handlers
	h := handlers.NewHandlers(config)

	switch {
	case config.Token != "" || config.RefreshToken != "":
		if config.RefreshToken != "" {
			data, err := h.Refresh(&config.RefreshToken)
			if err != nil {
				log.Println("refresh token, error:", err.Error())
			} else {
				config.Token = data
			}
		}
	case config.Login != "" && config.Password != "":
		data, err := h.Auth(&config.Login, &config.Password)
		if err != nil {
			log.Println("login error", err.Error())
		} else {
			config.Token = data
		}
	default:
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

	log.Println("start listening on", config.Addr, "with token", config.Token)
	err := http.ListenAndServe(config.Addr, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
