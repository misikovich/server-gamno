package env

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type EnvKey string

func (key EnvKey) Get() string {
	return os.Getenv(string(key))
}

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

const (
	Host    EnvKey = "HOST"
	Port    EnvKey = "PORT"
	APIHost EnvKey = "API_HOST"
	APIPort EnvKey = "API_PORT"
	DevMode EnvKey = "DEV"
)
