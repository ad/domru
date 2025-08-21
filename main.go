package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"strconv"

	"github.com/ad/domru/config"
	"github.com/ad/domru/handlers"
	"github.com/ad/domru/internal/api"
	myauth "github.com/ad/domru/internal/auth"
	"github.com/ad/domru/internal/constants"
)

// basicLogger implements api.Logger
type basicLogger struct{}

func (l basicLogger) Info(msg string, kv ...any)  { log.Println(append([]any{"INFO", msg}, kv...)...) }
func (l basicLogger) Error(msg string, kv ...any) { log.Println(append([]any{"ERROR", msg}, kv...)...) }

//go:embed templates/*
var templateFs embed.FS

func main() {
	// Init Config
	addonConfig := config.InitConfig()

	// Init auth + api
	tp := &myauth.ConfigProvider{Cfg: addonConfig}
	refresher := &myauth.TokenRefresherImpl{Cfg: addonConfig, Client: http.DefaultClient}
	autoClient := myauth.NewAutoClient(http.DefaultClient, tp, refresher, tp, constants.BaseUserAgentCore)
	apiWrapper := api.New(autoClient).WithLogger(basicLogger{}).WithDevice(addonConfig)

	// Init Handlers with API wrapper
	h := handlers.NewHandlers(addonConfig, templateFs, apiWrapper)

	switch {
	case addonConfig.Token != "" || addonConfig.RefreshToken != "":
		if addonConfig.RefreshToken != "" {
			access, refresh, err := apiWrapper.Refresh(context.Background(), addonConfig.RefreshToken, addonConfig.Operator, addonConfig.UUID)
			if err != nil {
				log.Println("refresh token error:", err)
			} else {
				addonConfig.Token = access
				addonConfig.RefreshToken = refresh
				_ = addonConfig.WriteConfig()
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

	log.Println("start listening on", addonConfig.Port, "with token", addonConfig.Token)

	if err := http.ListenAndServe(":"+strconv.Itoa(addonConfig.Port), nil); err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
