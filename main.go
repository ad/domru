package main

import (
	"embed"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ad/domru/config"
	"github.com/ad/domru/handlers"
	"github.com/ad/domru/multiconfig"
)

//go:embed templates/*
var templateFs embed.FS

func main() {
	log.Println("Starting Domofon application...")
	
	// Determine configuration directory
	configDir := "/share/domofon"
	if homeDir, err := os.UserHomeDir(); err == nil {
		configDir = filepath.Join(homeDir, ".domofon")
	}
	log.Printf("Using configuration directory: %s", configDir)

	// Initialize multi-account system
	log.Println("Initializing multi-account system...")
	
	// Ensure configuration directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Printf("WARNING: Failed to create configuration directory %s: %v", configDir, err)
		log.Println("This may cause issues with configuration persistence")
	}
	
	storage := multiconfig.NewAccountStorage(configDir)
	factory := &handlers.HandlerFactory{}
	aggregator := multiconfig.NewMultiAccountAggregator(storage, factory, templateFs)

	// Check for migration from legacy config
	legacyConfigPath := filepath.Join(configDir, "account.json")
	migration := multiconfig.NewMigrationService(storage, legacyConfigPath)
	
	if migration.NeedsMigration() {
		log.Println("Legacy configuration detected, starting migration process...")
		log.Printf("Legacy config path: %s", legacyConfigPath)
		log.Printf("Target config directory: %s", configDir)
		
		if err := migration.MigrateFromLegacy(); err != nil {
			log.Printf("ERROR: Migration failed: %v", err)
			log.Println("Migration process encountered an error, falling back to legacy mode...")
			log.Println("Your original configuration has been preserved and will continue to work")
			
			// Fallback to legacy mode
			addonConfig := config.InitConfig()
			if addonConfig == nil {
				log.Fatal("FATAL: Failed to initialize legacy configuration and migration failed")
			}
			h := handlers.NewHandlers(addonConfig, templateFs)
			log.Println("Successfully initialized legacy mode as fallback")
			initializeLegacyMode(h, addonConfig)
			return
		}
		
		log.Println("Migration completed successfully!")
		log.Println("Your account has been migrated to the new multi-account format")
		log.Println("A backup of your original configuration has been created")
		
		// Clean up old backup files to prevent disk space issues
		if err := migration.CleanupBackups(); err != nil {
			log.Printf("WARNING: Failed to cleanup old migration backups: %v", err)
		}
	} else {
		log.Println("No migration needed - using existing multi-account configuration")
	}

	// Load accounts after migration (if any)
	if err := aggregator.LoadAccounts(); err != nil {
		log.Printf("ERROR: Failed to load accounts configuration: %v", err)
		
		// Check if we have legacy config to fall back to
		if _, err := os.Stat(legacyConfigPath); err == nil {
			log.Println("Multi-account configuration failed, attempting fallback to legacy mode...")
			addonConfig := config.InitConfig()
			if addonConfig == nil {
				log.Fatal("FATAL: Both multi-account and legacy configurations failed to load")
			}
			h := handlers.NewHandlers(addonConfig, templateFs)
			log.Println("Successfully fell back to legacy mode")
			initializeLegacyMode(h, addonConfig)
			return
		}
		
		log.Println("No existing accounts found, starting in multi-account mode with empty configuration")
		log.Println("You can add accounts through the web interface at /accounts")
	}

	// Initialize multi-account handlers
	h := handlers.NewMultiAccountHandlers(aggregator, templateFs)

	// In multi-account mode, token refresh is handled per account by the aggregator
	// No need for global token refresh logic

	// Register existing endpoints
	http.HandleFunc("/", h.HomeHandler)
	http.HandleFunc("/login", h.LoginHandler)
	http.HandleFunc("/login/address", h.LoginAddressHandler)
	http.HandleFunc("/sms", h.LoginSMSHandler)

	http.HandleFunc("/cameras", h.CamerasHandler)
	http.HandleFunc("/door", h.DoorHandler)
	http.HandleFunc("/events/last", h.LastEventHandler)
	http.HandleFunc("/events", h.EventsHandler)
	http.HandleFunc("/finances", h.FinancesHandler)
	http.HandleFunc("/operators", h.OperatorsHandler)
	http.HandleFunc("/places", h.PlacesHandler)
	http.HandleFunc("/snapshot", h.SnapshotHandler)
	http.HandleFunc("/stream", h.StreamHandler)

	// Register new multi-account management endpoints
	http.HandleFunc("/accounts", h.MultiAccountsHandler)
	http.HandleFunc("/accounts/", func(w http.ResponseWriter, r *http.Request) {
		// Route to AccountByIDHandler for paths like /accounts/{id} and /accounts/{id}/action
		if strings.HasPrefix(r.URL.Path, "/accounts/") && r.URL.Path != "/accounts/" {
			h.AccountByIDHandler(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	// Get port from primary account or use default
	port := 18000
	if primaryAccount, err := aggregator.GetPrimaryAccount(); err == nil {
		// Port is not stored in AccountConfig, use default
		log.Printf("Starting multi-account mode with primary account: %s", primaryAccount.Name)
	} else {
		log.Println("Starting multi-account mode with no primary account")
	}

	log.Printf("Starting server on port %d in multi-account mode", port)

	if err := http.ListenAndServe(":"+strconv.Itoa(port), nil); err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

// initializeLegacyMode initializes the server in legacy single-account mode
func initializeLegacyMode(h *handlers.Handler, addonConfig *config.Config) {
	// Handle token refresh for legacy mode
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

	// Register legacy endpoints (no multi-account management endpoints)
	http.HandleFunc("/", h.HomeHandler)
	http.HandleFunc("/login", h.LoginHandler)
	http.HandleFunc("/login/address", h.LoginAddressHandler)
	http.HandleFunc("/sms", h.LoginSMSHandler)

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
