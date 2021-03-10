package config

import (
	"flag"
	"os"
)

// Config ...
type Config struct {
	Addr         string
	Token        string
	RefreshToken string
	Login        string
	Password     string
	Operator     string
}

// InitConfig ...
func InitConfig() *Config {
	config := &Config{}

	flag.StringVar(&config.Addr, "addr", lookupEnvOrString("DOMRU_ADDR", ":8080"), "listen address")
	flag.StringVar(&config.Token, "token", lookupEnvOrString("DOMRU_TOKEN", config.Token), "dom.ru token")
	flag.StringVar(&config.RefreshToken, "refresh", lookupEnvOrString("DOMRU_REFRESH", config.RefreshToken), "dom.ru refresh token")
	flag.StringVar(&config.Login, "login", lookupEnvOrString("DOMRU_LOGIN", config.Login), "dom.ru login(or phone in format 71231234567)")
	flag.StringVar(&config.Password, "password", lookupEnvOrString("DOMRU_PASSWORD", config.Password), "dom.ru password")
	flag.StringVar(&config.Operator, "operator", lookupEnvOrString("DOMRU_OPERATOR", config.Operator), "dom.ru operator")
	flag.Parse()

	return config
}

func lookupEnvOrString(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}

	return defaultVal
}

// func lookupEnvOrInt(key string, defaultVal int) int {
// 	if val, ok := os.LookupEnv(key); ok {
// 		if x, err := strconv.Atoi(val); err == nil {
// 			return x
// 		}
// 	}
// 	return defaultVal
// }

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
