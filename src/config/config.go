package config

import (
	"log"
	"os"
	"reflect"

	"github.com/joho/godotenv"
)

type Config struct {
	APP_ENV     string
	API_TOKEN   string
	SD_TOKEN    string
	LISTEN_PORT string
	REDIS_ADDR  string
	REDIS_PORT  string
	REDIS_DBNO  string
	TTL_SECONDS string
}

var config *Config

func GetConfig() Config {
	return *config
}

func loadEnvFile() {
	log.Println("Loading .env file.")
	err := godotenv.Load(".env")
	if err != nil {
		panic("Error loading .env file.")
	}
}

func init() {
	config = &Config{}
	_, found := os.LookupEnv("APP_ENV")
	if !found {
		loadEnvFile()
	}
	_, found = os.LookupEnv("LISTEN_PORT")
	if !found {
		os.Setenv("LISTEN_PORT", "80")
	}
	_, found = os.LookupEnv("SD_TOKEN")
	if !found {
		os.Setenv("SD_TOKEN", "")
	}
	refl := reflect.ValueOf(config).Elem()
	numFields := refl.NumField()
	for i := 0; i < numFields; i++ {
		envName := refl.Type().Field(i).Name
		envVal, foud := os.LookupEnv(envName)
		if !foud {
			panic("Environment [" + envName + "] not found.")
		}
		refl.Field(i).SetString(envVal)
	}
}
