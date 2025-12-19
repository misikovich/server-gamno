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
	Host           EnvKey = "HOST"
	Port           EnvKey = "PORT"
	APIHost        EnvKey = "API_HOST"
	APIPort        EnvKey = "API_PORT"
	DevMode        EnvKey = "DEV"
	VideosIDFile   EnvKey = "VIDEO_IDS_FILENAME"
	UseTLS         EnvKey = "USE_TLS"
	TLSCertPath    EnvKey = "TLS_CERT_PATH"
	TLSKeyPath     EnvKey = "TLS_KEY_PATH"
	AllowedOrigins EnvKey = "ALLOWED_ORIGINS"
	AllowedMethods EnvKey = "ALLOWED_METHODS"
	DBPath         EnvKey = "DB_PATH"
	YTDataAPIv3Key EnvKey = "YT_DATA_API_V3_KEY"
)
