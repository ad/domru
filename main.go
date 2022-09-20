package main

import (
	"embed"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ad/domru/config"
	"github.com/ad/domru/handlers"

	"github.com/gorilla/websocket"
)

//go:embed templates/*
var templateFs embed.FS

func main() {
	// Init Config
	addonConfig := config.InitConfig()

	// Init Handlers
	h := handlers.NewHandlers(addonConfig, templateFs)

	switch {
	case addonConfig.Token != "" || addonConfig.RefreshToken != "":
		if addonConfig.RefreshToken != "" {
			access, refresh, err := h.Refresh(&addonConfig.RefreshToken)
			if err != nil {
				log.Println("refresh token, error:", err.Error())
			} else {
				addonConfig.Token = access
				addonConfig.RefreshToken = refresh

				if err = addonConfig.WriteConfig(); err != nil {
					log.Println("error on write config file ", err)
				}
			}
		}
	default:
		log.Println("auth/refresh token or login and password must be provided")
	}

	http.HandleFunc("/", h.HomeHandler)
	http.HandleFunc("/login", h.LoginHandler)
	http.HandleFunc("/login/address", h.LoginAddressHandler)
	http.HandleFunc("/sms", h.LoginSMSHandler)
	// http.HandleFunc("/network", h.HANetworkHandler)

	http.HandleFunc("/cameras", h.CamerasHandler)
	http.HandleFunc("/door", h.DoorHandler)
	http.HandleFunc("/events/last", h.LastEventHandler)
	http.HandleFunc("/events", h.EventsHandler)
	http.HandleFunc("/finances", h.FinancesHandler)
	http.HandleFunc("/operators", h.OperatorsHandler)
	http.HandleFunc("/places", h.PlacesHandler)
	http.HandleFunc("/snapshot", h.SnapshotHandler)
	http.HandleFunc("/stream", h.StreamHandler)

	if addonConfig.UseWebsocket {
		u := url.URL{Scheme: "wss", Host: "api-mh.ertelecom.ru", Path: "/events"}
		log.Printf("connecting to %s", u.String())

		header := make(http.Header)
		header.Set("Authorization", "Bearer "+addonConfig.Token)
		header.Set("Operator", strconv.Itoa(addonConfig.Operator))
		header.Set("User-Agent", handlers.CLIENT_USERAGENT)

		connectWS := func(u url.URL, header http.Header) (c *websocket.Conn) {
			c, _, err := websocket.DefaultDialer.Dial(u.String(), header)
			if err != nil {
				log.Fatal("dial:", err)
			}

			return c
		}

		c := connectWS(u, header)
		defer c.Close()

		done := make(chan struct{})

		go func() {
			defer close(done)
			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					log.Println("read:", err, "reconnect")
					c = connectWS(u, header)
					continue
				}
				log.Printf("recv: %s", message)
			}
		}()
	}

	log.Println("start listening on", addonConfig.Port, "with token", addonConfig.Token)

	if err := http.ListenAndServe(":"+strconv.Itoa(addonConfig.Port), nil); err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
