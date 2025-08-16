package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
)

var ConfigFileName = "/share/domofon/account.json"

// Config ...
type Config struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh"`
	Login        int    `json:"login"`
	Operator     int    `json:"operator"`
	Port         int    `json:"port"`
	UUID         string `json:"uuid"`
}

// InitConfig ...
func InitConfig() *Config {
	config := &Config{Port: 18000}

	fileIsExists := false

	if err := ensureDir(ConfigFileName); err != nil {
		log.Println("directory for config not writable", ConfigFileName, err)
		// return config
	}

	if _, err := os.Stat(ConfigFileName); err == nil {
		// log.Println("trying to load config from file", ConfigFileName)

		byteValue, _ := os.ReadFile(ConfigFileName)
		if err = json.Unmarshal(byteValue, &config); err != nil {
			log.Println("error on unmarshal config from file ", err)
		}

		fileIsExists = true
	} else if os.IsNotExist(err) {
		// log.Println("trying to create config file", ConfigFileName)

		if err := TouchConfig(); err != nil {
			log.Println("error on create config file ", err)
		} else {
			fileIsExists = true
		}
	}

	// Генерируем UUID если его нет
	if config.UUID == "" {
		config.UUID = uuid.New().String()
	}

	flag.StringVar(&config.Token, "token", lookupEnvOrString("DOMRU_TOKEN", config.Token), "dom.ru token")
	flag.StringVar(&config.RefreshToken, "refresh", lookupEnvOrString("DOMRU_REFRESH", config.RefreshToken), "dom.ru refresh token")
	flag.IntVar(&config.Login, "login", lookupEnvOrInt("DOMRU_LOGIN", config.Login), "dom.ru login(or phone in format 71231234567)")
	flag.IntVar(&config.Operator, "operator", lookupEnvOrInt("DOMRU_OPERATOR", config.Operator), "dom.ru operator")
	flag.IntVar(&config.Port, "port", lookupEnvOrInt("DOMRU_PORT", config.Port), "listen port")
	flag.StringVar(&config.UUID, "uuid", lookupEnvOrString("DOMRU_UUID", config.UUID), "dom.ru device uuid")

	flag.Parse()

	log.Printf("config: %+v\n", config)
	if fileIsExists {
		if err := config.WriteConfig(); err != nil {
			log.Println("error on save config to file ", err)
		}
	}

	return config
}

func lookupEnvOrString(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}

	return defaultVal
}

func TouchConfig() error {
	file, err := os.OpenFile(ConfigFileName, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	return file.Close()
}

// WriteConfig ...
func (config *Config) WriteConfig() error {
	file, _ := json.MarshalIndent(config, "", " ")

	return os.WriteFile(ConfigFileName, file, 0644)
}

func lookupEnvOrInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if x, err := strconv.Atoi(val); err == nil {
			return x
		}
	}
	return defaultVal
}

func ensureDir(fileName string) error {
	dirName := filepath.Dir(fileName)
	if _, serr := os.Stat(dirName); serr != nil {
		merr := os.MkdirAll(dirName, os.ModeSticky|os.ModePerm)
		if merr != nil {
			return merr
		}
	}
	return nil
}

// func lookupEnvOrBool(key string, defaultVal bool) bool {
// 	if val, ok := os.LookupEnv(key); ok {
// 		if val == "true" {
// 			return true
// 		}
// 		if val == "false" {
// 			return false
// 		}
// 	}
// 	return defaultVal
// }
